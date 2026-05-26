// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package nfs

import (
	"io"
	"strings"
)

// parseAIXExports parses /etc/exports in the syntax used by the AIX
// NFS server. A line is
//
//	directory -option[,option...]
//
// where option is one of the flag keywords (`ro`, `rw`, etc.) or a
// `key=value` pair. Values for `rw`, `ro`, `root`, `access` are
// colon-separated host lists.
//
// Per AIX exports(5), when no `access=` list is given, named hosts
// in `rw=`/`ro=`/`root=` get the explicit per-host permission and
// **all other hosts** can still mount the share with the line's
// default permission. The parser captures the implicit "everyone
// else" access by emitting an additional `*` entry alongside each
// named host. `access=` overrides this — only the listed hosts can
// mount, and no wildcard row is emitted.
func parseAIXExports(r io.Reader) ([]ExportEntry, error) {
	var entries []ExportEntry
	err := scanExportLines(r, func(line string) error {
		fields := strings.Fields(line)
		if len(fields) == 0 || !strings.HasPrefix(fields[0], "/") {
			return nil
		}
		path := fields[0]
		var opts []string
		if len(fields) >= 2 {
			raw := strings.TrimPrefix(fields[1], "-")
			opts = splitOptionList(raw)
		}
		optMap := parseAIXOptMap(opts)
		globalRO := containsString(opts, "ro")
		rwHosts := optMap["rw"]
		roHosts := optMap["ro"]
		rootHosts := optMap["root"]

		for _, h := range aixHostList(optMap) {
			entries = append(entries, ExportEntry{
				Path:         path,
				Client:       h,
				Options:      opts,
				ReadOnly:     aixReadOnly(h, globalRO, rwHosts, roHosts),
				NoRootSquash: h != "*" && containsString(rootHosts, h),
			})
		}
		return nil
	})
	return entries, err
}

// parseAIXOptMap returns the values for the key=value options whose
// values are colon-separated host lists (rw, ro, root, access). Flag
// options (those without `=`) are not represented in the map.
func parseAIXOptMap(opts []string) map[string][]string {
	out := map[string][]string{}
	for _, o := range opts {
		key, value, ok := cutOnce(o, "=")
		if !ok {
			continue
		}
		out[key] = splitColonList(value)
	}
	return out
}

// aixHostList resolves the set of hosts an AIX exports line applies
// to. `access=` gates the share to exactly the named hosts. Without
// it, the result is the deduplicated union of `rw=`, `ro=`, and
// `root=` plus a trailing `*` row representing every other host
// (which AIX still permits to mount per exports(5)). When no host
// directives are given at all, the only row is the `*` row.
func aixHostList(optMap map[string][]string) []string {
	if list, ok := optMap["access"]; ok && len(list) > 0 {
		return list
	}
	seen := map[string]bool{}
	var hosts []string
	for _, key := range []string{"rw", "ro", "root"} {
		for _, h := range optMap[key] {
			if !seen[h] {
				seen[h] = true
				hosts = append(hosts, h)
			}
		}
	}
	return append(hosts, "*")
}

// aixReadOnly determines whether the (path, host) pair is exported
// read-only. Per AIX exports(5):
//   - A host explicitly named in `rw=` is read-write.
//   - A host explicitly named in `ro=` is read-only.
//   - Any other host (including `*` when no `access=` gates the line,
//     and `access=` hosts that aren't in `rw=` or `ro=`) falls back
//     to the line's default: read-only when bare `-ro` is set OR
//     when `rw=` names specific hosts (rest get ro per the docs),
//     otherwise read-write.
func aixReadOnly(host string, globalRO bool, rwHosts, roHosts []string) bool {
	if containsString(rwHosts, host) {
		return false
	}
	if containsString(roHosts, host) {
		return true
	}
	return globalRO || len(rwHosts) > 0
}

func splitColonList(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ":")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
