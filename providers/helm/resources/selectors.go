// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"fmt"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/helm/connection"
)

// stringArg reads a required string selector argument. ok is false when the arg
// is absent or empty, which is the legitimate "bare resource" fast path rather
// than a lookup miss.
func stringArg(args map[string]*llx.RawData, key string) (string, bool) {
	if len(args) == 0 {
		return "", false
	}
	raw, ok := args[key]
	if !ok || raw == nil {
		return "", false
	}
	v, _ := raw.Value.(string)
	return v, v != ""
}

// initHelmFile resolves the selector the schema documents,
// `helm.file(path: "NOTES.txt")`, against the loaded charts.
//
// Without it the runtime built the resource from the `path` arg alone:
// `content` came back "" because the Internal cache was never populated (a
// fabricated empty answer), `size`/`isBinary` were left UNSET rather than null,
// and with no `__id` every bare lookup in a session shared the cache key
// `helm.file\x00` — so the second returned the first one's path.
//
// A miss returns an error rather than falling through to `args, nil, nil`,
// which would rebuild the same husk.
func initHelmFile(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	path, ok := stringArg(args, "path")
	if !ok {
		return args, nil, nil
	}
	conn, ok := runtime.Connection.(*connection.HelmConnection)
	if !ok {
		return args, nil, nil
	}

	for _, loaded := range conn.Charts() {
		if loaded.Chart == nil {
			continue
		}
		for _, f := range loaded.Chart.Files {
			if f == nil || f.Name != path {
				continue
			}
			mqlFile, err := newMqlHelmFile(runtime, loaded.Chart.Name(), f)
			if err != nil {
				return nil, nil, err
			}
			return args, mqlFile, nil
		}
	}
	return nil, nil, fmt.Errorf("helm: no chart file at path %q", path)
}

// initHelmTemplate resolves `helm.template(name: "templates/deployment.yaml")`
// against the loaded charts, for the same reasons as initHelmFile.
//
// The template is returned unrendered (no `renderedContent`), because rendering
// is a chart-wide operation reached through `helm.chart.templates`. That is a
// visible null rather than a fabricated empty string.
func initHelmTemplate(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	name, ok := stringArg(args, "name")
	if !ok {
		return args, nil, nil
	}
	conn, ok := runtime.Connection.(*connection.HelmConnection)
	if !ok {
		return args, nil, nil
	}

	for _, loaded := range conn.Charts() {
		if loaded.Chart == nil {
			continue
		}
		for _, t := range loaded.Chart.Templates {
			if t == nil || t.Name != name {
				continue
			}
			mqlTemplate, err := newMqlHelmTemplate(runtime, loaded.Chart.Name(), t, "", nil)
			if err != nil {
				return nil, nil, err
			}
			return args, mqlTemplate, nil
		}
	}
	return nil, nil, fmt.Errorf("helm: no chart template named %q", name)
}
