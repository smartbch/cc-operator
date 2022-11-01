package sbch

import (
	"fmt"
	"reflect"
	"sync"
)

var _ RpcClient = (*ClusterClient)(nil)

type ClusterClient struct {
	clients []RpcClient
}

// TODO: check cert hash of nodes
func NewClusterRpcClientOfNodes(nodesGovAddr string, nodes []NodeInfo,
	minNodeCount int, skipCert bool) (RpcClient, []NodeInfo, error) {

	rpcUrls := make([]string, len(nodes))
	for idx, node := range nodes {
		rpcUrls[idx] = node.RpcUrl
	}
	return NewClusterClient(nodesGovAddr, rpcUrls), nodes, nil
}

func NewClusterClient(nodesGovAddr string, rpcUrls []string) RpcClient {
	clients := make([]RpcClient, len(rpcUrls))
	for idx, url := range rpcUrls {
		clients[idx] = NewSimpleRpcClient(nodesGovAddr, url)
	}
	return ClusterClient{clients: clients}
}

func (cluster ClusterClient) RpcURL() string {
	return "clusterRpcClient"
}

func (cluster ClusterClient) GetSbchdNodes() ([]NodeInfo, error) {
	result, err := cluster.GetFromAllNodes("GetSbchdNodes")
	if err != nil {
		return nil, err
	}
	return result.([]NodeInfo), err
}

func (cluster ClusterClient) GetRedeemingUtxoSigHashes() ([]string, error) {
	result, err := cluster.GetFromAllNodes("GetRedeemingUtxoSigHashes")
	if err != nil {
		return nil, err
	}
	return result.([]string), err
}

func (cluster ClusterClient) GetToBeConvertedUtxoSigHashes() ([]string, error) {
	result, err := cluster.GetFromAllNodes("GetToBeConvertedUtxoSigHashes")
	if err != nil {
		return nil, err
	}
	return result.([]string), err
}

func (cluster ClusterClient) GetFromAllNodes(methodName string) (any, error) {
	nClients := len(cluster.clients)
	resps := make([]any, nClients)
	errors := make([]error, nClients)

	// send post to nodes concurrently
	wg := sync.WaitGroup{}
	wg.Add(nClients)
	for i, client := range cluster.clients {
		go func(idx int, client RpcClient) {
			resps[idx], errors[idx] = getFromOneNode(client, methodName)
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
		if idx > 0 && !reflect.DeepEqual(resp0, resp) {
			return nil, fmt.Errorf("response not match between: %s, %s",
				cluster.clients[0].RpcURL(), cluster.clients[idx].RpcURL())
		}
	}

	//fmt.Println("resp:", string(resp0))
	return resp0, nil
}

func getFromOneNode(client RpcClient, methodName string) (any, error) {
	switch methodName {
	case "GetSbchdNodes":
		return client.GetSbchdNodes()
	case "GetRedeemingUtxoSigHashes":
		return client.GetRedeemingUtxoSigHashes()
	case "GetToBeConvertedUtxoSigHashes":
		return client.GetToBeConvertedUtxoSigHashes()
	default:
		panic("unknown method") // unreachable
	}
}
