package windows

import (
	"encoding/json"
	"io"
	"io/ioutil"
)

const PSGetComputerInfo = "Get-ComputerInfo | ConvertTo-Json"

func ParseComputerInfo(r io.Reader) (map[string]interface{}, error) {
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	var properties map[string]interface{}
	err = json.Unmarshal(data, &properties)
	if err != nil {
		return nil, err
	}

	return properties, nil
}
