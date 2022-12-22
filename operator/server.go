package operator

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/edgelesssys/ego/enclave"
	gethacc "github.com/ethereum/go-ethereum/accounts"
	gethcmn "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/smartbch/cc-operator/utils"
)

const (
	integrationTestMode = true // set this to false in production mode

	suspendTsDiffMaxSeconds = 60
	attestationProviderURL  = "https://shareduks.uks.attest.azure.net"
)

var (
	certBytes        []byte
	suspended        atomic.Value
	monitorAddresses []gethcmn.Address
)

var (
	errTsTooOld   = errors.New("ts too old")
	errTsTooNew   = errors.New("ts too new")
	errNotMonitor = errors.New("not monitor")
)

func Start(serverName, listenAddr, nodesGovAddr, monitorAddrList, signerKeyWIF string, bootstrapRpcURLs []string, privateUrls []string) {
	loadOrGenKey(signerKeyWIF)
	initRpcClients(nodesGovAddr, bootstrapRpcURLs, privateUrls)
	go getAndSignSigHashes()
	go watchSbchdNodes(privateUrls)
	go startHttpsServer(serverName, listenAddr, monitorAddrList)
	select {}
}

func loadOrGenKey(signerKeyWIF string) {
	if integrationTestMode {
		if signerKeyWIF != "" {
			loadKeyFromWIF(signerKeyWIF)
		} else {
			loadOrGenKeyNonEnclave()
		}
	} else {
		loadOrGenKeyInEnclave()
	}
}

func startHttpsServer(serverName, listenAddr, monitorAddrList string) {
	for _, addr := range strings.Split(monitorAddrList, ",") {
		monitorAddresses = append(monitorAddresses, gethcmn.HexToAddress(addr))
	}

	// Create a TLS config with a self-signed certificate and an embedded report.
	cert, _, tlsCfg := utils.CreateCertificate(serverName)
	certBytes = cert

	mux := createHttpHandlers()
	server := http.Server{
		Addr:         listenAddr,
		Handler:      mux,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 5 * time.Second,
		TLSConfig:    &tlsCfg,
	}
	fmt.Println("listening at:", listenAddr, "...")
	err := server.ListenAndServeTLS("", "")
	fmt.Println(err)
}

func createHttpHandlers() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/cert", handleCert)
	mux.HandleFunc("/cert-report", handleCertReport)
	mux.HandleFunc("/pubkey", handlePubKey)
	mux.HandleFunc("/pubkey-report", handlePubkeyReport)
	mux.HandleFunc("/pubkey-jwt", handlePubkeyJwt)
	mux.HandleFunc("/sig", handleSig)
	mux.HandleFunc("/info", handleOpInfo)
	mux.HandleFunc("/suspend", handleSuspend) // only monitor
	mux.HandleFunc("/redeeming-utxos-for-operators", handleGetRedeemingUtxosForOperators)
	mux.HandleFunc("/redeeming-utxos-for-monitors", handleGetRedeemingUtxosForMonitors)
	mux.HandleFunc("/to-be-converted-utxos-for-operators", handleGetToBeConvertedUtxosForOperators)
	mux.HandleFunc("/to-be-converted-utxos-for-monitors", handleGetToBeConvertedUtxosForMonitors)
	return mux
}

func handleCert(w http.ResponseWriter, r *http.Request) {
	if utils.GetQueryParam(r, "raw") != "" {
		w.Write(certBytes)
		return
	}
	NewOkResp("0x" + hex.EncodeToString(certBytes)).WriteTo(w)
}

func handleCertReport(w http.ResponseWriter, r *http.Request) {
	if integrationTestMode {
		NewErrResp("integration test mode").WriteTo(w)
		return
	}

	certHash := sha256.Sum256(certBytes)
	report, err := enclave.GetRemoteReport(certHash[:])
	if err != nil {
		NewErrResp(err.Error()).WriteTo(w)
		return
	}

	if utils.GetQueryParam(r, "raw") != "" {
		w.Write(report)
		return
	}
	NewOkResp("0x" + hex.EncodeToString(report)).WriteTo(w)
}

func handlePubKey(w http.ResponseWriter, r *http.Request) {
	NewOkResp("0x" + hex.EncodeToString(pubKeyBytes)).WriteTo(w)
}

func handlePubkeyReport(w http.ResponseWriter, r *http.Request) {
	if integrationTestMode {
		NewErrResp("integration test mode").WriteTo(w)
		return
	}

	pbkHash := sha256.Sum256(pubKeyBytes)
	report, err := enclave.GetRemoteReport(pbkHash[:])
	if err != nil {
		NewErrResp(err.Error()).WriteTo(w)
		return
	}

	NewOkResp("0x" + hex.EncodeToString(report)).WriteTo(w)
}

func handlePubkeyJwt(w http.ResponseWriter, r *http.Request) {
	if integrationTestMode {
		NewErrResp("integration test mode").WriteTo(w)
		return
	}

	token, err := enclave.CreateAzureAttestationToken(pubKeyBytes, attestationProviderURL)
	if err != nil {
		NewErrResp(err.Error()).WriteTo(w)
		return
	}

	NewOkResp(json.RawMessage(token)).WriteTo(w)
}

func handleSig(w http.ResponseWriter, r *http.Request) {
	if suspended.Load() != nil {
		NewErrResp("suspended").WriteTo(w)
		return
	}

	fmt.Println("handleSig:", r.URL.String())
	hash := utils.GetQueryParam(r, "hash")
	if len(hash) == 0 {
		NewErrResp("missing query parameter: hash").WriteTo(w)
		return
	}

	sig, err := getSig(hash)
	if err != nil {
		NewErrResp("no signature found:" + err.Error()).WriteTo(w)
		return
	}

	NewOkResp("0x" + hex.EncodeToString(sig)).WriteTo(w)
}

func handleOpInfo(w http.ResponseWriter, r *http.Request) {
	opInfo := getNodesInfo()
	opInfo.Status = "ok"
	if suspended.Load() != nil {
		opInfo.Status = "suspended"
	}

	NewOkResp(opInfo).WriteTo(w)
}

// only monitors can call this
func handleSuspend(w http.ResponseWriter, r *http.Request) {
	sig := utils.GetQueryParam(r, "sig")
	ts := utils.GetQueryParam(r, "ts")

	if sig == "" {
		NewErrResp("missing query parameter: sig").WriteTo(w)
		return
	}
	if ts == "" {
		NewErrResp("missing query parameter: ts").WriteTo(w)
		return
	}

	if err := parseAndCheckTs(ts); err != nil {
		NewErrResp(err.Error()).WriteTo(w)
		return
	}
	if err := checkSig(ts, sig); err != nil {
		NewErrResp(err.Error()).WriteTo(w)
		return
	}

	suspended.Store(true)
	NewOkResp("ok").WriteTo(w)
}
func parseAndCheckTs(tsParam string) error {
	ts, err := strconv.ParseInt(tsParam, 10, 64)
	if err != nil {
		return err
	}

	now := time.Now().Unix()
	if now-ts > suspendTsDiffMaxSeconds {
		return errTsTooOld
	}
	if ts-now > suspendTsDiffMaxSeconds {
		return errTsTooNew
	}
	return nil
}
func checkSig(ts, sig string) error {
	pk := "0x" + hex.EncodeToString(pubKeyBytes)
	hash := gethacc.TextHash([]byte(pk + "," + ts))
	pbk, err := crypto.SigToPub(hash[:], gethcmn.FromHex(sig))
	if err != nil {
		return err
	}

	addr := crypto.PubkeyToAddress(*pbk)
	for _, monitor := range monitorAddresses {
		if addr == monitor {
			return nil
		}
	}

	return errNotMonitor
}

func handleGetRedeemingUtxosForOperators(w http.ResponseWriter, r *http.Request) {
	utxos, err := currClusterClient.GetRedeemingUtxosForOperators()
	NewResp(utxos, err).WriteTo(w)
}
func handleGetRedeemingUtxosForMonitors(w http.ResponseWriter, r *http.Request) {
	utxos, err := currClusterClient.GetRedeemingUtxosForMonitors()
	NewResp(utxos, err).WriteTo(w)
}
func handleGetToBeConvertedUtxosForOperators(w http.ResponseWriter, r *http.Request) {
	utxos, err := currClusterClient.GetToBeConvertedUtxosForOperators()
	NewResp(utxos, err).WriteTo(w)
}
func handleGetToBeConvertedUtxosForMonitors(w http.ResponseWriter, r *http.Request) {
	utxos, err := currClusterClient.GetToBeConvertedUtxosForMonitors()
	NewResp(utxos, err).WriteTo(w)
}
