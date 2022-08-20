package sbch

import (
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io/ioutil"
	"net/http"
	"strings"
)

var _ BasicRpcClient = (*basicRpcClient)(nil)

type basicRpcClient struct {
	url        string
	httpClient *http.Client
}

func (client *basicRpcClient) RpcURL() string {
	return client.url
}

func (client *basicRpcClient) SendPost(reqStr string) ([]byte, error) {
	body := strings.NewReader(reqStr)
	req, err := http.NewRequest("POST", client.url, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	resp, err := client.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	//if resp.StatusCode != http.StatusOK {
	//	return nil, fmt.Errorf("StatusCode: %d", resp.StatusCode)
	//}

	defer resp.Body.Close()
	respData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return respData, nil
}

func NewSimpleRpcClient(url string) *basicRpcClient {
	return &basicRpcClient{
		url:        url,
		httpClient: http.DefaultClient,
	}
}

func NewRpcClientOfNode(node NodeInfo) (*basicRpcClient, error) {
	certData, err := getCertData(node.CertUrl)
	if err != nil {
		return nil, err
	}

	certHash := sha256.Sum256(certData)
	if certHash != node.CertHash {
		return nil, errors.New("cert data and hash not match")
	}

	return NewRpcClientWithCertData(node.RpcUrl, certData)
}

func NewRpcClientWithCertData(rpcUrl string, caCert []byte) (*basicRpcClient, error) {
	caCertPool := x509.NewCertPool()
	if ok := caCertPool.AppendCertsFromPEM(caCert); !ok {
		return nil, errors.New("failed to parse cert data")
	}

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs: caCertPool,
			},
		},
	}
	return &basicRpcClient{
		url:        rpcUrl,
		httpClient: client,
	}, nil
}

func getCertData(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	return ioutil.ReadAll(resp.Body)
}
