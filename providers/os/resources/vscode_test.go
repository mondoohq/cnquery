// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewVscodeExtensionPurl(t *testing.T) {
	cases := []struct {
		name      string
		publisher string
		extName   string
		version   string
		want      string
	}{
		{
			name:      "Roo Code (real marketplace extension)",
			publisher: "RooVeterinaryInc",
			extName:   "roo-cline",
			version:   "3.34.7",
			want:      "pkg:vscode-extension/RooVeterinaryInc/roo-cline@3.34.7",
		},
		{
			name:      "Microsoft Python (well-known prebuilt extension)",
			publisher: "ms-python",
			extName:   "python",
			version:   "2026.6.0",
			want:      "pkg:vscode-extension/ms-python/python@2026.6.0",
		},
		{
			name:      "version omitted (unknown / dev build)",
			publisher: "golang",
			extName:   "go",
			version:   "",
			want:      "pkg:vscode-extension/golang/go",
		},
		{
			name:      "empty publisher (incomplete metadata) yields empty PURL",
			publisher: "",
			extName:   "go",
			version:   "0.50.0",
			want:      "",
		},
		{
			name:      "empty name (incomplete metadata) yields empty PURL",
			publisher: "golang",
			extName:   "",
			version:   "0.50.0",
			want:      "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := newVscodeExtensionPurl(tc.publisher, tc.extName, tc.version)
			assert.Equal(t, tc.want, got)
		})
	}
}
