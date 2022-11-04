package proxy

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"github.com/smartbch/ccoperator/utils"
)

func StartProxyServerWithCert(
	baseOperatorUrl, listenAddr string,
	certFile, keyFile string) {

	proxy := newProxy(baseOperatorUrl)
	server := http.Server{
		Addr:         listenAddr,
		Handler:      proxy,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 5 * time.Second,
	}
	fmt.Println("cc-operator proxy listening at:", listenAddr, "...")
	err := server.ListenAndServeTLS(certFile, keyFile)
	if err != nil {
		panic(err)
	}
}

func StartProxyServerWithName(baseOperatorUrl, listenAddr string,
	serverName string) {

	proxy := newProxy(baseOperatorUrl)
	_, _, tlsCfg := utils.CreateCertificate(serverName)

	server := http.Server{
		Addr:         listenAddr,
		Handler:      proxy,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 5 * time.Second,
		TLSConfig:    &tlsCfg,
	}
	fmt.Println("cc-operator proxy listening at:", listenAddr, "...")
	err := server.ListenAndServeTLS("", "")
	if err != nil {
		panic(err)
	}
}

func newProxy(targetHost string) *httputil.ReverseProxy {
	targetUrl, err := url.Parse(targetHost)
	if err != nil {
		panic(err)
	}

	return httputil.NewSingleHostReverseProxy(targetUrl)
}
