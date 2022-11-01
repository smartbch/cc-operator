package sbch

import (
	"encoding/hex"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	testNodesGovAddr     = "0x8f1Cc6B6f276B776f3b7dB417c65fE356a164715"
	getNodeCountCallData = `{"jsonrpc":"2.0","id":1,"method":"eth_call","params":[{"data":"0x39bf397e","from":"0x0000000000000000000000000000000000000000","to":"0x8f1cc6b6f276b776f3b7db417c65fe356a164715"},"latest"]}`
	getNodeCountRetData  = `{"jsonrpc":"2.0","id":1,"result":"0x0000000000000000000000000000000000000000000000000000000000000003"}`
	getNode0CallData     = `{"jsonrpc":"2.0","id":1,"method":"eth_call","params":[{"data":"0x1c53c2800000000000000000000000000000000000000000000000000000000000000000","from":"0x0000000000000000000000000000000000000000","to":"0x8f1cc6b6f276b776f3b7db417c65fe356a164715"},"latest"]}`
	getNode0RetData      = `{"jsonrpc":"2.0","id":1,"result":"0x0000000000000000000000000000000000000000000000000000000000000001d86b49e3424e557beebf67bd06842cdb88e314c44887f3f265b7f81107dd61233132372e302e302e312f636572740000000000000000000000000000000000003132372e302e302e313a383534350000000000000000000000000000000000003132372e302e302e313a38353435000000000000000000000000000000000000"}`
	getNode1CallData     = `{"jsonrpc":"2.0","id":1,"method":"eth_call","params":[{"data":"0x1c53c2800000000000000000000000000000000000000000000000000000000000000001","from":"0x0000000000000000000000000000000000000000","to":"0x8f1cc6b6f276b776f3b7db417c65fe356a164715"},"latest"]}`
	getNode1RetData      = `{"jsonrpc":"2.0","id":1,"result":"0x0000000000000000000000000000000000000000000000000000000000000002d86b49e3424e557beebf67bd06842cdb88e314c44887f3f265b7f81107dd62223132372e302e302e322f636572740000000000000000000000000000000000003132372e302e302e323a383534350000000000000000000000000000000000003132372e302e302e323a38353435000000000000000000000000000000000000"}`
	getNode2CallData     = `{"jsonrpc":"2.0","id":1,"method":"eth_call","params":[{"data":"0x1c53c2800000000000000000000000000000000000000000000000000000000000000002","from":"0x0000000000000000000000000000000000000000","to":"0x8f1cc6b6f276b776f3b7db417c65fe356a164715"},"latest"]}`
	getNode2RetData      = `{"jsonrpc":"2.0","id":1,"result":"0x0000000000000000000000000000000000000000000000000000000000000003d86b49e3424e557beebf67bd06842cdb88e314c44887f3f265b7f81107dd63333132372e302e302e332f636572740000000000000000000000000000000000003132372e302e302e333a383534350000000000000000000000000000000000003132372e302e302e333a38353435000000000000000000000000000000000000"}`

	getRedeemingUtxosReq     = `{"jsonrpc":"2.0","id":1,"method":"sbch_getRedeemingUtxosForOperators"}`
	getToBeConvertedUtxosReq = `{"jsonrpc":"2.0","id":1,"method":"sbch_getToBeConvertedUtxosForOperators"}`
	getUtxosResp0            = `{"jsonrpc":"2.0","id":1,"result":null}`
	getUtxosResp             = `{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
	"infos": [
		{
		  "owner_of_lost": "0x1100000000000000000000000000000000000000",
		  "covenant_addr": "0x1200000000000000000000000000000000000000",
		  "is_redeemed": false,
		  "redeem_target": "0x1300000000000000000000000000000000000000",
		  "expected_sign_time": 1665561734,
		  "txid": "0x1400000000000000000000000000000000000000000000000000000000000000",
		  "index": 21,
		  "amount": "0x16",
		  "tx_sig_hash": "0x17"
		},
		{
		  "owner_of_lost": "0x2100000000000000000000000000000000000000",
		  "covenant_addr": "0x2200000000000000000000000000000000000000",
		  "is_redeemed": true,
		  "redeem_target": "0x2300000000000000000000000000000000000000",
		  "expected_sign_time": 1665561734,
		  "txid": "0x2400000000000000000000000000000000000000000000000000000000000000",
		  "index": 37,
		  "amount": "0x26",
		  "tx_sig_hash": "0x27"
		}
	  ]
  	}
}`
)

func fakeServerHandler(w http.ResponseWriter, r *http.Request) {
	req, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}

	resp, err := fakeServerLogic(string(req))
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}
	w.Write(resp)
}

func fakeServerLogic(reqStr string) ([]byte, error) {
	reqStr = regexp.MustCompile(`"id":\d+`).ReplaceAllString(reqStr, `"id":1`)
	switch reqStr {
	case getNodeCountCallData:
		return []byte(getNodeCountRetData), nil
	case getNode0CallData:
		return []byte(getNode0RetData), nil
	case getNode1CallData:
		return []byte(getNode1RetData), nil
	case getNode2CallData:
		return []byte(getNode2RetData), nil
	case getRedeemingUtxosReq:
		return []byte(getUtxosResp), nil
	case getToBeConvertedUtxosReq:
		return []byte(getUtxosResp), nil
	default:
		fmt.Println(reqStr)
		return nil, errors.New("unknown req: " + reqStr)
	}
}

func TestGetNodeCount(t *testing.T) {
	fakeServer := httptest.NewServer(http.HandlerFunc(fakeServerHandler))
	defer fakeServer.Close()
	c := NewSimpleRpcClient(testNodesGovAddr, fakeServer.URL)
	n, err := c.getNodeCount()
	require.NoError(t, err)
	require.Equal(t, uint64(3), n)
}

func TestGetNodeByIdx(t *testing.T) {
	fakeServer := httptest.NewServer(http.HandlerFunc(fakeServerHandler))
	defer fakeServer.Close()
	c := NewSimpleRpcClient(testNodesGovAddr, fakeServer.URL)
	node, err := c.getNodeByIdx(1)
	require.NoError(t, err)
	require.Equal(t, uint64(2), node.ID)
	require.Equal(t, "d86b49e3424e557beebf67bd06842cdb88e314c44887f3f265b7f81107dd6222",
		hex.EncodeToString(node.CertHash[:]))
	require.Equal(t, "127.0.0.2/cert", node.CertUrl)
	require.Equal(t, "127.0.0.2:8545", node.RpcUrl)
}

func TestGetSbchdNodes(t *testing.T) {
	fakeServer := httptest.NewServer(http.HandlerFunc(fakeServerHandler))
	defer fakeServer.Close()

	c1 := NewSimpleRpcClient(testNodesGovAddr, fakeServer.URL)
	c2 := NewClusterClient(testNodesGovAddr, []string{fakeServer.URL, fakeServer.URL})

	for _, c := range []RpcClient{c1, c2} {
		nodes, err := c.GetSbchdNodes()
		require.NoError(t, err)
		require.Len(t, nodes, 3)
	}
}

func TestGetRedeemingUtxoSigHashes(t *testing.T) {
	fakeServer := httptest.NewServer(http.HandlerFunc(fakeServerHandler))
	defer fakeServer.Close()

	c1 := NewSimpleRpcClient(testNodesGovAddr, fakeServer.URL)
	c2 := NewClusterClient(testNodesGovAddr, []string{fakeServer.URL, fakeServer.URL})

	for _, c := range []RpcClient{c1, c2} {
		hashes, err := c.GetRedeemingUtxoSigHashes()
		require.NoError(t, err)
		require.Equal(t, hashes, []string{"17", "27"})
	}
}

func TestGetToBeConvertedUtxoSigHashes(t *testing.T) {
	fakeServer := httptest.NewServer(http.HandlerFunc(fakeServerHandler))
	defer fakeServer.Close()

	c1 := NewSimpleRpcClient(testNodesGovAddr, fakeServer.URL)
	c2 := NewClusterClient(testNodesGovAddr, []string{fakeServer.URL, fakeServer.URL})

	for _, c := range []RpcClient{c1, c2} {
		hashes, err := c.GetToBeConvertedUtxoSigHashes()
		require.NoError(t, err)
		require.Equal(t, hashes, []string{"17", "27"})
	}
}
