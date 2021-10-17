package terraform

import (
	"encoding/json"
	"os"
	"path/filepath"
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

func ParseTerraformModuleManifest(fullPath string) (*ModuleManifest, error) {
	manifestPath := filepath.Join(fullPath, ".terraform/modules/modules.json")
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
