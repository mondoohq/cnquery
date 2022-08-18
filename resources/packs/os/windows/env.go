package windows

import (
	"encoding/json"
	"io"
	"io/ioutil"
)

type WindowsEnv struct {
	Key   string
	Value string
}

func ParseEnv(r io.Reader) (map[string]interface{}, error) {
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	var env []WindowsEnv
	err = json.Unmarshal(data, &env)
	if err != nil {
		return nil, err
	}

	res := map[string]interface{}{}
	for i := range env {
		envVar := env[i]
		res[envVar.Key] = envVar.Value
	}

	return res, nil
}
