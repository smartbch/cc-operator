package operator

import (
	"crypto/ecdsa"
	"encoding/hex"
	"io/ioutil"
	"os"

	"github.com/edgelesssys/ego/ecrypto"
	gethcmn "github.com/ethereum/go-ethereum/common"
	"github.com/gcash/bchd/bchec"
	"github.com/gcash/bchutil"
	log "github.com/sirupsen/logrus"

	"github.com/smartbch/cc-operator/utils"
	"github.com/smartbch/smartbch/crosschain/covenant"
)

const (
	keyFile = "/data/key.txt"
)

var (
	privKey     *bchec.PrivateKey
	pubKeyBytes []byte // compressed
)

// only used for testing
func loadKeyFromWIF(wifStr string) {
	wif, err := bchutil.DecodeWIF(wifStr)
	if err != nil {
		panic(err)
	}

	privKey = wif.PrivKey
	pubKeyBytes = privKey.PubKey().SerializeCompressed()
}

// only used for testing
func loadOrGenKeyNonEnclave() {
	fileData, err := os.ReadFile(keyFile)
	if err == nil {
		privKey, _ = bchec.PrivKeyFromBytes(bchec.S256(), fileData)
		pubKeyBytes = privKey.PubKey().SerializeCompressed()
		return
	}
	if os.IsNotExist(err) {
		genNewPrivKey()
		_ = ioutil.WriteFile(keyFile, privKey.Serialize(), 0600)
		pubKeyBytes = privKey.PubKey().SerializeCompressed()
		return
	}
	panic(err)
}

func loadOrGenKeyInEnclave() {
	log.Info("load private key from file:", keyFile)
	fileData, err := os.ReadFile(keyFile)
	if err != nil {
		log.Error("read file failed", err.Error())
		if os.IsNotExist(err) {
			// maybe it's first time to run this enclave app
			genAndSealPrivKey()
		} else {
			panic(err)
		}
	} else {
		unsealPrivKeyFromFile(fileData)
	}

	pubKeyBytes = privKey.PubKey().SerializeCompressed()
	log.Info("pubkey:", hex.EncodeToString(pubKeyBytes))
}

func genAndSealPrivKey() {
	genNewPrivKey()
	sealPrivKeyToFile()
}

func genNewPrivKey() {
	log.Info("generate new private key")
	key, err := ecdsa.GenerateKey(bchec.S256(), &utils.RandReader{})
	if err != nil {
		panic(err)
	}
	privKey = (*bchec.PrivateKey)(key)
	log.Info("generated new private key")
}

func sealPrivKeyToFile() {
	log.Info("seal private key to file:", keyFile)
	out, err := ecrypto.SealWithUniqueKey(privKey.Serialize(), nil)
	if err != nil {
		panic(err)
	}
	err = os.WriteFile(keyFile, out, 0600)
	if err != nil {
		panic(err)
	}
	log.Info("saved key to file")
}

func unsealPrivKeyFromFile(fileData []byte) {
	log.Info("unseal private key")
	rawData, err := ecrypto.Unseal(fileData, nil)
	if err != nil {
		log.Error("unseal file data failed", err.Error())
		return
	}
	privKey, _ = bchec.PrivKeyFromBytes(bchec.S256(), rawData)
	log.Info("loaded key from file")
}

func signSigHashECDSA(sigHashHex string) ([]byte, error) {
	sigHashBytes := gethcmn.FromHex(sigHashHex)
	return covenant.SignRedeemTxSigHashECDSA(privKey, sigHashBytes)
}

//func signSigHashSchnorr(sigHashHex string) ([]byte, error) {
//	sigHashBytes := gethcmn.FromHex(sigHashHex)
//	sig, err := privKey.SignSchnorr(sigHashBytes)
//	if err != nil {
//		return nil, err
//	}
//
//	return sig.Serialize(), nil
//}
