// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"bufio"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/types"
)

// cgroupV2Probe determines what cgroup mode the host is running in by
// looking for the v2 unified-hierarchy marker file and falling back to
// v1's per-controller directories. The output is one of `V2`, `V1`, or
// `NONE`.
const cgroupV2Probe = `if [ -r /sys/fs/cgroup/cgroup.controllers ]; then echo V2; elif [ -d /sys/fs/cgroup/memory ]; then echo V1; else echo NONE; fi`

// cgroupV2Walk dumps the entire v2 cgroup tree in a single shell call.
// Each cgroup directory is separated by a `===CGROUP===<path>` line,
// followed by `key=value` pairs for the interesting attribute files.
// `-maxdepth 4` bounds traversal on busy hosts (e.g., K8s nodes with
// thousands of pod cgroups); the typical layout is
// `slice/sub-slice/unit`, which fits comfortably within four levels.
// The `tr '\n' ' '` collapses multi-line values (notably cgroup.procs,
// which is one PID per line) into a single space-separated string; the
// parser trims the trailing space that `tr` leaves behind.
const cgroupV2Walk = `find /sys/fs/cgroup -maxdepth 4 -name cgroup.controllers 2>/dev/null | while read f; do
  d="${f%/cgroup.controllers}"
  echo "===CGROUP===$d"
  for n in cgroup.controllers cgroup.type memory.max memory.current memory.high memory.swap.max cpu.max cpu.weight pids.max pids.current cgroup.procs; do
    if [ -r "$d/$n" ]; then
      printf '%s=' "$n"
      tr '\n' ' ' < "$d/$n" 2>/dev/null
      echo
    fi
  done
done`

const cgroupRoot = "/sys/fs/cgroup"

type mqlCgroupsInternal struct {
	lock     sync.Mutex
	probed   atomic.Bool
	detected *cgroupDetection
	probeErr error
}

type mqlCgroupInternal struct {
	childResources []any
}

type cgroupDetection struct {
	version     int64
	controllers []string
	root        *mqlCgroup
	flat        []*mqlCgroup
}

func (c *mqlCgroups) id() (string, error) {
	return "cgroups", nil
}

func (c *mqlCgroups) probe() (*cgroupDetection, error) {
	if c.probed.Load() {
		return c.detected, c.probeErr
	}
	c.lock.Lock()
	defer c.lock.Unlock()
	if c.probed.Load() {
		return c.detected, c.probeErr
	}
	c.detected, c.probeErr = c.doProbe()
	c.probed.Store(true)
	return c.detected, c.probeErr
}

func (c *mqlCgroups) doProbe() (*cgroupDetection, error) {
	det := &cgroupDetection{controllers: []string{}, flat: []*mqlCgroup{}}

	probe, ok, err := runShellCmd(c.MqlRuntime, cgroupV2Probe)
	if err != nil {
		return det, err
	}
	if !ok {
		return det, nil
	}

	switch strings.TrimSpace(probe) {
	case "V2":
		det.version = 2
	case "V1":
		det.version = 1
		// v1's per-controller hierarchies are intentionally unmodeled.
		return det, nil
	default:
		det.version = 0
		return det, nil
	}

	stdout, ok, err := runShellCmd(c.MqlRuntime, cgroupV2Walk)
	if err != nil {
		return det, err
	}
	if !ok {
		return det, nil
	}

	parsed := parseCgroupWalk(stdout)
	if len(parsed) == 0 {
		return det, nil
	}

	byPath := make(map[string]*mqlCgroup, len(parsed))
	for _, p := range parsed {
		relPath := normalizeCgroupPath(p.rawPath)
		cg, err := buildCgroupResource(c.MqlRuntime, relPath, p)
		if err != nil {
			return det, err
		}
		byPath[relPath] = cg
		det.flat = append(det.flat, cg)
	}

	// Link parents -> children. Root's parent is itself; skip self-links.
	for path, cg := range byPath {
		parent := parentCgroupPath(path)
		if parent == path {
			continue
		}
		if p, ok := byPath[parent]; ok {
			p.childResources = append(p.childResources, cg)
		}
	}

	det.root = byPath["/"]
	if det.root != nil {
		// Copy out controllers as []string so we don't share the backing
		// array with the resource's RawData (mutating one would mutate
		// the other).
		raw := det.root.Controllers.Data
		det.controllers = make([]string, 0, len(raw))
		for _, v := range raw {
			if s, ok := v.(string); ok {
				det.controllers = append(det.controllers, s)
			}
		}
	}
	// If the v2 probe matched but the root has no controllers, the
	// unified hierarchy is effectively unusable from a query
	// standpoint. Report version=0 so callers don't see a confusing
	// "v2 with no controllers" state.
	if det.root != nil && len(det.controllers) == 0 {
		det.version = 0
	}
	return det, nil
}

func (c *mqlCgroups) version() (int64, error) {
	d, err := c.probe()
	if err != nil {
		return 0, err
	}
	return d.version, nil
}

func (c *mqlCgroups) controllers() ([]any, error) {
	d, err := c.probe()
	if err != nil {
		return nil, err
	}
	out := make([]any, len(d.controllers))
	for i, s := range d.controllers {
		out[i] = s
	}
	return out, nil
}

func (c *mqlCgroups) root() (*mqlCgroup, error) {
	d, err := c.probe()
	if err != nil {
		return nil, err
	}
	if d.root == nil {
		c.Root.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	return d.root, nil
}

func (c *mqlCgroups) list() ([]any, error) {
	d, err := c.probe()
	if err != nil {
		return nil, err
	}
	out := make([]any, len(d.flat))
	for i, cg := range d.flat {
		out[i] = cg
	}
	return out, nil
}

func (cg *mqlCgroup) id() (string, error) {
	return "cgroup:" + cg.Path.Data, nil
}

func (cg *mqlCgroup) children() ([]any, error) {
	if cg.childResources == nil {
		return []any{}, nil
	}
	return cg.childResources, nil
}

// parsedCgroup is the raw key/value capture from one cgroup directory's
// section of the walk output. Conversion to numeric fields happens in
// buildCgroupResource so the parser stays a flat string-to-string map.
type parsedCgroup struct {
	rawPath string
	attrs   map[string]string
}

func parseCgroupWalk(stdout string) []parsedCgroup {
	var out []parsedCgroup
	var cur *parsedCgroup

	scanner := bufio.NewScanner(strings.NewReader(stdout))
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "===CGROUP===") {
			if cur != nil {
				out = append(out, *cur)
			}
			cur = &parsedCgroup{
				rawPath: strings.TrimPrefix(line, "===CGROUP==="),
				attrs:   map[string]string{},
			}
			continue
		}
		if cur == nil {
			continue
		}
		idx := strings.IndexByte(line, '=')
		if idx <= 0 {
			continue
		}
		cur.attrs[line[:idx]] = strings.TrimSpace(line[idx+1:])
	}
	if cur != nil {
		out = append(out, *cur)
	}
	return out
}

func buildCgroupResource(runtime *plugin.Runtime, relPath string, p parsedCgroup) (*mqlCgroup, error) {
	controllers := stringsAsAny(strings.Fields(p.attrs["cgroup.controllers"]))
	pids := pidsAsAny(strings.Fields(p.attrs["cgroup.procs"]))

	cpuQuota, cpuPeriod := parseCpuMax(p.attrs["cpu.max"])

	resource, err := CreateResource(runtime, "cgroup", map[string]*llx.RawData{
		"path":             llx.StringData(relPath),
		"type":             llx.StringData(strings.TrimSpace(p.attrs["cgroup.type"])),
		"unitType":         llx.StringData(cgroupUnitTypeFromPath(relPath)),
		"controllers":      llx.ArrayData(controllers, types.String),
		"memoryMax":        llx.IntData(parseCgroupMax(p.attrs["memory.max"])),
		"memoryHigh":       llx.IntData(parseCgroupMax(p.attrs["memory.high"])),
		"memoryCurrent":    llx.IntData(parseCgroupInt(p.attrs["memory.current"], 0)),
		"memorySwapMax":    llx.IntData(parseCgroupMax(p.attrs["memory.swap.max"])),
		"cpuMaxQuotaUSec":  llx.IntData(cpuQuota),
		"cpuMaxPeriodUSec": llx.IntData(cpuPeriod),
		"cpuWeight":        llx.IntData(parseCgroupInt(p.attrs["cpu.weight"], 100)),
		"pidsMax":          llx.IntData(parseCgroupMax(p.attrs["pids.max"])),
		"pidsCurrent":      llx.IntData(parseCgroupInt(p.attrs["pids.current"], 0)),
		"pids":             llx.ArrayData(pids, types.Int),
	})
	if err != nil {
		return nil, err
	}
	return resource.(*mqlCgroup), nil
}

// normalizeCgroupPath converts an absolute filesystem path under
// /sys/fs/cgroup into a path relative to the cgroup root. The root
// itself becomes "/".
func normalizeCgroupPath(absPath string) string {
	rel := strings.TrimPrefix(absPath, cgroupRoot)
	if rel == "" {
		return "/"
	}
	return rel
}

// parentCgroupPath returns the path of the cgroup that contains this
// one. The root cgroup ("/") has no parent and returns itself, which
// callers use to skip the self-link.
func parentCgroupPath(path string) string {
	if path == "/" {
		return "/"
	}
	idx := strings.LastIndexByte(path, '/')
	if idx <= 0 {
		return "/"
	}
	return path[:idx]
}

// cgroupUnitTypeFromPath infers the systemd unit kind from the leaf
// name. This is distinct from the kernel's `cgroup.type` (which
// reports domain/threaded/etc.). We don't model arbitrary cgroup
// directories created outside systemd (e.g. Docker without
// systemd-cgroup) as a distinct kind — they get `other`.
func cgroupUnitTypeFromPath(path string) string {
	if path == "/" {
		return "root"
	}
	leaf := path
	if idx := strings.LastIndexByte(path, '/'); idx >= 0 {
		leaf = path[idx+1:]
	}
	switch {
	case strings.HasSuffix(leaf, ".slice"):
		return "slice"
	case strings.HasSuffix(leaf, ".scope"):
		return "scope"
	case strings.HasSuffix(leaf, ".service"):
		return "service"
	default:
		return "other"
	}
}

// parseCgroupMax parses a cgroup limit file. `max` (the kernel's
// literal for "no limit") becomes -1; anything else parses as an
// int64. Empty / missing / unparseable input becomes -1 too, so a
// cgroup that doesn't have the relevant controller enabled reads as
// "no limit" — semantically the same outcome as "max".
func parseCgroupMax(s string) int64 {
	s = strings.TrimSpace(s)
	if s == "" || s == "max" {
		return -1
	}
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return -1
	}
	return v
}

// parseCgroupInt parses a cgroup counter file (memory.current,
// pids.current, cpu.weight, etc.). On missing/invalid input it returns
// fallback so callers can distinguish "no value" from a real 0.
func parseCgroupInt(s string, fallback int64) int64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return fallback
	}
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return fallback
	}
	return v
}

// parseCpuMax parses the two-value `cpu.max` file (`<quota> <period>`).
// quota may be the literal `max` meaning unlimited (-1). period
// defaults to the kernel default of 100000 microseconds when absent.
func parseCpuMax(s string) (quota, period int64) {
	quota = -1
	period = 100000
	fields := strings.Fields(s)
	if len(fields) == 0 {
		return
	}
	if fields[0] != "max" {
		if v, err := strconv.ParseInt(fields[0], 10, 64); err == nil {
			quota = v
		}
	}
	if len(fields) >= 2 {
		if v, err := strconv.ParseInt(fields[1], 10, 64); err == nil {
			period = v
		}
	}
	return
}

func stringsAsAny(in []string) []any {
	out := make([]any, len(in))
	for i, s := range in {
		out[i] = s
	}
	return out
}

func pidsAsAny(in []string) []any {
	out := make([]any, 0, len(in))
	for _, s := range in {
		if v, err := strconv.ParseInt(s, 10, 64); err == nil {
			out = append(out, v)
		}
	}
	return out
}

// runShellCmd executes a command via the command resource. Returns
// (stdout, true, nil) on success and (empty, false, nil) when the
// command exits non-zero (e.g. shell unavailable, /sys/fs/cgroup
// inaccessible). This matches the pattern used by lvm.go and the
// systemd resources.
func runShellCmd(runtime *plugin.Runtime, cmdline string) (string, bool, error) {
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
