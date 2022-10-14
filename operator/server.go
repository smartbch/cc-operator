package operator

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/edgelesssys/ego/enclave"
)

type Resp struct {
	Success bool        `json:"success"`
	Error   string      `json:"error,omitempty"`
	Result  interface{} `json:"result,omitempty"`
}

func (resp Resp) toJSON() []byte {
	bytes, _ := json.Marshal(resp)
	return bytes
}
func (resp Resp) writeTo(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "*")
	w.Header().Set("Access-Control-Allow-Headers", "origin, content-type, accept")

	bytes, _ := json.Marshal(resp)
	_, _ = w.Write(bytes)
}

func startHttpServer(listenAddr string) {
	mux := createHttpHandlers()
	server := http.Server{
		Addr:         listenAddr,
		Handler:      mux,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 5 * time.Second,
	}
	fmt.Println("listening at:", listenAddr, "...")
	err := server.ListenAndServe()
	fmt.Println(err)
}

func createHttpHandlers() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/pubkey", handlePubKey)
	mux.HandleFunc("/report", handleReport)
	mux.HandleFunc("/jwt", handleJwtToken)
	mux.HandleFunc("/sig", handleSig)
	mux.HandleFunc("/nodes", handleCurrNodes)
	mux.HandleFunc("/newNodes", handleNewNodes)
	return mux
}

func handlePubKey(w http.ResponseWriter, r *http.Request) {
	resp := Resp{
		Success: true,
		Result:  "0x" + hex.EncodeToString(pubKeyBytes),
	}
	resp.writeTo(w)
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

	resp.writeTo(w)
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

	resp.writeTo(w)
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

	fmt.Println(string(resp.toJSON()))
	resp.writeTo(w)
}

func handleCurrNodes(w http.ResponseWriter, r *http.Request) {
	resp := Resp{
		Success: true,
		Result:  getCurrNodes(),
	}
	resp.writeTo(w)
}
func handleNewNodes(w http.ResponseWriter, r *http.Request) {
	resp := Resp{
		Success: true,
		Result:  getNewNodes(),
	}
	resp.writeTo(w)
}
