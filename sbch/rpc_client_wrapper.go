package sbch

import "encoding/json"

var _ RpcClient = (*rpcClientWrapper)(nil)

type rpcClientWrapper struct {
	client BasicRpcClient
}

func (wrapper rpcClientWrapper) GetEnclaveNodes() ([]EnclaveNodeInfo, error) {
	//TODO implement me
	panic("implement me")
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
