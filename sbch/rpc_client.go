package sbch

import (
	"fmt"
	"time"
)

func InitRpcClients(nodesGovAddr, bootstrapRpcUrl string,
	minNodeCount int, skipPbkCheck bool,
	clientReqTimeout time.Duration,
) (*RpcClientsInfo, error) {

	fmt.Println("InitRpcClients, nodesGovAddr:", nodesGovAddr,
		"bootstrapRpcUrl:", bootstrapRpcUrl, "minNodeCount:", minNodeCount)

	bootstrapClient := NewSimpleRpcClient(nodesGovAddr, bootstrapRpcUrl, 0)
	allNodes, err := bootstrapClient.GetSbchdNodes()
	if err != nil {
		return nil, err
	}

	clusterClient, validNodes, err := NewClusterRpcClientOfNodes(
		nodesGovAddr, allNodes, minNodeCount, skipPbkCheck, clientReqTimeout)
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
