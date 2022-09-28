package main

import (
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/bluele/gcache"
	"github.com/smartbch/ccoperator/sbch"
)

var rpcClientsInfoLock sync.RWMutex
var rpcClientsInfo *sbch.RpcClientsInfo
var newRpcClientsInfo *sbch.RpcClientsInfo
var nodesChangedTime time.Time

var sigCache = gcache.New(sigCacheMaxCount).Expiration(sigCacheExpiration).Simple().Build()

func initRpcClient() {
	var err error
	rpcClientsInfo, err = sbch.InitRpcClients(bootstrapRpcURL, minNodeCount)
	if err != nil {
		panic(err)
	}
}

func getAndSignSigHashes() {
	for {
		time.Sleep(getSigHashesInterval)

		rpcClientsInfoLock.RLock()
		rpcClients := rpcClientsInfo.ClusterRpcClient
		rpcClientsInfoLock.RUnlock()

		redeemingUtxoSigHashes, err := rpcClients.GetRedeemingUtxoSigHashes()
		if err != nil {
			fmt.Println("can not get sig hashes:", err.Error())
			continue
		}
		toBeConvertedUtxoSigHashes, err := rpcClients.GetToBeConvertedUtxoSigHashes()
		if err != nil {
			fmt.Println("can not get sig hashes:", err.Error())
			continue
		}
		allSigHashes := append(redeemingUtxoSigHashes, toBeConvertedUtxoSigHashes...)

		for _, sigHashHex := range allSigHashes {
			if sigCache.Has(sigHashHex) {
				continue
			}

			sigBytes, err := signSigHashECDSA(sigHashHex)
			if err != nil {
				fmt.Println("failed to sign sighash:", err.Error())
				continue
			}

			err = sigCache.SetWithExpire(sigHashHex, sigBytes, sigCacheExpiration)
			if err != nil {
				fmt.Println("failed to put sig into cache:", err.Error())
			}
		}
	}
}

func watchSbchdNodes() {
	// TODO: change to time.Ticker?
	for {
		time.Sleep(checkNodesInterval)

		latestNodes, err := rpcClientsInfo.BootstrapRpcClient.GetSbchdNodes()
		if err != nil {
			fmt.Println("failed to get sbchd nodes:", err.Error())
			continue
		}

		if nodesChanged(latestNodes) {
			newRpcClientsInfo = nil
			clusterClient, validNodes, err := sbch.NewClusterRpcClientOfNodes(
				latestNodes, minNodeCount)
			if err != nil {
				fmt.Println("failed to check sbchd nodes:", err.Error())
				continue
			}

			nodesChangedTime = time.Now()
			newRpcClientsInfo = &sbch.RpcClientsInfo{
				BootstrapRpcClient: rpcClientsInfo.BootstrapRpcClient,
				ClusterRpcClient:   clusterClient,
				AllNodes:           latestNodes,
				ValidNodes:         validNodes,
			}

			continue
		}

		if newRpcClientsInfo != nil {
			if time.Now().Sub(nodesChangedTime) > newNodesDelayTime {
				rpcClientsInfoLock.Lock()
				rpcClientsInfo = newRpcClientsInfo
				newRpcClientsInfo = nil
				rpcClientsInfoLock.Unlock()
			}
		}
	}
}

func nodesChanged(latestNodes []sbch.NodeInfo) bool {
	// second, compare with newRpcClientInfo.AllNodes
	if newRpcClientsInfo != nil {
		return nodesEqual(newRpcClientsInfo.AllNodes, latestNodes)
	}
	// first, compare with rpcClientInfo.AllNodes
	return nodesEqual(rpcClientsInfo.AllNodes, latestNodes)
}

func nodesEqual(s1, s2 []sbch.NodeInfo) bool {
	return reflect.DeepEqual(s1, s2)
}

func getSig(sigHashHex string) []byte {
	val, err := sigCache.Get(sigHashHex)
	if err != nil {
		return nil
	}

	if sig, ok := val.([]byte); ok {
		return sig
	}
	return nil
}

func getCurrNodes() []sbch.NodeInfo {
	rpcClientsInfoLock.RLock()
	defer rpcClientsInfoLock.RUnlock()
	return rpcClientsInfo.AllNodes
}
func getNewNodes() []sbch.NodeInfo {
	rpcClientsInfoLock.RLock()
	defer rpcClientsInfoLock.RUnlock()
	if newRpcClientsInfo == nil {
		return nil
	}
	return newRpcClientsInfo.AllNodes
}
