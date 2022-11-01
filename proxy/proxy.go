package proxy

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/smartbch/ccoperator/operator"
)

var baseOperatorUrl = "http://localhost:8801"

var pubKey *operator.Resp

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
	if pubKey != nil {
		pubKey.WriteTo(w)
		return
	}

	resp := getFromOperator(baseOperatorUrl + "/pubkey")
	if resp.Success {
		pubKey = &resp
		pubKey.WriteTo(w)
		return
	}
	resp.WriteTo(w)
}

func handleReport(w http.ResponseWriter, r *http.Request) {
	getFromOperator(baseOperatorUrl + "/report").WriteTo(w)
}

func handleJwtToken(w http.ResponseWriter, r *http.Request) {
	getFromOperator(baseOperatorUrl + "/jwt").WriteTo(w)
}

func handleSig(w http.ResponseWriter, r *http.Request) {
	hash := ""
	if vals := r.URL.Query()["hash"]; len(vals) > 0 {
		hash = vals[0]
	}
	getFromOperator(baseOperatorUrl + "/sig?hash=" + hash).WriteTo(w)
}

func handleCurrNodes(w http.ResponseWriter, r *http.Request) {
	getFromOperator(baseOperatorUrl + "/nodes").WriteTo(w)
}

func handleNewNodes(w http.ResponseWriter, r *http.Request) {
	getFromOperator(baseOperatorUrl + "/newNodes").WriteTo(w)
}

func getFromOperator(path string) operator.Resp {
	resp, err := http.Get(baseOperatorUrl + path)
	if err != nil {
		return operator.NewErrResp(err.Error())
	}

	defer resp.Body.Close()
	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return operator.NewErrResp(err.Error())
	}

	return operator.UnmarshalResp(bytes)
}
