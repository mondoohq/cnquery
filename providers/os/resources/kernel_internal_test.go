// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestParseModprobeConfig locks down the modprobe.d parser used by the
// kernel.module {blacklisted, installBypass, disabled} accessors. The
// shapes here are taken from CIS Linux benchmarks (cramfs, usb-storage,
// freevxfs, jffs2, hfs) and from the in-the-wild quirks the parser has
// to tolerate — `exec /bin/false`, leading whitespace, comments mid-line,
// and unrelated directives (alias, options, softdep) that must be ignored
// without poisoning a sibling module's rule.
func TestParseModprobeConfig(t *testing.T) {
	cases := []struct {
		name    string
		content string
		want    map[string]modprobeRule
	}{
		{
			name:    "simple blacklist",
			content: "blacklist cramfs\n",
			want: map[string]modprobeRule{
				"cramfs": {blacklisted: true},
			},
		},
		{
			name:    "install short-circuit to /bin/true",
			content: "install usb-storage /bin/true\n",
			want: map[string]modprobeRule{
				"usb-storage": {installBypass: true},
			},
		},
		{
			name:    "install short-circuit to /bin/false",
			content: "install usb-storage /bin/false\n",
			want: map[string]modprobeRule{
				"usb-storage": {installBypass: true},
			},
		},
		{
			name:    "install short-circuit to /usr/bin/true",
			content: "install usb-storage /usr/bin/true\n",
			want: map[string]modprobeRule{
				"usb-storage": {installBypass: true},
			},
		},
		{
			name:    "install short-circuit to /usr/bin/false",
			content: "install usb-storage /usr/bin/false\n",
			want: map[string]modprobeRule{
				"usb-storage": {installBypass: true},
			},
		},
		{
			name:    "install via exec wrapper to /bin/false",
			content: "install usb-storage exec /bin/false\n",
			want: map[string]modprobeRule{
				"usb-storage": {installBypass: true},
			},
		},
		{
			name:    "install via exec wrapper to /usr/bin/false",
			content: "install usb-storage exec /usr/bin/false\n",
			want: map[string]modprobeRule{
				"usb-storage": {installBypass: true},
			},
		},
		{
			name:    "install to real modprobe is not a bypass",
			content: "install usb-storage /sbin/modprobe usb-storage-real\n",
			want:    map[string]modprobeRule{},
		},
		{
			name: "comments and blank lines are ignored",
			content: `# CIS Linux Benchmark 1.1.1.1
# Disable mounting of cramfs
blacklist cramfs

# Disable mounting of freevxfs

blacklist freevxfs   # trailing comment
`,
			want: map[string]modprobeRule{
				"cramfs":   {blacklisted: true},
				"freevxfs": {blacklisted: true},
			},
		},
		{
			name: "multiple modules combine across lines",
			content: `blacklist cramfs
install usb-storage /bin/false
blacklist freevxfs
install jffs2 /bin/true
`,
			want: map[string]modprobeRule{
				"cramfs":      {blacklisted: true},
				"usb-storage": {installBypass: true},
				"freevxfs":    {blacklisted: true},
				"jffs2":       {installBypass: true},
			},
		},
		{
			name: "same module blacklisted and install-bypassed unions both flags",
			content: `blacklist usb-storage
install usb-storage /bin/false
`,
			want: map[string]modprobeRule{
				"usb-storage": {blacklisted: true, installBypass: true},
			},
		},
		{
			name:    "leading whitespace, tabs, and mixed indentation tolerated",
			content: "  blacklist cramfs\n\tinstall usb-storage\t/bin/false\n  \t blacklist  freevxfs \n",
			want: map[string]modprobeRule{
				"cramfs":      {blacklisted: true},
				"usb-storage": {installBypass: true},
				"freevxfs":    {blacklisted: true},
			},
		},
		{
			name: "alias / options / softdep / remove are ignored",
			content: `alias net-pf-10 ipv6
options ipv6 disable=1
softdep nf_conntrack pre: nf_defrag_ipv4
remove fuse /sbin/modprobe -r fuse
blacklist cramfs
`,
			want: map[string]modprobeRule{
				"cramfs": {blacklisted: true},
			},
		},
		{
			name:    "blacklist without a module name is dropped",
			content: "blacklist\n",
			want:    map[string]modprobeRule{},
		},
		{
			name:    "install without a command is dropped",
			content: "install usb-storage\n",
			want:    map[string]modprobeRule{},
		},
		{
			name:    "empty content yields empty map",
			content: "",
			want:    map[string]modprobeRule{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := parseModprobeConfig(tc.content)
			assert.Equal(t, tc.want, got)
		})
	}
}

// TestStripModprobeComment guards the modprobe-flavoured comment stripper
// against drift toward rsyslog's quote-aware shape — modprobe has no
// string literals, so `#` always introduces a comment.
func TestStripModprobeComment(t *testing.T) {
	cases := []struct {
		in  string
		out string
	}{
		{"blacklist cramfs", "blacklist cramfs"},
		{"blacklist cramfs # comment", "blacklist cramfs "},
		{"# entire line is a comment", ""},
		{"", ""},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			assert.Equal(t, tc.out, stripModprobeComment(tc.in))
		})
	}
}

// TestRpmKernelMatchesRunning is the unit-level reproducer for
// customer-issues #178: AL2023's `kernel` rpm carries epoch 1, so
// pkg.Version is "1:6.1.170-210.320.amzn2023" while /proc/version returns
// "6.1.170-210.320.amzn2023.x86_64". A naive `pkgVersion+"."+arch ==
// runningKernelVersion` check fails for every installed kernel image, and
// the entire kernel.installed list comes back with running:false.
func TestRpmKernelMatchesRunning(t *testing.T) {
	cases := []struct {
		name           string
		pkgVersion     string
		pkgArch        string
		runningKernel  string
		expectedResult bool
	}{
		{
			name:           "AL2023 epoch-1 kernel matches running",
			pkgVersion:     "1:6.1.170-210.320.amzn2023",
			pkgArch:        "x86_64",
			runningKernel:  "6.1.170-210.320.amzn2023.x86_64",
			expectedResult: true,
		},
		{
			name:           "AL2023 epoch-1 kernel at older ABI does not match running",
			pkgVersion:     "1:6.1.166-197.305.amzn2023",
			pkgArch:        "x86_64",
			runningKernel:  "6.1.170-210.320.amzn2023.x86_64",
			expectedResult: false,
		},
		{
			name:           "RHEL legacy kernel with no epoch still matches",
			pkgVersion:     "3.10.0-1160.11.1.el7",
			pkgArch:        "x86_64",
			runningKernel:  "3.10.0-1160.11.1.el7.x86_64",
			expectedResult: true,
		},
		{
			name:           "Oracle UEK kernel with epoch matches running",
			pkgVersion:     "1:6.12.0-105.51.5.el9uek",
			pkgArch:        "x86_64",
			runningKernel:  "6.12.0-105.51.5.el9uek.x86_64",
			expectedResult: true,
		},
		{
			name:           "different architectures never match",
			pkgVersion:     "1:6.1.170-210.320.amzn2023",
			pkgArch:        "aarch64",
			runningKernel:  "6.1.170-210.320.amzn2023.x86_64",
			expectedResult: false,
		},
		{
			name:           "running-kernel string is empty (kernel.info unavailable)",
			pkgVersion:     "1:6.1.170-210.320.amzn2023",
			pkgArch:        "x86_64",
			runningKernel:  "",
			expectedResult: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := rpmKernelMatchesRunning(tc.pkgVersion, tc.pkgArch, tc.runningKernel)
			assert.Equal(t, tc.expectedResult, got)
		})
	}
}

func TestStripRPMEpoch(t *testing.T) {
	cases := []struct {
		name string
		in   string
		out  string
	}{
		{"no epoch", "6.1.170-210.320.amzn2023", "6.1.170-210.320.amzn2023"},
		{"epoch 1", "1:6.1.170-210.320.amzn2023", "6.1.170-210.320.amzn2023"},
		{"epoch 10 (multi-digit)", "10:6.1.170", "6.1.170"},
		{"empty", "", ""},
		{"bare colon", ":6.1.170", "6.1.170"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.out, stripRPMEpoch(tc.in))
		})
	}
}

// TestPhotonKernelMatchesRunning locks down the Photon comparison shape
// (version + flavor-suffix-from-name == runningKernelVersion) and proves
// the shared stripRPMEpoch primitive keeps the comparison working should
// Photon ever ship a kernel rpm with an Epoch declared.
func TestPhotonKernelMatchesRunning(t *testing.T) {
	cases := []struct {
		name           string
		pkgVersion     string
		pkgName        string
		runningKernel  string
		expectedResult bool
	}{
		{
			name:           "bare linux package matches running",
			pkgVersion:     "4.19.97-1.ph3",
			pkgName:        "linux",
			runningKernel:  "4.19.97-1.ph3",
			expectedResult: true,
		},
		{
			name:           "linux-esx flavor matches running with -esx suffix",
			pkgVersion:     "4.19.97-1.ph3",
			pkgName:        "linux-esx",
			runningKernel:  "4.19.97-1.ph3-esx",
			expectedResult: true,
		},
		{
			name:           "older inactive kernel does not match",
			pkgVersion:     "4.19.90-1.ph3",
			pkgName:        "linux",
			runningKernel:  "4.19.97-1.ph3",
			expectedResult: false,
		},
		{
			name:           "wrong flavor does not match",
			pkgVersion:     "4.19.97-1.ph3",
			pkgName:        "linux-rt",
			runningKernel:  "4.19.97-1.ph3-esx",
			expectedResult: false,
		},
		{
			name:           "hypothetical epoch-1 photon kernel still matches running",
			pkgVersion:     "1:4.19.97-1.ph3",
			pkgName:        "linux-esx",
			runningKernel:  "4.19.97-1.ph3-esx",
			expectedResult: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := photonKernelMatchesRunning(tc.pkgVersion, tc.pkgName, tc.runningKernel)
			assert.Equal(t, tc.expectedResult, got)
		})
	}
}

// TestSuseKernelMatchesRunning locks down the SUSE comparison shape:
// running ends with the package's -flavor suffix AND the trimmed running
// version is a prefix of the package version (accounting for the extra
// dpkg-release segment on pkg.Version). stripRPMEpoch is in the path so
// the comparison still works if a SUSE kernel rpm ever declares an Epoch.
func TestSuseKernelMatchesRunning(t *testing.T) {
	cases := []struct {
		name           string
		pkgVersion     string
		pkgName        string
		runningKernel  string
		expectedResult bool
	}{
		{
			name:           "kernel-default matches running with extra release segment",
			pkgVersion:     "4.12.14-122.23.1",
			pkgName:        "kernel-default",
			runningKernel:  "4.12.14-122.23-default",
			expectedResult: true,
		},
		{
			name:           "kernel-default at older version does not match",
			pkgVersion:     "4.12.14-122.20.1",
			pkgName:        "kernel-default",
			runningKernel:  "4.12.14-122.23-default",
			expectedResult: false,
		},
		{
			name:           "kernel-rt does not match a -default running kernel",
			pkgVersion:     "4.12.14-122.23.1",
			pkgName:        "kernel-rt",
			runningKernel:  "4.12.14-122.23-default",
			expectedResult: false,
		},
		{
			name:           "hypothetical epoch-1 SUSE kernel still matches running",
			pkgVersion:     "1:4.12.14-122.23.1",
			pkgName:        "kernel-default",
			runningKernel:  "4.12.14-122.23-default",
			expectedResult: true,
		},
		{
			name:           "empty running-kernel string never matches",
			pkgVersion:     "4.12.14-122.23.1",
			pkgName:        "kernel-default",
			runningKernel:  "",
			expectedResult: false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := suseKernelMatchesRunning(tc.pkgVersion, tc.pkgName, tc.runningKernel)
			assert.Equal(t, tc.expectedResult, got)
		})
	}
}
