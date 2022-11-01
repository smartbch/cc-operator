package sbch

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"time"

	"github.com/ethereum/go-ethereum"
	gethcmn "github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"

	sbchrpcclient "github.com/smartbch/smartbch/rpc/client"
)

const (
	//nodesGovContractAddr = "0x0000000000000000000000000000000000001234" // TODO

	getNodeCountSel = "0x39bf397e" // ethers.utils.id('getNodeCount()')
	getNodeByIdxSel = "0x1c53c280" // ethers.utils.id('nodes(uint256)')
)

var _ RpcClient = (*SimpleRpcClient)(nil)

type SimpleRpcClient struct {
	nodesGovAddr  gethcmn.Address
	sbchRpcClient *sbchrpcclient.Client
	rpcUrl        string
	reqTimeout    time.Duration
}

func NewSimpleRpcClient(nodesGovAddr, rpcUrl string) SimpleRpcClient {
	sbchRpcClient, err := sbchrpcclient.DialHTTP(rpcUrl)
	if err != nil {
		panic(err) // TODO: return error
	}

	return SimpleRpcClient{
		nodesGovAddr:  gethcmn.HexToAddress(nodesGovAddr),
		sbchRpcClient: sbchRpcClient,
		rpcUrl:        rpcUrl,
	}
}

func (client SimpleRpcClient) RpcURL() string {
	return client.rpcUrl
}

func (client SimpleRpcClient) GetSbchdNodes() ([]NodeInfo, error) {
	ctx := context.Background()
	if client.reqTimeout > 0 {
		var cancelFn context.CancelFunc
		ctx, cancelFn = context.WithTimeout(context.Background(), client.reqTimeout)
		defer cancelFn()
	}

	nodeCount, err := client.getNodeCount(ctx)
	if err != nil {
		return nil, err
	}

	nodes := make([]NodeInfo, nodeCount)
	for i := uint64(0); i < nodeCount; i++ {
		nodes[i], err = client.getNodeByIdx(i, ctx)
		if err != nil {
			return nil, err
		}
	}

	return nodes, nil
}
func (client SimpleRpcClient) getNodeCount(ctx context.Context) (uint64, error) {
	callMsg := ethereum.CallMsg{
		To:   &client.nodesGovAddr,
		Data: gethcmn.FromHex(getNodeCountSel),
	}
	nodeCountData, err := client.sbchRpcClient.CallContract(ctx, callMsg, nil)
	if err != nil {
		return 0, err
	}
	return uint256.NewInt(0).SetBytes(nodeCountData).Uint64(), nil
}
func (client SimpleRpcClient) getNodeByIdx(n uint64, ctx context.Context) (node NodeInfo, err error) {
	callData := append(gethcmn.FromHex(getNodeByIdxSel), uint256.NewInt(n).PaddedBytes(32)...)
	callMsg := ethereum.CallMsg{
		To:   &client.nodesGovAddr,
		Data: callData,
	}
	nodeInfoData, err := client.sbchRpcClient.CallContract(ctx, callMsg, nil)
	if err != nil {
		return node, err
	}

	if len(nodeInfoData) != 32*5 {
		err = errors.New("invalid NodeInfo data: " + hex.EncodeToString(nodeInfoData))
		return
	}

	node.ID = uint256.NewInt(0).SetBytes(nodeInfoData[:32]).Uint64()
	copy(node.CertHash[:], nodeInfoData[32:32*2])
	node.CertUrl = string(bytes.TrimRight(nodeInfoData[32*2:32*3], string([]byte{0})))
	node.RpcUrl = string(bytes.TrimRight(nodeInfoData[32*3:32*4], string([]byte{0})))
	node.Intro = string(bytes.TrimRight(nodeInfoData[32*4:], string([]byte{0})))
	return
}

func (client SimpleRpcClient) GetRedeemingUtxoSigHashes() ([]string, error) {
	ctx := context.Background()
	if client.reqTimeout > 0 {
		var cancelFn context.CancelFunc
		ctx, cancelFn = context.WithTimeout(ctx, client.reqTimeout)
		defer cancelFn()
	}

	utxoInfos, err := client.sbchRpcClient.RedeemingUtxosForOperators(ctx)
	if err != nil {
		return nil, err
	}

	sigHashes := make([]string, len(utxoInfos.Infos))
	for i, utxoInfo := range utxoInfos.Infos {
		sigHashes[i] = hex.EncodeToString(utxoInfo.TxSigHash)
	}
	return sigHashes, nil
}
func (client SimpleRpcClient) GetToBeConvertedUtxoSigHashes() ([]string, error) {
	ctx := context.Background()
	if client.reqTimeout > 0 {
		var cancelFn context.CancelFunc
		ctx, cancelFn = context.WithTimeout(ctx, client.reqTimeout)
		defer cancelFn()
	}

	utxoInfos, err := client.sbchRpcClient.ToBeConvertedUtxosForOperators(ctx)
	if err != nil {
		return nil, err
	}

	sigHashes := make([]string, len(utxoInfos.Infos))
	for i, utxoInfo := range utxoInfos.Infos {
		sigHashes[i] = hex.EncodeToString(utxoInfo.TxSigHash)
	}
	return sigHashes, nil
}