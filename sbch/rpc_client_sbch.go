package sbch

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/holiman/uint256"
)

const (
	nodesGovContractAddr = "0x0000000000000000000000000000000000000000" // TODO

	getNodeCountSel = "0x39bf397e" // ethers.utils.id('getNodeCount()')
	getNodeByIdxSel = "0x1c53c280" // ethers.utils.id('nodes(uint256)')
)

const (
	reqRedeemingUTXOs = `{"jsonrpc": "2.0", "method": "sbch_getRedeemingUtxosForOperators", "params": [], "id":1}`
	reqCallTmpl       = `{"jsonrpc": "2.0", "method": "eth_call", "params": [{"to": "%s", "data": "%s"}, "latest"], "id":1}`
)

var _ RpcClient = (*sbchRpcClient)(nil)

// smartBCH JSON-RPC client
type sbchRpcClient struct {
	basicClient BasicRpcClient
}

func NewSimpleRpcClient(url string) RpcClient {
	return &sbchRpcClient{
		basicClient: newBasicRpcClient(url),
	}
}

func (client sbchRpcClient) GetOperatorSigHashes() ([]string, error) {
	resp, err := client.basicClient.SendPost(reqRedeemingUTXOs)
	if err != nil {
		return nil, err
	}

	var utxoInfos []UtxoInfo
	err = json.Unmarshal(resp, &utxoInfos)
	if err != nil {
		return nil, err
	}

	sigHashes := make([]string, len(utxoInfos))
	for i, utxoInfo := range utxoInfos {
		sigHashes[i] = utxoInfo.TxSigHash.String()
	}

	return sigHashes, nil
}

func (client sbchRpcClient) GetSbchdNodes() ([]NodeInfo, error) {
	nodeCount, err := client.getNodeCount()
	if err != nil {
		return nil, err
	}

	nodes := make([]NodeInfo, nodeCount)
	for i := uint64(0); i < nodeCount; i++ {
		nodes[i], err = client.getNodeByIdx(i)
		if err != nil {
			return nil, err
		}
	}

	return nodes, nil
}
func (client sbchRpcClient) getNodeCount() (uint64, error) {
	data := getNodeCountSel
	reqGetNodeCount := fmt.Sprintf(reqCallTmpl, nodesGovContractAddr, data)
	resp, err := client.basicClient.SendPost(reqGetNodeCount)
	if err != nil {
		return 0, err
	}

	return uint256.NewInt(0).SetBytes(resp).Uint64(), nil
}
func (client sbchRpcClient) getNodeByIdx(n uint64) (node NodeInfo, err error) {
	data := getNodeByIdxSel + hex.EncodeToString(uint256.NewInt(n).PaddedBytes(32))
	reqGetNodeByIdx := fmt.Sprintf(reqCallTmpl, nodesGovContractAddr, data)

	var resp []byte
	resp, err = client.basicClient.SendPost(reqGetNodeByIdx)
	if err != nil {
		return
	}

	if len(resp) != 64*5 {
		err = errors.New("invalid NodeInfo data: " + hex.EncodeToString(resp))
		return
	}

	node.ID = uint256.NewInt(0).SetBytes(resp[:64]).Uint64()
	copy(node.CertHash[:], resp[64:64*2])
	node.CertUrl = string(bytes.TrimRight(resp[64*2:64*3], string([]byte{0})))
	node.RpcUrl = string(bytes.TrimRight(resp[64*3:64*4], string([]byte{0})))
	node.Intro = string(bytes.TrimRight(resp[64*4:], string([]byte{0})))
	return
}
