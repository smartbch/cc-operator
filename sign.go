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
var enclaveNodesChangedTime time.Time

var sigCache = gcache.New(sigCacheMaxCount).Expiration(sigCacheExpiration).Simple().Build()

func initRpcClient() {
	var err error
	rpcClientsInfo, err = sbch.InitRpcClients(bootstrapRpcURL, minEnclaveNodeCount, minSameRespCount)
	if err != nil {
		panic(err)
	}
}

func getAndSignSigHashes() {
	for {
		time.Sleep(getSigHashesInterval)

		rpcClientsInfoLock.RLock()
		attestedRpcClients := rpcClientsInfo.AttestedRpcClients
		rpcClientsInfoLock.RUnlock()

		sigHashes, err := attestedRpcClients.GetOperatorSigHashes()
		if err != nil {
			fmt.Println("can not get sig hashes:", err.Error())
			continue
		}

		for _, sigHashHex := range sigHashes {
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

func watchEnclaveNodes() {
	// TODO: change to time.Ticker?
	for {
		time.Sleep(checkEnclaveNodesInterval)

		latestEnclaveNodes, err := rpcClientsInfo.BootstrapRpcClient.GetEnclaveNodes()
		if err != nil {
			fmt.Println("failed to get enclave nodes:", err.Error())
			continue
		}

		if enclaveNodesChanged(latestEnclaveNodes) {
			newRpcClientsInfo = nil
			attestedRpcClients, attestedEnclaveNodes, err := sbch.AttestEnclavesAndCreateRpcClient(
				latestEnclaveNodes, minEnclaveNodeCount, minSameRespCount)
			if err != nil {
				fmt.Println("failed to attest enclave nodes:", err.Error())
				continue
			}

			enclaveNodesChangedTime = time.Now()
			newRpcClientsInfo = &sbch.RpcClientsInfo{
				BootstrapRpcClient:   rpcClientsInfo.BootstrapRpcClient,
				AttestedRpcClients:   attestedRpcClients,
				EnclaveNodes:         latestEnclaveNodes,
				AttestedEnclaveNodes: attestedEnclaveNodes,
			}

			continue
		}

		if newRpcClientsInfo != nil {
			if time.Now().Sub(enclaveNodesChangedTime) > newEnclaveNodesDelayTime {
				rpcClientsInfoLock.Lock()
				rpcClientsInfo = newRpcClientsInfo
				newRpcClientsInfo = nil
				rpcClientsInfoLock.Unlock()
			}
		}
	}
}

func enclaveNodesChanged(latestEnclaveNodes []sbch.EnclaveNodeInfo) bool {
	// first, compare with rpcClientInfo.EnclaveNodes
	if newRpcClientsInfo != nil {
		return enclaveNodesEqual(newRpcClientsInfo.EnclaveNodes, latestEnclaveNodes)
	}
	// second, compare with newRpcClientInfo.EnclaveNodes
	return enclaveNodesEqual(rpcClientsInfo.EnclaveNodes, latestEnclaveNodes)
}

func enclaveNodesEqual(s1, s2 []sbch.EnclaveNodeInfo) bool {
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

func getCurrEnclaveNodes() []sbch.EnclaveNodeInfo {
	rpcClientsInfoLock.RLock()
	defer rpcClientsInfoLock.RUnlock()
	return rpcClientsInfo.EnclaveNodes
}
func getNewEnclaveNodes() []sbch.EnclaveNodeInfo {
	rpcClientsInfoLock.RLock()
	defer rpcClientsInfoLock.RUnlock()
	if newRpcClientsInfo == nil {
		return nil
	}
	return newRpcClientsInfo.EnclaveNodes
}
