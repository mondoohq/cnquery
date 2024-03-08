// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package ranger

import (
	"net/http"
	"runtime"

	"go.mondoo.com/cnquery/v10"
	"go.mondoo.com/ranger-rpc"
	"go.mondoo.com/ranger-rpc/plugins/scope"
)

func DefaultRangerPlugins(features cnquery.Features) []ranger.ClientPlugin {
	plugins := []ranger.ClientPlugin{}
	plugins = append(plugins, scope.NewRequestIDRangerPlugin())
	plugins = append(plugins, sysInfoHeader(features))
	return plugins
}

func sysInfoHeader(features cnquery.Features) ranger.ClientPlugin {
	const (
		HttpHeaderUserAgent      = "User-Agent"
		HttpHeaderClientFeatures = "Mondoo-Features"
		HttpHeaderPlatformID     = "Mondoo-PlatformID"
	)

	h := http.Header{}
	info := map[string]string{
		"cnquery": cnquery.Version,
		"build":   cnquery.Build,
	}
	info["PN"] = runtime.GOOS
	// info["PR"] = sysInfo.Platform.Version
	// info["PA"] = sysInfo.Platform.Arch
	// info["IP"] = sysInfo.IP
	// info["HN"] = sysInfo.Hostname
	// h.Set(HttpHeaderPlatformID, sysInfo.PlatformId)

	h.Set(HttpHeaderUserAgent, scope.XInfoHeader(info))
	h.Set(HttpHeaderClientFeatures, features.Encode())
	return scope.NewCustomHeaderRangerPlugin(h)
}
