package main

import (
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/smartbch/cc-operator/operator"
	"strings"
)

var (
	helpFlag        = false
	serverName      = "cc-operator"
	listenAddr      = "0.0.0.0:8801"
	bootstrapRpcURL = "http://localhost:8545"
	monitorAddrList = ""
	signerKeyWIF    = ""

	// TODO: change this to constant in production mode
	nodesGovAddr = "0x0000000000000000000000000000000000001234"

	fixedBootstrapRpcURls = []string{"http://sbch1:8545", "http://sbch2:8545"} //todo: replace in product
	bootstrapSetPubkey    = ""                                                 //todo: replace with centralized rpc pubkey in product
)

func main() {
	newFixedBootstrapRpcUrl := ""
	flag.BoolVar(&helpFlag, "help", false, "show help")
	flag.StringVar(&serverName, "serverName", serverName, "server name to generate TLS certificate")
	flag.StringVar(&listenAddr, "listenAddr", listenAddr, "listen addr, ip:port")
	flag.StringVar(&bootstrapRpcURL, "bootstrapRpcURL", bootstrapRpcURL, "bootstrap smartBCH RPC URL")
	flag.StringVar(&nodesGovAddr, "nodesGovAddr", nodesGovAddr, "address of NodesGov contract")
	flag.StringVar(&monitorAddrList, "monitorAddrList", monitorAddrList, "comma separated monitor addresses")
	flag.StringVar(&signerKeyWIF, "signerKeyWIF", signerKeyWIF, "signer key WIF, for integration test only")
	flag.StringVar(&newFixedBootstrapRpcUrl, "newFixedBootstrapUrl", newFixedBootstrapRpcUrl, "new fixed bootstrap urls with signature")
	flag.Parse()
	if helpFlag {
		flag.Usage()
		return
	}
	operator.Start(serverName, listenAddr, nodesGovAddr, monitorAddrList, signerKeyWIF, getBootstrapRpcUrls(newFixedBootstrapRpcUrl))
}

func getNewBootstrapRpcPubkey() []byte {
	pb, err := hex.DecodeString(bootstrapSetPubkey)
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
func getBootstrapRpcUrls(newFixedBootstrapRpcUrl string) []string {
	pb := getNewBootstrapRpcPubkey()
	if newFixedBootstrapRpcUrl != "" {
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
	repeatBootstrapUrl := false
	bootstrapRpcURLs = append(bootstrapRpcURLs, fixedBootstrapRpcURls...)
	for _, url := range fixedBootstrapRpcURls {
		if url == bootstrapRpcURL {
			repeatBootstrapUrl = true
		}
	}
	if !repeatBootstrapUrl {
		bootstrapRpcURLs = append(bootstrapRpcURLs, bootstrapRpcURL)
	}
	return bootstrapRpcURLs
}
