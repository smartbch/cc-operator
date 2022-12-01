package main

import (
	"flag"
	"github.com/smartbch/cc-operator/operator"
)

var (
	helpFlag        = false
	serverName      = "cc-operator"
	listenAddr      = "0.0.0.0:8801"
	bootstrapRpcURL = "http://localhost:8545"
	monitorAddrList = ""
	signerKeyWIF    = ""

	// TODO: change this to constant in production mode
	nodesGovAddr = "0x0000000000000000000000000000000000001234"

	fixedBootstrapRpcURls = []string{"http://sbch1:8545", "http://sbch2:8545"}
)

func main() {
	flag.BoolVar(&helpFlag, "help", false, "show help")
	flag.StringVar(&serverName, "serverName", serverName, "server name to generate TLS certificate")
	flag.StringVar(&listenAddr, "listenAddr", listenAddr, "listen addr, ip:port")
	flag.StringVar(&bootstrapRpcURL, "bootstrapRpcURL", bootstrapRpcURL, "bootstrap smartBCH RPC URL")
	flag.StringVar(&nodesGovAddr, "nodesGovAddr", nodesGovAddr, "address of NodesGov contract")
	flag.StringVar(&monitorAddrList, "monitorAddrList", monitorAddrList, "comma separated monitor addresses")
	flag.StringVar(&signerKeyWIF, "signerKeyWIF", signerKeyWIF, "signer key WIF, for integration test only")
	flag.Parse()

	if helpFlag {
		flag.Usage()
		return
	}
	var bootstrapRpcURLs []string
	repeatBootstrapUrl := false
	bootstrapRpcURLs = append(bootstrapRpcURLs, fixedBootstrapRpcURls...)
	for _, url := range fixedBootstrapRpcURls {
		if url == bootstrapRpcURL {
			repeatBootstrapUrl = true
		}
	}
	if !repeatBootstrapUrl {
		bootstrapRpcURLs = append(bootstrapRpcURLs, bootstrapRpcURL)
	}
	operator.Start(serverName, listenAddr, nodesGovAddr, monitorAddrList, signerKeyWIF, bootstrapRpcURLs)
}
