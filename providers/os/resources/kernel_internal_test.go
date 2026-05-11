// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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
