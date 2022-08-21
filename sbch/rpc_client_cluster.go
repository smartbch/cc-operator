package sbch

import (
	"bytes"
	"fmt"
)

var _ BasicRpcClient = (*clusterRpcClient)(nil)

// cluster JSON-RPC basicClient
type clusterRpcClient struct {
	clients          []BasicRpcClient
	minSameRespCount int
}

func (cluster *clusterRpcClient) RpcURL() string {
	return "clusterRpcClient"
}

func (cluster *clusterRpcClient) SendPost(reqStr string) ([]byte, error) {
	resps := cluster.sendPosts(reqStr)
	sameResp, sameRespCount := findSameResps(resps)
	if sameRespCount < cluster.minSameRespCount {
		return nil, fmt.Errorf("not enough same resp: %d < %d",
			sameRespCount, cluster.minSameRespCount)
	}
	return sameResp, nil
}

func (cluster *clusterRpcClient) sendPosts(reqStr string) [][]byte {
	var resps [][]byte

	// TODO: call SendPost() parallel
	for _, subClient := range cluster.clients {
		resp, err := subClient.SendPost(reqStr)
		if err != nil {
			fmt.Println("failed to send post! node:", subClient.RpcURL(), "error:", err.Error())
		} else {
			resps = append(resps, resp)
		}
	}

	return resps
}

func findSameResps(resps [][]byte) (sameResp []byte, sameRespCount int) {
	for _, resp1 := range resps {
		sameCount := 0
		for _, resp2 := range resps {
			if bytes.Equal(resp2, resp1) {
				sameCount++
			}
		}
		if sameCount > sameRespCount {
			sameRespCount = sameCount
			sameResp = resp1
		}
	}
	return
}

func NewClusterRpcClientOfNodes(sbchdNodes []NodeInfo,
	minNodeCount int, minSameRespCount int) (RpcClient, []NodeInfo, error) {

	if len(sbchdNodes) < minNodeCount {
		return nil, nil, fmt.Errorf("not enough sbchd nodes: %d < %d",
			len(sbchdNodes), minNodeCount)
	}

	validNodes := make([]NodeInfo, 0, len(sbchdNodes))
	rpcClients := make([]BasicRpcClient, 0, len(sbchdNodes))
	for _, node := range sbchdNodes {
		client, err := newBasicRpcClientOfNode(node)
		if err != nil {
			fmt.Println("failed to create rpc basicClient, node:", node.ID, "error:", err.Error())
		} else {
			validNodes = append(validNodes, node)
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

	clusterClient := newClusterRpcClient(rpcClients, minSameRespCount)
	return clusterClient, validNodes, nil
}

func newClusterRpcClient(clients []BasicRpcClient, minSameRespCount int) RpcClient {
	return &sbchRpcClient{
		basicClient: &clusterRpcClient{
			clients:          clients,
			minSameRespCount: minSameRespCount,
		},
	}
}
