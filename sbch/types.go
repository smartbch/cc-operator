package sbch

type BasicRpcClient interface {
	RpcURL() string
	SendPost(reqStr string) ([]byte, error)
}

type RpcClient interface {
	//GetBlockNumber() (uint64, error)
	GetSbchdNodes() ([]NodeInfo, error)
	GetOperatorSigHashes() ([]string, error)
}

type NodeInfo struct {
	ID       uint64
	CertHash [32]byte
	CertUrl  string
	RpcUrl   string
	Intro    string
}

type RpcClientsInfo struct {
	BootstrapRpcClient RpcClient
	ClusterRpcClient   RpcClient
	AllNodes           []NodeInfo
	ValidNodes         []NodeInfo // used by clusterRpcClient
}
