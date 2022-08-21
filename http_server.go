package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/edgelesssys/ego/enclave"
)

func startHttpServer(listenAddr string) {
	initHttpHandlers()

	server := http.Server{Addr: listenAddr, ReadTimeout: 3 * time.Second, WriteTimeout: 5 * time.Second}
	fmt.Println("listening ...")
	err := server.ListenAndServe()
	fmt.Println(err)
}

func initHttpHandlers() {
	http.HandleFunc("/pubkey", handlePubKey)
	http.HandleFunc("/report", handleReport)
	http.HandleFunc("/token", handleJwtToken)
	http.HandleFunc("/sig", handleSig)
	http.HandleFunc("/nodes", handleCurrNodes)
	http.HandleFunc("/newNodes", handleNewNodes)
}

func handlePubKey(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(hex.EncodeToString(pubKeyBytes)))
	return
}

func handleReport(w http.ResponseWriter, r *http.Request) {
	hash := sha256.Sum256(pubKeyBytes)
	report, err := enclave.GetRemoteReport(hash[:])
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}
	w.Write([]byte(hex.EncodeToString(report)))
}

func handleJwtToken(w http.ResponseWriter, r *http.Request) {
	token, err := enclave.CreateAzureAttestationToken(pubKeyBytes, attestationProviderURL)
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}
	w.Write([]byte(token))
}

func handleSig(w http.ResponseWriter, r *http.Request) {
	vals := r.URL.Query()["hash"]
	if len(vals) == 0 {
		return
	}

	sig := getSig(vals[0])
	w.Write(sig)
}

func handleCurrNodes(w http.ResponseWriter, r *http.Request) {
	nodes := getCurrNodes()
	bytes, err := json.Marshal(nodes)
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}
	w.Write(bytes)
}
func handleNewNodes(w http.ResponseWriter, r *http.Request) {
	nodes := getNewNodes()
	bytes, err := json.Marshal(nodes)
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}
	w.Write(bytes)
}
