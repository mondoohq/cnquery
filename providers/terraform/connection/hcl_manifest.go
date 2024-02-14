// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/hashicorp/hcl/v2"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/vault"
)

func ParseTerraformModuleManifest(manifestPath string) (*ModuleManifest, error) {
	_, err := os.Stat(manifestPath)
	if err != nil {
		return nil, err
	}

	f, err := os.Open(manifestPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var manifest ModuleManifest
	if err := json.NewDecoder(f).Decode(&manifest); err != nil {
		return nil, err
	}
	return &manifest, nil
}

// e.g. mondoo-operator/.github/terraform/aws/.terraform/modules/vpc/examples/secondary-cidr-blocks/main.tf/1/1
var MODULE_EXAMPLES = regexp.MustCompile(`^.*/modules/.+/examples/.+`)

func NewHclConnection(id uint32, asset *inventory.Asset) (*Connection, error) {
	cc := asset.Connections[0]
	path := cc.Options["path"]
	return newHclConnection(id, path, asset)
}

func newHclConnection(id uint32, path string, asset *inventory.Asset) (*Connection, error) {
	// NOTE: right now we are only supporting to load either state, plan or hcl files but not at the same time
	if len(asset.Connections) != 1 {
		return nil, errors.New("only one connection is supported")
	}

	confOptions := asset.Connections[0].Options
	includeDotTerraform := true
	if confOptions["ignore-dot-terraform"] == "true" {
		includeDotTerraform = false
	}

	var assetType terraformAssetType
	// hcl files
	loader := NewHCLFileLoader()
	tfVars := make(map[string]*hcl.Attribute)
	var modulesManifest *ModuleManifest

	assetType = configurationfiles
	// FIXME: cannot handle relative paths
	stat, err := os.Stat(path)
	if os.IsNotExist(err) {
		return nil, errors.New("path is not a valid file or directory")
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

			// if user asked to ignore .terraform, we skip all files in .terraform
			if strings.Contains(path, ".terraform") && !includeDotTerraform {
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

	return &Connection{
		Connection: plugin.NewConnection(id, asset),
		asset:      asset,
		assetType:  assetType,

		parsed:          loader.GetParser(),
		tfVars:          tfVars,
		modulesManifest: modulesManifest,
	}, nil
}

func NewHclGitConnection(id uint32, asset *inventory.Asset) (*Connection, error) {
	cc := asset.Connections[0]

	if len(cc.Options) == 0 {
		return nil, errors.New("missing URLs in options for HCL over Git connection")
	}

	user := ""
	token := ""
	for i := range cc.Credentials {
		cred := cc.Credentials[i]
		if cred.Type == vault.CredentialType_password {
			user = cred.User
			token = string(cred.Secret)
			if token == "" && cred.Password != "" {
				token = string(cred.Password)
			}
		}
	}

	gitUrl := ""

	// If a token is provided, it will be used to clone the repo
	// gitlab: git clone https://oauth2:ACCESS_TOKEN@somegitlab.com/vendor/package.git
	// if sshUrl := cc.Options["ssh-url"]; sshUrl != "" { ... not doing ssh url right now
	if httpUrl := cc.Options["http-url"]; httpUrl != "" {
		u, err := url.Parse(httpUrl)
		if err != nil {
			return nil, errors.New("failed to parse url for git repo: " + httpUrl)
		}

		if user != "" && token != "" {
			u.User = url.UserPassword(user, token)
		} else if token != "" {
			u.User = url.User(token)
		}

		gitUrl = u.String()
	}

	if gitUrl == "" {
		return nil, errors.New("missing url for git repo " + asset.Name)
	}

	path, closer, err := gitClone(gitUrl)
	if err != nil {
		return nil, err
	}
	conn, err := newHclConnection(id, path, asset)
	if err != nil {
		return nil, err
	}
	conn.closer = closer
	return conn, nil
}

func gitClone(gitUrl string) (string, func(), error) {
	cloneDir, err := os.MkdirTemp(os.TempDir(), "gitClone")
	if err != nil {
		return "", nil, errors.Wrap(err, "failed to create temporary dir for git processing")
	}

	closer := func() {
		log.Info().Str("path", cloneDir).Msg("cleaning up git clone")
		if err = os.RemoveAll(cloneDir); err != nil {
			log.Error().Err(err).Msg("failed to remove temporary dir for git processing")
		}
	}

	// Note: DO NOT leak credentials into logs!!
	var infoUrl string
	if u, err := url.Parse(gitUrl); err == nil {
		if u.User != nil {
			u.User = url.User("_obfuscated_")
		}
		infoUrl = u.String()
	}

	log.Info().Str("url", infoUrl).Str("path", cloneDir).Msg("git clone")
	repo, err := git.PlainClone(cloneDir, false, &git.CloneOptions{
		URL:               gitUrl,
		Progress:          os.Stderr,
		Depth:             1,
		RecurseSubmodules: git.DefaultSubmoduleRecursionDepth,
	})
	if err != nil {
		closer()
		return "", nil, errors.Wrap(err, "failed to clone git repo "+infoUrl)
	}

	ref, err := repo.Head()
	if err != nil {
		closer()
		return "", nil, errors.Wrap(err, "failed to get head of git repo "+infoUrl)
	}

	log.Info().Str("url", infoUrl).Str("path", cloneDir).Str("head", ref.Hash().String()).Msg("finished git clone")

	return cloneDir, closer, nil
}
