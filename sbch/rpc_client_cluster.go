package sbch

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"reflect"
	"sort"
	"sync"
	"time"

	gethcmn "github.com/ethereum/go-ethereum/common"

	sbchrpctypes "github.com/smartbch/smartbch/rpc/types"
)

var _ RpcClient = (*ClusterClient)(nil)

type ClusterClient struct {
	clients     []RpcClient
	PublicNodes []NodeInfo
}

func NewClusterRpcClient(nodesGovAddr string, nodes []NodeInfo, privateUrls []string,
	reqTimeout time.Duration) (*ClusterClient, error) {

	clients := make([]RpcClient, 0, len(nodes))
	for _, node := range nodes {
		client, err := NewSimpleRpcClient(nodesGovAddr, node.RpcUrl, reqTimeout)
		if err != nil {
			return nil, fmt.Errorf("dail %s failed: %w", node.RpcUrl, err)
		}
		pbk, err := client.GetRpcPubkey()
		if err != nil {
			return nil, fmt.Errorf("get pubkey from %s failed: %w", node.RpcUrl, err)
		}
		if sha256.Sum256(pbk) != node.PbkHash {
			return nil, fmt.Errorf("pubkey not match: %s", node.RpcUrl)
		}
		clients = append(clients, client)
	}
	for _, url := range privateUrls {
		client, err := NewSimpleRpcClient(nodesGovAddr, url, reqTimeout)
		if err != nil {
			return nil, fmt.Errorf("dail %s failed: %w", url, err)
		}
		clients = append(clients, client)
	}
	return &ClusterClient{
		clients:     clients,
		PublicNodes: nodes,
	}, nil
}

func (cluster *ClusterClient) RpcURL() string {
	return "clusterRpcClient"
}

func (cluster *ClusterClient) GetRpcPubkey() ([]byte, error) {
	return nil, fmt.Errorf("unsupported operation")
}

func (cluster *ClusterClient) GetSbchdNodes() ([]NodeInfo, error) {
	result, err := cluster.getFromAllNodes("GetSbchdNodes")
	if err != nil {
		return nil, err
	}
	return result.([]NodeInfo), err
}

func (cluster *ClusterClient) GetSbchdNodesSorted() ([]NodeInfo, error) {
	nodes, err := cluster.GetSbchdNodes()
	if err == nil {
		sortNodes(nodes)
	}
	return nodes, err
}
func sortNodes(nodes []NodeInfo) {
	sort.Slice(nodes, func(i, j int) bool {
		return bytes.Compare(nodes[i].PbkHash[:], nodes[j].PbkHash[:]) < 0
	})
}

func (cluster *ClusterClient) GetRedeemingUtxosForOperators() ([]*sbchrpctypes.UtxoInfo, error) {
	result, err := cluster.getFromAllNodes("GetRedeemingUtxosForOperators")
	if err != nil {
		return nil, err
	}
	return result.([]*sbchrpctypes.UtxoInfo), err
}
func (cluster *ClusterClient) GetRedeemingUtxosForMonitors() ([]*sbchrpctypes.UtxoInfo, error) {
	result, err := cluster.getFromAllNodes("GetRedeemingUtxosForMonitors")
	if err != nil {
		return nil, err
	}
	return result.([]*sbchrpctypes.UtxoInfo), err
}
func (cluster *ClusterClient) GetToBeConvertedUtxosForOperators() ([]*sbchrpctypes.UtxoInfo, error) {
	result, err := cluster.getFromAllNodes("GetToBeConvertedUtxosForOperators")
	if err != nil {
		return nil, err
	}
	return result.([]*sbchrpctypes.UtxoInfo), err
}
func (cluster *ClusterClient) GetToBeConvertedUtxosForMonitors() ([]*sbchrpctypes.UtxoInfo, error) {
	result, err := cluster.getFromAllNodes("GetToBeConvertedUtxosForMonitors")
	if err != nil {
		return nil, err
	}
	return result.([]*sbchrpctypes.UtxoInfo), err
}

func (cluster *ClusterClient) GetMonitors() ([]gethcmn.Address, error) {
	result, err := cluster.getFromAllNodes("GetMonitors")
	if err != nil {
		return nil, err
	}
	return result.([]gethcmn.Address), err
}

func (cluster *ClusterClient) getFromAllNodes(methodName string) (any, error) {
	if len(cluster.clients) == 0 {
		return nil, fmt.Errorf("no clients")
	}

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
			return nil, fmt.Errorf("failed to call %s: %w",
				cluster.clients[idx].RpcURL(), err)
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
	case "GetMonitors":
		return client.GetMonitors()
	default:
		panic("unknown method") // unreachable
	}
}
