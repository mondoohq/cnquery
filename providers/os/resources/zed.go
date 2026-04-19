// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/spf13/afero"
	"github.com/tailscale/hujson"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
)

const defaultZedConfigDir = ".config/zed"

func initZed(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	return initConfigPath(runtime, args, "zed", defaultZedConfigDir)
}

func (r *mqlZed) id() (string, error) {
	return "zed/" + r.ConfigPath.Data, nil
}

func (r *mqlZed) settings() (interface{}, error) {
	afs := connectionAfs(r.MqlRuntime)
	data, err := afs.ReadFile(filepath.Join(r.ConfigPath.Data, "settings.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]interface{}{}, nil
		}
		return nil, err
	}

	// Zed's settings.json is JSONC (allows // and /* */ comments, trailing commas).
	// Use hujson to normalize to standard JSON before parsing.
	clean, err := hujson.Standardize(data)
	if err != nil {
		return nil, err
	}
	var settings map[string]interface{}
	if err := json.Unmarshal(clean, &settings); err != nil {
		return nil, err
	}
	return settings, nil
}

func (r *mqlZed) extensions() ([]interface{}, error) {
	afs := connectionAfs(r.MqlRuntime)
	extensionsDir := filepath.Join(r.ConfigPath.Data, "extensions")

	data, err := afs.ReadFile(filepath.Join(extensionsDir, "installed.json"))
	if err != nil {
		if os.IsNotExist(err) {
			// Fallback: scan extensions directory for subdirectories
			return zedExtensionsFromDir(afs, extensionsDir)
		}
		return nil, err
	}

	// installed.json contains extension names
	var installed map[string]json.RawMessage
	if err := json.Unmarshal(data, &installed); err != nil {
		return zedExtensionsFromDir(afs, extensionsDir)
	}

	var result []interface{}
	for name := range installed {
		result = append(result, name)
	}
	return result, nil
}

// zedExtensionsFromDir lists extension names by scanning subdirectories.
func zedExtensionsFromDir(afs *afero.Afero, extensionsDir string) ([]interface{}, error) {
	subdirs, err := listSubdirsAfero(afs, extensionsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var result []interface{}
	for _, dir := range subdirs {
		result = append(result, dir.name)
	}
	return result, nil
}
