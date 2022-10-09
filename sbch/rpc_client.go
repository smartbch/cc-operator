package sbch

import "fmt"

func InitRpcClients(bootstrapRpcUrl string, minNodeCount int) (*RpcClientsInfo, error) {
	fmt.Println("InitRpcClients, bootstrapRpcUrl:", bootstrapRpcUrl, "minNodeCount:", minNodeCount)
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
