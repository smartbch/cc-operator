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
	reqOperatorSigHashes = `{"jsonrpc": "2.0", "method": "sbch_operatorSigHashes", "params": [], "id":1}`
	reqCallTmpl          = `{"jsonrpc": "2.0", "method": "eth_call", "params": [{"to": "%s", "data": "%s"}, "latest"], "id":1}`
)

var _ RpcClient = (*rpcClientWrapper)(nil)

type rpcClientWrapper struct {
	client BasicRpcClient
}

func wrapSimpleRpcClient(url string) RpcClient {
	return &rpcClientWrapper{
		client: NewSimpleRpcClient(url),
	}
}

func (wrapper rpcClientWrapper) GetOperatorSigHashes() ([]string, error) {
	resp, err := wrapper.client.SendPost(reqOperatorSigHashes)
	if err != nil {
		return nil, err
	}

	var sigHashes []string
	err = json.Unmarshal(resp, &sigHashes)
	if err != nil {
		return nil, err
	}

	return sigHashes, nil
}

func (wrapper rpcClientWrapper) GetSbchdNodes() ([]NodeInfo, error) {
	nodeCount, err := wrapper.getNodeCount()
	if err != nil {
		return nil, err
	}

	nodes := make([]NodeInfo, nodeCount)
	for i := uint64(0); i < nodeCount; i++ {
		nodes[i], err = wrapper.getNodeByIdx(i)
		if err != nil {
			return nil, err
		}
	}

	return nodes, nil
}
func (wrapper rpcClientWrapper) getNodeCount() (uint64, error) {
	data := getNodeCountSel
	reqGetNodeCount := fmt.Sprintf(reqCallTmpl, nodesGovContractAddr, data)
	resp, err := wrapper.client.SendPost(reqGetNodeCount)
	if err != nil {
		return 0, err
	}

	return uint256.NewInt(0).SetBytes(resp).Uint64(), nil
}
func (wrapper rpcClientWrapper) getNodeByIdx(n uint64) (node NodeInfo, err error) {
	data := getNodeByIdxSel + hex.EncodeToString(uint256.NewInt(n).PaddedBytes(32))
	reqGetNodeByIdx := fmt.Sprintf(reqCallTmpl, nodesGovContractAddr, data)

	var resp []byte
	resp, err = wrapper.client.SendPost(reqGetNodeByIdx)
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
