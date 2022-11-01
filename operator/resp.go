package operator

import (
	"encoding/json"
	"net/http"
)

type Resp struct {
	Success bool        `json:"success"`
	Error   string      `json:"error,omitempty"`
	Result  interface{} `json:"result,omitempty"`
}

func NewErrResp(err string) Resp {
	return Resp{
		Success: false,
		Error:   err,
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