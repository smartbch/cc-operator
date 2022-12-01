package operator

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"reflect"
	"sort"
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

	getSigHashesInterval = 3 * time.Second
	checkNodesInterval   = 6 * time.Minute
	newNodesDelayTime    = 6 * time.Hour
	clientReqTimeout     = 5 * time.Minute

	redeemPublicityPeriod  = 25 * 60
	convertPublicityPeriod = 100 * 60
)

var (
	nodesGovAddr      string
	rpcClientLock     sync.RWMutex
	currClusterClient *sbch.ClusterClient
	newClusterClient  *sbch.ClusterClient
	nodesChangedTime  time.Time
	skipPbkCheck      bool

	sigCache  = gcache.New(sigCacheMaxCount).Expiration(sigCacheExpiration).Simple().Build()
	timeCache = gcache.New(timeCacheMaxCount).Expiration(sigCacheExpiration).Simple().Build()
)

func initRpcClients(_nodesGovAddr string, bootstrapRpcURLs []string, _skipPbkCheck bool) {
	nodesGovAddr = _nodesGovAddr
	skipPbkCheck = _skipPbkCheck

	bootstrapClient := sbch.NewClusterRpcClient(nodesGovAddr, bootstrapRpcURLs, clientReqTimeout)
	allNodes, err := bootstrapClient.GetSbchdNodes()
	if err != nil {
		panic(err)
	}

	sortNodes(allNodes)
	clusterClient, err := sbch.NewClusterRpcClientOfNodes(
		nodesGovAddr, allNodes, skipPbkCheck, clientReqTimeout)
	if err != nil {
		panic(err)
	}

	latestNodes, err := clusterClient.GetSbchdNodes()
	if err != nil {
		panic(err)
	}

	sortNodes(latestNodes)
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

		sortNodes(latestNodes)
		if nodesChanged(latestNodes) {
			newClusterClient = nil
			clusterClient, err := sbch.NewClusterRpcClientOfNodes(
				nodesGovAddr, latestNodes, skipPbkCheck, clientReqTimeout)
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

func sortNodes(nodes []sbch.NodeInfo) {
	sort.Slice(nodes, func(i, j int) bool {
		return bytes.Compare(nodes[i].PbkHash[:], nodes[j].PbkHash[:]) < 0
	})
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
		return nil, errors.New("still too early to sign")
	}

	sig, ok := val.([]byte)
	if !ok {
		return nil, errors.New("invalid cached signature")
	}
	return sig, nil
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
