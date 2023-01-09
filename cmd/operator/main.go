package main

import (
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"strings"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/smartbch/cc-operator/operator"
)

var (
	helpFlag       = false
	serverName     = "cc-operator"
	listenAddr     = "0.0.0.0:8801"
	privateRpcURLs = ""
	signerKeyWIF   = ""    // test only
	withChaos      = false // test only

	// TODO: change this to constant in production mode
	nodesGovAddr = "0x0000000000000000000000000000000000001234"

	// TODO: replace in product
	fixedBootstrapRpcURls = []string{
		"http://18.141.161.139:8545",
		"http://18.141.161.139:8545",
	}
	// TODO: replace with centralized rpc pubkey in product
	bootstrapSetPubkey = ""
)

func main() {
	newFixedBootstrapRpcUrl := ""
	flag.BoolVar(&helpFlag, "help", false, "show help")
	flag.StringVar(&serverName, "serverName", serverName, "server name to generate TLS certificate")
	flag.StringVar(&listenAddr, "listenAddr", listenAddr, "listen addr, ip:port")
	flag.StringVar(&nodesGovAddr, "nodesGovAddr", nodesGovAddr, "address of NodesGov contract")
	flag.StringVar(&newFixedBootstrapRpcUrl, "newFixedBootstrapUrl", newFixedBootstrapRpcUrl, "new fixed bootstrap urls with signature separated with comma")
	flag.StringVar(&privateRpcURLs, "privateRpcUrls", privateRpcURLs, "comma separated private rpc urls")
	flag.StringVar(&signerKeyWIF, "signerKeyWIF", signerKeyWIF, "signer key WIF, for integration test only")
	flag.BoolVar(&withChaos, "withChaos", withChaos, "return chaos, for integration test only")

	flag.Parse()
	if helpFlag {
		flag.Usage()
		return
	}

	bootstrapRpcURLs := getBootstrapRpcUrls(newFixedBootstrapRpcUrl, bootstrapSetPubkey)

	var privateRpcURLList []string
	if privateRpcURLs != "" {
		privateRpcURLList = strings.Split(privateRpcURLs, ",")
	}

	operator.Start(serverName, listenAddr, nodesGovAddr, signerKeyWIF,
		bootstrapRpcURLs, privateRpcURLList, withChaos)
}

func getNewBootstrapRpcPubkey(pbkHex string) []byte {
	pb, err := hex.DecodeString(pbkHex)
	if err != nil {
		panic(err)
	}
	_, err = crypto.UnmarshalPubkey(pb)
	if err != nil {
		panic(err)
	}
	return pb
}

// newFixedBootstrapRpcUrl format: url0,url1,sig
func getBootstrapRpcUrls(newFixedBootstrapRpcUrl, pbkHex string) []string {
	if newFixedBootstrapRpcUrl != "" {
		pb := getNewBootstrapRpcPubkey(pbkHex)

		parts := strings.Split(newFixedBootstrapRpcUrl, ",")
		if len(parts) != 3 {
			panic("new fixed bootstrap url should has 3 parts")
		}
		hash := sha256.Sum256([]byte(parts[0] + parts[1]))
		sig, err := hex.DecodeString(parts[2])
		if err != nil {
			panic(err)
		}
		if !crypto.VerifySignature(pb, hash[:], sig) {
			panic("verify new fixed bootstrap url signature failed")
		}
		fixedBootstrapRpcURls = []string{parts[0], parts[1]}
	}

	var bootstrapRpcURLs []string
	bootstrapRpcURLs = append(bootstrapRpcURLs, fixedBootstrapRpcURls...)
	return bootstrapRpcURLs
}
