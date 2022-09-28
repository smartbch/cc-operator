package sbch

func InitRpcClients(bootstrapRpcUrl string, minNodeCount int) (*RpcClientsInfo, error) {
	bootstrapClient := NewSimpleRpcClient(bootstrapRpcUrl)
	allNodes, err := bootstrapClient.GetSbchdNodes()
	if err != nil {
		return nil, err
	}

	clusterClient, validNodes, err := NewClusterRpcClientOfNodes(allNodes, minNodeCount)
	return &RpcClientsInfo{
		BootstrapRpcClient: bootstrapClient,
		ClusterRpcClient:   clusterClient,
		AllNodes:           allNodes,
		ValidNodes:         validNodes,
	}, nil
}
