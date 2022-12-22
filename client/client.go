package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/smartbch/cc-operator/operator"
	"github.com/smartbch/cc-operator/sbch"
	sbchrpctypes "github.com/smartbch/smartbch/rpc/types"
)

type OpResp struct {
	Success bool            `json:"success"`
	Error   string          `json:"error,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
}

type Client struct {
	rpcUrl     string
	reqTimeout time.Duration
}

func NewClient(rpcUrl string, reqTimeout time.Duration) *Client {
	if strings.HasSuffix(rpcUrl, "/") {
		rpcUrl = rpcUrl[:len(rpcUrl)-1]
	}
	return &Client{
		rpcUrl:     rpcUrl,
		reqTimeout: reqTimeout,
	}
}

func (client *Client) RpcURL() string {
	return client.rpcUrl
}

func (client *Client) GetInfo() (info operator.OpInfo, err error) {
	err = client.getWithTimeout("/info", &info)
	return
}

func (client *Client) GetNodes() ([]sbch.NodeInfo, error) {
	info, err := client.GetInfo()
	return info.CurrNodes, err
}
func (client *Client) GetNewNodes() ([]sbch.NodeInfo, error) {
	info, err := client.GetInfo()
	return info.NewNodes, err
}
func (client *Client) GetStatus() (string, error) {
	info, err := client.GetInfo()
	return info.Status, err
}

func (client *Client) GetRedeemingUtxosForOperators() (utxoList []*sbchrpctypes.UtxoInfo, err error) {
	err = client.getWithTimeout("/redeeming-utxos-for-operators", &utxoList)
	return
}

func (client *Client) GetRedeemingUtxosForMonitors() (utxoList []*sbchrpctypes.UtxoInfo, err error) {
	err = client.getWithTimeout("/redeeming-utxos-for-monitors", &utxoList)
	return
}

func (client *Client) GetToBeConvertedUtxosForOperators() (utxoList []*sbchrpctypes.UtxoInfo, err error) {
	err = client.getWithTimeout("/to-be-converted-utxos-for-operators", &utxoList)
	return
}

func (client *Client) GetToBeConvertedUtxosForMonitors() (utxoList []*sbchrpctypes.UtxoInfo, err error) {
	err = client.getWithTimeout("/to-be-converted-utxos-for-monitors", &utxoList)
	return
}

func (client *Client) GetPubkeyBytes() (result []byte, err error) {
	err = client.getWithTimeout("/pubkey", &result)
	return
}

func (client *Client) Suspend(sig string, ts int64) error {
	pathAndQuery := fmt.Sprintf("/suspend?sig=%s&ts=%d", sig, ts)
	return client.getWithTimeout(pathAndQuery, nil)
}

func (client *Client) getWithTimeout(pathAndQuery string, result any) error {
	ctx := context.Background()
	if client.reqTimeout > 0 {
		var cancelFn context.CancelFunc
		ctx, cancelFn = context.WithTimeout(ctx, client.reqTimeout)
		defer cancelFn()
	}

	return client.httpGet(ctx, pathAndQuery, result)
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
		if bzPtr, ok := result.(*[]byte); ok {
			*bzPtr = respObj.Result
		} else {
			err = json.Unmarshal(respObj.Result, result)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
