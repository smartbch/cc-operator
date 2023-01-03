package operator

import (
	"encoding/json"
	"net/http"

	gethcmn "github.com/ethereum/go-ethereum/common"

	"github.com/smartbch/cc-operator/sbch"
)

type OpInfo struct {
	Status           string            `json:"status"`
	CurrNodes        []sbch.NodeInfo   `json:"currNodes,omitempty"`
	NewNodes         []sbch.NodeInfo   `json:"newNodes,omitempty"`
	NodesChangedTime int64             `json:"nodesChangedTime,omitempty"`
	Monitors         []gethcmn.Address `json:"monitors,omitempty"`
}

type Resp struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
	Result  any    `json:"result,omitempty"`
}

func NewResp(result any, err error) Resp {
	if err != nil {
		return NewErrResp(err.Error())
	} else {
		return NewOkResp(result)
	}
}
func NewErrResp(err string) Resp {
	return Resp{
		Success: false,
		Error:   err,
	}
}
func NewOkResp(result any) Resp {
	return Resp{
		Success: true,
		Result:  result,
	}
}

func UnmarshalResp(jsonBytes []byte) (resp Resp) {
	_ = json.Unmarshal(jsonBytes, &resp)
	return
}

func (resp Resp) ToJSON() []byte {
	bytes, _ := json.Marshal(resp)
	return bytes
}

func (resp Resp) WriteTo(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "*")
	w.Header().Set("Access-Control-Allow-Headers", "origin, content-type, accept")

	bytes, _ := json.Marshal(resp)
	_, _ = w.Write(bytes)
}
