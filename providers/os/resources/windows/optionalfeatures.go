// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package windows

import (
	"encoding/json"
	"io"
)

const QUERY_OPTIONAL_FEATURES = "Get-WindowsOptionalFeature -Online -FeatureName * | Select-Object -Property FeatureName,DisplayName,Description,State | ConvertTo-Json"

type WindowsOptionalFeature struct {
	Name        string `json:"FeatureName"`
	DisplayName string `json:"DisplayName"`
	Description string `json:"Description"`
	Enabled     bool   `json:"Enabled"`
	State       int64  `json:"State"`
}

func ParseWindowsOptionalFeatures(input io.Reader) ([]WindowsOptionalFeature, error) {
	data, err := io.ReadAll(input)
	if err != nil {
		return nil, err
	}

	// for empty result set do not get the '{}', therefore lets abort here
	if len(data) == 0 {
		return []WindowsOptionalFeature{}, nil
	}

	var winFeatures []WindowsOptionalFeature
	err = json.Unmarshal(data, &winFeatures)
	if err != nil {
		return nil, err
	}

	for i := range winFeatures {
		if winFeatures[i].State == 2 {
			winFeatures[i].Enabled = true
		}
	}

	return winFeatures, nil
}
