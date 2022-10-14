package sbch

import (
	gethcmn "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

type BasicRpcClient interface {
	RpcURL() string
	SendPost(reqStr string) ([]byte, error)
}

type RpcClient interface {
	//GetBlockNumber() (uint64, error)
	GetSbchdNodes() ([]NodeInfo, error)
	GetRedeemingUtxoSigHashes() ([]string, error)
	GetToBeConvertedUtxoSigHashes() ([]string, error)
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

type UtxoInfo struct {
	OwnerOfLost      gethcmn.Address `json:"owner_of_lost"`
	CovenantAddr     gethcmn.Address `json:"covenant_addr"`
	IsRedeemed       bool            `json:"is_redeemed"`
	RedeemTarget     gethcmn.Address `json:"redeem_target"`
	ExpectedSignTime int64           `json:"expected_sign_time"`
	Txid             gethcmn.Hash    `json:"txid"`
	Index            uint32          `json:"index"`
	Amount           hexutil.Uint64  `json:"amount"` // in satoshi
	TxSigHash        hexutil.Bytes   `json:"tx_sig_hash"`
}
