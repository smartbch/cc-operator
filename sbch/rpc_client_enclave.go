package sbch

import (
	"bytes"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/binary"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/edgelesssys/ego/attestation"
	"github.com/edgelesssys/ego/attestation/tcbstatus"
	"github.com/edgelesssys/ego/eclient"
)

var _ BasicRpcClient = (*enclaveRpcClient)(nil)

type enclaveRpcClient struct {
	enclaveNodeInfo EnclaveNodeInfo
	tlsConfig       *tls.Config
}

func wrapEnclaveRpcClient(enclaveNodeInfo EnclaveNodeInfo) RpcClient {
	return &rpcClientWrapper{
		client: newEnclaveRpcClient(enclaveNodeInfo),
	}
}
func newEnclaveRpcClient(enclaveNodeInfo EnclaveNodeInfo) *enclaveRpcClient {
	return &enclaveRpcClient{
		enclaveNodeInfo: enclaveNodeInfo,
	}
}

func (client *enclaveRpcClient) RpcURL() string {
	return client.enclaveNodeInfo.ServerAddr
}

func (client *enclaveRpcClient) SendPost(reqStr string) ([]byte, error) {
	return client.httpPost(reqStr)
}

func (client *enclaveRpcClient) httpPost(bodyStr string) ([]byte, error) {
	if client.tlsConfig == nil {
		return nil, errors.New("not attested")
	}

	url := "https://" + client.enclaveNodeInfo.ServerAddr
	return httpPost(client.tlsConfig, url, bodyStr)
}

func (client *enclaveRpcClient) remoteAttest() error {
	url := "https://" + client.enclaveNodeInfo.ServerAddr

	// Get server certificate and its report. Skip TLS certificate verification because
	// the certificate is self-signed and we will verify it using the report instead.
	tlsConfig := &tls.Config{InsecureSkipVerify: true}
	certBytes := httpGet(tlsConfig, url+"/cert")
	reportBytes := httpGet(tlsConfig, url+"/report")

	if err := client.verifyReport(reportBytes, certBytes); err != nil {
		return err
	}

	// Create a TLS config that uses the server certificate as root
	// CA so that future connections to the server can be verified.
	cert, _ := x509.ParseCertificate(certBytes)
	tlsConfig = &tls.Config{RootCAs: x509.NewCertPool(), ServerName: "localhost"}
	tlsConfig.RootCAs.AddCert(cert)

	client.tlsConfig = tlsConfig
	return nil
}

func (client *enclaveRpcClient) verifyReport(reportBytes, certBytes []byte) error {
	report, err := eclient.VerifyRemoteReport(reportBytes)
	if err == attestation.ErrTCBLevelInvalid {
		fmt.Printf("Warning: TCB level is invalid: %v\n%v\n", report.TCBStatus, tcbstatus.Explain(report.TCBStatus))
		fmt.Println("We'll ignore this issue in this sample. For an app that should run in production, you must decide which of the different TCBStatus values are acceptable for you to continue.")
	} else if err != nil {
		return err
	}

	hash := sha256.Sum256(certBytes)
	if !bytes.Equal(report.Data[:len(hash)], hash[:]) {
		return errors.New("report data does not match the certificate's hash")
	}

	// You can either verify the UniqueID or the tuple (SignerID, ProductID, SecurityVersion, Debug).

	if report.SecurityVersion < client.enclaveNodeInfo.SecurityVersion {
		return errors.New("invalid security version")
	}
	if binary.LittleEndian.Uint16(report.ProductID) != client.enclaveNodeInfo.ProductID {
		return errors.New("invalid product")
	}
	if !bytes.Equal(report.SignerID, client.enclaveNodeInfo.SignerID) {
		return errors.New("invalid signer")
	}

	// For production, you must also verify that report.Debug == false
	if report.Debug {
		return errors.New("debug != false")
	}

	return nil
}

func httpGet(tlsConfig *tls.Config, url string) []byte {
	client := http.Client{Transport: &http.Transport{TLSClientConfig: tlsConfig}}
	resp, err := client.Get(url)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		panic(resp.Status)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	return body
}

func httpPost(tlsConfig *tls.Config, url, bodyStr string) ([]byte, error) {
	body := strings.NewReader(bodyStr)
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return nil, err
	}

	client := http.Client{Transport: &http.Transport{TLSClientConfig: tlsConfig}}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("status: " + resp.Status)
	}

	respData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return respData, nil
}
