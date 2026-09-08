// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package provider

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/llx"
	"go.mondoo.com/mql/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/providers/terraform/connection"
)

// connectToDir connects to a directory holding the given files and returns the
// resulting asset, after platform detection has run.
func connectToDir(t *testing.T, files map[string]string, options map[string]string) *inventory.Asset {
	t.Helper()

	dir := t.TempDir()
	for name, content := range files {
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644))
	}

	opts := map[string]string{"path": dir}
	for k, v := range options {
		opts[k] = v
	}

	srv := &Service{Service: plugin.NewService()}
	res, err := srv.Connect(&plugin.ConnectReq{
		Asset: &inventory.Asset{
			Connections: []*inventory.Config{{Type: HclConnectionType, Options: opts}},
		},
	}, nil)
	require.NoError(t, err)
	return res.Asset
}

const bucket = `resource "aws_s3_bucket" "b" { bucket = "example" }`

func TestDetectPlatformByDialect(t *testing.T) {
	t.Run("terraform files produce a terraform platform", func(t *testing.T) {
		asset := connectToDir(t, map[string]string{"main.tf": bucket}, nil)

		assert.Equal(t, "terraform-hcl", asset.Platform.Name)
		assert.Equal(t, "Terraform HCL", asset.Platform.Title)
		assert.Equal(t, []string{"terraform"}, asset.Platform.Family)
		assert.Contains(t, asset.Name, "Terraform HCL")
	})

	t.Run("tofu files produce an opentofu platform", func(t *testing.T) {
		asset := connectToDir(t, map[string]string{"main.tofu": bucket}, nil)

		assert.Equal(t, "opentofu-hcl", asset.Platform.Name)
		assert.Equal(t, "OpenTofu HCL", asset.Platform.Title)
		assert.Contains(t, asset.Name, "OpenTofu HCL")
	})

	t.Run("an opentofu asset stays in the terraform family", func(t *testing.T) {
		// This is what lets a policy written against the terraform family apply
		// unchanged to an OpenTofu asset. Dropping "terraform" from this chain
		// would silently stop every existing terraform policy from matching.
		asset := connectToDir(t, map[string]string{"main.tofu": bucket}, nil)

		assert.Equal(t, []string{"opentofu", "terraform"}, asset.Platform.Family)
		assert.Contains(t, asset.Platform.Family, "terraform")
	})

	t.Run("forcing terraform on a mixed directory keeps the terraform platform", func(t *testing.T) {
		asset := connectToDir(t,
			map[string]string{"main.tf": bucket, "main.tofu": bucket},
			map[string]string{connection.OptionDialect: "terraform"})

		assert.Equal(t, "terraform-hcl", asset.Platform.Name)
	})

	t.Run("the connection runtime agrees with the platform catalog", func(t *testing.T) {
		// Other providers build their platform from conn.Runtime(). If this
		// provider is ever wired up that way, the two must not disagree: a
		// connection reporting "terraform" while its platform declares
		// "opentofu" would produce an asset that contradicts itself.
		for files, wantRuntime := range map[string]string{
			"main.tf":   "terraform",
			"main.tofu": "opentofu",
		} {
			asset := connectToDir(t, map[string]string{files: bucket}, nil)
			assert.Equal(t, wantRuntime, asset.Platform.Runtime, "platform runtime for %s", files)
			assert.True(t, PlatformByName(asset.Platform.Name).Consistent(asset.Platform),
				"platform %s should be consistent with its catalog entry", asset.Platform.Name)
		}
	})

	t.Run("both dialects share a platform ID for the same path", func(t *testing.T) {
		// The platform ID identifies the project, not the tool applying it, so
		// a repository that migrates from Terraform to OpenTofu stays the same
		// asset rather than forking into a second one.
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "main.tf"), []byte(bucket), 0o644))

		srv := &Service{Service: plugin.NewService()}
		connect := func(opts map[string]string) string {
			o := map[string]string{"path": dir}
			for k, v := range opts {
				o[k] = v
			}
			res, err := srv.Connect(&plugin.ConnectReq{
				Asset: &inventory.Asset{
					Connections: []*inventory.Config{{Type: HclConnectionType, Options: o}},
				},
			}, nil)
			require.NoError(t, err)
			return res.Asset.PlatformIds[0]
		}

		asTerraform := connect(nil)
		asOpenTofu := connect(map[string]string{connection.OptionDialect: "opentofu"})
		assert.Equal(t, asTerraform, asOpenTofu)
	})
}

func TestParseCLIDialect(t *testing.T) {
	parse := func(connector string, flags map[string]*llx.Primitive) *inventory.Config {
		srv := &Service{Service: plugin.NewService()}
		res, err := srv.ParseCLI(&plugin.ParseCLIReq{
			Connector: connector,
			Args:      []string{"/some/path"},
			Flags:     flags,
		})
		require.NoError(t, err)
		return res.Asset.Connections[0]
	}

	t.Run("the terraform connector records no dialect", func(t *testing.T) {
		// Left unset so an HCL configuration can detect its own dialect; only an
		// explicit choice is recorded.
		conf := parse("terraform", nil)
		assert.Empty(t, conf.Options[connection.OptionDialect])
	})

	for _, tool := range []string{"opentofu", "tofu"} {
		t.Run("--iac-tool "+tool+" selects opentofu", func(t *testing.T) {
			conf := parse("terraform", map[string]*llx.Primitive{
				"iac-tool": llx.StringPrimitive(tool),
			})
			assert.Equal(t, "opentofu", conf.Options[connection.OptionDialect])
		})
	}

	t.Run("--iac-tool terraform forces terraform", func(t *testing.T) {
		conf := parse("terraform", map[string]*llx.Primitive{
			"iac-tool": llx.StringPrimitive("terraform"),
		})
		assert.Equal(t, "terraform", conf.Options[connection.OptionDialect])
	})

	t.Run("an empty --iac-tool does not force a dialect", func(t *testing.T) {
		conf := parse("terraform", map[string]*llx.Primitive{
			"iac-tool": llx.StringPrimitive(""),
		})
		assert.Empty(t, conf.Options[connection.OptionDialect])
	})
}
