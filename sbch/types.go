package sbch

import (
	gethcmn "github.com/ethereum/go-ethereum/common"
)

type RpcClient interface {
	RpcURL() string
	GetSbchdNodes() ([]NodeInfo, error)
	GetRedeemingUtxoSigHashes() ([]string, error)
	GetToBeConvertedUtxoSigHashes() ([]string, error)
	GetRpcPubkey() ([]byte, error)
}

type NodeInfo struct {
	ID      uint64       `json:"id"`
	PbkHash gethcmn.Hash `json:"pbkHash"`
	RpcUrl  string       `json:"rpcUrl"`
	Intro   string       `json:"intro"`
}
