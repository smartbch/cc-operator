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
	gethcmn "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/smartbch/ccoperator/utils"
)

var certBytes []byte
var suspended atomic.Value
var monitorAddresses []gethcmn.Address

var (
	errTsTooOld   = errors.New("ts too old")
	errTsTooNew   = errors.New("ts too new")
	errNotMonitor = errors.New("not monitor")
)

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
	err := server.ListenAndServe()
	fmt.Println(err)
}

func createHttpHandlers() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/cert", handleCert)
	mux.HandleFunc("/pubkey", handlePubKey)
	mux.HandleFunc("/report", handleReport)
	mux.HandleFunc("/jwt", handleJwtToken)
	mux.HandleFunc("/sig", handleSig)
	mux.HandleFunc("/nodes", handleCurrNodes)
	mux.HandleFunc("/newNodes", handleNewNodes)
	mux.HandleFunc("/suspend", handleSuspend)
	mux.HandleFunc("/status", handleStatus)
	return mux
}

func handleCert(w http.ResponseWriter, r *http.Request) {
	resp := Resp{
		Success: true,
		Result:  "0x" + hex.EncodeToString(certBytes),
	}
	resp.WriteTo(w)
}

func handlePubKey(w http.ResponseWriter, r *http.Request) {
	resp := Resp{
		Success: true,
		Result:  "0x" + hex.EncodeToString(pubKeyBytes),
	}
	resp.WriteTo(w)
}

func handleReport(w http.ResponseWriter, r *http.Request) {
	var resp Resp

	if integrationTestMode {
		resp.Success = false
		resp.Error = "integrationTestMode!"
	} else {
		hash := sha256.Sum256(pubKeyBytes)
		report, err := enclave.GetRemoteReport(hash[:])
		if err != nil {
			resp.Success = false
			resp.Error = err.Error()
		} else {
			resp.Success = true
			resp.Result = "0x" + hex.EncodeToString(report)
		}
	}

	resp.WriteTo(w)
}

func handleJwtToken(w http.ResponseWriter, r *http.Request) {
	var resp Resp

	if integrationTestMode {
		resp.Success = false
		resp.Error = "integrationTestMode!"
	} else {
		token, err := enclave.CreateAzureAttestationToken(pubKeyBytes, attestationProviderURL)
		if err != nil {
			resp.Success = false
			resp.Error = err.Error()
		} else {
			resp.Success = true
			resp.Result = json.RawMessage(token)
		}
	}

	resp.WriteTo(w)
}

func handleSig(w http.ResponseWriter, r *http.Request) {
	if suspended.Load() != nil {
		NewErrResp("suspended").WriteTo(w)
		return
	}

	fmt.Println("handleSig:", r.URL.String())
	hash := getQueryParam(r, "hash")
	if len(hash) == 0 {
		NewErrResp("missing query parameter: hash").WriteTo(w)
		return
	}

	sig := getSig(hash)
	if sig == nil {
		NewErrResp("no signature found").WriteTo(w)
		return
	}

	NewOkResp("0x" + hex.EncodeToString(sig)).WriteTo(w)
}

func handleCurrNodes(w http.ResponseWriter, r *http.Request) {
	resp := Resp{
		Success: true,
		Result:  getCurrNodes(),
	}
	resp.WriteTo(w)
}
func handleNewNodes(w http.ResponseWriter, r *http.Request) {
	resp := Resp{
		Success: true,
		Result:  getNewNodes(),
	}
	resp.WriteTo(w)
}

func handleStatus(w http.ResponseWriter, r *http.Request) {
	resp := Resp{
		Success: true,
		Result:  "ok",
	}
	if suspended.Load() != nil {
		resp.Result = "suspended"
	}
	resp.WriteTo(w)
}

// only monitors can call this
func handleSuspend(w http.ResponseWriter, r *http.Request) {
	sig := getQueryParam(r, "sig")
	ts := getQueryParam(r, "ts")

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
	if now-ts > 60 {
		return errTsTooOld
	}
	if ts-now > 60 {
		return errTsTooNew
	}
	return nil
}

func checkSig(ts, sig string) error {
	hash := sha256.Sum256([]byte(ts))
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

func getQueryParam(r *http.Request, name string) string {
	params := r.URL.Query()[name]
	if len(params) == 0 {
		return ""
	}
	return params[0]
}
