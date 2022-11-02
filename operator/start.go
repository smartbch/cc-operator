package operator

func Start(serverName, listenAddr, bootstrapRpcURL, nodesGovAddr, monitorAddrList string) {
	loadOrGenKey()
	initRpcClient(nodesGovAddr, bootstrapRpcURL)
	go getAndSignSigHashes()
	go watchSbchdNodes()
	go startHttpsServer(serverName, listenAddr, monitorAddrList)
	select {}
}
