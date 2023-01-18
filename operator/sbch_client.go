package operator

import (
	"encoding/hex"
	"fmt"
	"reflect"
	"sync"
	"time"

	gethcmn "github.com/ethereum/go-ethereum/common"
	log "github.com/sirupsen/logrus"

	"github.com/smartbch/cc-operator/sbch"
)

type sbchRpcClient struct {
	// never changed
	nodesGovAddr string
	privateUrls  []string

	// curr/new clients, protected by mutex
	rpcClientLock     sync.RWMutex
	currClusterClient *sbch.ClusterClient
	newClusterClient  *sbch.ClusterClient
	nodesChangedTime  time.Time

	// monitors info
	currMonitors  []gethcmn.Address
	allMonitors   []gethcmn.Address
	allMonitorMap map[gethcmn.Address]bool
}

func newSbchClient(nodesGovAddr string, bootstrapRpcURLs, privateUrls []string) (*sbchRpcClient, error) {
	log.Info("initRpcClient, nodesGovAddr:", nodesGovAddr,
		", bootstrapRpcURLs:", bootstrapRpcURLs, ", privateUrls:", privateUrls)

	// create bootstrapClient and use it to get all nodes
	bootstrapClient, err := sbch.NewClusterRpcClient(nodesGovAddr, bootstrapRpcURLs, clientReqTimeout)
	if err != nil {
		return nil, fmt.Errorf("failed to create bootstrapClient: %w", err)
	}
	bootNodes, err := bootstrapClient.GetSbchdNodesSorted()
	if err != nil {
		return nil, fmt.Errorf("failed to get bootNodes: %w", err)
	}

	// create clusterClient and check nodes
	clusterClient, err := sbch.NewClusterRpcClientOfNodes(nodesGovAddr, bootNodes, privateUrls, clientReqTimeout)
	if err != nil {
		return nil, fmt.Errorf("failed to create clusterClient: %w", err)
	}
	latestNodes, err := clusterClient.GetSbchdNodesSorted()
	if err != nil {
		return nil, fmt.Errorf("failed to get latestNodes: %w", err)
	}
	if !nodesEqual(latestNodes, bootNodes) {
		log.Info("bootNodes:", toJSON(bootNodes))
		log.Info("latestNodes:", toJSON(latestNodes))
		return nil, fmt.Errorf("nodes not match")
	}

	client := &sbchRpcClient{
		nodesGovAddr:      nodesGovAddr,
		privateUrls:       privateUrls,
		currClusterClient: clusterClient,
		allMonitorMap:     map[gethcmn.Address]bool{},
	}
	return client, nil
}

func (client *sbchRpcClient) getAllSigHashes4Op() ([]string, error) {
	client.rpcClientLock.RLock()
	rpcClient := client.currClusterClient
	client.rpcClientLock.RUnlock()

	log.Info("call GetRedeemingUtxosForOperators ...")
	redeemingUtxos4Op, err := rpcClient.GetRedeemingUtxosForOperators()
	if err != nil {
		log.Error("failed to call GetRedeemingUtxosForOperators:", err.Error())
		return nil, err
	}

	log.Info("call GetToBeConvertedUtxosForOperators ...")
	toBeConvertedUtxos4Op, err := rpcClient.GetToBeConvertedUtxosForOperators()
	if err != nil {
		log.Error("failed to call GetToBeConvertedUtxosForOperators:", err.Error())
		return nil, err
	}

	sigHashes := make([]string, 0, len(redeemingUtxos4Op)+len(toBeConvertedUtxos4Op))
	for _, utxo := range redeemingUtxos4Op {
		sigHashes = append(sigHashes, hex.EncodeToString(utxo.TxSigHash))
	}
	for _, utxo := range toBeConvertedUtxos4Op {
		sigHashes = append(sigHashes, hex.EncodeToString(utxo.TxSigHash))
	}
	log.Info("allSigHashes4Op:", sigHashes)
	return sigHashes, nil
}

func (client *sbchRpcClient) getAllSigHashes4Mo() ([]string, []string, error) {
	client.rpcClientLock.RLock()
	rpcClient := client.currClusterClient
	client.rpcClientLock.RUnlock()

	log.Info("call GetRedeemingUtxosForMonitors ...")
	redeemingUtxos4Mo, err := rpcClient.GetRedeemingUtxosForMonitors()
	if err != nil {
		log.Error("failed to call GetRedeemingUtxosForOperators:", err.Error())
		return nil, nil, err
	}

	log.Info("call GetToBeConvertedUtxosForMonitors ...")
	toBeConvertedUtxos4Mo, err := rpcClient.GetToBeConvertedUtxosForMonitors()
	if err != nil {
		log.Error("failed to call GetToBeConvertedUtxosForMonitors:", err.Error())
		return nil, nil, err
	}

	redeemingSigHashes := make([]string, len(redeemingUtxos4Mo))
	for i, utxo := range redeemingUtxos4Mo {
		redeemingSigHashes[i] = hex.EncodeToString(utxo.TxSigHash)
	}
	log.Info("redeemingSigHashes4Mo:", redeemingSigHashes)

	toBeConvertedSigHashes := make([]string, len(toBeConvertedUtxos4Mo))
	for i, utxo := range toBeConvertedUtxos4Mo {
		toBeConvertedSigHashes[i] = hex.EncodeToString(utxo.TxSigHash)
	}
	log.Info("toBeConvertedSigHashes4Mo:", toBeConvertedSigHashes)
	return redeemingSigHashes, toBeConvertedSigHashes, nil
}

// run this in a goroutine
func (client *sbchRpcClient) watchMonitorsAndSbchdNodes() {
	log.Info("start to watchMonitorsAndSbchdNodes ...")
	// TODO: change to time.Ticker?
	for {
		time.Sleep(checkNodesInterval)

		client.watchMonitors()
		client.watchSbchdNodes()
	}
}

func (client *sbchRpcClient) watchMonitors() {
	log.Info("get monitors ...")
	latestMonitors, err := client.currClusterClient.GetMonitors()
	if err != nil {
		log.Error("failed to get monitors:", err.Error())
	} else if !reflect.DeepEqual(latestMonitors, client.currMonitors) {
		log.Info("monitors changed:", toJSON(latestMonitors))
		client.rpcClientLock.Lock()
		client.currMonitors = latestMonitors
		for _, monitor := range latestMonitors {
			if !client.allMonitorMap[monitor] {
				client.allMonitorMap[monitor] = true
				client.allMonitors = append(client.allMonitors, monitor)
			}
		}
		client.rpcClientLock.Unlock()
	}
}

func (client *sbchRpcClient) watchSbchdNodes() {
	log.Info("get latest nodes ...")
	latestNodes, err := client.currClusterClient.GetSbchdNodesSorted()
	if err != nil {
		log.Error("failed to get sbchd nodes:", err.Error())
		return
	}

	if client.nodesChanged(latestNodes) {
		log.Info("nodes changed:", toJSON(latestNodes))
		client.newClusterClient = nil
		clusterClient, err := sbch.NewClusterRpcClientOfNodes(
			client.nodesGovAddr, latestNodes, client.privateUrls, clientReqTimeout)
		if err != nil {
			log.Error("failed to check sbchd nodes:", err.Error())
			return
		}

		client.nodesChangedTime = time.Now()
		client.newClusterClient = clusterClient
		return
	} else {
		log.Info("nodes not changed")
	}

	if client.newClusterClient != nil {
		if time.Now().Sub(client.nodesChangedTime) > newNodesDelayTime {
			log.Info("switch to new cluster client")
			client.rpcClientLock.Lock()
			client.currClusterClient = client.newClusterClient
			client.newClusterClient = nil
			client.rpcClientLock.Unlock()
		}
	}
}
func (client *sbchRpcClient) nodesChanged(latestNodes []sbch.NodeInfo) bool {
	if client.newClusterClient != nil {
		return !nodesEqual(client.newClusterClient.AllNodes, latestNodes)
	}
	return !nodesEqual(client.currClusterClient.AllNodes, latestNodes)
}
func nodesEqual(s1, s2 []sbch.NodeInfo) bool {
	return reflect.DeepEqual(s1, s2)
}

func (client *sbchRpcClient) isMonitor(addr gethcmn.Address) bool {
	client.rpcClientLock.RLock()
	defer client.rpcClientLock.RUnlock()

	return client.allMonitorMap[addr]
}

func (client *sbchRpcClient) fillMonitorsAndNodesInfo(opInfo *OpInfo) {
	client.rpcClientLock.RLock()
	defer client.rpcClientLock.RUnlock()

	opInfo.Monitors = client.allMonitors
	if client.currClusterClient != nil {
		opInfo.CurrNodes = client.currClusterClient.AllNodes
	}
	if client.newClusterClient != nil {
		opInfo.NewNodes = client.newClusterClient.AllNodes
		opInfo.NodesChangedTime = client.nodesChangedTime.Unix()
	}
}
