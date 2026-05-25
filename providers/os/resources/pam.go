// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"io"
	"sort"
	"strconv"
	"strings"

	"go.mondoo.com/mql/v13/checksums"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/os/connection/shared"
	"go.mondoo.com/mql/v13/providers/os/resources/pam"
	"go.mondoo.com/mql/v13/types"
)

const (
	defaultPamConf = "/etc/pam.conf"
	defaultPamDir  = "/etc/pam.d"
)

func initPamConf(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if x, ok := args["path"]; ok {
		path, ok := x.Value.(string)
		if !ok {
			return nil, nil, errors.New("wrong type for 'path' it must be a string")
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

func (s *mqlPamConf) id() (string, error) {
	checksum := checksums.New
	for i := range s.Files.Data {
		path := s.Files.Data[i].(*mqlFile).Path.Data
		checksum = checksum.Add(path)
	}

	return checksum.String(), nil
}

func (se *mqlPamConfServiceEntry) id() (string, error) {
	ptype := se.PamType.Data
	mod := se.Module.Data
	s := se.Service.Data
	ln := se.LineNumber.Data
	lnstr := strconv.FormatInt(ln, 10)

	id := s + "/" + lnstr + "/" + ptype

	// for include mod is empty
	if mod != "" {
		id += "/" + mod
	}

	return id, nil
}

// GetFiles is called when the user has not provided a custom path. Otherwise files are set in the init
// method and this function is never called then since the data is already cached.
func (s *mqlPamConf) files() ([]any, error) {
	// check if the pam.d directory exists and is a directory
	// according to the pam spec, pam prefers the directory if it  exists over the single file config
	// see http://www.linux-pam.org/Linux-PAM-html/sag-configuration.html
	raw, err := CreateResource(s.MqlRuntime, "file", map[string]*llx.RawData{
		"path": llx.StringData(defaultPamDir),
	})
	if err != nil {
		return nil, err
	}
	f := raw.(*mqlFile)
	exist := f.GetExists()
	if exist.Error != nil {
		return nil, exist.Error
	}

	if exist.Data {
		return getSortedPathFiles(s.MqlRuntime, defaultPamDir)
	} else {
		return getSortedPathFiles(s.MqlRuntime, defaultPamConf)
	}
}

func (s *mqlPamConf) content(files []any) (string, error) {
	conn := s.MqlRuntime.Connection.(shared.Connection)

	var res strings.Builder
	var notReadyError error = nil

	for i := range files {
		file := files[i].(*mqlFile)

		f, err := conn.FileSystem().Open(file.Path.Data)
		if err != nil {
			return "", err
		}

		raw, err := io.ReadAll(f)
		f.Close()
		if err != nil {
			return "", err
		}

		res.WriteString(string(raw))
		res.WriteString("\n")
	}

	if notReadyError != nil {
		return "", notReadyError
	}

	return res.String(), nil
}

func (s *mqlPamConf) services(files []any) (map[string]any, error) {
	conn := s.MqlRuntime.Connection.(shared.Connection)

	contents := map[string]string{}
	var notReadyError error = nil

	for i := range files {
		file := files[i].(*mqlFile)

		f, err := conn.FileSystem().Open(file.Path.Data)
		if err != nil {
			return nil, err
		}

		raw, err := io.ReadAll(f)
		f.Close()
		if err != nil {
			return nil, err
		}

		contents[file.Path.Data] = string(raw)
	}

	if notReadyError != nil {
		return nil, notReadyError
	}

	services := map[string]any{}
	for basename, content := range contents {
		lines := strings.Split(content, "\n")
		settings := []any{}
		var line string
		for i := range lines {
			line = lines[i]

			if idx := strings.Index(line, "#"); idx >= 0 {
				line = line[0:idx]
			}
			line = strings.Trim(line, " \t\r")

			if line != "" {
				settings = append(settings, line)
			}
		}
		services[basename] = settings
	}

	return services, nil
}

// canonicalizePamModuleName strips a leading path and `.so` suffix from a PAM
// module reference so callers can look modules up by short name. Case is
// preserved — PAM module names are conventionally lowercase, but we don't
// fold them.
func canonicalizePamModuleName(name string) string {
	if idx := strings.LastIndex(name, "/"); idx >= 0 {
		name = name[idx+1:]
	}
	name = strings.TrimSuffix(name, ".so")
	return name
}

// aggregatePamParams merges `key=value` options from one or more service
// entries into a single dict. Bare options without `=` are stored with an
// empty string value so `params["use_authtok"] != null` works as an
// existence check. Later occurrences of the same key overwrite earlier
// ones, matching how PAM itself evaluates duplicate flags.
func aggregatePamParams(optionLists ...[]any) map[string]any {
	params := map[string]any{}
	for _, opts := range optionLists {
		for _, raw := range opts {
			token, ok := raw.(string)
			if !ok {
				continue
			}
			if token == "" {
				continue
			}
			if idx := strings.Index(token, "="); idx >= 0 {
				key := strings.ToLower(token[:idx])
				value := token[idx+1:]
				params[key] = value
			} else {
				params[strings.ToLower(token)] = ""
			}
		}
	}
	return params
}

// isPamControlEnabled reports whether a PAM control directive counts as
// "the module is loaded". Bracketed controls that explicitly route the
// module to ignore/skip don't count.
func isPamControlEnabled(control string) bool {
	c := strings.TrimSpace(control)
	if c == "" {
		return false
	}
	// Bracketed controls: `[default=ignore]` / `[default=skip]` mean the
	// module is referenced but its result is discarded — treat as not
	// enabled. Any other bracketed form (e.g. `[success=1 default=bad]`,
	// `[default=die]`) is a real load.
	if strings.HasPrefix(c, "[") {
		lower := strings.ToLower(c)
		if strings.Contains(lower, "default=ignore") || strings.Contains(lower, "default=skip") {
			return false
		}
		return true
	}
	// Bare controls: required, requisite, sufficient, optional,
	// substack, include — all count as loaded.
	return true
}

func initPamModule(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}

	nameRaw := args["name"]
	if nameRaw == nil {
		return args, nil, nil
	}
	name, ok := nameRaw.Value.(string)
	if !ok {
		return nil, nil, errors.New("wrong type for 'name', it must be a string")
	}
	name = canonicalizePamModuleName(name)

	conf, err := CreateResource(runtime, "pam.conf", map[string]*llx.RawData{})
	if err != nil {
		return nil, nil, err
	}
	pamConf := conf.(*mqlPamConf)

	modules := pamConf.GetModules()
	if modules.Error != nil {
		return nil, nil, modules.Error
	}

	for _, m := range modules.Data {
		mod, ok := m.(*mqlPamModule)
		if !ok {
			continue
		}
		if mod.Name.Data == name {
			return nil, mod, nil
		}
	}

	// Module is not referenced by any service — return an empty husk.
	res, err := CreateResource(runtime, "pam.module", map[string]*llx.RawData{
		"name":    llx.StringData(name),
		"params":  llx.MapData(map[string]any{}, types.String),
		"enabled": llx.BoolData(false),
		"entries": llx.ArrayData([]any{}, types.Resource("pam.conf.serviceEntry")),
	})
	if err != nil {
		return nil, nil, err
	}
	return nil, res, nil
}

func (m *mqlPamModule) id() (string, error) {
	return "pam.module/" + m.Name.Data, nil
}

func (s *mqlPamConf) modules(entries map[string]any) ([]any, error) {
	// Collect entries grouped by canonical module name, preserving source
	// order across files for last-write-wins option aggregation.
	type moduleAgg struct {
		name       string
		entries    []any
		optionSets [][]any
		anyEnabled bool
	}

	byName := map[string]*moduleAgg{}
	order := []string{}

	// Iterate services in a stable order so the resulting []pam.module
	// list is deterministic across calls.
	serviceNames := make([]string, 0, len(entries))
	for svc := range entries {
		serviceNames = append(serviceNames, svc)
	}
	sort.Strings(serviceNames)

	for _, svc := range serviceNames {
		raw := entries[svc]
		list, ok := raw.([]any)
		if !ok {
			continue
		}
		for _, e := range list {
			entry, ok := e.(*mqlPamConfServiceEntry)
			if !ok {
				continue
			}
			rawModule := entry.Module.Data
			if rawModule == "" {
				// Skip @include lines and anything that doesn't reference
				// a real module.
				continue
			}
			name := canonicalizePamModuleName(rawModule)
			agg, ok := byName[name]
			if !ok {
				agg = &moduleAgg{name: name}
				byName[name] = agg
				order = append(order, name)
			}
			agg.entries = append(agg.entries, entry)
			agg.optionSets = append(agg.optionSets, entry.Options.Data)
			if isPamControlEnabled(entry.Control.Data) {
				agg.anyEnabled = true
			}
		}
	}

	out := make([]any, 0, len(order))
	for _, name := range order {
		agg := byName[name]
		params := aggregatePamParams(agg.optionSets...)
		res, err := CreateResource(s.MqlRuntime, "pam.module", map[string]*llx.RawData{
			"name":    llx.StringData(agg.name),
			"params":  llx.MapData(params, types.String),
			"enabled": llx.BoolData(agg.anyEnabled),
			"entries": llx.ArrayData(agg.entries, types.Resource("pam.conf.serviceEntry")),
		})
		if err != nil {
			return nil, err
		}
		out = append(out, res)
	}
	return out, nil
}

func (s *mqlPamConf) entries(files []any) (map[string]any, error) {
	conn := s.MqlRuntime.Connection.(shared.Connection)

	contents := map[string]string{}
	var notReadyError error = nil

	for i := range files {
		file := files[i].(*mqlFile)

		f, err := conn.FileSystem().Open(file.Path.Data)
		if err != nil {
			return nil, err
		}

		raw, err := io.ReadAll(f)
		f.Close()
		if err != nil {
			return nil, err
		}

		contents[file.Path.Data] = string(raw)
	}

	if notReadyError != nil {
		return nil, notReadyError
	}

	services := map[string]any{}
	for basename, content := range contents {
		lines := strings.Split(content, "\n")
		settings := []any{}
		var line string
		for i := range lines {
			line = lines[i]

			entry, err := pam.ParseLine(line)
			if err != nil {
				return nil, err
			}

			// empty lines parse as empty object
			if entry == nil {
				continue
			}

			pamEntry, err := CreateResource(s.MqlRuntime, "pam.conf.serviceEntry", map[string]*llx.RawData{
				"service":    llx.StringData(basename),
				"lineNumber": llx.IntData(int64(i)), // Used for ID
				"pamType":    llx.StringData(entry.PamType),
				"control":    llx.StringData(entry.Control),
				"module":     llx.StringData(entry.Module),
				"options":    llx.ArrayData(entry.Options, types.String),
			})
			if err != nil {
				return nil, err
			}
			settings = append(settings, pamEntry.(*mqlPamConfServiceEntry))

		}

		services[basename] = settings
	}

	return services, nil
}
