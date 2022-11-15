package proxy

import (
	"bytes"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/edgelesssys/ego/attestation"
	"github.com/edgelesssys/ego/eclient"

	"github.com/smartbch/cc-operator/utils"
)

func StartProxyServerWithCert(
	operatorUrl, operatorName string, signer, uniqueID []byte,
	listenAddr, certFile, keyFile string) {

	proxy := newProxy(operatorUrl, operatorName, signer, uniqueID)
	server := http.Server{
		Addr:         listenAddr,
		Handler:      proxy,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 5 * time.Second,
	}
	fmt.Println("cc-operator proxy listening at:", listenAddr, "...")
	err := server.ListenAndServeTLS(certFile, keyFile)
	if err != nil {
		panic(err)
	}
}

func StartProxyServerWithName(operatorUrl, operatorName string, signer, uniqueID []byte,
	listenAddr, proxyName string) {

	proxy := newProxy(operatorUrl, operatorName, signer, uniqueID)
	_, _, tlsCfg := utils.CreateCertificate(proxyName)

	server := http.Server{
		Addr:         listenAddr,
		Handler:      proxy,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 5 * time.Second,
		TLSConfig:    &tlsCfg,
	}
	fmt.Println("cc-operator proxy listening at:", listenAddr, "...")
	err := server.ListenAndServeTLS("", "")
	if err != nil {
		panic(err)
	}
}

func newProxy(operatorUrl, operatorName string, signer, uniqueID []byte) *httputil.ReverseProxy {
	certBytes := verifyOperator(operatorUrl, signer, uniqueID)
	cert, _ := x509.ParseCertificate(certBytes)
	tlsConfig := &tls.Config{RootCAs: x509.NewCertPool(), ServerName: operatorName}
	tlsConfig.RootCAs.AddCert(cert)

	targetUrl, err := url.Parse(operatorUrl)
	if err != nil {
		panic(err)
	}

	proxy := httputil.NewSingleHostReverseProxy(targetUrl)
	proxy.Transport = &http.Transport{TLSClientConfig: tlsConfig}
	return proxy
}

func verifyOperator(operatorUrl string, signer, uniqueID []byte) []byte {
	if !strings.HasPrefix(operatorUrl, "https://") {
		operatorUrl = "https://" + operatorUrl
	}
	tlsConfig := &tls.Config{InsecureSkipVerify: true}

	var certStr string
	var reportStr string
	var certBytes []byte
	var reportBytes []byte
	var err error

	certStr = string(utils.HttpsGet(tlsConfig, operatorUrl+"/cert?raw=true"))
	reportStr = string(utils.HttpsGet(tlsConfig, operatorUrl+"/cert-report?raw=true"))

	certBytes, err = hex.DecodeString(certStr)
	if err != nil {
		panic(err)
	}
	reportBytes, err = hex.DecodeString(reportStr)
	if err != nil {
		panic(err)
	}
	if err := verifyReport(reportBytes, certBytes, signer, uniqueID); err != nil {
		panic(err)
	}
	fmt.Printf("verify operator:%s passed\n", operatorUrl)
	return certBytes
}

func verifyReport(reportBytes, certBytes, signer, uniqueID []byte) error {
	report, err := eclient.VerifyRemoteReport(reportBytes)
	if err != nil {
		return err
	}
	return checkReport(report, certBytes, signer, uniqueID)
}

func checkReport(report attestation.Report, certBytes, signer, uniqueID []byte) error {
	hash := sha256.Sum256(certBytes)
	if !bytes.Equal(report.Data[:len(hash)], hash[:]) {
		return errors.New("report data does not match the certificate's hash")
	}
	if !bytes.Equal(report.UniqueID, uniqueID) {
		return errors.New("invalid unique id")
	}
	if report.SecurityVersion < 2 {
		return errors.New("invalid security version")
	}
	if binary.LittleEndian.Uint16(report.ProductID) != 0x001 {
		return errors.New("invalid product")
	}
	if !bytes.Equal(report.SignerID, signer) {
		return errors.New("invalid signer")
	}
	if report.Debug {
		return errors.New("should not open debug")
	}
	return nil
}
