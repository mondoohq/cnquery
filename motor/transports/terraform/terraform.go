package terraform

import (
	"io/ioutil"

	"github.com/cockroachdb/errors"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/spf13/afero"
	"go.mondoo.io/mondoo/motor/transports"
	"go.mondoo.io/mondoo/motor/transports/fsutil"
)

func New(tc *transports.TransportConfig) (*Transport, error) {
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

	return &Transport{
		parsed:          parsed,
		tfVars:          tfVars,
		modulesManifest: modulesManifest,
	}, nil
}

// References:
// - https://www.terraform.io/docs/language/syntax/configuration.html
// - https://github.com/hashicorp/hcl/blob/main/hclsyntax/spec.md
type Transport struct {
	parsed          *hclparse.Parser
	tfVars          map[string]*hcl.Attribute
	modulesManifest *ModuleManifest
	opts            map[string]string
}

func (t *Transport) RunCommand(command string) (*transports.Command, error) {
	return nil, errors.New("terraform does not implement RunCommand")
}

func (t *Transport) FileInfo(path string) (transports.FileInfoDetails, error) {
	return transports.FileInfoDetails{}, errors.New("terraform does not implement FileInfo")
}

func (t *Transport) FS() afero.Fs {
	return &fsutil.NoFs{}
}

func (t *Transport) Close() {}

func (t *Transport) Capabilities() transports.Capabilities {
	return transports.Capabilities{}
}

func (t *Transport) Kind() transports.Kind {
	return transports.Kind_KIND_CODE
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

// TODO: we need to fix that
func (t *Transport) Identifier() (string, error) {
	return "terraform", nil
}
