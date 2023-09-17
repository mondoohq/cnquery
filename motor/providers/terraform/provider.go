package terraform

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/go-git/go-git/v5"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/motor/providers"
	"go.mondoo.com/cnquery/motor/vault"
)

var (
	_ providers.Instance           = (*Provider)(nil)
	_ providers.PlatformIdentifier = (*Provider)(nil)
	// e.g. mondoo-operator/.github/terraform/aws/.terraform/modules/vpc/examples/secondary-cidr-blocks/main.tf/1/1
	MODULE_EXAMPLES = regexp.MustCompile(`^.*/modules/.+/examples/.+`)
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

	// Terraform from git, temporary before v9
	var err error
	var closer func()
	path := tc.Options["path"]
	if strings.HasPrefix(path, "git+https://") || strings.HasPrefix(path, "git+ssh://") {
		path, closer, err = processGitForTerraform(path, tc.Credentials)
		if err != nil {
			return nil, err
		}
		tc.Options["asset-type"] = "hcl"
		tc.Options["path"] = path
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
			return nil, errors.New("path '" + path + "'is not a valid file or directory")
		}

		if stat.IsDir() {
			filepath.WalkDir(path, func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					return err
				}

				// skip terraform module examples
				foundExamples := MODULE_EXAMPLES.FindString(path)
				if foundExamples != "" {
					log.Debug().Str("path", path).Msg("ignoring terraform module example")
					return nil
				}

				if !d.IsDir() {
					if strings.HasSuffix(path, ".terraform/modules/modules.json") {
						modulesManifest, err = ParseTerraformModuleManifest(path)
						if errors.Is(err, os.ErrNotExist) {
							log.Debug().Str("path", path).Msg("no terraform module manifest found")
						} else {
							return errors.Wrap(err, fmt.Sprintf("could not parse terraform module manifest %s", path))
						}
					}

					// we do not want to parse hcl files from terraform modules .terraform files
					if strings.Contains(path, ".terraform") {
						return nil
					}

					log.Debug().Str("path", path).Msg("parsing hcl file")
					err = loader.ParseHclFile(path)
					if err != nil {
						return errors.Wrap(err, "could not parse hcl file")
					}

					err = ReadTfVarsFromFile(path, tfVars)
					if err != nil {
						return errors.Wrap(err, "could not parse tfvars file")
					}
				}
				return nil
			})
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

		state:  &state,
		plan:   &plan,
		closer: closer,
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
	closer          func()
}

func (t *Provider) Close() {
	if t.closer != nil {
		t.closer()
	}
}

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

// TODO: Migrate to v9

var reGitHttps = regexp.MustCompile(`git\+https://([^/]+)/(.*)`)

// gitCloneUrl returns a git clone url from a git+https url
// If a token is provided, it will be used to clone the repo
// gitlab: git clone https://oauth2:ACCESS_TOKEN@somegitlab.com/vendor/package.git
func gitCloneUrl(url string, credentials []*vault.Credential) (string, error) {

	user := ""
	token := ""
	for i := range credentials {
		cred := credentials[i]
		if cred.Type == vault.CredentialType_password {
			user = cred.User
			token = string(cred.Secret)
			if token == "" && cred.Password != "" {
				token = string(cred.Password)
			}
		}
	}

	m := reGitHttps.FindStringSubmatch(url)
	if len(m) == 3 {
		if strings.Contains(m[1], ":") {
			return "", errors.New("url cannot contain a port! (" + m[1] + ")")
		}

		if user != "" && token != "" {
			// e.g. used by GitLab
			url = "https://" + user + ":" + token + "@" + m[1] + "/" + m[2]
		} else if token != "" {
			// e.g. used by GitHub
			url = "https://" + token + "@" + m[1] + "/" + m[2]
		} else {
			url = "git@" + m[1] + ":" + m[2]
		}
	}
	// url = strings.ReplaceAll(url, "git+https://gitlab.com/", "git@gitlab.com:")
	url = strings.ReplaceAll(url, "git+ssh://", "")

	if !strings.HasSuffix(url, ".git") {
		url += ".git"
	}
	return url, nil
}

// processGitForTerraform clones a git repo and returns the path to the clone
func processGitForTerraform(url string, credentials []*vault.Credential) (string, func(), error) {
	cloneUrl, err := gitCloneUrl(url, credentials)
	if err != nil {
		return "", nil, errors.Wrap(err, "failed to parse git clone url "+url)
	}

	cloneDir, err := os.MkdirTemp(os.TempDir(), "gitClone")
	if err != nil {
		return "", nil, errors.Wrap(err, "failed to create temporary dir for git processing")
	}

	closer := func() {
		// FIXME: needs to be added in v9
		// log.Info().Str("path", cloneDir).Msg("cleaning up git clone")
		// if err = os.RemoveAll(cloneDir); err != nil {
		// 	log.Error().Err(err).Msg("failed to remove temporary dir for git processing")
		// }
	}

	log.Info().Str("url", url).Str("path", cloneDir).Msg("git clone")
	repo, err := git.PlainClone(cloneDir, false, &git.CloneOptions{
		URL:               cloneUrl,
		Progress:          os.Stderr,
		Depth:             1,
		RecurseSubmodules: git.DefaultSubmoduleRecursionDepth,
	})
	if err != nil {
		closer()
		return "", nil, errors.Wrap(err, "failed to clone git repo "+url)
	}

	ref, err := repo.Head()
	if err != nil {
		closer()
		return "", nil, errors.Wrap(err, "failed to get head of git repo "+url)
	}
	log.Info().Str("url", url).Str("path", cloneDir).Str("head", ref.Hash().String()).Msg("finshed git clone")

	return cloneDir, closer, nil
}
