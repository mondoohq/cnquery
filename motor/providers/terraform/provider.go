package terraform

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/rs/zerolog/log"
	"os"
	"path/filepath"

	"github.com/cockroachdb/errors"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
	"go.mondoo.com/cnquery/motor/providers"
)

var (
	_ providers.Instance           = (*Provider)(nil)
	_ providers.PlatformIdentifier = (*Provider)(nil)
)

type terraformAssetType int32

const (
	configurationfiles terraformAssetType = 0
	planfile           terraformAssetType = 1
	statefile          terraformAssetType = 2
)

func New(tc *providers.Config) (*Provider, error) {
	if tc.Options == nil {
		return nil, errors.New("path is required")
	}

	projectPath := ""
	// NOTE: right now we are only supporting to load either state, plan or hcl files but not at the same time

	var assetType terraformAssetType
	var state State
	var plan Plan
	// hcl files
	loader := NewHCLFileLoader()
	tfVars := make(map[string]*hcl.Attribute)
	var modulesManifest *ModuleManifest

	if tc.Options["asset-type"] == "state" {
		assetType = statefile
		stateFilePath := tc.Options["path"]
		projectPath = stateFilePath
		log.Debug().Str("path", stateFilePath).Msg("load terraform state file")
		data, err := os.ReadFile(stateFilePath)
		if err != nil {
			return nil, err
		}
		err = json.Unmarshal(data, &state)
		if err != nil {
			return nil, err
		}
	} else if tc.Options["asset-type"] == "plan" {
		assetType = planfile
		planfile := tc.Options["path"]
		projectPath = planfile
		log.Debug().Str("path", projectPath).Msg("load terraform plan file")
		data, err := os.ReadFile(planfile)
		if err != nil {
			return nil, err
		}
		err = json.Unmarshal(data, &plan)
		if err != nil {
			return nil, err
		}
	} else if tc.Options["path"] != "" {
		assetType = configurationfiles
		path := tc.Options["path"]
		projectPath = path
		stat, err := os.Stat(path)
		if os.IsNotExist(err) {
			return nil, errors.New("path is not a valid file or directory")
		}

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
	}

	// build project hash to identify the project
	absPath, _ := filepath.Abs(projectPath)
	h := sha256.New()
	h.Write([]byte(absPath))
	hash := hex.EncodeToString(h.Sum(nil))
	platformID := "//platformid.api.mondoo.app/runtime/terraform/hash/" + hash

	return &Provider{
		platformID: platformID,
		assetType:  assetType,

		parsed:          loader.GetParser(),
		tfVars:          tfVars,
		modulesManifest: modulesManifest,

		state: &state,
		plan:  &plan,
	}, nil
}

// References:
// - https://www.terraform.io/docs/language/syntax/configuration.html
// - https://github.com/hashicorp/hcl/blob/main/hclsyntax/spec.md
type Provider struct {
	platformID      string
	assetType       terraformAssetType
	parsed          *hclparse.Parser
	tfVars          map[string]*hcl.Attribute
	modulesManifest *ModuleManifest
	state           *State
	plan            *Plan
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

func (p *Provider) State() (*State, error) {
	return p.state, nil
}

func (p *Provider) Plan() (*Plan, error) {
	return p.plan, nil
}
