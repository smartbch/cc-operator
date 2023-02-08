package operator

import (
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/bluele/gcache"
	gethcmn "github.com/ethereum/go-ethereum/common"
	"github.com/gcash/bchd/bchec"
	log "github.com/sirupsen/logrus"

	"github.com/smartbch/cc-operator/utils"
	"github.com/smartbch/smartbch/crosschain/covenant"
)

type txSigner struct {
	privKey    *bchec.PrivateKey
	sbchClient *sbchRpcClient

	sigCache  gcache.Cache
	timeCache gcache.Cache
}

func newSigner(privKey *bchec.PrivateKey, sbchClient *sbchRpcClient) *txSigner {
	return &txSigner{
		privKey:    privKey,
		sbchClient: sbchClient,
		sigCache:   gcache.New(sigCacheMaxCount).Expiration(sigCacheExpiration).Simple().Build(),
		timeCache:  gcache.New(timeCacheMaxCount).Expiration(sigCacheExpiration).Simple().Build(),
	}
}

// run this in a goroutine
func (signer *txSigner) getAndSignSigHashes() {
	log.Info("start to getAndSignSigHashes ...")
	for {
		time.Sleep(getSigHashesInterval)

		allSigHashes4Op, err := signer.sbchClient.getAllSigHashes4Op()
		if err != nil {
			continue
		}
		signer.signSigHashes4Op(allSigHashes4Op)

		redeemingSigHashes4Mo, toBeConvertedSigHashes4Mo, err := signer.sbchClient.getAllSigHashes4Mo()
		if err != nil {
			continue
		}
		signer.cacheSigHashes4Mo(redeemingSigHashes4Mo, toBeConvertedSigHashes4Mo)
	}
}

func (signer *txSigner) signSigHashes4Op(allSigHashes4Op []string) {
	for _, sigHashHex := range allSigHashes4Op {
		if signer.sigCache.Has(sigHashHex) {
			continue
		}

		sigBytes, err := signer.signSigHashECDSA(sigHashHex)
		if err != nil {
			log.Error("failed to sign sigHash:", err.Error())
			continue
		}

		log.Info("sigHash:", sigHashHex, "sig:", hex.EncodeToString(sigBytes))
		err = signer.sigCache.SetWithExpire(sigHashHex, sigBytes, sigCacheExpiration)
		if err != nil {
			log.Error("failed to put sig into cache:", err.Error())
		}
	}
}

func (signer *txSigner) signSigHashECDSA(sigHashHex string) ([]byte, error) {
	sigHashBytes := gethcmn.FromHex(sigHashHex)
	return covenant.SignRedeemTxSigHashECDSA(signer.privKey, sigHashBytes)
}

func (signer *txSigner) cacheSigHashes4Mo(redeemingSigHashes4Mo, toBeConvertedSigHashes4Mo []string) {
	ts := utils.GetTimestampFromTSC()

	redeemOkTs := ts + uint64(redeemPublicityPeriod)
	for _, sigHashHex := range redeemingSigHashes4Mo {
		if signer.timeCache.Has(sigHashHex) {
			continue
		}

		err := signer.timeCache.SetWithExpire(sigHashHex, redeemOkTs, timeCacheExpiration)
		if err != nil {
			log.Error("failed to put sigHash into cache:", err.Error())
		}
	}

	convertOkTs := ts + uint64(convertPublicityPeriod)
	for _, sigHashHex := range toBeConvertedSigHashes4Mo {
		if signer.timeCache.Has(sigHashHex) {
			continue
		}

		err := signer.timeCache.SetWithExpire(sigHashHex, convertOkTs, timeCacheExpiration)
		if err != nil {
			log.Error("failed to put sigHash into cache:", err.Error())
		}
	}
}

func (signer *txSigner) getSig(sigHashHex string) ([]byte, error) {
	sigHashHex = strings.TrimPrefix(sigHashHex, "0x")

	val, err := signer.sigCache.Get(sigHashHex)
	if err != nil {
		return nil, err
	}

	timestampIfc, err := signer.timeCache.Get(sigHashHex)
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

func (signer *txSigner) isMonitor(addr gethcmn.Address) bool {
	return signer.sbchClient.isMonitor(addr)
}

func (signer *txSigner) fillMonitorsAndNodesInfo(opInfo *OpInfo) {
	signer.sbchClient.fillMonitorsAndNodesInfo(opInfo)
}
