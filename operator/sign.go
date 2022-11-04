package operator

import (
	"encoding/hex"
	"encoding/binary"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/bluele/gcache"

	sbchrpctypes "github.com/smartbch/smartbch/rpc/types"

	"github.com/smartbch/ccoperator/sbch"
	"github.com/smartbch/ccoperator/utils"
)

const (
	minNodeCount = 2

	sigCacheMaxCount   = 100000
	sigCacheExpiration = 24 * time.Hour
	timeCacheMaxCount   = 200000
	timeCacheExpiration = 24 * time.Hour

	getSigHashesInterval = 3 * time.Second
	checkNodesInterval   = 1 * time.Hour
	newNodesDelayTime    = 6 * time.Hour
	clientReqTimeout     = 5 * time.Minute

	publicityPeriod = 6 * time.Hour //TODO
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
	timeCache = gcache.New(timeCacheMaxCount).Expiration(sigCacheExpiration).Simple().Build()
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

		fmt.Println("GetRedeemingUtxoSigHashes for Operators ...")
		redeemingUtxos4Op, err := rpcClient.GetRedeemingUtxosForOperators()
		redeemingUtxoSigHashes4Op := utxosToSigHashes(redeemingUtxos4Op)
		if err != nil {
			fmt.Println("can not get sig hashes:", err.Error())
			continue
		}
		fmt.Println("sigHashes:", redeemingUtxoSigHashes4Op)

		fmt.Println("GetToBeConvertedUtxoSigHashes for Operators ...")
		toBeConvertedUtxos4Op, err := rpcClient.GetToBeConvertedUtxosForOperators()
		if err != nil {
			fmt.Println("can not get sig hashes:", err.Error())
			continue
		}
		toBeConvertedUtxoSigHashes4Op := utxosToSigHashes(toBeConvertedUtxos4Op)
		fmt.Println("sigHashes:", toBeConvertedUtxoSigHashes4Op)

		allSigHashes4Op := append(redeemingUtxoSigHashes4Op, toBeConvertedUtxoSigHashes4Op...)
		for _, sigHashHex := range allSigHashes4Op {
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

		fmt.Println("GetRedeemingUtxoSigHashes for Monitors ...")
		redeemingUtxos4Mo, err := rpcClient.GetRedeemingUtxosForOperators()
		redeemingUtxoSigHashes4Mo := utxosToSigHashes(redeemingUtxos4Mo)
		if err != nil {
			fmt.Println("can not get sig hashes:", err.Error())
			continue
		}
		fmt.Println("sigHashes:", redeemingUtxoSigHashes4Mo)

		fmt.Println("GetToBeConvertedUtxoSigHashes for Monitors ...")
		toBeConvertedUtxos4Mo, err := rpcClient.GetToBeConvertedUtxosForOperators()
		if err != nil {
			fmt.Println("can not get sig hashes:", err.Error())
			continue
		}
		toBeConvertedUtxoSigHashes4Mo := utxosToSigHashes(toBeConvertedUtxos4Mo)
		fmt.Println("sigHashes:", toBeConvertedUtxoSigHashes4Mo)

		allSigHashes4Mo := append(redeemingUtxoSigHashes4Mo, toBeConvertedUtxoSigHashes4Mo...)
		var timestampBz [8]byte
		binary.BigEndian.PutUint64(timestampBz[:], utils.GetTimestampFromTSC())
		for _, sigHashHex := range allSigHashes4Mo {
			if timeCache.Has(sigHashHex) {
				continue
			}

			err = timeCache.SetWithExpire(sigHashHex, timestampBz, timeCacheExpiration)
			if err != nil {
				fmt.Println("failed to put sig into cache:", err.Error())
			}
		}
	}
}

func utxosToSigHashes(utxos []*sbchrpctypes.UtxoInfo) []string {
	sigHashes := make([]string, len(utxos))
	for i, utxoInfo := range utxos {
		sigHashes[i] = hex.EncodeToString(utxoInfo.TxSigHash)
	}
	return sigHashes
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
	timestampBz, ok := timestampIfc.([]byte)
	if !ok {
		return nil, errors.New("invalid cached timestamp")
	}
	okToSignTime := binary.BigEndian.Uint64(timestampBz) + uint64(publicityPeriod)
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
