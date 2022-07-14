package sbch

import (
	"fmt"
)

const (
	reqOperatorSigHashes = `{"jsonrpc": "2.0", "method": "sbch_operatorSigHashes", "params": [], "id":1}`
)

func InitRpcClients(bootstrapRpcUrl string, minEnclaveNodeCount int, minSameRespCount int) (
	*RpcClientsInfo, error) {

	bootstrapClient := wrapSimpleRpcClient(bootstrapRpcUrl)
	enclaveNodes, err := bootstrapClient.GetEnclaveNodes()
	if err != nil {
		return nil, err
	}

	attestedClients, attestedEnclaveNodes, err := AttestEnclavesAndCreateRpcClient(enclaveNodes, minEnclaveNodeCount, minSameRespCount)
	return &RpcClientsInfo{
		BootstrapRpcClient:   bootstrapClient,
		AttestedRpcClients:   attestedClients,
		EnclaveNodes:         enclaveNodes,
		AttestedEnclaveNodes: attestedEnclaveNodes,
	}, nil
}

func AttestEnclavesAndCreateRpcClient(enclaveNodes []EnclaveNodeInfo,
	minEnclaveNodeCount int, minSameRespCount int) (RpcClient, []EnclaveNodeInfo, error) {

	if len(enclaveNodes) < minEnclaveNodeCount {
		return nil, nil, fmt.Errorf("not enough enclave nodes: %d < %d",
			len(enclaveNodes), minEnclaveNodeCount)
	}

	attestedEnclaveNodes := make([]EnclaveNodeInfo, 0, len(enclaveNodes))
	attestedRpcClients := make([]BasicRpcClient, 0, len(enclaveNodes))
	for _, info := range enclaveNodes {
		client := newEnclaveRpcClient(info)
		if err := client.remoteAttest(); err != nil {
			fmt.Println("failed to attest enclave node:", client.RpcURL(), "error:", err.Error())
		} else {
			attestedEnclaveNodes = append(attestedEnclaveNodes, info)
			attestedRpcClients = append(attestedRpcClients, client)
		}
	}
	if len(attestedRpcClients) < minEnclaveNodeCount {
		return nil, nil, fmt.Errorf("not enough attested enclave nodes: %d < %d",
			len(enclaveNodes), minEnclaveNodeCount)
	}
	if minSameRespCount <= len(attestedRpcClients)/2 {
		return nil, nil, fmt.Errorf("minSameRespCount is not greater than half of clients: %d <= %d/2",
			minSameRespCount, len(attestedRpcClients))
	}

	cli := wrapAggregatedRpcClient(attestedRpcClients, minSameRespCount)
	return cli, attestedEnclaveNodes, nil
}
