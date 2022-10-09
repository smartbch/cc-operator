package main

import (
	"flag"

	"github.com/smartbch/ccoperator/operator"
)

var (
	listenAddr      = "0.0.0.0:8080"
	bootstrapRpcURL = "http://localhost:8545"
)

func main() {
	flag.StringVar(&listenAddr, "listenAddr", "0.0.0.0:8080", "listen addr, ip:port")
	flag.StringVar(&bootstrapRpcURL, "bootstrapRpcURL", "http://localhost:8545", "bootstrap smartBCH RPC URL")
	flag.Parse()

	operator.Start(listenAddr, bootstrapRpcURL)
}
