// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"strings"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/os/resources/parsers"
)

// yumConfigPaths lists the default global package-manager configuration
// file locations. Classic yum uses /etc/yum.conf; dnf-based systems use
// /etc/dnf/dnf.conf (on which /etc/yum.conf is often just a symlink).
// The first one that exists wins.
var yumConfigPaths = []string{
	"/etc/yum.conf",
	"/etc/dnf/dnf.conf",
}

func initYumConfig(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if x, ok := args["path"]; ok {
		path, ok := x.Value.(string)
		if !ok {
			return nil, nil, errors.New("wrong type for 'path' in yum.config initialization, it must be a string")
		}

		f, err := CreateResource(runtime, "file", map[string]*llx.RawData{
			"path": llx.StringData(path),
		})
		if err != nil {
			return nil, nil, err
		}
		args["file"] = llx.ResourceData(f, "file")
		delete(args, "path")
	}

	return args, nil, nil
}

func (y *mqlYumConfig) id() (string, error) {
	file := y.GetFile()
	if file.Error != nil {
		return "", file.Error
	}
	if file.Data == nil {
		return "", errors.New("cannot get file for yum.config")
	}
	return file.Data.Path.Data, nil
}

func (y *mqlYumConfig) file() (*mqlFile, error) {
	for _, candidate := range yumConfigPaths {
		f, err := CreateResource(y.MqlRuntime, "file", map[string]*llx.RawData{
			"path": llx.StringData(candidate),
		})
		if err != nil {
			return nil, err
		}
		mqlFile := f.(*mqlFile)
		if exists := mqlFile.GetExists(); exists.Error == nil && exists.Data {
			return mqlFile, nil
		}
	}

	// none exist; return the primary path so callers can still inspect it
	f, err := CreateResource(y.MqlRuntime, "file", map[string]*llx.RawData{
		"path": llx.StringData(yumConfigPaths[0]),
	})
	if err != nil {
		return nil, err
	}
	return f.(*mqlFile), nil
}

func (y *mqlYumConfig) content(file *mqlFile) (string, error) {
	return fileContentOrEmpty(file)
}

// params returns every directive in the [main] section as a string map.
func (y *mqlYumConfig) params(content string) (map[string]any, error) {
	ini := parsers.ParseIni(content, "=")

	res := map[string]any{}
	main, ok := ini.Fields["main"].(map[string]any)
	if !ok {
		return res, nil
	}
	for k, v := range main {
		if s, ok := v.(string); ok {
			res[k] = s
		}
	}
	return res, nil
}

// yumBoolParam interprets a yum/dnf boolean directive. yum accepts
// 1/0, true/false, yes/no, on/off (case-insensitive). Absent or
// unrecognized values are treated as false.
func yumBoolParam(params map[string]any, key string) bool {
	v, ok := params[key]
	if !ok {
		return false
	}
	s, ok := v.(string)
	if !ok {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func (y *mqlYumConfig) gpgcheck(params map[string]any) (bool, error) {
	return yumBoolParam(params, "gpgcheck"), nil
}

func (y *mqlYumConfig) localPkgGpgcheck(params map[string]any) (bool, error) {
	return yumBoolParam(params, "localpkg_gpgcheck"), nil
}

func (y *mqlYumConfig) repoGpgcheck(params map[string]any) (bool, error) {
	return yumBoolParam(params, "repo_gpgcheck"), nil
}

func (y *mqlYumConfig) cleanRequirementsOnRemove(params map[string]any) (bool, error) {
	return yumBoolParam(params, "clean_requirements_on_remove"), nil
}
