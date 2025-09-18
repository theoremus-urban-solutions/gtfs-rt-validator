package validator

import (
	"encoding/json"
)

func MarshalJSON(res Result) ([]byte, error) {
	return json.MarshalIndent(res, "", "  ")
}
