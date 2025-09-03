package report

import (
	"encoding/json"

	"your.org/gtfsrt/validator/notice"
)

func MarshalJSON(res notice.Result) ([]byte, error) {
	return json.MarshalIndent(res, "", "  ")
}
