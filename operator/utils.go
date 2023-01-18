package operator

import (
	"encoding/json"
)

func toJSON(v any) string {
	bs, _ := json.Marshal(v)
	return string(bs)
}
