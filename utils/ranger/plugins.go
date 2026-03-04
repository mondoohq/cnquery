// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package ranger

import (
	"net/http"
	"runtime"

	"go.mondoo.com/mql/v13"
	"go.mondoo.com/ranger-rpc"
	"go.mondoo.com/ranger-rpc/plugins/scope"
)

// ClientSysInfo holds system information for upstream request headers.
// This is a lightweight struct to avoid import cycles with the sysinfo package.
type ClientSysInfo struct {
	PlatformName    string
	PlatformVersion string
	PlatformArch    string
	IP              string
	Hostname        string
	PlatformID      string
}

func DefaultRangerPlugins(features mql.Features, si *ClientSysInfo) []ranger.ClientPlugin {
	plugins := []ranger.ClientPlugin{}
	plugins = append(plugins, scope.NewRequestIDRangerPlugin())
	plugins = append(plugins, sysInfoHeader(features, si))
	return plugins
}

func sysInfoHeader(features mql.Features, si *ClientSysInfo) ranger.ClientPlugin {
	const (
		HttpHeaderUserAgent      = "User-Agent"
		HttpHeaderClientFeatures = "Mondoo-Features"
		HttpHeaderPlatformID     = "Mondoo-PlatformID"
	)

	h := http.Header{}
	info := map[string]string{
		"mql":   mql.Version,
		"build": mql.Build,
	}
	if si != nil {
		info["PN"] = si.PlatformName
		info["PR"] = si.PlatformVersion
		info["PA"] = si.PlatformArch
		info["IP"] = si.IP
		info["HN"] = si.Hostname
		h.Set(HttpHeaderPlatformID, si.PlatformID)
	}

	if info["PN"] == "" {
		info["PN"] = runtime.GOOS
	}

	h.Set(HttpHeaderUserAgent, scope.XInfoHeader(info))
	h.Set(HttpHeaderClientFeatures, features.Encode())
	return scope.NewCustomHeaderRangerPlugin(h)
}
