package sbch

import (
	"bytes"
	"fmt"
)

var _ BasicRpcClient = (*aggregatedRpcClient)(nil)

type aggregatedRpcClient struct {
	clients          []BasicRpcClient
	minSameRespCount int
}

func wrapAggregatedRpcClient(clients []BasicRpcClient, minSameRespCount int) RpcClient {
	return &rpcClientWrapper{
		client: &aggregatedRpcClient{
			clients:          clients,
			minSameRespCount: minSameRespCount,
		},
	}
}

func (client *aggregatedRpcClient) RpcURL() string {
	return "aggregatedRpcClient"
}

func (client *aggregatedRpcClient) SendPost(reqStr string) ([]byte, error) {
	resps := client.sendPosts(reqStr)
	sameResp, sameRespCount := findSameResps(resps)
	if sameRespCount < client.minSameRespCount {
		return nil, fmt.Errorf("not enough same resp: %d < %d",
			sameRespCount, client.minSameRespCount)
	}
	return sameResp, nil
}

func (client *aggregatedRpcClient) sendPosts(reqStr string) [][]byte {
	var resps [][]byte

	// TODO: call SendPost() parallel
	for _, subClient := range client.clients {
		resp, err := subClient.SendPost(reqStr)
		if err != nil {
			fmt.Println("failed to send post! node:", subClient.RpcURL(), "error:", err.Error())
		} else {
			resps = append(resps, resp)
		}
	}

	return resps
}

func findSameResps(resps [][]byte) (sameResp []byte, sameRespCount int) {
	for _, resp1 := range resps {
		sameCount := 0
		for _, resp2 := range resps {
			if bytes.Equal(resp2, resp1) {
				sameCount++
			}
		}
		if sameCount > sameRespCount {
			sameRespCount = sameCount
			sameResp = resp1
		}
	}
	return
}
