package terraform

import (
	"go.mondoo.com/cnquery/motor/vault"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/motor/providers"
)

func TestTerraform(t *testing.T) {
	p, err := New(&providers.Config{
		Options: map[string]string{
			"path": "./testdata/hcl",
		},
	})
	require.NoError(t, err)

	files := p.Parser().Files()
	assert.Equal(t, len(files), 2)
}

func TestModuleManifestIssue676(t *testing.T) {
	// See https://github.com/mondoohq/cnquery/issues/676
	p, err := New(&providers.Config{
		Options: map[string]string{
			"path": "./testdata/issue676",
		},
	})
	require.NoError(t, err)

	require.NotNil(t, p.modulesManifest)
	require.Len(t, p.modulesManifest.Records, 3)
}

func TestGitCloneUrl(t *testing.T) {
	cloneUrl, err := gitCloneUrl("git+https://somegitlab.com/vendor/package.git", nil)
	require.NoError(t, err)
	assert.Equal(t, "git@somegitlab.com:vendor/package.git", cloneUrl)

	cloneUrl, err = gitCloneUrl("git+https://somegitlab.com/vendor/package.git", []*vault.Credential{{
		Type:     vault.CredentialType_password,
		User:     "oauth2",
		Password: "ACCESS_TOKEN",
	}})
	require.NoError(t, err)
	assert.Equal(t, "https://oauth2:ACCESS_TOKEN@somegitlab.com/vendor/package.git", cloneUrl)
}
