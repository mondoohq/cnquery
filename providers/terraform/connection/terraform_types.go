// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package connection

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

type terraformAssetType int32

const (
	configurationfiles terraformAssetType = 0
	planfile           terraformAssetType = 1
	statefile          terraformAssetType = 2
)
