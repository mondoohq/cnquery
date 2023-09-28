// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package sysinfo

import (
	"errors"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/providers/os/resources/networkinterface"

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

	log.Debug().Msg("Gathering system information")
	if cfg.runtime == nil {

		cfg.runtime = providers.Coordinator.NewRuntime()

		// TODO: we need to ensure that the os provider is available here

		// init runtime
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
	// TODO: it is not returning it as a MQL SingleValue, therefore we need to force it with return
	raw, err := exec.Exec("return asset { name arch title family build version kind runtime labels ids }", nil)
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

		platformID := llx.TRaw2TArr[string](vals["ids"])
		if len(platformID) > 0 {
			sysInfo.PlatformId = platformID[0]
		}
	} else {
		return sysInfo, errors.New("returned asset detection type is incorrect")
	}

	// determine hostname
	osRaw, err := exec.Exec("return os.hostname", nil)
	if err != nil {
		return sysInfo, err
	}

	if hostname, ok := osRaw.Value.(string); ok {
		sysInfo.Hostname = hostname
	}

	// determine ip address
	// TODO: move this to MQL and expose that information in the graph
	ipAddr, err := networkinterface.GetOutboundIP()
	if err == nil {
		sysInfo.IP = ipAddr.String()
	}

	// detect the execution runtime
	execEnv := execruntime.Detect()
	sysInfo.Labels = map[string]string{
		"environment": execEnv.Id,
	}

	return sysInfo, nil
}
