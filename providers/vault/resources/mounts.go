// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"strconv"

	vaultapi "github.com/hashicorp/vault/api"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
)

func (r *mqlVault) auditDevices() ([]any, error) {
	client, err := vaultClient(r.MqlRuntime)
	if err != nil {
		return nil, err
	}

	devices, err := client.Sys().ListAudit()
	if err != nil {
		return nil, err
	}

	res := make([]any, 0, len(devices))
	for path, device := range devices {
		if device == nil {
			continue
		}

		options, err := convert.JsonToDict(device.Options)
		if err != nil {
			return nil, err
		}

		mqlDevice, err := CreateResource(r.MqlRuntime, "vault.auditDevice", map[string]*llx.RawData{
			"__id":        llx.StringData(path),
			"path":        llx.StringData(path),
			"type":        llx.StringData(device.Type),
			"description": llx.StringData(device.Description),
			"local":       llx.BoolData(device.Local),
			"options":     llx.DictData(options),
			"logRaw":      llx.BoolData(optionBool(device.Options, "log_raw")),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlDevice)
	}
	return res, nil
}

func (r *mqlVault) authMethods() ([]any, error) {
	client, err := vaultClient(r.MqlRuntime)
	if err != nil {
		return nil, err
	}

	mounts, err := client.Sys().ListAuth()
	if err != nil {
		return nil, err
	}

	res := make([]any, 0, len(mounts))
	for path, mount := range mounts {
		if mount == nil {
			continue
		}

		mqlMethod, err := CreateResource(r.MqlRuntime, "vault.authMethod", map[string]*llx.RawData{
			"__id":              llx.StringData(path),
			"path":              llx.StringData(path),
			"type":              llx.StringData(mount.Type),
			"description":       llx.StringData(mount.Description),
			"local":             llx.BoolData(mount.Local),
			"defaultLeaseTtl":   llx.IntData(int64(mount.Config.DefaultLeaseTTL)),
			"maxLeaseTtl":       llx.IntData(int64(mount.Config.MaxLeaseTTL)),
			"tokenType":         llx.StringData(mount.Config.TokenType),
			"listedInLoginForm": llx.BoolData(mount.Config.ListingVisibility == "unauth"),
			"pluginVersion":     llx.StringData(mount.PluginVersion),
			"deprecationStatus": llx.StringData(mount.DeprecationStatus),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlMethod)
	}
	return res, nil
}

func (r *mqlVault) secretEngines() ([]any, error) {
	client, err := vaultClient(r.MqlRuntime)
	if err != nil {
		return nil, err
	}

	mounts, err := client.Sys().ListMounts()
	if err != nil {
		return nil, err
	}

	res := make([]any, 0, len(mounts))
	for path, mount := range mounts {
		if mount == nil {
			continue
		}

		mqlEngine, err := CreateResource(r.MqlRuntime, "vault.secretEngine", map[string]*llx.RawData{
			"__id":              llx.StringData(path),
			"path":              llx.StringData(path),
			"type":              llx.StringData(mount.Type),
			"description":       llx.StringData(mount.Description),
			"local":             llx.BoolData(mount.Local),
			"sealWrap":          llx.BoolData(mount.SealWrap),
			"defaultLeaseTtl":   llx.IntData(int64(mount.Config.DefaultLeaseTTL)),
			"maxLeaseTtl":       llx.IntData(int64(mount.Config.MaxLeaseTTL)),
			"kvVersion":         llx.IntData(kvVersion(mount)),
			"pluginVersion":     llx.StringData(mount.PluginVersion),
			"deprecationStatus": llx.StringData(mount.DeprecationStatus),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlEngine)
	}
	return res, nil
}

// kvVersion reads the key/value major version off a mount. Only kv mounts carry
// it, and a version 1 mount often omits the option entirely, so an absent value
// on a kv mount means 1 rather than unknown. Non-kv mounts report 0.
func kvVersion(mount *vaultapi.MountOutput) int64 {
	if mount == nil || mount.Type != "kv" {
		return 0
	}
	raw, ok := mount.Options["version"]
	if !ok || raw == "" {
		return 1
	}
	version, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 1
	}
	return version
}

// optionBool reads a mount option that the server renders as a string. Vault
// writes booleans in audit device options as "true"/"false" text, so a plain
// type assertion would always miss.
func optionBool(options map[string]string, key string) bool {
	raw, ok := options[key]
	if !ok {
		return false
	}
	value, err := strconv.ParseBool(raw)
	if err != nil {
		return false
	}
	return value
}
