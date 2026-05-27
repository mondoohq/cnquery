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

// TestModuleNameFromPath locks down the extraction of a bare module name
// from the path entries found in modules.dep and modules.builtin. The tricky
// parts are the compression suffixes that modern kernels apply to .ko files
// (.xz / .zst / .gz) and the dash↔underscore normalization the kernel applies
// to module names.
func TestModuleNameFromPath(t *testing.T) {
	cases := []struct {
		in  string
		out string
	}{
		{"kernel/net/netfilter/nf_conntrack.ko", "nf_conntrack"},
		{"kernel/fs/cramfs/cramfs.ko.xz", "cramfs"},
		{"kernel/fs/squashfs/squashfs.ko.zst", "squashfs"},
		{"kernel/drivers/usb/storage/usb-storage.ko.gz", "usb_storage"},
		// leading/trailing whitespace (modules.dep keeps the colon on the LHS,
		// which the caller strips, but tabs around builtin entries appear too)
		{"\tkernel/fs/ext4/ext4.ko\t", "ext4"},
		{"snd-hda-intel.ko", "snd_hda_intel"},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			assert.Equal(t, tc.out, moduleNameFromPath(tc.in))
		})
	}
}

// TestNormalizeModuleName confirms dashes collapse to underscores so a lookup
// by either spelling resolves the same module (the kernel treats them as
// equivalent and lsmod always reports underscores).
func TestNormalizeModuleName(t *testing.T) {
	assert.Equal(t, "usb_storage", normalizeModuleName("usb-storage"))
	assert.Equal(t, "usb_storage", normalizeModuleName("usb_storage"))
	assert.Equal(t, "nf_conntrack", normalizeModuleName("nf_conntrack"))
	assert.Equal(t, "", normalizeModuleName(""))
}

// TestParseModulesDep locks down the modules.dep parser behind
// kernel.module.onDisk. Real Debian/Ubuntu indexes compress modules
// (.ko.zst / .ko.xz), list each loadable module as the left-hand side of a
// "module: deps" line, and may carry hundreds of dependency paths on the
// right that must NOT be treated as separately installed modules unless they
// appear as their own left-hand entry.
func TestParseModulesDep(t *testing.T) {
	cases := []struct {
		name    string
		content string
		present []string // expected onDisk
		absent  []string // expected NOT onDisk
	}{
		{
			name:    "single module, no deps, trailing colon",
			content: "kernel/fs/overlayfs/overlay.ko.zst:",
			present: []string{"overlay"},
		},
		{
			name: "module with deps records only the left-hand side",
			content: "kernel/net/netfilter/nf_conntrack.ko: " +
				"kernel/net/netfilter/nf_defrag_ipv4.ko kernel/lib/libcrc32c.ko",
			present: []string{"nf_conntrack"},
			// deps on the RHS are not independently installed unless they
			// also appear as their own left-hand entry.
			absent: []string{"nf_defrag_ipv4", "libcrc32c"},
		},
		{
			name: "dep that also has its own line is present",
			content: "kernel/net/netfilter/nf_conntrack.ko: kernel/lib/libcrc32c.ko\n" +
				"kernel/lib/libcrc32c.ko:",
			present: []string{"nf_conntrack", "libcrc32c"},
		},
		{
			name: "all compression suffixes and dash normalization",
			content: "kernel/fs/cramfs/cramfs.ko.xz:\n" +
				"kernel/fs/squashfs/squashfs.ko.zst:\n" +
				"kernel/drivers/usb/storage/usb-storage.ko.gz:\n" +
				"kernel/sound/pci/hda/snd-hda-intel.ko:",
			present: []string{"cramfs", "squashfs", "usb_storage", "snd_hda_intel"},
		},
		{
			name:    "blank lines, whitespace-only lines, and a trailing newline are ignored",
			content: "\n  \n\t\nkernel/fs/jffs2/jffs2.ko:\n\n",
			present: []string{"jffs2"},
		},
		{
			name:    "duplicate module across lines is idempotent",
			content: "kernel/fs/hfs/hfs.ko:\nkernel/fs/hfs/hfs.ko: kernel/dep.ko",
			present: []string{"hfs"},
		},
		{
			name:    "empty content yields no modules",
			content: "",
			absent:  []string{"anything"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := parseModulesDep(tc.content)
			for _, name := range tc.present {
				assert.True(t, got[name], "expected %q to be on disk", name)
			}
			for _, name := range tc.absent {
				assert.False(t, got[name], "expected %q NOT to be on disk", name)
			}
		})
	}
}

// TestParseModulesBuiltin locks down the modules.builtin parser behind
// kernel.module.builtIn. Each non-blank line is one module path; the file
// uses the ".ko" suffix on modern kernels but historically listed bare paths
// without an extension, and dash/underscore normalization applies the same
// way as modules.dep.
func TestParseModulesBuiltin(t *testing.T) {
	cases := []struct {
		name    string
		content string
		present []string
		absent  []string
	}{
		{
			name:    "modern .ko-suffixed entries",
			content: "kernel/fs/ext4/ext4.ko\nkernel/net/ipv4/tcp_cubic.ko",
			present: []string{"ext4", "tcp_cubic"},
		},
		{
			name:    "legacy entries without a .ko extension",
			content: "kernel/fs/ext4/ext4\nkernel/drivers/char/random",
			present: []string{"ext4", "random"},
		},
		{
			name:    "dash normalization",
			content: "kernel/drivers/usb/storage/usb-storage.ko",
			present: []string{"usb_storage"},
		},
		{
			name:    "blank lines and trailing newline ignored",
			content: "\nkernel/fs/ext4/ext4.ko\n  \n",
			present: []string{"ext4"},
		},
		{
			name:    "empty content yields no modules",
			content: "",
			absent:  []string{"ext4"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := parseModulesBuiltin(tc.content)
			for _, name := range tc.present {
				assert.True(t, got[name], "expected %q to be builtin", name)
			}
			for _, name := range tc.absent {
				assert.False(t, got[name], "expected %q NOT to be builtin", name)
			}
		})
	}
}
