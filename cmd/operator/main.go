package main

import (
	"flag"

	"github.com/smartbch/cc-operator/operator"
)

var (
	serverName      = "sbch-operator"
	listenAddr      = "0.0.0.0:8801"
	bootstrapRpcURL = "http://localhost:8545"
	monitorAddrList = ""

	// TODO: change this to constant in production mode
	nodesGovAddr = "0x0000000000000000000000000000000000001234"
)

func main() {
	flag.StringVar(&serverName, "serverName", "cc-operator", "server name to generate TLS certificate")
	flag.StringVar(&listenAddr, "listenAddr", "0.0.0.0:8801", "listen addr, ip:port")
	flag.StringVar(&bootstrapRpcURL, "bootstrapRpcURL", "http://localhost:8545", "bootstrap smartBCH RPC URL")
	flag.StringVar(&nodesGovAddr, "nodesGovAddr", "0x0000000000000000000000000000000000001234", "address of NodesGov contract")
	flag.StringVar(&monitorAddrList, "monitorAddrList", "", "comma separated monitor addresses")
	flag.Parse()

	operator.Start(serverName, listenAddr, bootstrapRpcURL, nodesGovAddr, monitorAddrList)
}
