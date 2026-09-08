// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/providers-sdk/v1/inventory"
)

func TestIsTofuConfigFile(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		// OpenTofu-specific configuration, per the OpenTofu "Specific Code
		// Override" RFC. These override the identically-named .tf file.
		{"main.tofu", true},
		{"main.tofu.json", true},
		{"prod.tofuvars", true},
		{"prod.tofuvars.json", true},
		{"main_override.tofu", true},
		{"/abs/path/to/main.tofu", true},

		// Terraform files, which this provider does read.
		{"main.tf", false},
		{"main.tf.json", false},
		{"prod.tfvars", false},
		{"prod.tfvars.json", false},

		// Test files carry no deployed configuration and the .tftest.*
		// equivalents are not read either, so they must not trip the error.
		{"main.tofutest.hcl", false},
		{"main.tofutest.json", false},

		// Unrelated files.
		{"README.md", false},
		{"main.go", false},
		{"tofu", false},
		{"", false},
		// a directory that merely contains the word must not match on its path
		{"/repo/tofu/main.tf", false},
	}

	for _, tc := range tests {
		t.Run(tc.path, func(t *testing.T) {
			assert.Equal(t, tc.want, isTofuConfigFile(tc.path))
		})
	}
}

func TestNewUnsupportedTofuFilesError(t *testing.T) {
	t.Run("nil when nothing was skipped", func(t *testing.T) {
		assert.NoError(t, newUnsupportedTofuFilesError(nil))
		assert.NoError(t, newUnsupportedTofuFilesError([]string{}))
	})

	t.Run("sorts the reported files", func(t *testing.T) {
		err := newUnsupportedTofuFilesError([]string{"c.tofu", "a.tofu", "b.tofu"})
		require.Error(t, err)
		var tofuErr *UnsupportedTofuFilesError
		require.ErrorAs(t, err, &tofuErr)
		assert.Equal(t, []string{"a.tofu", "b.tofu", "c.tofu"}, tofuErr.Files)
	})

	t.Run("does not mutate the caller's slice", func(t *testing.T) {
		input := []string{"c.tofu", "a.tofu"}
		_ = newUnsupportedTofuFilesError(input)
		assert.Equal(t, []string{"c.tofu", "a.tofu"}, input)
	})

	t.Run("message names the count and the files", func(t *testing.T) {
		err := newUnsupportedTofuFilesError([]string{"a.tofu", "b.tofu"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "found 2 OpenTofu configuration file(s)")
		assert.Contains(t, err.Error(), "a.tofu")
		assert.Contains(t, err.Error(), "b.tofu")
	})

	t.Run("truncates a long list but keeps the true count", func(t *testing.T) {
		err := newUnsupportedTofuFilesError([]string{
			"a.tofu", "b.tofu", "c.tofu", "d.tofu", "e.tofu", "f.tofu", "g.tofu",
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "found 7 OpenTofu configuration file(s)")
		assert.Contains(t, err.Error(), "(and 2 more)")
		assert.NotContains(t, err.Error(), "g.tofu")
	})
}

func TestClassifyDocument(t *testing.T) {
	tests := []struct {
		name    string
		doc     string
		want    documentKind
		wantErr bool
	}{
		{
			name: "terraform show -json output",
			doc:  `{"format_version":"1.0","terraform_version":"1.9.0","values":{}}`,
			want: documentShowJSON,
		},
		{
			name: "OpenTofu encrypted envelope",
			doc:  `{"encrypted_data":"AAAA","encryption_version":"v0","meta":{}}`,
			want: documentEncrypted,
		},
		{
			// An encrypted envelope must be recognised as encrypted even if a
			// future version also carries a format_version alongside it.
			name: "encryption wins over format_version",
			doc:  `{"format_version":"1.0","encrypted_data":"AAAA"}`,
			want: documentEncrypted,
		},
		{
			name: "raw terraform.tfstate",
			doc:  `{"version":4,"terraform_version":"1.9.0","serial":3,"resources":[]}`,
			want: documentNativeState,
		},
		{
			name: "empty JSON object",
			doc:  `{}`,
			want: documentUnrecognized,
		},
		{
			name:    "not JSON at all (binary plan file)",
			doc:     "\x00\x01binary",
			want:    documentUnrecognized,
			wantErr: true,
		},
		{
			name:    "JSON array rather than an object",
			doc:     `[]`,
			want:    documentUnrecognized,
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := classifyDocument([]byte(tc.doc))
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestValidateStateDocument(t *testing.T) {
	t.Run("accepts show -json output", func(t *testing.T) {
		assert.NoError(t, validateStateDocument(
			[]byte(`{"format_version":"1.0","values":{"root_module":{}}}`)))
	})

	t.Run("rejects an encrypted state file", func(t *testing.T) {
		err := validateStateDocument([]byte(`{"encrypted_data":"AAAA","encryption_version":"v0"}`))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "encrypted")
	})

	t.Run("rejects a raw state file", func(t *testing.T) {
		err := validateStateDocument([]byte(`{"version":4,"resources":[]}`))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "raw state file")
	})

	t.Run("rejects an object with no format_version", func(t *testing.T) {
		err := validateStateDocument([]byte(`{"something":"else"}`))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "format_version")
	})

	t.Run("rejects malformed JSON", func(t *testing.T) {
		assert.Error(t, validateStateDocument([]byte(`{not json`)))
	})
}

func TestValidatePlanDocument(t *testing.T) {
	t.Run("accepts show -json output", func(t *testing.T) {
		assert.NoError(t, validatePlanDocument(
			[]byte(`{"format_version":"1.1","resource_changes":[]}`)))
	})

	t.Run("rejects an encrypted plan file", func(t *testing.T) {
		err := validatePlanDocument([]byte(`{"encrypted_data":"AAAA"}`))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "encrypted")
	})

	t.Run("rejects a binary plan file", func(t *testing.T) {
		// real binary plan files start with a zip header
		err := validatePlanDocument([]byte("PK\x03\x04binary plan"))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "terraform show -json")
	})
}

// writeFiles creates a directory holding the given name -> content files.
func writeFiles(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, content := range files {
		path := filepath.Join(dir, name)
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
		require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	}
	return dir
}

func hclConnectionFor(path string) (*Connection, error) {
	return NewHclConnection(0, &inventory.Asset{
		Connections: []*inventory.Config{
			{Options: map[string]string{"path": path}, Type: "hcl"},
		},
	})
}

const validResource = `resource "aws_s3_bucket" "b" { bucket = "example" }`

func TestHclConnectionRejectsTofuFiles(t *testing.T) {
	// Regression test for a silent false-clean scan: .tofu files used to be
	// skipped without error, so an OpenTofu configuration scanned as an empty
	// one and every assertion over terraform.resources passed vacuously.
	t.Run("directory containing .tofu files", func(t *testing.T) {
		dir := writeFiles(t, map[string]string{
			"main.tofu":      validResource,
			"variables.tofu": `variable "region" { default = "us-east-1" }`,
		})

		_, err := hclConnectionFor(dir)
		require.Error(t, err)

		var tofuErr *UnsupportedTofuFilesError
		require.ErrorAs(t, err, &tofuErr)
		assert.Len(t, tofuErr.Files, 2)
	})

	t.Run("a single .tofu file given directly", func(t *testing.T) {
		dir := writeFiles(t, map[string]string{"main.tofu": validResource})

		_, err := hclConnectionFor(filepath.Join(dir, "main.tofu"))
		require.Error(t, err)

		var tofuErr *UnsupportedTofuFilesError
		require.ErrorAs(t, err, &tofuErr)
		assert.Len(t, tofuErr.Files, 1)
	})

	t.Run("a .tofu file alongside .tf files still fails", func(t *testing.T) {
		// OpenTofu loads main.tofu *instead of* main.tf, so reporting on the .tf
		// file here would describe configuration that is never deployed.
		dir := writeFiles(t, map[string]string{
			"main.tf":   `resource "aws_s3_bucket" "b" { acl = "private" }`,
			"main.tofu": `resource "aws_s3_bucket" "b" { acl = "public-read" }`,
		})

		_, err := hclConnectionFor(dir)
		require.Error(t, err)

		var tofuErr *UnsupportedTofuFilesError
		require.ErrorAs(t, err, &tofuErr)
		assert.Len(t, tofuErr.Files, 1)
	})

	t.Run("tofuvars files are reported too", func(t *testing.T) {
		dir := writeFiles(t, map[string]string{
			"main.tf":       validResource,
			"prod.tofuvars": `region = "us-east-1"`,
		})

		_, err := hclConnectionFor(dir)
		require.Error(t, err)

		var tofuErr *UnsupportedTofuFilesError
		require.ErrorAs(t, err, &tofuErr)
		assert.Len(t, tofuErr.Files, 1)
	})
}

func TestHclConnectionAcceptsTerraformFiles(t *testing.T) {
	// Guards the Terraform path against over-eager rejection: adding an
	// extension to tofuConfigExtensions that overlaps .tf would fail here.
	dir := writeFiles(t, map[string]string{
		"main.tf":      validResource,
		"prod.tfvars":  `region = "us-east-1"`,
		"sub/other.tf": `variable "x" { default = 1 }`,
		"README.md":    "not terraform",
		// test files must not trip the OpenTofu check
		"main.tofutest.hcl": `run "x" {}`,
	})

	conn, err := hclConnectionFor(dir)
	require.NoError(t, err)
	assert.Len(t, conn.Parser().Files(), 2)
	assert.Len(t, conn.TfVars(), 1)
}

func TestHclConnectionSurfacesParseErrors(t *testing.T) {
	// The WalkDir return value used to be discarded, so a malformed .tf file
	// was skipped silently and scanned as an empty configuration. Re-discarding
	// it would make this test fail.
	dir := writeFiles(t, map[string]string{
		"broken.tf": `resource "aws_s3_bucket" "b" {`,
	})

	_, err := hclConnectionFor(dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "could not parse hcl file")
}

func TestStateConnectionRejectsUnreadableFiles(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		wantMessage string
	}{
		{
			name:        "encrypted state",
			content:     `{"encrypted_data":"AAAA","encryption_version":"v0"}`,
			wantMessage: "encrypted",
		},
		{
			name:        "raw state file",
			content:     `{"version":4,"terraform_version":"1.9.0","resources":[]}`,
			wantMessage: "raw state file",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := writeFiles(t, map[string]string{"state.json": tc.content})

			_, err := NewStateConnection(0, &inventory.Asset{
				Connections: []*inventory.Config{
					{Options: map[string]string{"path": filepath.Join(dir, "state.json")}, Type: "state"},
				},
			})
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.wantMessage)
		})
	}
}
