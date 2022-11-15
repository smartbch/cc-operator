package main

import (
	"flag"

	gethcmn "github.com/ethereum/go-ethereum/common"
	"github.com/smartbch/cc-operator/proxy"
)

var (
	operatorUrl  = "https://localhost:8801"
	operatorName = "cc-operator"
	signer       = ""
	uniqueID     = ""
	proxyName    = "sbch-operator"
	listenAddr   = "0.0.0.0:8901"
	certFile     = ""
	keyFile      = ""
)

func main() {
	flag.StringVar(&operatorUrl, "operatorUrl", "https://localhost:8801", "base URL of cc-operator")
	flag.StringVar(&operatorName, "operatorName", "cc-operator", "operator name to verify TLS certificate")
	flag.StringVar(&signer, "signer", "", "signer of cc-operator")
	flag.StringVar(&uniqueID, "uniqueID", "", "uniqueID of cc-operator")
	flag.StringVar(&listenAddr, "listenAddr", "0.0.0.0:8901", "listen addr, ip:port")
	flag.StringVar(&proxyName, "proxyName", "sbch-operator-proxy", "server name to generate TLS certificate")
	flag.StringVar(&certFile, "certFile", "", "cert file")
	flag.StringVar(&keyFile, "keyFile", "", "key file")
	flag.Parse()

	if certFile != "" && keyFile != "" {
		proxy.StartProxyServerWithCert(operatorUrl, operatorName,
			gethcmn.FromHex(signer), gethcmn.FromHex(uniqueID),
			listenAddr, certFile, keyFile)
	} else {
		proxy.StartProxyServerWithName(operatorUrl, operatorName,
			gethcmn.FromHex(signer), gethcmn.FromHex(uniqueID),
			listenAddr, proxyName)
	}
}
