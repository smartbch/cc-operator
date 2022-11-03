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
	clients    []RpcClient
	AllNodes   []NodeInfo
	ValidNodes []NodeInfo
}

func NewClusterRpcClientOfNodes(nodesGovAddr string, nodes []NodeInfo,
	minNodeCount int, skipPbkCheck bool, clientReqTimeout time.Duration) (*ClusterClient, error) {

	okNodes := make([]NodeInfo, 0, len(nodes))
	clients := make([]RpcClient, 0, len(nodes))
	for _, node := range nodes {
		client := NewSimpleRpcClient(nodesGovAddr, node.RpcUrl, clientReqTimeout)
		pbk, err := client.GetRpcPubkey()
		if err != nil {
			fmt.Println("failed to get pubkey from node:", node.RpcUrl, err)
			continue
		}

		if !skipPbkCheck && sha256.Sum256(pbk) != node.PbkHash {
			fmt.Println("pubkey not match:", node.RpcUrl)
			continue
		}

		okNodes = append(okNodes, node)
		clients = append(clients, client)
	}

	if len(okNodes) < minNodeCount {
		return nil, fmt.Errorf("not enough nodes to connect")
	}

	return &ClusterClient{
		clients:    clients,
		AllNodes:   nodes,
		ValidNodes: okNodes,
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
