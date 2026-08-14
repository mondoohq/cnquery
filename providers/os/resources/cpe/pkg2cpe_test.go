// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package cpe

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPkg2Gen(t *testing.T) {
	tests := []struct {
		name     string
		vendor   string
		pkg      string
		version  string
		release  string
		arch     string
		expected []string
	}{
		{
			name:    "escapes a plus in the version",
			vendor:  "tar",
			pkg:     "tar",
			version: "1.34+dfsg-1",
			expected: []string{
				"cpe:2.3:a:tar:tar:1.34\\+dfsg-1:*:*:*:*:*:*:*",
			},
		},
		{
			name:    "escapes a scoped npm name",
			vendor:  "@coreui/vue",
			pkg:     "@coreui/vue",
			version: "2.1.2",
			expected: []string{
				"cpe:2.3:a:\\@coreui\\/vue:\\@coreui\\/vue:2.1.2:*:*:*:*:*:*:*",
			},
		},
		{
			name:    "strips a zero epoch and keeps the arch",
			vendor:  "nextgen",
			pkg:     "mirthconnect",
			version: "0:4.4.0.b2948-1",
			arch:    "i386",
			expected: []string{
				"cpe:2.3:a:nextgen:mirthconnect:4.4.0.b2948-1:*:*:*:*:*:i386:*",
			},
		},
		{
			name:    "strips a multi digit epoch",
			vendor:  "acme",
			pkg:     "widget",
			version: "12:1.2.3-4",
			expected: []string{
				"cpe:2.3:a:acme:widget:1.2.3-4:*:*:*:*:*:*:*",
			},
		},
		{
			// The epoch pattern is anchored to leading digits, so it leaves
			// "v1:2.3" alone. WFNize then truncates at the colon because a
			// colon separates WFN components -- long-standing behavior of the
			// nvdtools binding, pinned here so the epoch handling above is not
			// mistaken for the cause.
			name:    "a non-epoch colon is not stripped as an epoch",
			vendor:  "acme",
			pkg:     "widget",
			version: "v1:2.3",
			expected: []string{
				"cpe:2.3:a:acme:widget:v1:*:*:*:*:*:*:*",
			},
		},
		{
			name:    "carries the release through as the update field",
			vendor:  "acme",
			pkg:     "widget",
			version: "1.2.3",
			release: "5",
			expected: []string{
				"cpe:2.3:a:acme:widget:1.2.3:5:*:*:*:*:*:*",
			},
		},
		{
			name:    "uppercase input is lowercased",
			vendor:  "ACME",
			pkg:     "Widget",
			version: "1.2.3",
			arch:    "X86_64",
			expected: []string{
				"cpe:2.3:a:acme:widget:1.2.3:*:*:*:*:*:x86_64:*",
			},
		},
		{
			// A CPE needs a product and a version, so these are not errors --
			// they simply produce nothing to match against.
			name:     "no name yields no cpe",
			vendor:   "acme",
			pkg:      "",
			version:  "1.2.3",
			expected: []string{},
		},
		{
			name:     "no version yields no cpe",
			vendor:   "acme",
			pkg:      "widget",
			version:  "",
			expected: []string{},
		},
		{
			name:     "an epoch with nothing after it yields no cpe",
			vendor:   "acme",
			pkg:      "widget",
			version:  "3:",
			expected: []string{},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cpes, err := NewPackage2Cpe(test.vendor, test.pkg, test.version, test.release, test.arch)
			require.NoError(t, err)
			assert.Equal(t, test.expected, cpes)
		})
	}
}

// NewPackage2Cpe runs two to three times per package, so the regexp and the
// field-name table are package level rather than rebuilt per call. This pins
// the allocation count so a future edit cannot quietly move them back into the
// function body.
func TestNewPackage2CpeAllocations(t *testing.T) {
	allocs := testing.AllocsPerRun(100, func() {
		_, _ = NewPackage2Cpe("red hat, inc.", "libfoo-dev", "2:1.2.3-2.el9", "", "x86_64")
	})
	assert.LessOrEqual(t, allocs, float64(30), "NewPackage2Cpe allocations regressed")
}

func BenchmarkNewPackage2Cpe(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = NewPackage2Cpe("red hat, inc.", "libfoo-dev", "2:1.2.3-2.el9", "", "x86_64")
	}
}
