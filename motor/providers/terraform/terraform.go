package terraform

import (
	"crypto/sha256"
	"encoding/hex"
	"io/ioutil"
	"path/filepath"

	"github.com/cockroachdb/errors"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/spf13/afero"
	"go.mondoo.io/mondoo/motor/providers"
	"go.mondoo.io/mondoo/motor/providers/fsutil"
)

var (
	_ providers.Transport                   = (*Transport)(nil)
	_ providers.TransportPlatformIdentifier = (*Transport)(nil)
)

func New(tc *providers.TransportConfig) (*Transport, error) {
	if tc.Options == nil || tc.Options["path"] == "" {
		return nil, errors.New("path is required")
	}

	path := tc.Options["path"]
	fileList, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, err
	}

	parsed, err := ParseHclDirectory(path, fileList)
	if err != nil {
		return nil, errors.Wrap(err, "could not parse hcl files")
	}

	tfVars, err := ParseTfVars(path, fileList)
	if err != nil {
		return nil, errors.Wrap(err, "could not parse tfvars files")
	}

	modulesManifest, err := ParseTerraformModuleManifest(path)

	absPath, _ := filepath.Abs(path)
	h := sha256.New()
	h.Write([]byte(absPath))
	hash := hex.EncodeToString(h.Sum(nil))

	platformID := "//platformid.api.mondoo.app/runtime/terraform/hash/" + hash

	return &Transport{
		platformID:      platformID,
		path:            path,
		parsed:          parsed,
		tfVars:          tfVars,
		modulesManifest: modulesManifest,
	}, nil
}

// References:
// - https://www.terraform.io/docs/language/syntax/configuration.html
// - https://github.com/hashicorp/hcl/blob/main/hclsyntax/spec.md
type Transport struct {
	platformID      string
	path            string
	parsed          *hclparse.Parser
	tfVars          map[string]*hcl.Attribute
	modulesManifest *ModuleManifest
}

func (t *Transport) RunCommand(command string) (*providers.Command, error) {
	return nil, errors.New("terraform does not implement RunCommand")
}

func (t *Transport) FileInfo(path string) (providers.FileInfoDetails, error) {
	return providers.FileInfoDetails{}, errors.New("terraform does not implement FileInfo")
}

func (t *Transport) FS() afero.Fs {
	return &fsutil.NoFs{}
}

func (t *Transport) Close() {}

func (t *Transport) Capabilities() providers.Capabilities {
	return providers.Capabilities{}
}

func (t *Transport) Kind() providers.Kind {
	return providers.Kind_KIND_CODE
}

func (t *Transport) PlatformIdDetectors() []providers.PlatformIdDetector {
	return []providers.PlatformIdDetector{
		providers.TransportPlatformIdentifierDetector,
	}
}

func (t *Transport) Runtime() string {
	return ""
}

func (t *Transport) Parser() *hclparse.Parser {
	return t.parsed
}

func (t *Transport) TfVars() map[string]*hcl.Attribute {
	return t.tfVars
}

func (t *Transport) ModulesManifest() *ModuleManifest {
	return t.modulesManifest
}

func (t *Transport) Identifier() (string, error) {
	return t.platformID, nil
}
