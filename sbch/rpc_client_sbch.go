package sbch

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"

	gethcmn "github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

const (
	//nodesGovContractAddr = "0x0000000000000000000000000000000000001234" // TODO

	getNodeCountSel = "0x39bf397e" // ethers.utils.id('getNodeCount()')
	getNodeByIdxSel = "0x1c53c280" // ethers.utils.id('nodes(uint256)')
)

const (
	reqCallTmpl           = `{"jsonrpc": "2.0", "method": "eth_call", "params": [{"to": "%s", "data": "%s"}, "latest"], "id":1}`
	reqRedeemingUTXOs     = `{"jsonrpc": "2.0", "method": "sbch_getRedeemingUtxosForOperators", "params": [], "id":1}`
	reqToBeConvertedUTXOs = `{"jsonrpc": "2.0", "method": "sbch_getToBeConvertedUtxosForOperators", "params": [], "id":1}`
)

var _ RpcClient = (*sbchRpcClient)(nil)

type JsonRpcError struct {
	Code    int64  `json:"code"`
	Message string `json:"message"`
}

type GetUTXOsResp struct {
	Version string        `json:"jsonrpc"`
	Id      int64         `json:"id"`
	Error   *JsonRpcError `json:"error"`
	Result  []UtxoInfo    `json:"result"`
}

// smartBCH JSON-RPC client
type sbchRpcClient struct {
	basicClient  BasicRpcClient
	nodesGovAddr string
}

func NewSimpleRpcClient(nodesGovAddr, rpcUrl string) RpcClient {
	return &sbchRpcClient{
		nodesGovAddr: nodesGovAddr,
		basicClient:  newBasicRpcClient(rpcUrl),
	}
}

func (client sbchRpcClient) GetRedeemingUtxoSigHashes() ([]string, error) {
	return client.getSigHashes(reqRedeemingUTXOs)
}
func (client sbchRpcClient) GetToBeConvertedUtxoSigHashes() ([]string, error) {
	return client.getSigHashes(reqToBeConvertedUTXOs)
}
func (client sbchRpcClient) getSigHashes(reqStr string) ([]string, error) {
	respBytes, err := client.basicClient.SendPost(reqStr)
	if err != nil {
		return nil, err
	}

	var resp GetUTXOsResp
	err = json.Unmarshal(respBytes, &resp)
	if err != nil {
		return nil, err
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("failed to getSigHashes, code:%d, msg:%s",
			resp.Error.Code, resp.Error.Message)
	}

	utxoInfos := resp.Result
	sigHashes := make([]string, len(utxoInfos))
	for i, utxoInfo := range utxoInfos {
		sigHashes[i] = hex.EncodeToString(utxoInfo.TxSigHash)
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
	jsonRpcReq := fmt.Sprintf(reqCallTmpl, client.nodesGovAddr, data)
	jsonRpcResp, err := client.basicClient.SendPost(jsonRpcReq)
	if err != nil {
		return 0, err
	}

	var respMap map[string]interface{}
	if err = json.Unmarshal(jsonRpcResp, &respMap); err != nil {
		return 0, err
	}
	result, ok := respMap["result"].(string)
	if !ok {
		return 0, fmt.Errorf("invalid result: %s", string(jsonRpcResp))
	}

	resultBytes := gethcmn.FromHex(result)
	return uint256.NewInt(0).SetBytes(resultBytes).Uint64(), nil
}
func (client sbchRpcClient) getNodeByIdx(n uint64) (node NodeInfo, err error) {
	data := getNodeByIdxSel + hex.EncodeToString(uint256.NewInt(n).PaddedBytes(32))
	jsonRpcReq := fmt.Sprintf(reqCallTmpl, client.nodesGovAddr, data)
	jsonRpcResp, err := client.basicClient.SendPost(jsonRpcReq)
	if err != nil {
		return node, err
	}

	var respMap map[string]interface{}
	if err = json.Unmarshal(jsonRpcResp, &respMap); err != nil {
		return node, err
	}
	result, ok := respMap["result"].(string)
	if !ok {
		return node, fmt.Errorf("invalid result: %s", string(jsonRpcResp))
	}

	nodeInfoData := gethcmn.FromHex(result)
	if len(nodeInfoData) != 32*5 {
		err = errors.New("invalid NodeInfo data: " + result)
		return
	}

	node.ID = uint256.NewInt(0).SetBytes(nodeInfoData[:32]).Uint64()
	copy(node.CertHash[:], nodeInfoData[32:32*2])
	node.CertUrl = string(bytes.TrimRight(nodeInfoData[32*2:32*3], string([]byte{0})))
	node.RpcUrl = string(bytes.TrimRight(nodeInfoData[32*3:32*4], string([]byte{0})))
	node.Intro = string(bytes.TrimRight(nodeInfoData[32*4:], string([]byte{0})))
	return
}
