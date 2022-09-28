package main

import (
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"os"

	"github.com/edgelesssys/ego/ecrypto"
	"github.com/gcash/bchd/bchec"
	"github.com/smartbch/ccoperator/utils"
)

const keyFile = "/data/key.txt"

var privKey *bchec.PrivateKey
var pubKeyBytes []byte // compressed

func loadOrGenKey() {
	fileData, err := os.ReadFile(keyFile)
	if err != nil {
		fmt.Printf("read file failed, %s\n", err.Error())
		if os.IsNotExist(err) {
			// maybe first run this enclave app
			genPrivKey()
		}
		return
	}

	unsealPrivKeyFromFile(fileData)

	pubKeyBytes = privKey.PubKey().SerializeCompressed()
	fmt.Printf("pubkey: %s\n", hex.EncodeToString(pubKeyBytes))
}

func genPrivKey() {
	newPrivKey()
	sealPrivKeyToFile()
}

func newPrivKey() {
	key, err := ecdsa.GenerateKey(bchec.S256(), &utils.RandReader{})
	if err != nil {
		panic(err)
	}
	privKey = (*bchec.PrivateKey)(key)
	fmt.Println("generated new private key")
}

func sealPrivKeyToFile() {
	out, err := ecrypto.SealWithUniqueKey(privKey.Serialize(), nil)
	if err != nil {
		panic(err)
	}
	err = os.WriteFile(keyFile, out, 0600)
	if err != nil {
		panic(err)
	}
	fmt.Println("saved key to file")
}

func unsealPrivKeyFromFile(fileData []byte) {
	rawData, err := ecrypto.Unseal(fileData, nil)
	if err != nil {
		fmt.Printf("unseal file data failed, %s\n", err.Error())
		return
	}
	privKey, _ = bchec.PrivKeyFromBytes(bchec.S256(), rawData)
	fmt.Println("loaded key from file")
}

func signSigHashSchnorr(sigHashHex string) ([]byte, error) {
	sigHashBytes, err := hex.DecodeString(sigHashHex)
	if err != nil {
		return nil, err
	}

	sig, err := privKey.SignSchnorr(sigHashBytes)
	if err != nil {
		return nil, err
	}

	return sig.Serialize(), nil
}

func signSigHashECDSA(sigHashHex string) ([]byte, error) {
	sigHashBytes, err := hex.DecodeString(sigHashHex)
	if err != nil {
		return nil, err
	}

	sig, err := privKey.SignECDSA(sigHashBytes)
	if err != nil {
		return nil, err
	}

	return sig.Serialize(), nil
}
