// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"encoding/json"
	"os"
)

type ModuleManifest struct {
	Records []Record `json:"Modules"`
}

// Record represents some metadata about an installed module
type Record struct {
	// Key is a unique identifier for this particular module
	Key string `json:"Key"`

	// SourceAddr indicates where the modules was loaded from
	SourceAddr string `json:"Source"`

	// Version is the exact version of the module
	Version string `json:"Version"`

	// Path to the directory where the module is stored
	Dir string `json:"Dir"`
}

func ParseTerraformModuleManifest(manifestPath string) (*ModuleManifest, error) {
	_, err := os.Stat(manifestPath)
	if err != nil {
		return nil, err
	}

	f, err := os.Open(manifestPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var manifest ModuleManifest
	if err := json.NewDecoder(f).Decode(&manifest); err != nil {
		return nil, err
	}
	return &manifest, nil
}
