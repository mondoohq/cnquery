// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package sysinfo

import (
	"errors"

	"go.mondoo.com/cnquery"
	"go.mondoo.com/cnquery/cli/execruntime"
	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/mql"
	"go.mondoo.com/cnquery/providers"
	"go.mondoo.com/cnquery/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/providers-sdk/v1/plugin"
)

type sysInfoConfig struct {
	runtime *providers.Runtime
}

type SystemInfoOption func(t *sysInfoConfig) error

func WithRuntime(r *providers.Runtime) SystemInfoOption {
	return func(c *sysInfoConfig) error {
		c.runtime = r
		return nil
	}
}

type SystemInfo struct {
	Version    string              `json:"version,omitempty"`
	Build      string              `json:"build,omitempty"`
	Platform   *inventory.Platform `json:"platform,omitempty"`
	IP         string              `json:"ip,omitempty"`
	Hostname   string              `json:"platform_hostname,omitempty"`
	Labels     map[string]string   `json:"labels,omitempty"`
	PlatformId string              `json:"platform_id,omitempty"`
}

func GatherSystemInfo(opts ...SystemInfoOption) (*SystemInfo, error) {
	cfg := &sysInfoConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	if cfg.runtime == nil {
		cfg.runtime = providers.Coordinator.NewRuntime()
		if err := cfg.runtime.UseProvider(providers.DefaultOsID); err != nil {
			return nil, err
		}

		args, err := cfg.runtime.Provider.Instance.Plugin.ParseCLI(&plugin.ParseCLIReq{
			Connector: "local",
		})
		if err != nil {
			return nil, err
		}

		if err = cfg.runtime.Connect(&plugin.ConnectReq{
			Asset: args.Asset,
		}); err != nil {
			return nil, err
		}
	}

	sysInfo := &SystemInfo{
		Version: cnquery.GetVersion(),
		Build:   cnquery.GetBuild(),
	}

	exec := mql.New(cfg.runtime, nil)
	raw, err := exec.Exec("asset { name arch title family build version kind runtime labels }", nil)
	if err != nil {
		return sysInfo, err
	}

	if vals, ok := raw.Value.(map[string]interface{}); ok {
		sysInfo.Platform = &inventory.Platform{
			Name:    llx.TRaw2T[string](vals["name"]),
			Arch:    llx.TRaw2T[string](vals["arch"]),
			Title:   llx.TRaw2T[string](vals["title"]),
			Family:  llx.TRaw2TArr[string](vals["family"]),
			Build:   llx.TRaw2T[string](vals["build"]),
			Version: llx.TRaw2T[string](vals["version"]),
			Kind:    llx.TRaw2T[string](vals["kind"]),
			Runtime: llx.TRaw2T[string](vals["runtime"]),
			Labels:  llx.TRaw2TMap[string](vals["labels"]),
		}
	} else {
		return sysInfo, errors.New("returned asset detection type is incorrect")
	}

	// TODO: platform IDs
	// 	idDetector := providers.HostnameDetector
	// 	if pi.IsFamily(platform.FAMILY_WINDOWS) {
	// 		idDetector = providers.MachineIdDetector
	// 	}
	// 		sysInfo.PlatformId = info.IDs[0]
	// TODO: outbound ip
	// sysInfo.IP = ip
	// TODO: hostname
	// sysInfo.Hostname = hn

	// detect the execution runtime
	execEnv := execruntime.Detect()
	sysInfo.Labels = map[string]string{
		"environment": execEnv.Id,
	}

	return sysInfo, nil
}
