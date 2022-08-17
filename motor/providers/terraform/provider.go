package terraform

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"

	"github.com/cockroachdb/errors"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/spf13/afero"
	"go.mondoo.io/mondoo/motor/providers"
	"go.mondoo.io/mondoo/motor/providers/os/fsutil"
)

var (
	_ providers.Transport                   = (*Provider)(nil)
	_ providers.TransportPlatformIdentifier = (*Provider)(nil)
)

func New(tc *providers.Config) (*Provider, error) {
	if tc.Options == nil || tc.Options["path"] == "" {
		return nil, errors.New("path is required")
	}

	path := tc.Options["path"]
	stat, err := os.Stat(path)
	if os.IsNotExist(err) {
		return nil, errors.New("path is not a valid file or directory")
	}

	loader := NewHCLFileLoader()
	tfVars := make(map[string]*hcl.Attribute)
	var modulesManifest *ModuleManifest

	if stat.IsDir() {
		fileList, err := os.ReadDir(path)
		if err != nil {
			return nil, err
		}

		err = loader.ParseHclDirectory(path, fileList)
		if err != nil {
			return nil, errors.Wrap(err, "could not parse hcl files")
		}

		err = ReadTfVarsFromDir(path, fileList, tfVars)
		if err != nil {
			return nil, errors.Wrap(err, "could not parse tfvars files")
		}

		modulesManifest, err = ParseTerraformModuleManifest(path)
	} else {
		err = loader.ParseHclFile(path)
		if err != nil {
			return nil, errors.Wrap(err, "could not parse hcl file")
		}

		err = ReadTfVarsFromFile(path, tfVars)
		if err != nil {
			return nil, errors.Wrap(err, "could not parse tfvars file")
		}
	}

	absPath, _ := filepath.Abs(path)
	h := sha256.New()
	h.Write([]byte(absPath))
	hash := hex.EncodeToString(h.Sum(nil))

	platformID := "//platformid.api.mondoo.app/runtime/terraform/hash/" + hash

	return &Provider{
		platformID:      platformID,
		path:            path,
		parsed:          loader.GetParser(),
		tfVars:          tfVars,
		modulesManifest: modulesManifest,
	}, nil
}

// References:
// - https://www.terraform.io/docs/language/syntax/configuration.html
// - https://github.com/hashicorp/hcl/blob/main/hclsyntax/spec.md
type Provider struct {
	platformID      string
	path            string
	parsed          *hclparse.Parser
	tfVars          map[string]*hcl.Attribute
	modulesManifest *ModuleManifest
}

func (t *Provider) RunCommand(command string) (*providers.Command, error) {
	return nil, providers.ErrRunCommandNotImplemented
}

func (t *Provider) FileInfo(path string) (providers.FileInfoDetails, error) {
	return providers.FileInfoDetails{}, providers.ErrFileInfoNotImplemented
}

func (t *Provider) FS() afero.Fs {
	return &fsutil.NoFs{}
}

func (t *Provider) Close() {}

func (t *Provider) Capabilities() providers.Capabilities {
	return providers.Capabilities{}
}

func (t *Provider) Kind() providers.Kind {
	return providers.Kind_KIND_CODE
}

func (t *Provider) PlatformIdDetectors() []providers.PlatformIdDetector {
	return []providers.PlatformIdDetector{
		providers.TransportPlatformIdentifierDetector,
	}
}

func (t *Provider) Runtime() string {
	return ""
}

func (t *Provider) Parser() *hclparse.Parser {
	return t.parsed
}

func (t *Provider) TfVars() map[string]*hcl.Attribute {
	return t.tfVars
}

func (t *Provider) ModulesManifest() *ModuleManifest {
	return t.modulesManifest
}

func (t *Provider) Identifier() (string, error) {
	return t.platformID, nil
}
