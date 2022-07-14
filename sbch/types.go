package sbch

type BasicRpcClient interface {
	RpcURL() string
	SendPost(reqStr string) ([]byte, error)
}

type RpcClient interface {
	//GetBlockNumber() (uint64, error)
	GetEnclaveNodes() ([]EnclaveNodeInfo, error)
	GetOperatorSigHashes() ([]string, error)
}

type EnclaveNodeInfo struct {
	ServerAddr      string // like localhost:8080
	SignerID        []byte
	ProductID       uint16
	SecurityVersion uint
}

type RpcClientsInfo struct {
	BootstrapRpcClient   RpcClient
	AttestedRpcClients   RpcClient
	EnclaveNodes         []EnclaveNodeInfo
	AttestedEnclaveNodes []EnclaveNodeInfo
}
