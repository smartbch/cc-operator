package main

import (
	"flag"

	"github.com/smartbch/ccoperator/proxy"
)

var (
	operatorUrl = "https://localhost:8801"
	proxyName   = "sbch-operator"
	listenAddr  = "0.0.0.0:8901"
	certFile    = ""
	keyFile     = ""
)

func main() {
	flag.StringVar(&operatorUrl, "operatorUrl", "https://localhost:8801", "base URL of cc-operator")
	flag.StringVar(&listenAddr, "listenAddr", "0.0.0.0:8901", "listen addr, ip:port")
	flag.StringVar(&proxyName, "proxyName", "sbch-operator-proxy", "server name to generate TLS certificate")
	flag.StringVar(&certFile, "certFile", "", "cert file")
	flag.StringVar(&keyFile, "keyFile", "", "key file")
	flag.Parse()

	if certFile != "" && keyFile != "" {
		proxy.StartProxyServerWithCert(operatorUrl, listenAddr, certFile, keyFile)
	} else {
		proxy.StartProxyServerWithName(operatorUrl, listenAddr, proxyName)
	}
}
