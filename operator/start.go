package operator

func Start(listenAddr, bootstrapRpcURL, nodesGovAddr string) {
	loadOrGenKey()
	initRpcClient(nodesGovAddr, bootstrapRpcURL)
	go getAndSignSigHashes()
	go watchSbchdNodes()
	go startHttpServer(listenAddr)
	select {}
}
