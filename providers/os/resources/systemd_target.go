// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"bufio"
	"strings"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/types"
)

func (t *mqlSystemdTarget) id() (string, error) {
	return "systemd.target:" + t.Name.Data, nil
}

func (x *mqlSystemdTargets) id() (string, error) {
	return "systemd.targets", nil
}

func (x *mqlSystemdTargets) list() ([]any, error) {
	names, err := listSystemdTargetNames(x.MqlRuntime)
	if err != nil {
		return nil, err
	}
	if len(names) == 0 {
		return []any{}, nil
	}

	res := make([]any, 0, len(names))
	for _, name := range names {
		props, err := showSystemdTargetProperties(x.MqlRuntime, name)
		if err != nil {
			// `systemctl show` returns exit 0 with empty properties for an
			// unknown unit. A genuine error here means systemctl itself is
			// gone, so surface it instead of swallowing it for every target.
			return nil, err
		}
		mqlTarget, err := CreateResource(x.MqlRuntime, "systemd.target", map[string]*llx.RawData{
			"name":          llx.StringData(name),
			"description":   llx.StringData(props["Description"]),
			"loadState":     llx.StringData(props["LoadState"]),
			"activeState":   llx.StringData(props["ActiveState"]),
			"subState":      llx.StringData(props["SubState"]),
			"unitFileState": llx.StringData(props["UnitFileState"]),
			"fragmentPath":  llx.StringData(props["FragmentPath"]),
			"wants":         llx.ArrayData(splitSystemdUnitList(props["Wants"]), types.String),
			"requires":      llx.ArrayData(splitSystemdUnitList(props["Requires"]), types.String),
			"after":         llx.ArrayData(splitSystemdUnitList(props["After"]), types.String),
			"before":        llx.ArrayData(splitSystemdUnitList(props["Before"]), types.String),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlTarget)
	}
	return res, nil
}

// listSystemdTargetNames returns the de-duplicated set of every .target
// unit known to systemd — both installed unit files and any
// transient/active targets — without the .target suffix. Empty list is
// returned when systemctl isn't installed (e.g. on a non-systemd host).
func listSystemdTargetNames(runtime *plugin.Runtime) ([]string, error) {
	installed, ok, err := runSystemctl(runtime,
		"systemctl list-unit-files --type=target --no-legend --plain --no-pager")
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, nil
	}
	active, ok, err := runSystemctl(runtime,
		"systemctl list-units --type=target --all --no-legend --plain --no-pager")
	if err != nil {
		return nil, err
	}
	if !ok {
		active = ""
	}

	seen := map[string]struct{}{}
	var names []string
	for _, line := range strings.Split(installed+"\n"+active, "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		unit := fields[0]
		if !strings.HasSuffix(unit, ".target") {
			continue
		}
		name := strings.TrimSuffix(unit, ".target")
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		names = append(names, name)
	}
	return names, nil
}

func showSystemdTargetProperties(runtime *plugin.Runtime, name string) (map[string]string, error) {
	stdout, ok, err := runSystemctl(runtime,
		"systemctl show "+shellQuoteUnit(name+".target")+" --no-pager")
	if err != nil {
		return nil, err
	}
	if !ok {
		return map[string]string{}, nil
	}
	return parseSystemdShowOutput(stdout), nil
}

// parseSystemdShowOutput consumes the `Key=Value` lines emitted by
// `systemctl show`. Values may themselves contain `=` (e.g.
// `Environment=FOO=bar`) so we split on the first `=` only.
func parseSystemdShowOutput(stdout string) map[string]string {
	out := map[string]string{}
	scanner := bufio.NewScanner(strings.NewReader(stdout))
	// Some unit properties (e.g. ExecStart) can be very long; raise the
	// per-line cap so we don't drop them.
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		idx := strings.IndexByte(line, '=')
		if idx <= 0 {
			continue
		}
		out[line[:idx]] = line[idx+1:]
	}
	return out
}

// splitSystemdUnitList splits the space-separated unit list that systemd
// emits in Wants=/Requires=/After=/Before= fields, returning []any for
// llx.ArrayData.
func splitSystemdUnitList(s string) []any {
	if s == "" {
		return []any{}
	}
	parts := strings.Fields(s)
	out := make([]any, 0, len(parts))
	for _, p := range parts {
		out = append(out, p)
	}
	return out
}

// runSystemctl executes a systemctl/resolvectl/timedatectl invocation via
// the command resource. Returns (stdout, true, nil) on success; (empty,
// false, nil) when the binary is missing or the command exits non-zero.
// Reusing this from systemd_resolved.go and systemd_timesyncd.go too.
func runSystemctl(runtime *plugin.Runtime, cmdline string) (string, bool, error) {
	o, err := CreateResource(runtime, "command", map[string]*llx.RawData{
		"command": llx.StringData(cmdline),
	})
	if err != nil {
		return "", false, err
	}
	cmd := o.(*mqlCommand)
	if exit := cmd.GetExitcode(); exit.Data != 0 {
		return "", false, nil
	}
	return cmd.Stdout.Data, true, nil
}

// shellQuoteUnit single-quotes a unit name for safe inclusion in a shell
// command line. systemd unit names normally contain only alnum + `-._@:\`
// (no whitespace or shell metacharacters), so the fast path returns the
// string unchanged. Defense-in-depth in case a future unit name carries
// something unexpected.
func shellQuoteUnit(s string) string {
	if s == "" {
		return "''"
	}
	if !strings.ContainsAny(s, " \t\n'\"\\$`") {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
