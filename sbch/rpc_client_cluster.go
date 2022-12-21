package sbch

import (
	"crypto/sha256"
	"fmt"
	"reflect"
	"sync"
	"time"

	sbchrpctypes "github.com/smartbch/smartbch/rpc/types"
)

var _ RpcClient = (*ClusterClient)(nil)

type ClusterClient struct {
	clients  []RpcClient
	AllNodes []NodeInfo
}

func NewClusterRpcClient(nodesGovAddr string, nodeUrls []string,
	clientReqTimeout time.Duration) (*ClusterClient, error) {

	clients := make([]RpcClient, len(nodeUrls))
	for i, url := range nodeUrls {
		client, err := NewSimpleRpcClient(nodesGovAddr, url, clientReqTimeout)
		if err != nil {
			return nil, fmt.Errorf("dail %s failed: %w", url, err)
		}
		clients[i] = client
	}
	return &ClusterClient{clients: clients}, nil
}

func NewClusterRpcClientOfNodes(nodesGovAddr string, nodes []NodeInfo, skipPbkCheck bool,
	privateUrls []string, clientReqTimeout time.Duration) (*ClusterClient, error) {

	clients := make([]RpcClient, 0, len(nodes))
	for _, node := range nodes {
		client, err := NewSimpleRpcClient(nodesGovAddr, node.RpcUrl, clientReqTimeout)
		if err != nil {
			return nil, fmt.Errorf("dail %s failed: %w", node.RpcUrl, err)
		}
		pbk, err := client.GetRpcPubkey()
		if err != nil {
			return nil, fmt.Errorf("get pubkey from %s failed: %w", node.RpcUrl, err)
		}
		if !skipPbkCheck && sha256.Sum256(pbk) != node.PbkHash {
			return nil, fmt.Errorf("pubkey not match: %s", node.RpcUrl)
		}
		clients = append(clients, client)
	}
	for _, url := range privateUrls {
		client, err := NewSimpleRpcClient(nodesGovAddr, url, clientReqTimeout)
		if err != nil {
			return nil, fmt.Errorf("dail %s failed: %w", url, err)
		}
		clients = append(clients, client)
	}
	return &ClusterClient{
		clients:  clients,
		AllNodes: nodes,
	}, nil
}

func (cluster *ClusterClient) RpcURL() string {
	return "clusterRpcClient"
}

func (cluster *ClusterClient) GetRpcPubkey() ([]byte, error) {
	panic("not supported")
}

func (cluster *ClusterClient) GetSbchdNodes() ([]NodeInfo, error) {
	result, err := cluster.GetFromAllNodes("GetSbchdNodes")
	if err != nil {
		return nil, err
	}
	return result.([]NodeInfo), err
}

func (cluster *ClusterClient) GetRedeemingUtxosForOperators() ([]*sbchrpctypes.UtxoInfo, error) {
	result, err := cluster.GetFromAllNodes("GetRedeemingUtxosForOperators")
	if err != nil {
		return nil, err
	}
	return result.([]*sbchrpctypes.UtxoInfo), err
}
func (cluster *ClusterClient) GetRedeemingUtxosForMonitors() ([]*sbchrpctypes.UtxoInfo, error) {
	result, err := cluster.GetFromAllNodes("GetRedeemingUtxosForMonitors")
	if err != nil {
		return nil, err
	}
	return result.([]*sbchrpctypes.UtxoInfo), err
}
func (cluster *ClusterClient) GetToBeConvertedUtxosForOperators() ([]*sbchrpctypes.UtxoInfo, error) {
	result, err := cluster.GetFromAllNodes("GetToBeConvertedUtxosForOperators")
	if err != nil {
		return nil, err
	}
	return result.([]*sbchrpctypes.UtxoInfo), err
}
func (cluster *ClusterClient) GetToBeConvertedUtxosForMonitors() ([]*sbchrpctypes.UtxoInfo, error) {
	result, err := cluster.GetFromAllNodes("GetToBeConvertedUtxosForMonitors")
	if err != nil {
		return nil, err
	}
	return result.([]*sbchrpctypes.UtxoInfo), err
}

func (cluster *ClusterClient) GetFromAllNodes(methodName string) (any, error) {
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
	case "GetRedeemingUtxosForOperators":
		return client.GetRedeemingUtxosForOperators()
	case "GetRedeemingUtxosForMonitors":
		return client.GetRedeemingUtxosForMonitors()
	case "GetToBeConvertedUtxosForOperators":
		return client.GetToBeConvertedUtxosForOperators()
	case "GetToBeConvertedUtxosForMonitors":
		return client.GetToBeConvertedUtxosForMonitors()
	default:
		panic("unknown method") // unreachable
	}
}
