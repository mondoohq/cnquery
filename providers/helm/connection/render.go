// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import "encoding/json"

// Option keys carried on inventory.Config.Options. Scalars are stored
// verbatim; repeatable flags are JSON-encoded string arrays (Options is
// a map[string]string, so a list has to be serialized).
const (
	OptionPath        = "path"
	OptionValues      = "values"       // -f / --values, JSON []string
	OptionSet         = "set"          // --set, JSON []string of key=val
	OptionSetString   = "set-string"   // --set-string, JSON []string
	OptionSetJSON     = "set-json"     // --set-json, JSON []string
	OptionSetFile     = "set-file"     // --set-file, JSON []string
	OptionReleaseName = "release-name" // release name override
	OptionNamespace   = "namespace"    // release namespace override
	OptionKubeVersion = "kube-version" // target .Capabilities.KubeVersion
	OptionAPIVersions = "api-versions" // -a / --api-versions, JSON []string
	OptionIsUpgrade   = "is-upgrade"   // ".Release.IsUpgrade" toggle ("true")

	// remote-fetch options (workstream 3)
	OptionRepo             = "repo"              // chart repository URL for a named chart
	OptionVersion          = "version"           // chart version to pull
	OptionUsername         = "username"          // registry/repo basic-auth user
	OptionPassword         = "password"          // registry/repo basic-auth password
	OptionRepositoryConfig = "repository-config" // helm repositories.yaml path
	OptionRepositoryCache  = "repository-cache"  // helm repo cache dir
)

// RenderOptions is the parsed, typed view of the render-affecting
// connection options. It mirrors helm's own install/template inputs so
// the rendered manifests match what `helm install` would produce.
type RenderOptions struct {
	ValueFiles   []string
	Values       []string
	StringValues []string
	JSONValues   []string
	FileValues   []string
	ReleaseName  string
	Namespace    string
	KubeVersion  string
	APIVersions  []string
	IsUpgrade    bool
}

// RenderOptions returns the parsed render configuration for this
// connection. Namespace defaults to "default" to match helm.
func (c *HelmConnection) RenderOptions() RenderOptions {
	return c.renderOpts
}

func parseRenderOptions(opts map[string]string) RenderOptions {
	ro := RenderOptions{
		ValueFiles:   decodeStringList(opts[OptionValues]),
		Values:       decodeStringList(opts[OptionSet]),
		StringValues: decodeStringList(opts[OptionSetString]),
		JSONValues:   decodeStringList(opts[OptionSetJSON]),
		FileValues:   decodeStringList(opts[OptionSetFile]),
		ReleaseName:  opts[OptionReleaseName],
		Namespace:    opts[OptionNamespace],
		KubeVersion:  opts[OptionKubeVersion],
		APIVersions:  decodeStringList(opts[OptionAPIVersions]),
		IsUpgrade:    opts[OptionIsUpgrade] == "true",
	}
	if ro.Namespace == "" {
		ro.Namespace = "default"
	}
	return ro
}

// decodeStringList parses a JSON-encoded []string option. An empty or
// malformed value yields nil — render options are best-effort, never a
// hard connection failure.
func decodeStringList(s string) []string {
	if s == "" {
		return nil
	}
	var out []string
	if err := json.Unmarshal([]byte(s), &out); err != nil {
		return nil
	}
	return out
}

// EncodeStringList serializes a []string for storage in
// inventory.Config.Options. Exposed so ParseCLI can encode list flags.
func EncodeStringList(vals []string) string {
	if len(vals) == 0 {
		return ""
	}
	b, err := json.Marshal(vals)
	if err != nil {
		return ""
	}
	return string(b)
}
