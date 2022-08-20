package sbch

import (
	"fmt"
)

func InitRpcClients(bootstrapRpcUrl string,
	minNodeCount int, minSameRespCount int) (*RpcClientsInfo, error) {

	bootstrapClient := wrapSimpleRpcClient(bootstrapRpcUrl)
	sbchdNodes, err := bootstrapClient.GetSbchdNodes()
	if err != nil {
		return nil, err
	}

	rpcClients, okNodes, err := CheckNodesAndCreateRpcClient(sbchdNodes, minNodeCount, minSameRespCount)
	return &RpcClientsInfo{
		BootstrapRpcClient: bootstrapClient,
		RpcClients:         rpcClients,
		AllNodes:           sbchdNodes,
		UsedNodes:          okNodes,
	}, nil
}

func CheckNodesAndCreateRpcClient(sbchdNodes []NodeInfo,
	minNodeCount int, minSameRespCount int) (RpcClient, []NodeInfo, error) {

	if len(sbchdNodes) < minNodeCount {
		return nil, nil, fmt.Errorf("not enough sbchd nodes: %d < %d",
			len(sbchdNodes), minNodeCount)
	}

	okSbchdNodes := make([]NodeInfo, 0, len(sbchdNodes))
	rpcClients := make([]BasicRpcClient, 0, len(sbchdNodes))
	for _, node := range sbchdNodes {
		client, err := NewRpcClientOfNode(node)
		if err != nil {
			fmt.Println("failed to create rpc client, node:", node.ID, "error:", err.Error())
		} else {
			okSbchdNodes = append(okSbchdNodes, node)
			rpcClients = append(rpcClients, client)
		}
	}
	if len(rpcClients) < minNodeCount {
		return nil, nil, fmt.Errorf("not enough checked nodes: %d < %d",
			len(sbchdNodes), minNodeCount)
	}
	if minSameRespCount <= len(rpcClients)/2 {
		return nil, nil, fmt.Errorf("minSameRespCount is not greater than half of clients: %d <= %d/2",
			minSameRespCount, len(rpcClients))
	}

	cli := wrapAggregatedRpcClient(rpcClients, minSameRespCount)
	return cli, okSbchdNodes, nil
}
