// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"strconv"
	"strings"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/types"
)

// =============================================================================
// kernel.cmdline — /proc/cmdline
// =============================================================================

func (k *mqlKernel) cmdline() (*mqlKernelCmdline, error) {
	raw, ok, err := readKernelHardeningFile(k.MqlRuntime, "/proc/cmdline")
	if err != nil {
		return nil, err
	}
	if !ok {
		raw = ""
	}
	raw = strings.TrimRight(raw, "\n")

	params, flags := parseKernelCmdline(raw)

	resource, err := CreateResource(k.MqlRuntime, "kernel.cmdline", map[string]*llx.RawData{
		"raw":        llx.StringData(raw),
		"parameters": llx.MapData(params, types.String),
		"flags":      llx.ArrayData(flags, types.String),
	})
	if err != nil {
		return nil, err
	}
	return resource.(*mqlKernelCmdline), nil
}

func (c *mqlKernelCmdline) id() (string, error) {
	return "kernel.cmdline", nil
}

// parseKernelCmdline splits /proc/cmdline into `key=value` parameters
// and bare flags. Duplicate parameters collapse to the last occurrence
// (extremely rare outside of `console=`); the raw string is preserved
// on the resource for callers who need full fidelity.
func parseKernelCmdline(raw string) (map[string]any, []any) {
	params := map[string]any{}
	var flags []any
	for _, tok := range strings.Fields(raw) {
		if idx := strings.IndexByte(tok, '='); idx > 0 {
			params[tok[:idx]] = tok[idx+1:]
			continue
		}
		flags = append(flags, tok)
	}
	return params, flags
}

// =============================================================================
// kernel.taint — /proc/sys/kernel/tainted
// =============================================================================

func (k *mqlKernel) taint() (*mqlKernelTaint, error) {
	raw, ok, err := readKernelHardeningFile(k.MqlRuntime, "/proc/sys/kernel/tainted")
	if err != nil {
		return nil, err
	}

	var bitmask int64
	if ok {
		if v, perr := strconv.ParseInt(strings.TrimSpace(raw), 10, 64); perr == nil {
			bitmask = v
		}
	}

	reasons := taintReasons(bitmask)

	resource, err := CreateResource(k.MqlRuntime, "kernel.taint", map[string]*llx.RawData{
		"bitmask": llx.IntData(bitmask),
		"tainted": llx.BoolData(bitmask != 0),
		"reasons": llx.ArrayData(stringsAsAnySlice(reasons), types.String),
	})
	if err != nil {
		return nil, err
	}
	return resource.(*mqlKernelTaint), nil
}

func (t *mqlKernelTaint) id() (string, error) {
	return "kernel.taint", nil
}

// taintBits maps bit positions to the human-readable reasons documented
// in Documentation/admin-guide/tainted-kernels.rst. The list is bit-
// ordered; new bits append at the end.
var taintBits = []string{
	"proprietary module loaded",    // 0  G/P
	"module force-loaded",          // 1  F
	"SMP with CPU mismatch",        // 2  S
	"module force-unloaded",        // 3  R
	"machine check exception",      // 4  M
	"bad page",                     // 5  B
	"taint requested by userspace", // 6  U
	"kernel oops or BUG",           // 7  D
	"ACPI table overridden",        // 8  A
	"kernel issued warning",        // 9  W
	"staging driver loaded",        // 10 C
	"firmware workaround applied",  // 11 I
	"out-of-tree module loaded",    // 12 O
	"unsigned module loaded",       // 13 E
	"soft lockup occurred",         // 14 L
	"kernel was live-patched",      // 15 K
	"auxiliary taint",              // 16 X
	"struct randomization plugin",  // 17 T
	"in-kernel test",               // 18
}

func taintReasons(bitmask int64) []string {
	if bitmask == 0 {
		return []string{}
	}
	out := []string{}
	for i, reason := range taintBits {
		if bitmask&(1<<uint(i)) != 0 {
			out = append(out, reason)
		}
	}
	return out
}

// =============================================================================
// kernel.lockdown — /sys/kernel/security/lockdown
// =============================================================================

func (k *mqlKernel) lockdown() (*mqlKernelLockdown, error) {
	raw, ok, err := readKernelHardeningFile(k.MqlRuntime, "/sys/kernel/security/lockdown")
	if err != nil {
		return nil, err
	}

	mode := "unavailable"
	if ok {
		mode = parseLockdownMode(raw)
	}

	resource, err := CreateResource(k.MqlRuntime, "kernel.lockdown", map[string]*llx.RawData{
		"mode":    llx.StringData(mode),
		"enabled": llx.BoolData(mode == "integrity" || mode == "confidentiality"),
	})
	if err != nil {
		return nil, err
	}
	return resource.(*mqlKernelLockdown), nil
}

func (l *mqlKernelLockdown) id() (string, error) {
	return "kernel.lockdown", nil
}

// parseLockdownMode extracts the active mode from
// `/sys/kernel/security/lockdown`. The file lists every supported mode
// with the active one in square brackets, e.g. `[none] integrity
// confidentiality`. We return the bracketed token, or `unknown` if
// the file is malformed and no bracketed token is found.
func parseLockdownMode(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "unknown"
	}
	for _, tok := range strings.Fields(raw) {
		if len(tok) >= 2 && tok[0] == '[' && tok[len(tok)-1] == ']' {
			return tok[1 : len(tok)-1]
		}
	}
	return "unknown"
}

// =============================================================================
// kernel.aslr — /proc/sys/kernel/randomize_va_space
// =============================================================================

func (k *mqlKernel) aslr() (*mqlKernelAslr, error) {
	raw, ok, err := readKernelHardeningFile(k.MqlRuntime, "/proc/sys/kernel/randomize_va_space")
	if err != nil {
		return nil, err
	}

	mode := int64(-1)
	level := "unknown"
	if ok {
		if v, perr := strconv.ParseInt(strings.TrimSpace(raw), 10, 64); perr == nil {
			mode = v
			level = aslrLevel(v)
		}
	}

	resource, err := CreateResource(k.MqlRuntime, "kernel.aslr", map[string]*llx.RawData{
		"mode":    llx.IntData(mode),
		"level":   llx.StringData(level),
		"enabled": llx.BoolData(mode > 0),
	})
	if err != nil {
		return nil, err
	}
	return resource.(*mqlKernelAslr), nil
}

func (a *mqlKernelAslr) id() (string, error) {
	return "kernel.aslr", nil
}

func aslrLevel(mode int64) string {
	switch mode {
	case 0:
		return "disabled"
	case 1:
		return "conservative"
	case 2:
		return "full"
	default:
		return "unknown"
	}
}

// =============================================================================
// shared helpers
// =============================================================================

// readKernelHardeningFile reads a file in /proc or /sys via the file
// resource, returning (content, true, nil) on success and ("", false,
// nil) when the file is unavailable (missing kernel feature, non-Linux
// host, or unreadable). Honest errors from the runtime still surface.
func readKernelHardeningFile(runtime *plugin.Runtime, path string) (string, bool, error) {
	o, err := CreateResource(runtime, "file", map[string]*llx.RawData{
		"path": llx.StringData(path),
	})
	if err != nil {
		return "", false, err
	}
	f := o.(*mqlFile)
	content := f.GetContent()
	if content.Error != nil {
		return "", false, nil
	}
	return content.Data, true, nil
}

// stringsAsAnySlice converts a []string to []any so it can be wrapped
// in llx.ArrayData without colliding with the cgroup/firewalld helpers
// defined elsewhere in this package.
func stringsAsAnySlice(in []string) []any {
	out := make([]any, len(in))
	for i, s := range in {
		out[i] = s
	}
	return out
}
