package operator

import (
	"encoding/hex"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/bluele/gcache"
	"github.com/smartbch/ccoperator/sbch"
)

const (
	minNodeCount = 2

	sigCacheMaxCount   = 10000
	sigCacheExpiration = 2 * time.Hour

	getSigHashesInterval = 3 * time.Second
	checkNodesInterval   = 1 * time.Hour
	newNodesDelayTime    = 6 * time.Hour
	clientReqTimeout     = 5 * time.Minute
)

var (
	nodesGovAddr      string
	rpcClientLock     sync.RWMutex
	bootstrapClient   *sbch.SimpleRpcClient
	currClusterClient *sbch.ClusterClient
	newClusterClient  *sbch.ClusterClient
	nodesChangedTime  time.Time
	skipNodeCert      bool

	sigCache = gcache.New(sigCacheMaxCount).Expiration(sigCacheExpiration).Simple().Build()
)

func initRpcClients(_nodesGovAddr, bootstrapRpcURL string, _skipNodeCert bool) {
	nodesGovAddr = _nodesGovAddr
	skipNodeCert = _skipNodeCert

	bootstrapClient = sbch.NewSimpleRpcClient(nodesGovAddr, bootstrapRpcURL, clientReqTimeout)
	allNodes, err := bootstrapClient.GetSbchdNodes()
	if err != nil {
		panic(err)
	}

	clusterClient, err := sbch.NewClusterRpcClientOfNodes(
		nodesGovAddr, allNodes, minNodeCount, skipNodeCert, clientReqTimeout)
	if err != nil {
		panic(err)
	}

	rpcClientLock.Lock()
	currClusterClient = clusterClient
	rpcClientLock.Unlock()
}

// run this in a goroutine
func getAndSignSigHashes() {
	fmt.Println("start to getAndSignSigHashes ...")
	for {
		time.Sleep(getSigHashesInterval)

		rpcClientLock.RLock()
		rpcClient := currClusterClient
		rpcClientLock.RUnlock()

		fmt.Println("GetRedeemingUtxoSigHashes ...")
		redeemingUtxoSigHashes, err := rpcClient.GetRedeemingUtxoSigHashes()
		if err != nil {
			fmt.Println("can not get sig hashes:", err.Error())
			continue
		}
		fmt.Println("sigHashes:", redeemingUtxoSigHashes)

		fmt.Println("GetToBeConvertedUtxoSigHashes ...")
		toBeConvertedUtxoSigHashes, err := rpcClient.GetToBeConvertedUtxoSigHashes()
		if err != nil {
			fmt.Println("can not get sig hashes:", err.Error())
			continue
		}
		fmt.Println("sigHashes:", toBeConvertedUtxoSigHashes)

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

			fmt.Println("sigHash:", sigHashHex, "sig:", hex.EncodeToString(sigBytes))
			err = sigCache.SetWithExpire(sigHashHex, sigBytes, sigCacheExpiration)
			if err != nil {
				fmt.Println("failed to put sig into cache:", err.Error())
			}
		}
	}
}

// run this in a goroutine
func watchSbchdNodes() {
	fmt.Println("start to watchSbchdNodes ...")
	// TODO: change to time.Ticker?
	for {
		time.Sleep(checkNodesInterval)

		latestNodes, err := currClusterClient.GetSbchdNodes()
		if err != nil {
			fmt.Println("failed to get sbchd nodes:", err.Error())
			continue
		}

		if nodesChanged(latestNodes) {
			newClusterClient = nil
			clusterClient, err := sbch.NewClusterRpcClientOfNodes(
				nodesGovAddr, latestNodes, minNodeCount, skipNodeCert, clientReqTimeout)
			if err != nil {
				fmt.Println("failed to check sbchd nodes:", err.Error())
				continue
			}

			nodesChangedTime = time.Now()
			newClusterClient = clusterClient
			continue
		}

		if newClusterClient != nil {
			if time.Now().Sub(nodesChangedTime) > newNodesDelayTime {
				rpcClientLock.Lock()
				currClusterClient = newClusterClient
				newClusterClient = nil
				rpcClientLock.Unlock()
			}
		}
	}
}

func nodesChanged(latestNodes []sbch.NodeInfo) bool {
	if newClusterClient != nil {
		return nodesEqual(newClusterClient.AllNodes, latestNodes)
	}
	return nodesEqual(currClusterClient.AllNodes, latestNodes)
}

func nodesEqual(s1, s2 []sbch.NodeInfo) bool {
	return reflect.DeepEqual(s1, s2)
}

func getSig(sigHashHex string) []byte {
	if strings.HasPrefix(sigHashHex, "0x") {
		sigHashHex = sigHashHex[2:]
	}

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
	rpcClientLock.RLock()
	defer rpcClientLock.RUnlock()
	return currClusterClient.AllNodes
}
func getNewNodes() []sbch.NodeInfo {
	rpcClientLock.RLock()
	defer rpcClientLock.RUnlock()
	if newClusterClient == nil {
		return nil
	}
	return newClusterClient.AllNodes
}
