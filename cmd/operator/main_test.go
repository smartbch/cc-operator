package main

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

func TestGetBootstrapRpcUrls(t *testing.T) {
	privKey, err := ecdsa.GenerateKey(crypto.S256(), rand.Reader)
	require.NoError(t, err)

	url1 := "http://sbch1:8545"
	url2 := "http://sbch2:8545"
	hash := sha256.Sum256([]byte(url1 + url2))
	sig, err := crypto.Sign(hash[:], privKey)
	require.NoError(t, err)

	option := fmt.Sprintf("%s,%s,%s", url1, url2, hex.EncodeToString(sig[:64]))
	pbk := crypto.FromECDSAPub(&privKey.PublicKey)
	pbkHex := hex.EncodeToString(pbk)
	urls := getBootstrapRpcUrls(option, pbkHex)
	fmt.Println(urls)
}
