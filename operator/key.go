package operator

import (
	"crypto/ecdsa"
	"encoding/hex"
	"os"

	"github.com/edgelesssys/ego/ecrypto"
	"github.com/gcash/bchd/bchec"
	"github.com/gcash/bchutil"
	log "github.com/sirupsen/logrus"

	"github.com/smartbch/cc-operator/utils"
)

func loadOrGenKey(signerKeyWIF string) (privKey *bchec.PrivateKey, pbkBytes []byte, err error) {
	if sgxMode {
		if signerKeyWIF != "" && integrationTestMode {
			privKey, err = loadKeyFromWIF(signerKeyWIF)
		} else {
			privKey, err = loadOrGenKeyInEnclave()
		}
	} else {
		if signerKeyWIF != "" {
			privKey, err = loadKeyFromWIF(signerKeyWIF)
		} else {
			privKey, err = loadOrGenKeyNonEnclave()
		}
	}

	if err != nil {
		return nil, nil, err
	}

	pbkBytes = privKey.PubKey().SerializeCompressed()
	log.Info("pubkey:", hex.EncodeToString(pbkBytes))
	return
}

// only used for testing
func loadKeyFromWIF(wifStr string) (*bchec.PrivateKey, error) {
	log.Info("load private key from WIF")
	wif, err := bchutil.DecodeWIF(wifStr)
	if err != nil {
		return nil, err
	}
	return wif.PrivKey, nil
}

// only used for testing
func loadOrGenKeyNonEnclave() (*bchec.PrivateKey, error) {
	log.Info("load private key from file:", keyFile)
	fileData, err := os.ReadFile(keyFile)
	if err == nil {
		privKey, _ := bchec.PrivKeyFromBytes(bchec.S256(), fileData)
		return privKey, nil
	}
	if os.IsNotExist(err) {
		privKey, err := genNewPrivKey()
		if err == nil {
			err = os.WriteFile(keyFile, privKey.Serialize(), 0600)
		}
		return privKey, err
	}
	return nil, err
}

func loadOrGenKeyInEnclave() (privKey *bchec.PrivateKey, err error) {
	log.Info("load sealed private key from file:", keyFile)
	fileData, _err := os.ReadFile(keyFile)
	if _err != nil {
		log.Error("read file failed", _err.Error())
		if os.IsNotExist(_err) {
			// maybe it's first time to run this enclave app
			privKey, err = genAndSealPrivKey()
			if err != nil {
				return
			}
		} else {
			err = _err
			return
		}
	} else {
		privKey = unsealPrivKeyFromFile(fileData)
	}

	return
}

func genAndSealPrivKey() (*bchec.PrivateKey, error) {
	privKey, err := genNewPrivKey()
	if err != nil {
		return nil, err
	}

	err = sealPrivKeyToFile(privKey)
	if err != nil {
		return nil, err
	}

	return privKey, nil
}

func genNewPrivKey() (*bchec.PrivateKey, error) {
	log.Info("generate new private key")
	key, err := ecdsa.GenerateKey(bchec.S256(), &utils.RandReader{})
	if err != nil {
		return nil, err
	}
	privKey := (*bchec.PrivateKey)(key)
	log.Info("generated new private key")
	return privKey, nil
}

func sealPrivKeyToFile(privKey *bchec.PrivateKey) error {
	log.Info("seal private key to file:", keyFile)
	out, err := ecrypto.SealWithUniqueKey(privKey.Serialize(), nil)
	if err != nil {
		return err
	}
	err = os.WriteFile(keyFile, out, 0600)
	if err != nil {
		return err
	}
	log.Info("saved key to file")
	return nil
}

func unsealPrivKeyFromFile(fileData []byte) *bchec.PrivateKey {
	log.Info("unseal private key")
	rawData, err := ecrypto.Unseal(fileData, nil)
	if err != nil {
		log.Error("unseal file data failed", err.Error())
		return nil
	}
	privKey, _ := bchec.PrivKeyFromBytes(bchec.S256(), rawData)
	log.Info("loaded key from file")
	return privKey
}

//
//func signSigHashECDSA(sigHashHex string) ([]byte, error) {
//	sigHashBytes := gethcmn.FromHex(sigHashHex)
//	return covenant.SignRedeemTxSigHashECDSA(privKey, sigHashBytes)
//}

//func signSigHashSchnorr(sigHashHex string) ([]byte, error) {
//	sigHashBytes := gethcmn.FromHex(sigHashHex)
//	sig, err := privKey.SignSchnorr(sigHashBytes)
//	if err != nil {
//		return nil, err
//	}
//
//	return sig.Serialize(), nil
//}
