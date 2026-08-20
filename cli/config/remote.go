// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package config

import "strings"

// RemoteConfigLoader loads configuration from a remote source identified by a
// URI scheme prefix (for example "aws-ssm-ps://"). It loads the resolved
// configuration into the global viper instance, mirroring the local config
// path.
type RemoteConfigLoader func(path string) error

// remoteConfigLoaders maps a scheme prefix to the loader that handles it.
var remoteConfigLoaders = map[string]RemoteConfigLoader{}

// RegisterRemoteConfigLoader associates a loader with a config path prefix.
//
// Remote backends depend on cloud SDKs (e.g. the AWS SDK for SSM Parameter
// Store), so they live in subpackages that register themselves from init().
// Binaries blank-import the backends they need; this keeps those SDKs out of
// the import graph of this package — and therefore out of every package that
// transitively imports cli/config, including the query evaluation path.
func RegisterRemoteConfigLoader(prefix string, loader RemoteConfigLoader) {
	remoteConfigLoaders[prefix] = loader
}

// loadRemoteConfig dispatches to the registered loader whose prefix matches
// path. matched reports whether any loader handled it.
func loadRemoteConfig(path string) (matched bool, err error) {
	for prefix, loader := range remoteConfigLoaders {
		if strings.HasPrefix(path, prefix) {
			return true, loader(path)
		}
	}
	return false, nil
}
