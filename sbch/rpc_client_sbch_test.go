package sbch

import (
	"encoding/hex"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	testNodesGovAddr     = "0x8f1Cc6B6f276B776f3b7dB417c65fE356a164715"
	getNodeCountCallData = `{"jsonrpc": "2.0", "method": "eth_call", "params": [{"to": "0x8f1Cc6B6f276B776f3b7dB417c65fE356a164715", "data": "0x39bf397e"}, "latest"], "id":1}`
	getNodeCountRetData  = `{"jsonrpc":"2.0","id":1,"result":"0x0000000000000000000000000000000000000000000000000000000000000003"}`
	getNode0CallData     = `{"jsonrpc": "2.0", "method": "eth_call", "params": [{"to": "0x8f1Cc6B6f276B776f3b7dB417c65fE356a164715", "data": "0x1c53c2800000000000000000000000000000000000000000000000000000000000000000"}, "latest"], "id":1}`
	getNode0RetData      = `{"jsonrpc":"2.0","id":1,"result":"0x0000000000000000000000000000000000000000000000000000000000000001d86b49e3424e557beebf67bd06842cdb88e314c44887f3f265b7f81107dd61233132372e302e302e312f636572740000000000000000000000000000000000003132372e302e302e313a383534350000000000000000000000000000000000003132372e302e302e313a38353435000000000000000000000000000000000000"}`
	getNode1CallData     = `{"jsonrpc": "2.0", "method": "eth_call", "params": [{"to": "0x8f1Cc6B6f276B776f3b7dB417c65fE356a164715", "data": "0x1c53c2800000000000000000000000000000000000000000000000000000000000000001"}, "latest"], "id":1}`
	getNode1RetData      = `{"jsonrpc":"2.0","id":1,"result":"0x0000000000000000000000000000000000000000000000000000000000000002d86b49e3424e557beebf67bd06842cdb88e314c44887f3f265b7f81107dd62223132372e302e302e322f636572740000000000000000000000000000000000003132372e302e302e323a383534350000000000000000000000000000000000003132372e302e302e323a38353435000000000000000000000000000000000000"}`
	getNode2CallData     = `{"jsonrpc": "2.0", "method": "eth_call", "params": [{"to": "0x8f1Cc6B6f276B776f3b7dB417c65fE356a164715", "data": "0x1c53c2800000000000000000000000000000000000000000000000000000000000000002"}, "latest"], "id":1}`
	getNode2RetData      = `{"jsonrpc":"2.0","id":1,"result":"0x0000000000000000000000000000000000000000000000000000000000000003d86b49e3424e557beebf67bd06842cdb88e314c44887f3f265b7f81107dd63333132372e302e302e332f636572740000000000000000000000000000000000003132372e302e302e333a383534350000000000000000000000000000000000003132372e302e302e333a38353435000000000000000000000000000000000000"}`
)

type mockBasicRpcClient struct {
}

func (m mockBasicRpcClient) RpcURL() string {
	return "mockBasicRpcClient"
}

func (m mockBasicRpcClient) SendPost(reqStr string) ([]byte, error) {
	switch reqStr {
	case getNodeCountCallData:
		return []byte(getNodeCountRetData), nil
	case getNode0CallData:
		return []byte(getNode0RetData), nil
	case getNode1CallData:
		return []byte(getNode1RetData), nil
	case getNode2CallData:
		return []byte(getNode2RetData), nil
	default:
		return nil, errors.New("unknown req: " + reqStr)
	}
}

func TestGetNodeCount(t *testing.T) {
	c := sbchRpcClient{
		basicClient:  mockBasicRpcClient{},
		nodesGovAddr: testNodesGovAddr,
	}
	n, err := c.getNodeCount()
	require.NoError(t, err)
	require.Equal(t, uint64(3), n)
}

func TestGetNodeByIdx(t *testing.T) {
	c := sbchRpcClient{
		basicClient:  mockBasicRpcClient{},
		nodesGovAddr: testNodesGovAddr,
	}
	node, err := c.getNodeByIdx(1)
	require.NoError(t, err)
	require.Equal(t, uint64(2), node.ID)
	require.Equal(t, "d86b49e3424e557beebf67bd06842cdb88e314c44887f3f265b7f81107dd6222",
		hex.EncodeToString(node.CertHash[:]))
	require.Equal(t, "127.0.0.2/cert", node.CertUrl)
	require.Equal(t, "127.0.0.2:8545", node.RpcUrl)
}

func TestGetSbchdNodes(t *testing.T) {
	c := sbchRpcClient{
		basicClient:  mockBasicRpcClient{},
		nodesGovAddr: testNodesGovAddr,
	}
	nodes, err := c.GetSbchdNodes()
	require.NoError(t, err)
	require.Len(t, nodes, 3)
}
