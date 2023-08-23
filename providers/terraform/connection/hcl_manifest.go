// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/providers-sdk/v1/inventory"
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

	// NOTE: right now we are only supporting to load either state, plan or hcl files but not at the same time

	var assetType terraformAssetType
	// hcl files
	loader := NewHCLFileLoader()
	tfVars := make(map[string]*hcl.Attribute)
	var modulesManifest *ModuleManifest

	assetType = configurationfiles
	path := cc.Options["path"]
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
		asset:     asset,
		assetType: assetType,

		parsed:          loader.GetParser(),
		tfVars:          tfVars,
		modulesManifest: modulesManifest,
	}, nil
}
