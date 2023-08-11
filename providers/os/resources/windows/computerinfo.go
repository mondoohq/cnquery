package windows

import (
	"encoding/json"
)

const PSGetComputerInfo = "Get-ComputerInfo | ConvertTo-Json"

func ParseComputerInfo(data []byte) (map[string]interface{}, error) {
	var properties map[string]interface{}
	return properties, json.Unmarshal(data, &properties)
}
