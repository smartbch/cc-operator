package sbch

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"

	gethcmn "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

const (
	testNodesGovAddr     = "0x8f1Cc6B6f276B776f3b7dB417c65fE356a164715"
	getNodeCountCallData = `{"jsonrpc":"2.0","id":1,"method":"eth_call","params":[{"data":"0x39bf397e","from":"0x0000000000000000000000000000000000000000","to":"0x8f1cc6b6f276b776f3b7db417c65fe356a164715"},"latest"]}`
	getNodeCountRetData  = `{"jsonrpc":"2.0","id":1,"result":"0x0000000000000000000000000000000000000000000000000000000000000003"}`
	getNode0CallData     = `{"jsonrpc":"2.0","id":1,"method":"eth_call","params":[{"data":"0x1c53c2800000000000000000000000000000000000000000000000000000000000000000","from":"0x0000000000000000000000000000000000000000","to":"0x8f1cc6b6f276b776f3b7db417c65fe356a164715"},"latest"]}`
	getNode0RetData      = `{"jsonrpc":"2.0","id":1,"result":"0x0000000000000000000000000000000000000000000000000000000000000001d86b49e3424e557beebf67bd06842cdb88e314c44887f3f265b7f81107dd61233132372e302e302e313a383534350000000000000000000000000000000000003132372e302e302e313a38353435000000000000000000000000000000000000"}`
	getNode1CallData     = `{"jsonrpc":"2.0","id":1,"method":"eth_call","params":[{"data":"0x1c53c2800000000000000000000000000000000000000000000000000000000000000001","from":"0x0000000000000000000000000000000000000000","to":"0x8f1cc6b6f276b776f3b7db417c65fe356a164715"},"latest"]}`
	getNode1RetData      = `{"jsonrpc":"2.0","id":1,"result":"0x0000000000000000000000000000000000000000000000000000000000000002d86b49e3424e557beebf67bd06842cdb88e314c44887f3f265b7f81107dd62223132372e302e302e323a383534350000000000000000000000000000000000003132372e302e302e323a38353435000000000000000000000000000000000000"}`
	getNode2CallData     = `{"jsonrpc":"2.0","id":1,"method":"eth_call","params":[{"data":"0x1c53c2800000000000000000000000000000000000000000000000000000000000000002","from":"0x0000000000000000000000000000000000000000","to":"0x8f1cc6b6f276b776f3b7db417c65fe356a164715"},"latest"]}`
	getNode2RetData      = `{"jsonrpc":"2.0","id":1,"result":"0x0000000000000000000000000000000000000000000000000000000000000003d86b49e3424e557beebf67bd06842cdb88e314c44887f3f265b7f81107dd63333132372e302e302e333a383534350000000000000000000000000000000000003132372e302e302e333a38353435000000000000000000000000000000000000"}`

	getRpcPubkeyReq  = `{"jsonrpc":"2.0","id":1,"method":"sbch_getRpcPubkey"}`
	getRpcPubkeyResp = `{"jsonrpc":"2.0","id":1,"result":"049791f89a61c582c3584b3409147ace7df3c41bdef296b77b0d50d95bc6a10d8d378ed2167c43b21265ea819896c2fdddc41581e1deceb029f79675cfcb46f56c"}`

	getRedeemingUtxosReq     = `{"jsonrpc":"2.0","id":1,"method":"sbch_getRedeemingUtxosForOperators"}`
	getToBeConvertedUtxosReq = `{"jsonrpc":"2.0","id":1,"method":"sbch_getToBeConvertedUtxosForOperators"}`
	getUtxosResp0            = `{"jsonrpc":"2.0","id":1,"result":null}`
	getUtxosResp             = `{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
	"signature": "0x828249d898a94021df93f143ad0028a1f1812296b40336b001f44cc5352da23e51c8539e284828e403b16f6bfd05efa7d81f797653677d1b235da58b71c17316",
	"infos": [
		{
		  "ownerOfLost": "0x1100000000000000000000000000000000000000",
		  "covenantAddr": "0x1200000000000000000000000000000000000000",
		  "isRedeemed": false,
		  "redeemTarget": "0x1300000000000000000000000000000000000000",
		  "expectedSignTime": 1665561734,
		  "txid": "0x1400000000000000000000000000000000000000000000000000000000000000",
		  "index": 21,
		  "amount": "0x16",
		  "txSigHash": "0x17"
		},
		{
		  "ownerOfLost": "0x2100000000000000000000000000000000000000",
		  "covenantAddr": "0x2200000000000000000000000000000000000000",
		  "isRedeemed": true,
		  "redeemTarget": "0x2300000000000000000000000000000000000000",
		  "expectedSignTime": 1665561734,
		  "txid": "0x2400000000000000000000000000000000000000000000000000000000000000",
		  "index": 37,
		  "amount": "0x26",
		  "txSigHash": "0x27"
		}
	  ]
  	}
}`
)

var testKey, _ = crypto.HexToECDSA("5c4870e61de2941b517030005d576f1460e8c5eb82c2e02b3b3b599bd01287b0")

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
	case getRpcPubkeyReq:
		return []byte(getRpcPubkeyResp), nil
	default:
		fmt.Println(reqStr)
		return nil, errors.New("unknown req: " + reqStr)
	}
}

func TestGetNodeCount(t *testing.T) {
	fakeServer := httptest.NewServer(http.HandlerFunc(fakeServerHandler))
	defer fakeServer.Close()
	c := NewSimpleRpcClient(testNodesGovAddr, fakeServer.URL, 0)
	n, err := c.getNodeCount(context.Background())
	require.NoError(t, err)
	require.Equal(t, uint64(3), n)
}

func TestGetNodeByIdx(t *testing.T) {
	fakeServer := httptest.NewServer(http.HandlerFunc(fakeServerHandler))
	defer fakeServer.Close()
	c := NewSimpleRpcClient(testNodesGovAddr, fakeServer.URL, 0)
	node, err := c.getNodeByIdx(1, context.Background())
	require.NoError(t, err)
	require.Equal(t, uint64(2), node.ID)
	require.Equal(t, "d86b49e3424e557beebf67bd06842cdb88e314c44887f3f265b7f81107dd6222",
		hex.EncodeToString(node.PbkHash[:]))
	require.Equal(t, "127.0.0.2:8545", node.RpcUrl)
}

func TestGetSbchdNodes(t *testing.T) {
	fakeServer := httptest.NewServer(http.HandlerFunc(fakeServerHandler))
	defer fakeServer.Close()

	c1 := NewSimpleRpcClient(testNodesGovAddr, fakeServer.URL, 0)
	c2 := &ClusterClient{clients: []RpcClient{c1, c1}}

	for _, c := range []RpcClient{c1, c2} {
		nodes, err := c.GetSbchdNodes()
		require.NoError(t, err)
		require.Len(t, nodes, 3)
	}
}

func TestGetRedeemingUtxoSigHashes(t *testing.T) {
	fakeServer := httptest.NewServer(http.HandlerFunc(fakeServerHandler))
	defer fakeServer.Close()

	c1 := NewSimpleRpcClient(testNodesGovAddr, fakeServer.URL, 0)
	c2 := &ClusterClient{clients: []RpcClient{c1, c1}}

	for _, c := range []RpcClient{c1, c2} {
		utxos, err := c.GetRedeemingUtxosForOperators()
		require.NoError(t, err)
		require.Len(t, utxos, 2)
	}
}

func TestGetToBeConvertedUtxoSigHashes(t *testing.T) {
	fakeServer := httptest.NewServer(http.HandlerFunc(fakeServerHandler))
	defer fakeServer.Close()

	c1 := NewSimpleRpcClient(testNodesGovAddr, fakeServer.URL, 0)
	c2 := &ClusterClient{clients: []RpcClient{c1, c1}}

	for _, c := range []RpcClient{c1, c2} {
		utxos, err := c.GetToBeConvertedUtxosForOperators()
		require.NoError(t, err, c.RpcURL())
		require.Len(t, utxos, 2)
	}
}

func TestSig(t *testing.T) {
	keyHex := hex.EncodeToString(crypto.FromECDSA(testKey))
	fmt.Println("privkey:", keyHex)
	pubkey := crypto.FromECDSAPub(&testKey.PublicKey)
	fmt.Println("pubkey:", hex.EncodeToString(pubkey))

	hash := gethcmn.FromHex("41f1d614287f32cec18860d4c5344966784576457b472a3d1bd7bd9430fd51e1")

	sig, _ := crypto.Sign(hash, testKey)
	fmt.Println("sig:", hex.EncodeToString(sig[:64]))

	ok := crypto.VerifySignature(pubkey, hash, sig[:64])
	fmt.Println(ok)
}
