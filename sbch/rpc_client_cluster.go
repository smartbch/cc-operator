package sbch

import (
	"bytes"
	"fmt"
	"sync"
)

var _ BasicRpcClient = (*clusterRpcClient)(nil)

// cluster JSON-RPC basicClient
type clusterRpcClient struct {
	clients []BasicRpcClient
}

func (cluster *clusterRpcClient) RpcURL() string {
	return "clusterRpcClient"
}

func (cluster *clusterRpcClient) SendPost(reqStr string) ([]byte, error) {
	nClients := len(cluster.clients)
	resps := make([][]byte, nClients)
	errors := make([]error, nClients)

	// send post to nodes concurrently
	wg := sync.WaitGroup{}
	wg.Add(nClients)
	for i, client := range cluster.clients {
		go func(idx int, client BasicRpcClient) {
			resps[idx], errors[idx] = client.SendPost(reqStr)
			wg.Done()
		}(i, client)
	}
	wg.Wait()

	// fail if one of node return error
	for idx, err := range errors {
		if err != nil {
			return nil, fmt.Errorf("failed to call %s: %s",
				cluster.clients[idx].RpcURL(), err.Error())
		}
	}

	// all responses should be same
	resp0 := resps[0]
	for idx, resp := range resps {
		if idx > 0 && bytes.Equal(resp0, resp) {
			return nil, fmt.Errorf("response not match between: %s, %s",
				cluster.clients[0].RpcURL(), cluster.clients[idx].RpcURL())
		}
	}

	return resp0, nil
}

func NewClusterRpcClientOfNodes(sbchdNodes []NodeInfo,
	minNodeCount int) (RpcClient, []NodeInfo, error) {

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

	clusterClient := newClusterRpcClient(rpcClients)
	return clusterClient, validNodes, nil
}

func newClusterRpcClient(clients []BasicRpcClient) RpcClient {
	return &sbchRpcClient{
		nodesGovAddr: nodesGovContractAddr,
		basicClient: &clusterRpcClient{
			clients: clients,
		},
	}
}
