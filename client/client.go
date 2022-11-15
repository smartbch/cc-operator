package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/smartbch/cc-operator/sbch"
)

type OpResp struct {
	Success bool            `json:"success"`
	Error   string          `json:"error,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
}

type Client struct {
	rpcUrl string
}

func NewClient(rpcUrl string) *Client {
	if strings.HasSuffix(rpcUrl, "/") {
		rpcUrl = rpcUrl[:len(rpcUrl)-1]
	}
	return &Client{rpcUrl: rpcUrl}
}

func (client *Client) RpcURL() string {
	return client.rpcUrl
}

func (client *Client) GetNodes() ([]sbch.NodeInfo, error) {
	var nodes []sbch.NodeInfo
	err := client.httpGet(context.Background(), "/nodes", &nodes)
	return nodes, err
}

func (client *Client) GetNewNodes() ([]sbch.NodeInfo, error) {
	var nodes []sbch.NodeInfo
	err := client.httpGet(context.Background(), "/newNodes", &nodes)
	return nodes, err
}

func (client *Client) Suspend(sig string, ts int64) error {
	pathAndQuery := fmt.Sprintf("/suspend?sig=%s&ts=%d", sig, ts)
	return client.httpGet(context.Background(), pathAndQuery, nil)
}

func (client *Client) httpGet(ctx context.Context, pathAndQuery string, result any) error {
	req, err := http.NewRequestWithContext(ctx, "GET", client.rpcUrl+pathAndQuery, nil)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	respData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var respObj OpResp
	err = json.Unmarshal(respData, &respObj)
	if err != nil {
		return err
	}

	if !respObj.Success {
		return errors.New(respObj.Error)
	}

	if result != nil {
		err = json.Unmarshal(respObj.Result, result)
		if err != nil {
			return err
		}
	}

	return nil
}
