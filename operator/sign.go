package operator

import (
	"encoding/hex"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/bluele/gcache"

	"github.com/smartbch/cc-operator/sbch"
	"github.com/smartbch/cc-operator/utils"
)

const (
	sigCacheMaxCount    = 100000
	sigCacheExpiration  = 24 * time.Hour
	timeCacheMaxCount   = 200000
	timeCacheExpiration = 24 * time.Hour

	getSigHashesInterval = 10 * time.Second
	checkNodesInterval   = 6 * time.Minute
	newNodesDelayTime    = 6 * time.Hour
	clientReqTimeout     = 5 * time.Minute

	redeemPublicityPeriod  = 25  // * 60
	convertPublicityPeriod = 100 // * 60
)

var (
	nodesGovAddr string // never changed

	rpcClientLock     sync.RWMutex
	currClusterClient *sbch.ClusterClient
	newClusterClient  *sbch.ClusterClient
	nodesChangedTime  time.Time

	sigCache  = gcache.New(sigCacheMaxCount).Expiration(sigCacheExpiration).Simple().Build()
	timeCache = gcache.New(timeCacheMaxCount).Expiration(sigCacheExpiration).Simple().Build()
)

func initRpcClients(_nodesGovAddr string, bootstrapRpcURLs, privateUrls []string) {
	nodesGovAddr = _nodesGovAddr

	// create bootstrapClient and use it to get all nodes
	bootstrapClient, err := sbch.NewClusterRpcClient(nodesGovAddr, bootstrapRpcURLs, clientReqTimeout)
	if err != nil {
		panic(err)
	}
	allNodes, err := bootstrapClient.GetSbchdNodesSorted()
	if err != nil {
		panic(err)
	}

	// create clusterClient and check nodes
	clusterClient, err := sbch.NewClusterRpcClientOfNodes(nodesGovAddr, allNodes, privateUrls, clientReqTimeout)
	if err != nil {
		panic(err)
	}
	latestNodes, err := clusterClient.GetSbchdNodesSorted()
	if err != nil {
		panic(err)
	}
	if !nodesEqual(latestNodes, allNodes) {
		panic("Invalid Bootstrap Client")
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

		allSigHashes4Op, err := getAllSigHashes4Op(rpcClient)
		if err != nil {
			continue
		}
		signSigHashes4Op(allSigHashes4Op)

		redeemingSigHashes4Mo, toBeConvertedSigHashes4Mo, err := getAllSigHashes4Mo(rpcClient)
		if err != nil {
			continue
		}
		cacheSigHashes4Mo(redeemingSigHashes4Mo, toBeConvertedSigHashes4Mo)
	}
}

func getAllSigHashes4Op(rpcClient *sbch.ClusterClient) ([]string, error) {
	fmt.Println("call GetRedeemingUtxosForOperators ...")
	redeemingUtxos4Op, err := rpcClient.GetRedeemingUtxosForOperators()
	if err != nil {
		fmt.Println("failed to call GetRedeemingUtxosForOperators:", err.Error())
		return nil, err
	}

	fmt.Println("call GetToBeConvertedUtxosForOperators ...")
	toBeConvertedUtxos4Op, err := rpcClient.GetToBeConvertedUtxosForOperators()
	if err != nil {
		fmt.Println("failed to call GetToBeConvertedUtxosForOperators:", err.Error())
		return nil, err
	}

	sigHashes := make([]string, 0, len(redeemingUtxos4Op)+len(toBeConvertedUtxos4Op))
	for _, utxo := range redeemingUtxos4Op {
		sigHashes = append(sigHashes, hex.EncodeToString(utxo.TxSigHash))
	}
	for _, utxo := range toBeConvertedUtxos4Op {
		sigHashes = append(sigHashes, hex.EncodeToString(utxo.TxSigHash))
	}
	fmt.Println("allSigHashes4Op:", sigHashes)
	return sigHashes, nil
}

func getAllSigHashes4Mo(rpcClient *sbch.ClusterClient) ([]string, []string, error) {
	fmt.Println("call GetRedeemingUtxosForMonitors ...")
	redeemingUtxos4Mo, err := rpcClient.GetRedeemingUtxosForMonitors()
	if err != nil {
		fmt.Println("failed to call GetRedeemingUtxosForOperators:", err.Error())
		return nil, nil, err
	}

	fmt.Println("call GetToBeConvertedUtxosForMonitors ...")
	toBeConvertedUtxos4Mo, err := rpcClient.GetToBeConvertedUtxosForMonitors()
	if err != nil {
		fmt.Println("failed to call GetToBeConvertedUtxosForMonitors:", err.Error())
		return nil, nil, err
	}

	redeemingSigHashes := make([]string, len(redeemingUtxos4Mo))
	for i, utxo := range redeemingUtxos4Mo {
		redeemingSigHashes[i] = hex.EncodeToString(utxo.TxSigHash)
	}
	fmt.Println("redeemingSigHashes4Mo:", redeemingSigHashes)

	toBeConvertedSigHashes := make([]string, len(toBeConvertedUtxos4Mo))
	for i, utxo := range toBeConvertedUtxos4Mo {
		toBeConvertedSigHashes[i] = hex.EncodeToString(utxo.TxSigHash)
	}
	fmt.Println("toBeConvertedSigHashes4Mo:", toBeConvertedSigHashes)
	return redeemingSigHashes, toBeConvertedSigHashes, nil
}

func signSigHashes4Op(allSigHashes4Op []string) {
	for _, sigHashHex := range allSigHashes4Op {
		if sigCache.Has(sigHashHex) {
			continue
		}

		sigBytes, err := signSigHashECDSA(sigHashHex)
		if err != nil {
			fmt.Println("failed to sign sigHash:", err.Error())
			continue
		}

		fmt.Println("sigHash:", sigHashHex, "sig:", hex.EncodeToString(sigBytes))
		err = sigCache.SetWithExpire(sigHashHex, sigBytes, sigCacheExpiration)
		if err != nil {
			fmt.Println("failed to put sig into cache:", err.Error())
		}
	}
}

func cacheSigHashes4Mo(redeemingSigHashes4Mo, toBeConvertedSigHashes4Mo []string) {
	ts := utils.GetTimestampFromTSC()

	redeemOkTs := ts + uint64(redeemPublicityPeriod)
	for _, sigHashHex := range redeemingSigHashes4Mo {
		if timeCache.Has(sigHashHex) {
			continue
		}

		err := timeCache.SetWithExpire(sigHashHex, redeemOkTs, timeCacheExpiration)
		if err != nil {
			fmt.Println("failed to put sigHash into cache:", err.Error())
		}
	}

	convertOkTs := ts + uint64(convertPublicityPeriod)
	for _, sigHashHex := range toBeConvertedSigHashes4Mo {
		if timeCache.Has(sigHashHex) {
			continue
		}

		err := timeCache.SetWithExpire(sigHashHex, convertOkTs, timeCacheExpiration)
		if err != nil {
			fmt.Println("failed to put sigHash into cache:", err.Error())
		}
	}
}

// run this in a goroutine
func watchSbchdNodes(privateUrls []string) {
	fmt.Println("start to watchSbchdNodes ...")
	// TODO: change to time.Ticker?
	for {
		time.Sleep(checkNodesInterval)

		fmt.Println("get latest nodes ...")
		latestNodes, err := currClusterClient.GetSbchdNodesSorted()
		if err != nil {
			fmt.Println("failed to get sbchd nodes:", err.Error())
			continue
		}

		if nodesChanged(latestNodes) {
			fmt.Println("nodes changed")
			newClusterClient = nil
			clusterClient, err := sbch.NewClusterRpcClientOfNodes(
				nodesGovAddr, latestNodes, privateUrls, clientReqTimeout)
			if err != nil {
				fmt.Println("failed to check sbchd nodes:", err.Error())
				continue
			}

			nodesChangedTime = time.Now()
			newClusterClient = clusterClient
			continue
		} else {
			fmt.Println("nodes not changed")
		}

		if newClusterClient != nil {
			if time.Now().Sub(nodesChangedTime) > newNodesDelayTime {
				fmt.Println("switch to new cluster client")
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
		return !nodesEqual(newClusterClient.AllNodes, latestNodes)
	}
	return !nodesEqual(currClusterClient.AllNodes, latestNodes)
}
func nodesEqual(s1, s2 []sbch.NodeInfo) bool {
	return reflect.DeepEqual(s1, s2)
}

func getSig(sigHashHex string) ([]byte, error) {
	if strings.HasPrefix(sigHashHex, "0x") {
		sigHashHex = sigHashHex[2:]
	}

	val, err := sigCache.Get(sigHashHex)
	if err != nil {
		return nil, err
	}

	timestampIfc, err := timeCache.Get(sigHashHex)
	if err != nil {
		return nil, err
	}
	okToSignTime, ok := timestampIfc.(uint64)
	if !ok {
		return nil, errors.New("invalid cached timestamp")
	}
	currentTime := utils.GetTimestampFromTSC()
	if currentTime < okToSignTime { // Cannot Sign
		return nil, fmt.Errorf("still too early to sign: %d < %d", currentTime, okToSignTime)
	}

	sig, ok := val.([]byte)
	if !ok {
		return nil, errors.New("invalid cached signature")
	}
	return sig, nil
}

func getNodesInfo() (nodesInfo OpInfo) {
	rpcClientLock.RLock()
	defer rpcClientLock.RUnlock()

	if currClusterClient != nil {
		nodesInfo.CurrNodes = currClusterClient.AllNodes
	}
	if newClusterClient != nil {
		nodesInfo.NewNodes = newClusterClient.AllNodes
		nodesInfo.NodesChangedTime = nodesChangedTime.Unix()
	}
	return nodesInfo
}
