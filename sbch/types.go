package sbch

import (
	gethcmn "github.com/ethereum/go-ethereum/common"
)

type RpcClient interface {
	GetSbchdNodes() ([]NodeInfo, error)
	GetRedeemingUtxoSigHashes() ([]string, error)
	GetToBeConvertedUtxoSigHashes() ([]string, error)
	RpcURL() string
}

type NodeInfo struct {
	ID       uint64       `json:"id"`
	CertHash gethcmn.Hash `json:"certHash"`
	CertUrl  string       `json:"certUrl"`
	RpcUrl   string       `json:"rpcUrl"`
	Intro    string       `json:"intro"`
}

type RpcClientsInfo struct {
	BootstrapRpcClient RpcClient
	ClusterRpcClient   RpcClient
	AllNodes           []NodeInfo
	ValidNodes         []NodeInfo // used by clusterRpcClient
}
