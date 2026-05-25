// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeCgroupPath(t *testing.T) {
	assert.Equal(t, "/", normalizeCgroupPath("/sys/fs/cgroup"))
	assert.Equal(t, "/system.slice", normalizeCgroupPath("/sys/fs/cgroup/system.slice"))
	assert.Equal(t,
		"/user.slice/user-1000.slice",
		normalizeCgroupPath("/sys/fs/cgroup/user.slice/user-1000.slice"))
	assert.Equal(t,
		"/system.slice/docker-abc123.scope",
		normalizeCgroupPath("/sys/fs/cgroup/system.slice/docker-abc123.scope"))
}

func TestParentCgroupPath(t *testing.T) {
	// Root is its own parent — callers use this to break the self-link.
	assert.Equal(t, "/", parentCgroupPath("/"))
	assert.Equal(t, "/", parentCgroupPath("/system.slice"))
	assert.Equal(t, "/system.slice", parentCgroupPath("/system.slice/docker.scope"))
	assert.Equal(t,
		"/user.slice/user-1000.slice",
		parentCgroupPath("/user.slice/user-1000.slice/session-c1.scope"))
}

func TestCgroupUnitTypeFromPath(t *testing.T) {
	assert.Equal(t, "root", cgroupUnitTypeFromPath("/"))
	assert.Equal(t, "slice", cgroupUnitTypeFromPath("/system.slice"))
	assert.Equal(t, "slice", cgroupUnitTypeFromPath("/user.slice/user-1000.slice"))
	assert.Equal(t, "scope", cgroupUnitTypeFromPath("/system.slice/docker-abc123.scope"))
	assert.Equal(t, "service", cgroupUnitTypeFromPath("/system.slice/nginx.service"))
	// Unrecognized leaves (e.g. raw kubelet/podman trees) fall through to "other".
	assert.Equal(t, "other", cgroupUnitTypeFromPath("/kubepods/burstable"))
}

func TestParseCgroupMax(t *testing.T) {
	// "max" is the kernel's literal for "no limit"; we surface it as -1
	// so MQL queries can use `< 0` to mean unlimited.
	assert.Equal(t, int64(-1), parseCgroupMax("max"))
	assert.Equal(t, int64(-1), parseCgroupMax(""))
	assert.Equal(t, int64(-1), parseCgroupMax("   "))
	// Missing controllers' files also read as "unlimited" — same outcome.
	assert.Equal(t, int64(-1), parseCgroupMax("not a number"))
	assert.Equal(t, int64(1073741824), parseCgroupMax("1073741824"))
	assert.Equal(t, int64(1073741824), parseCgroupMax(" 1073741824 "))
}

func TestParseCgroupInt(t *testing.T) {
	assert.Equal(t, int64(0), parseCgroupInt("", 0))
	assert.Equal(t, int64(100), parseCgroupInt("", 100))
	assert.Equal(t, int64(100), parseCgroupInt("garbage", 100))
	assert.Equal(t, int64(42), parseCgroupInt("42", 0))
	assert.Equal(t, int64(42), parseCgroupInt("  42  ", 0))
}

func TestParseCpuMax(t *testing.T) {
	// Default values for an empty file (controller not enabled).
	q, p := parseCpuMax("")
	assert.Equal(t, int64(-1), q)
	assert.Equal(t, int64(100000), p)

	// `max <period>` — quota unlimited, custom period.
	q, p = parseCpuMax("max 100000")
	assert.Equal(t, int64(-1), q)
	assert.Equal(t, int64(100000), p)

	// 50% CPU cap.
	q, p = parseCpuMax("50000 100000")
	assert.Equal(t, int64(50000), q)
	assert.Equal(t, int64(100000), p)

	// quota-only (rare, but defensible — keep period at default).
	q, p = parseCpuMax("50000")
	assert.Equal(t, int64(50000), q)
	assert.Equal(t, int64(100000), p)
}

func TestParseCgroupWalk(t *testing.T) {
	input := `===CGROUP===/sys/fs/cgroup
cgroup.controllers=cpuset cpu io memory hugetlb pids rdma misc
memory.max=max
memory.current=104857600
memory.high=max
memory.swap.max=max
cpu.max=max 100000
cpu.weight=100
pids.max=max
pids.current=42
cgroup.procs=1
===CGROUP===/sys/fs/cgroup/system.slice
cgroup.controllers=cpuset cpu io memory pids
memory.max=8589934592
memory.current=536870912
memory.high=max
memory.swap.max=max
cpu.max=200000 100000
cpu.weight=100
pids.max=4096
pids.current=85
cgroup.procs=
===CGROUP===/sys/fs/cgroup/system.slice/nginx.service
cgroup.controllers=
memory.max=104857600
memory.current=83886080
memory.swap.max=0
cpu.max=50000 100000
cpu.weight=50
pids.max=512
pids.current=4
cgroup.procs=1234 1235 1236 1237
`
	parsed := parseCgroupWalk(input)
	require.Len(t, parsed, 3)

	// Root
	assert.Equal(t, "/sys/fs/cgroup", parsed[0].rawPath)
	assert.Equal(t, "cpuset cpu io memory hugetlb pids rdma misc", parsed[0].attrs["cgroup.controllers"])
	assert.Equal(t, "max", parsed[0].attrs["memory.max"])
	assert.Equal(t, "104857600", parsed[0].attrs["memory.current"])
	assert.Equal(t, "42", parsed[0].attrs["pids.current"])
	assert.Equal(t, "1", parsed[0].attrs["cgroup.procs"])

	// system.slice with concrete memory cap
	assert.Equal(t, "/sys/fs/cgroup/system.slice", parsed[1].rawPath)
	assert.Equal(t, "8589934592", parsed[1].attrs["memory.max"])
	assert.Equal(t, "200000 100000", parsed[1].attrs["cpu.max"])
	assert.Equal(t, "", parsed[1].attrs["cgroup.procs"], "empty cgroup.procs file becomes empty string")

	// Service with explicit limits and a multi-pid procs file
	require.Equal(t, "/sys/fs/cgroup/system.slice/nginx.service", parsed[2].rawPath)
	assert.Equal(t, "104857600", parsed[2].attrs["memory.max"])
	assert.Equal(t, "1234 1235 1236 1237", parsed[2].attrs["cgroup.procs"])
}

func TestParseCgroupWalkEmpty(t *testing.T) {
	assert.Empty(t, parseCgroupWalk(""))
}

func TestPidsAsAny(t *testing.T) {
	// Numeric strings parse to int64; non-numeric tokens are dropped so
	// stray output from `tr` collapsing an empty file can't crash MQL.
	out := pidsAsAny([]string{"1234", "1235", "abc", "1236"})
	require.Len(t, out, 3)
	assert.Equal(t, int64(1234), out[0])
	assert.Equal(t, int64(1235), out[1])
	assert.Equal(t, int64(1236), out[2])

	assert.Empty(t, pidsAsAny(nil))
	assert.Empty(t, pidsAsAny([]string{}))
}

func TestStringsAsAny(t *testing.T) {
	out := stringsAsAny([]string{"memory", "cpu", "io"})
	require.Len(t, out, 3)
	assert.Equal(t, "memory", out[0])
	assert.Equal(t, "io", out[2])
	assert.Empty(t, stringsAsAny(nil))
}
