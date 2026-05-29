// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package windows

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"
)

const QUERY_OPTIONAL_FEATURES = "Get-WindowsOptionalFeature -Online -FeatureName * | Select-Object -Property FeatureName,DisplayName,Description,State | ConvertTo-Json"

// OptionalFeatureQuery builds a PowerShell command that retrieves a single
// optional feature by name. Looking up one feature is significantly cheaper
// than enumerating every feature in the image with `-FeatureName *`. The name
// is wrapped in a single-quoted string (with embedded single quotes doubled)
// so it is treated literally — no wildcard expansion, no command injection.
func OptionalFeatureQuery(name string) string {
	escaped := strings.ReplaceAll(name, "'", "''")
	return "Get-WindowsOptionalFeature -Online -FeatureName '" + escaped + "' | Select-Object -Property FeatureName,DisplayName,Description,State | ConvertTo-Json"
}

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
	data = bytes.TrimSpace(data)
	if len(data) == 0 {
		return []WindowsOptionalFeature{}, nil
	}

	// ConvertTo-Json emits a single JSON object (not an array) when only one
	// feature is returned, e.g. when querying a single feature by name. Handle
	// both shapes so callers get a consistent slice.
	var winFeatures []WindowsOptionalFeature
	if data[0] == '{' {
		var single WindowsOptionalFeature
		if err = json.Unmarshal(data, &single); err != nil {
			return nil, err
		}
		winFeatures = []WindowsOptionalFeature{single}
	} else if err = json.Unmarshal(data, &winFeatures); err != nil {
		return nil, err
	}

	for i := range winFeatures {
		if winFeatures[i].State == 2 {
			winFeatures[i].Enabled = true
		}
	}

	return winFeatures, nil
}
