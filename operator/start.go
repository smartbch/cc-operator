package operator

func Start(serverName, listenAddr, bootstrapRpcURL, nodesGovAddr string) {
	loadOrGenKey()
	initRpcClient(nodesGovAddr, bootstrapRpcURL)
	go getAndSignSigHashes()
	go watchSbchdNodes()
	go startHttpsServer(serverName, listenAddr)
	select {}
}
