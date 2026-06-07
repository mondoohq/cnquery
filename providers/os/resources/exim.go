// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"io"
	"io/fs"
	"strings"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/os/connection/shared"
)

const (
	// Debian's exim4 split config keeps the dc_* control values, including
	// dc_local_interfaces, in this file.
	debianEximConfig = "/etc/exim4/update-exim4.conf.conf"
	// the monolithic configuration used by RHEL and most source installs
	monolithicEximConfig = "/etc/exim/exim.conf"
)

func (e *mqlExim) id() (string, error) {
	// a constant id keeps creation cheap; configPath() may shell out to
	// `exim -bP` to locate a non-standard config, which shouldn't run just
	// to compute the cache key
	return "exim", nil
}

func initExim(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if x, ok := args["path"]; ok && x != nil {
		path, ok := x.Value.(string)
		if !ok {
			return nil, nil, errors.New("wrong type for 'path' in exim initialization, it must be a string")
		}
		args["configPath"] = llx.StringData(path)
		delete(args, "path")
	}
	return args, nil, nil
}

func (e *mqlExim) configPath() (string, error) {
	// only reached when no path was supplied via init
	conn := e.MqlRuntime.Connection.(shared.Connection)
	if asset := conn.Asset(); asset != nil && asset.Platform != nil && asset.Platform.IsFamily("debian") {
		// Debian's exim4-config package keeps dc_local_interfaces in its
		// split-config control file regardless of where the assembled config
		// lives, so this is the file to read on Debian-family systems
		return debianEximConfig, nil
	}

	// otherwise ask Exim where its configuration lives — this handles BSD
	// ports/pkgsrc, source installs, and other non-standard prefixes rather
	// than assuming /etc/exim/exim.conf
	if path := e.eximConfigureFile(); path != "" {
		return path, nil
	}
	return monolithicEximConfig, nil
}

// eximConfigureFile returns the configuration file Exim itself reports via
// `exim -bP configure_file`, or "" when the binary is unavailable.
func (e *mqlExim) eximConfigureFile() string {
	o, err := CreateResource(e.MqlRuntime, "command", map[string]*llx.RawData{
		"command": llx.StringData("exim -bP configure_file"),
	})
	if err != nil {
		return ""
	}
	cmd := o.(*mqlCommand)

	exit := cmd.GetExitcode()
	if exit.Error != nil || exit.Data != 0 {
		return ""
	}
	stdout := cmd.GetStdout()
	if stdout.Error != nil {
		return ""
	}

	// configure_file reports the single config file in use; take the first line
	path := strings.TrimSpace(stdout.Data)
	if idx := strings.IndexByte(path, '\n'); idx >= 0 {
		path = strings.TrimSpace(path[:idx])
	}
	// Exim prints just the path here, but normalize defensively in case a build
	// emits the general `optionname = value` form so the prefix doesn't leak
	// into the path and break the file read
	if rest, ok := strings.CutPrefix(path, "configure_file"); ok {
		if after, ok := strings.CutPrefix(strings.TrimSpace(rest), "="); ok {
			path = strings.TrimSpace(after)
		}
	}
	return path
}

func (e *mqlExim) params() (map[string]any, error) {
	configPath := e.GetConfigPath()
	if configPath.Error != nil {
		return nil, configPath.Error
	}
	content, err := e.readFile(configPath.Data)
	if err != nil {
		return nil, err
	}
	return parseEximConfig(content), nil
}

func (e *mqlExim) localInterfaces() ([]any, error) {
	params := e.GetParams()
	if params.Error != nil {
		return nil, params.Error
	}

	// Debian split config: dc_local_interfaces is a ' ; '-separated string
	if v, ok := params.Data["dc_local_interfaces"].(string); ok && v != "" {
		return splitEximList(v, ";"), nil
	}
	// monolithic exim.conf: local_interfaces is an Exim list (default ':'
	// separator, overridable with a leading '<sep')
	if v, ok := params.Data["local_interfaces"].(string); ok && v != "" {
		return parseEximListValue(v), nil
	}
	return []any{}, nil
}

// readFile reads a config file from the target's filesystem. A missing file
// yields empty content rather than an error, so an MTA that isn't installed
// resolves cleanly.
func (e *mqlExim) readFile(path string) (string, error) {
	conn := e.MqlRuntime.Connection.(shared.Connection)
	f, err := conn.FileSystem().Open(path)
	if err != nil {
		// the connection's virtual filesystem may not return *os.PathError,
		// so match the wrapped sentinel rather than using os.IsNotExist
		if errors.Is(err, fs.ErrNotExist) {
			return "", nil
		}
		return "", err
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// parseEximConfig parses the macros and main-section options of an Exim config.
// It handles both the Debian `KEY='value'` form and the monolithic `key = value`
// form, folds backslash continuations, strips comments, and stops at the first
// `begin <section>` block (routers, transports, acl, …) so only the main
// section and macros are returned. Keys that are not simple assignments (named
// lists like `domainlist x = …`, ACL conditions) are skipped.
func parseEximConfig(content string) map[string]any {
	params := map[string]any{}
	for _, line := range joinEximContinuations(content) {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		// the main section ends at the first `begin <section>` directive
		if trimmed == "begin" || strings.HasPrefix(trimmed, "begin ") {
			break
		}

		idx := strings.Index(trimmed, "=")
		if idx < 0 {
			continue
		}
		key := strings.TrimSpace(trimmed[:idx])
		key = strings.TrimSpace(strings.TrimPrefix(key, "hide ")) // `hide key = value`
		// a simple option or macro has a single-token name; skip named lists
		// (`domainlist x = …`) and other multi-token left-hand sides
		if key == "" || strings.ContainsAny(key, " \t") {
			continue
		}

		params[key] = unquoteEximValue(strings.TrimSpace(trimmed[idx+1:]))
	}
	return params
}

// joinEximContinuations joins lines ending in a backslash with the following
// line, matching Exim's line-continuation rule.
func joinEximContinuations(content string) []string {
	var logical []string
	var buf strings.Builder
	continuing := false

	for _, raw := range strings.Split(content, "\n") {
		line := strings.TrimRight(raw, "\r")
		if continuing {
			buf.WriteString(strings.TrimLeft(line, " \t"))
		} else {
			buf.Reset()
			buf.WriteString(line)
		}

		if strings.HasSuffix(strings.TrimRight(line, " \t"), "\\") {
			s := strings.TrimSuffix(strings.TrimRight(buf.String(), " \t"), "\\")
			buf.Reset()
			buf.WriteString(s)
			continuing = true
			continue
		}

		logical = append(logical, buf.String())
		continuing = false
	}
	if continuing {
		logical = append(logical, buf.String())
	}
	return logical
}

// unquoteEximValue strips a matching pair of surrounding single or double
// quotes (the Debian split config quotes its values).
func unquoteEximValue(value string) string {
	if len(value) >= 2 {
		first, last := value[0], value[len(value)-1]
		if (first == '\'' && last == '\'') || (first == '"' && last == '"') {
			return value[1 : len(value)-1]
		}
	}
	return value
}

// parseEximListValue parses an Exim list, honoring the optional `<sep` prefix
// that selects a non-default separator (needed for IPv6 addresses, which
// otherwise collide with the default ':' separator).
func parseEximListValue(value string) []any {
	value = strings.TrimSpace(value)
	sep := ":"
	if strings.HasPrefix(value, "<") && len(value) >= 2 {
		sep = string(value[1])
		value = strings.TrimSpace(value[2:])
	}
	return splitEximList(value, sep)
}

// splitEximList splits value on sep and trims each token, dropping empties.
func splitEximList(value, sep string) []any {
	res := []any{}
	for _, tok := range strings.Split(value, sep) {
		if tok = strings.TrimSpace(tok); tok != "" {
			res = append(res, tok)
		}
	}
	return res
}
