package main

import (
	"flag"

	"github.com/smartbch/ccoperator/operator"
)

var (
	listenAddr      = "0.0.0.0:8801"
	bootstrapRpcURL = "http://localhost:8545"

	// TODO: change this to constant in production mode
	nodesGovAddr = "0x0000000000000000000000000000000000001234"
)

func main() {
	flag.StringVar(&listenAddr, "listenAddr", "0.0.0.0:8080", "listen addr, ip:port")
	flag.StringVar(&bootstrapRpcURL, "bootstrapRpcURL", "http://localhost:8545", "bootstrap smartBCH RPC URL")
	flag.StringVar(&nodesGovAddr, "nodesGovAddr", "0x0000000000000000000000000000000000001234", "address of NodesGov contract")
	flag.Parse()

	operator.Start(listenAddr, bootstrapRpcURL, nodesGovAddr)
}
