package sbch

import (
	"fmt"
)

func InitRpcClients(nodesGovAddr, bootstrapRpcUrl string, minNodeCount int, skipPbkCheck bool) (*RpcClientsInfo, error) {
	fmt.Println("InitRpcClients, nodesGovAddr:", nodesGovAddr, "bootstrapRpcUrl:", bootstrapRpcUrl, "minNodeCount:", minNodeCount)

	bootstrapClient := NewSimpleRpcClient(nodesGovAddr, bootstrapRpcUrl)
	allNodes, err := bootstrapClient.GetSbchdNodes()
	if err != nil {
		return nil, err
	}

	clusterClient, validNodes, err := NewClusterRpcClientOfNodes(nodesGovAddr, allNodes, minNodeCount, skipPbkCheck)
	if err != nil {
		return nil, err
	}

	return &RpcClientsInfo{
		BootstrapRpcClient: bootstrapClient,
		ClusterRpcClient:   clusterClient,
		AllNodes:           allNodes,
		ValidNodes:         validNodes,
	}, nil
}
