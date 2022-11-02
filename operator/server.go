package operator

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/edgelesssys/ego/enclave"

	"github.com/smartbch/ccoperator/utils"
)

var certBytes []byte

func startHttpsServer(serverName, listenAddr string) {
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
	var resp Resp

	fmt.Println("handleSig:", r.URL.String())
	vals := r.URL.Query()["hash"]
	if len(vals) == 0 {
		resp.Success = false
		resp.Error = "missing query parameter: hash"
	} else {
		sig := getSig(vals[0])
		if sig == nil {
			resp.Success = false
			resp.Error = "no signature"
		} else {
			resp.Success = true
			resp.Result = "0x" + hex.EncodeToString(sig)
		}
	}

	fmt.Println(string(resp.ToJSON()))
	resp.WriteTo(w)
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
