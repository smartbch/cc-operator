package operator

func Start(listenAddr, bootstrapRpcURL string) {
	loadOrGenKey()
	initRpcClient(bootstrapRpcURL)
	go getAndSignSigHashes()
	go watchSbchdNodes()
	go startHttpServer(listenAddr)
	select {}
}
