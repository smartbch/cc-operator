package sbch

import (
	gethcmn "github.com/ethereum/go-ethereum/common"
	sbchrpctypes "github.com/smartbch/smartbch/rpc/types"
)

type RpcClient interface {
	RpcURL() string
	GetSbchdNodes() ([]NodeInfo, error)
	GetRedeemingUtxosForOperators() ([]*sbchrpctypes.UtxoInfo, error)
	GetRedeemingUtxosForMonitors() ([]*sbchrpctypes.UtxoInfo, error)
	GetToBeConvertedUtxosForOperators() ([]*sbchrpctypes.UtxoInfo, error)
	GetToBeConvertedUtxosForMonitors() ([]*sbchrpctypes.UtxoInfo, error)
	GetRpcPubkey() ([]byte, error)
}

type NodeInfo struct {
	ID      uint64       `json:"id"`
	PbkHash gethcmn.Hash `json:"pbkHash"`
	RpcUrl  string       `json:"rpcUrl"`
	Intro   string       `json:"intro"`
}
