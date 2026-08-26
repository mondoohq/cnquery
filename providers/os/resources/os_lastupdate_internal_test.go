// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/llx"
	"go.mondoo.com/mql/providers-sdk/v1/inventory"
)

// isRpmPlatform decides whether the answer comes from the rpm database or from
// the updates package's file readers. A platform that falls on the wrong side
// reads null instead of a timestamp, which no error surfaces.
func TestIsRpmPlatform(t *testing.T) {
	tests := []struct {
		name     string
		platform *inventory.Platform
		want     bool
	}{
		{"rhel", &inventory.Platform{Name: "redhat", Family: []string{"redhat", "linux", "unix", "os"}}, true},
		{"fedora", &inventory.Platform{Name: "fedora", Family: []string{"redhat", "linux", "unix", "os"}}, true},
		{"centos", &inventory.Platform{Name: "centos", Family: []string{"redhat", "linux", "unix", "os"}}, true},
		{"sles", &inventory.Platform{Name: "sles", Family: []string{"suse", "linux", "unix", "os"}}, true},
		{"euler", &inventory.Platform{Name: "euleros", Family: []string{"euler", "linux", "unix", "os"}}, true},
		{"amazonlinux", &inventory.Platform{Name: "amazonlinux", Family: []string{"linux", "unix", "os"}}, true},
		{"photon", &inventory.Platform{Name: "photon", Family: []string{"linux", "unix", "os"}}, true},
		{"bottlerocket", &inventory.Platform{Name: "bottlerocket", Family: []string{"linux", "unix", "os"}}, true},
		{"azurelinux", &inventory.Platform{Name: "azurelinux", Family: []string{"linux", "unix", "os"}}, true},
		{"wrlinux", &inventory.Platform{Name: "wrlinux", Family: []string{"linux", "unix", "os"}}, true},
		{"mageia", &inventory.Platform{Name: "mageia", Family: []string{"linux", "unix", "os"}}, true},
		{"ubuntu", &inventory.Platform{Name: "ubuntu", Family: []string{"debian", "linux", "unix", "os"}}, false},
		{"debian", &inventory.Platform{Name: "debian", Family: []string{"debian", "linux", "unix", "os"}}, false},
		{"alpine", &inventory.Platform{Name: "alpine", Family: []string{"linux", "unix", "os"}}, false},
		{"arch", &inventory.Platform{Name: "arch", Family: []string{"arch", "linux", "unix", "os"}}, false},
		{"macos", &inventory.Platform{Name: "macos", Family: []string{"darwin", "bsd", "unix", "os"}}, false},
		{"windows", &inventory.Platform{Name: "windows", Family: []string{"windows", "os"}}, false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.want, isRpmPlatform(test.platform))
		})
	}
}

func TestLastUpdateAge(t *testing.T) {
	installed := time.Now().Add(-48 * time.Hour)

	age := lastUpdateAge(installed)
	require.NotNil(t, age)

	// The result is a duration-typed time, the same encoding uptime uses, so it
	// is read back through the epoch rather than as a wall-clock instant.
	seconds := llx.TimeToDuration(age)
	assert.InDelta(t, (48 * time.Hour).Seconds(), float64(seconds), 5)
}

// A log written in a zone ahead of the scanner's, or a skewed clock, puts the
// install in the future. A negative age would render as a nonsense duration, so
// it clamps to zero.
func TestLastUpdateAgeClampsFutureInstall(t *testing.T) {
	age := lastUpdateAge(time.Now().Add(24 * time.Hour))
	require.NotNil(t, age)
	assert.Equal(t, int64(0), llx.TimeToDuration(age))
}
