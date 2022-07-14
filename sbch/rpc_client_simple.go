package sbch

import (
	"io/ioutil"
	"net/http"
	"strings"
)

var _ BasicRpcClient = (*simpleRpcClient)(nil)

type simpleRpcClient struct {
	url string
}

func wrapSimpleRpcClient(url string) RpcClient {
	return &rpcClientWrapper{
		client: &simpleRpcClient{
			url: url,
		},
	}
}

func (client *simpleRpcClient) RpcURL() string {
	return client.url
}

func (client *simpleRpcClient) SendPost(reqStr string) ([]byte, error) {
	return sendRequest(client.url, reqStr)
}

func sendRequest(url, bodyStr string) ([]byte, error) {
	body := strings.NewReader(bodyStr)
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	respData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return respData, nil
}
