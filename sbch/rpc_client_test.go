package sbch

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
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
	getRpcPubkeyResp = `{"jsonrpc":"2.0","id":1,"result":"04b48c5986dcdd12746db4fdc14a9546c220a91e230a2204fc279acddc4387a0b211b7615c2e971e25647ab46a80a5c6b269d86ccfcada4719d69b3a82992c8793"}`

	getRedeemingUtxosReq     = `{"jsonrpc":"2.0","id":1,"method":"sbch_getRedeemingUtxosForOperators"}`
	getToBeConvertedUtxosReq = `{"jsonrpc":"2.0","id":1,"method":"sbch_getToBeConvertedUtxosForOperators"}`
	getUtxosResp             = `{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
	"signature": "0x94ee995e0006ea0bf9b7e47c6b7153d9f62bab44b41e19ea154a415a93ee686c67d6ec3ce543030a8054efa3ecde24928cb7e26b0dee82c56be294af447279a3",
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
	getCcInfoReq  = `{"jsonrpc":"2.0","id":1,"method":"sbch_getCcInfo"}`
	getCcInfoResp = `{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "monitorsWithPauseCommand": null,
    "operators": [
      {
        "address": "0x324e132eac14affea5df80e04b919d97035dc952",
        "pubkey": "0x02d86b49e3424e557beebf67bd06842cdb88e314c44887f3f265b7f81107dd6994",
        "rpcUrl": "https://3.1.26.210:8801",
        "intro": "shagate2-testnet1-op1"
      },
      {
        "address": "0xac63c42b6007122d554c586d3766cb44924ea9fd",
        "pubkey": "0x035c0a0cb8987290ea0a7a926e8aa8978ac042b4c0be8553eb4422461ce1a17cd8",
        "rpcUrl": "https://3.1.26.210:8802",
        "intro": "shagate2-testnet1-op2"
      },
      {
        "address": "0x8b1c9950aa5c6ff3bb038ff31878dd6a268958f8",
        "pubkey": "0x03fdec69ef6ec640264045229ca7cf0f170927b87fc8d2047844f8a766ead467e4",
        "rpcUrl": "https://3.1.26.210:8803",
        "intro": "shagate2-testnet1-op3"
      },
      {
        "address": "0x61d8f9889ae4e30d5d7ce7151711ae9c47079fae",
        "pubkey": "0x038fd3d33474e1bd453614f85d8fb1edecae92255867d18a9048669119fb710af5",
        "rpcUrl": "https://3.1.26.210:8804",
        "intro": "shagate2-testnet1-op4"
      },
      {
        "address": "0xfb25b245144b523f3b4f70ed9792d2cb2516a499",
        "pubkey": "0x0394ec324d59305638ead14b4f4da9a50c793f1e328e180f92c04a4990bb573af1",
        "rpcUrl": "https://3.1.26.210:8805",
        "intro": "shagate2-testnet1-op5"
      },
      {
        "address": "0x28305473e074d90ab25ad31d9df904c7a7d31922",
        "pubkey": "0x0271ea0c254ebbb7ed78668ba8653abe222b9f7177642d3a75709d95912a8d9d2c",
        "rpcUrl": "https://3.1.26.210:8806",
        "intro": "shagate2-testnet1-op6"
      },
      {
        "address": "0x88c64f117172cdb67bc29c37bc983ce7fc6d5731",
        "pubkey": "0x02fbbc3870035c2ee30cfa3102aff15e58bdfc0d0f95998cd7e1eeebc09cdb6873",
        "rpcUrl": "https://3.1.26.210:8807",
        "intro": "shagate2-testnet1-op7"
      },
      {
        "address": "0x250f6d5b3a5f4d29fe5550bb12aac3b1d9d1fd97",
        "pubkey": "0x0386f450b1bee3b220c6a9a25515f15f05bd80a23e5f707873dfbac52db933b27d",
        "rpcUrl": "https://3.1.26.210:8808",
        "intro": "shagate2-testnet1-op8"
      },
      {
        "address": "0xf84f85fe48c08d8f8a1f46226a60978249c58043",
        "pubkey": "0x03bfe6f6ecb5e10662481aeb6f6408db2a32b9b86a660acbb8c5374dbb976e53ca",
        "rpcUrl": "https://3.1.26.210:8809",
        "intro": "shagate2-testnet1-op9"
      },
      {
        "address": "0x417d9f3fe6ebf314f4a14916e8ac2b4d0a27e958",
        "pubkey": "0x03883b732620e238e74041e5fab900234dc80f7a48d56a1bf41e8523c4661f8243",
        "rpcUrl": "https://3.1.26.210:8810",
        "intro": "shagate2-testnet1-op0"
      }
    ],
    "monitors": [
      {
        "address": "0x765fd1f0e3d125b36de29b5f88295a247814276e",
        "pubkey": "0x024a899d685daf6b1999a5c8f2fd3c9ed640d58e92fd0e00cf87cacee8ff1504b8",
        "intro": "shagate2-testnet1-mo1"
      },
      {
        "address": "0xfb6ffd0802f41f387bca3168f676a66dc2216d61",
        "pubkey": "0x0374ac9ab3415253dbb7e29f46a69a3e51b5d2d66f125b0c9f2dc990b1d2e87e17",
        "intro": "shagate2-testnet1-mo2"
      },
      {
        "address": "0xdcb8fc457b40cdb5e2338e5e53fc463ede0ab73e",
        "pubkey": "0x024cc911ba9d2c7806a217774618b7ba4848ccd33fe664414fc3144d144cdebf7b",
        "intro": "shagate2-testnet1-mo3"
      }
    ],
    "oldOperators": null,
    "oldMonitors": null,
    "lastCovenantAddress": "0x0000000000000000000000000000000000000000",
    "currCovenantAddress": "0x6Ad3f81523c87aa17f1dFA08271cF57b6277C98e",
    "lastRescannedHeight": 1534590,
    "rescannedHeight": 1534594,
    "rescanTime": 1673230046,
    "utxoAlreadyHandled": true,
    "latestEpochHandled": 7,
    "covenantAddrLastChangeTime": 0,
    "signature": "0x3cada8f3e79dbc7e1cfd72656c1d3ba233db80640e5912b20b62137422d18ac010051f995f8c508d113e908bf3513610bf7f59e50638c9df622adb38115bb94101"
  }
}`
)

var testKey, _ = crypto.HexToECDSA("14542dfb851d5a19b0b8f4951d3e392819c87d83bd75bbedaeb99b4d34086aad")

func fakeServerHandler(w http.ResponseWriter, r *http.Request) {
	req, err := io.ReadAll(r.Body)
	if err != nil {
		_, _ = w.Write([]byte(err.Error()))
		return
	}

	resp, err := fakeServerLogic(string(req))
	if err != nil {
		_, _ = w.Write([]byte(err.Error()))
		return
	}
	_, _ = w.Write(resp)
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
	case getCcInfoReq:
		return []byte(getCcInfoResp), nil
	default:
		fmt.Println(reqStr)
		return nil, errors.New("unknown req: " + reqStr)
	}
}

func TestGetNodeCount(t *testing.T) {
	fakeServer := httptest.NewServer(http.HandlerFunc(fakeServerHandler))
	defer fakeServer.Close()
	c, _ := NewSimpleRpcClient(testNodesGovAddr, fakeServer.URL, 0)
	n, err := c.getNodeCount(context.Background())
	require.NoError(t, err)
	require.Equal(t, uint64(3), n)
}

func TestGetNodeByIdx(t *testing.T) {
	fakeServer := httptest.NewServer(http.HandlerFunc(fakeServerHandler))
	defer fakeServer.Close()
	c, _ := NewSimpleRpcClient(testNodesGovAddr, fakeServer.URL, 0)
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

	c1, _ := NewSimpleRpcClient(testNodesGovAddr, fakeServer.URL, 0)
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

	c1, _ := NewSimpleRpcClient(testNodesGovAddr, fakeServer.URL, 0)
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

	c1, _ := NewSimpleRpcClient(testNodesGovAddr, fakeServer.URL, 0)
	c2 := &ClusterClient{clients: []RpcClient{c1, c1}}

	for _, c := range []RpcClient{c1, c2} {
		utxos, err := c.GetToBeConvertedUtxosForOperators()
		require.NoError(t, err, c.RpcURL())
		require.Len(t, utxos, 2)
	}
}

func TestGetMonitors(t *testing.T) {
	fakeServer := httptest.NewServer(http.HandlerFunc(fakeServerHandler))
	defer fakeServer.Close()

	c1, _ := NewSimpleRpcClient(testNodesGovAddr, fakeServer.URL, 0)
	c2 := &ClusterClient{clients: []RpcClient{c1, c1}}

	expectedMonitors := []gethcmn.Address{
		gethcmn.HexToAddress("0x765fd1f0e3d125b36de29b5f88295a247814276e"),
		gethcmn.HexToAddress("0xfb6ffd0802f41f387bca3168f676a66dc2216d61"),
		gethcmn.HexToAddress("0xdcb8fc457b40cdb5e2338e5e53fc463ede0ab73e"),
	}

	for _, c := range []RpcClient{c1, c2} {
		monitors, err := c.GetMonitors()
		require.NoError(t, err)
		require.Equal(t, expectedMonitors, monitors)
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
