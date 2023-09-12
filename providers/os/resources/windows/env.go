// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package windows

import (
	"encoding/json"
	"io"
)

type WindowsEnv struct {
	Key   string
	Value string
}

func ParseEnv(r io.Reader) (map[string]interface{}, error) {
	data, err := io.ReadAll(r)
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
