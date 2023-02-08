package main

import (
	"bytes"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"flag"
	"fmt"

	"github.com/edgelesssys/ego/attestation"
	gethcmn "github.com/ethereum/go-ethereum/common"

	"github.com/smartbch/cc-operator/utils"
)

//var (
//	signer    string
//	uniqueID  []byte
//	proxyAddr *string
//)

func main() {
	signerID := flag.String("signer-id", "", "signer ID")
	uniqueID := flag.String("unique-id", "", "unique ID")
	opAddr := flag.String("operator-addr", "localhost:8801", "operator address")
	flag.Parse()

	signerIDBytes := gethcmn.FromHex(*signerID)
	uniqueIDBytes := gethcmn.FromHex(*uniqueID)

	pubkeyBytes, err := getPubkey(*opAddr)
	if err != nil {
		println("failed to get pubkey data: ", err.Error())
		return
	}

	reportBytes, err := getPubkeyReport(*opAddr)
	if err != nil {
		println("failed to get pubkey report: ", err.Error())
		return
	}

	report, err := utils.VerifyRemoteReport(reportBytes)
	if err != nil {
		println("failed to verify remote report: ", err.Error())
		return
	}

	err = checkReport(report, signerIDBytes, uniqueIDBytes, pubkeyBytes)
	if err != nil {
		println("failed to check report: ", err.Error())
		return
	}

	fmt.Println("verification passed!")
	fmt.Println("pubkey: ", "0x"+hex.EncodeToString(pubkeyBytes))
}

func getPubkey(opAddr string) ([]byte, error) {
	url := "https://" + opAddr + "/pubkey?raw=true"
	tlsConfig := &tls.Config{InsecureSkipVerify: true}
	return utils.HttpsGet(tlsConfig, url)
}
func getPubkeyReport(opAddr string) ([]byte, error) {
	url := "https://" + opAddr + "/pubkey-report?raw=true"
	tlsConfig := &tls.Config{InsecureSkipVerify: true}
	return utils.HttpsGet(tlsConfig, url)
}

func checkReport(report attestation.Report, signer, uniqueID, pubkey []byte) error {
	if !bytes.Equal(report.SignerID, signer) {
		return fmt.Errorf("signer-id not match! expected: %x, got: %x", signer, report.SignerID)
	}
	if !bytes.Equal(report.UniqueID, uniqueID) {
		return fmt.Errorf("unique-id not match! expected: %x, got: %x", uniqueID, report.UniqueID)
	}

	//if report.SecurityVersion < 2 {
	//	return errors.New("invalid security version")
	//}
	//if binary.LittleEndian.Uint16(report.ProductID) != 0x001 {
	//	return errors.New("invalid product")
	//}
	//if report.Debug {
	//	return errors.New("should not open debug")
	//}

	hash := sha256.Sum256(pubkey)
	if !bytes.Equal(report.Data[:len(hash)], hash[:]) {
		return fmt.Errorf("pubkey hash not match! expected: %x, got: %x", hash, report.Data[:len(hash)])
	}

	return nil
}
