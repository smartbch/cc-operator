package main

// TODO
const (
	listenAddr             = "0.0.0.0:8080"
	attestationProviderURL = "https://shareduks.uks.attest.azure.net"
	bootstrapRpcURL        = "http://localhost:8545"
)

func main() {
	loadOrGenKey()
	initRpcClient()
	go getAndSignSigHashes()
	go watchSbchdNodes()
	go startHttpServer(listenAddr)
	select {}
}
