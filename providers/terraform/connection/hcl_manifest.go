// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/providers-sdk/v1/plugin"
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

// dotTerraformDir is the vendored module cache Terraform writes next to a
// configuration. Its contents are copies of upstream modules, not the code
// under review.
const dotTerraformDir = ".terraform"

// hasPathSegment reports whether rel — a slash-separated path relative to the
// scan root — contains segment as a whole path element.
//
// Matching a whole segment rather than a substring matters: a substring test
// for ".terraform" also swallows a scan root named `.terraform-configs/`, a
// repository directory called `.terraform-modules/`, and a configuration file
// named `main.terraform.tf`. Those are the user's own code, and skipping them
// reported an empty configuration with no error — every policy over it then
// passed vacuously.
func hasPathSegment(rel, segment string) bool {
	for _, part := range strings.Split(rel, "/") {
		if part == segment {
			return true
		}
	}
	return false
}

// MODULE_EXAMPLES matches the `examples/` trees shipped inside modules that
// Terraform vendored into `.terraform/modules/`. Those are upstream sample
// configurations, not deployed code.
//
// The pattern is anchored on the `.terraform/` cache deliberately. Without that
// anchor it also matched `modules/<name>/examples/<case>/`, which is the
// canonical layout of a first-party Terraform module repository — so a
// repository's own examples, including any misconfiguration in them, were
// silently dropped from the scan.
var MODULE_EXAMPLES = regexp.MustCompile(`(^|/)\.terraform/modules/.+/examples/.+`)

func NewHclConnection(id uint32, asset *inventory.Asset) (*Connection, error) {
	if len(asset.Connections) == 0 {
		return nil, errors.New("no connection options for asset")
	}
	cc := asset.Connections[0]
	path := cc.Options["path"]
	return newHclConnection(id, path, asset)
}

// tfvarsCandidate is a variable-definitions file found during the walk,
// together with the precedence rank it is applied at.
type tfvarsCandidate struct {
	path string
	rank int
}

// Terraform's variable-definition precedence, lowest first. Files later in this
// order override earlier ones.
//
//	rankExplicit  — `prod.tfvars` and friends. Terraform only loads these when
//	                named with `-var-file`, never automatically. We keep loading
//	                them so a scan of a directory that only carries such files
//	                still sees values, but they must not outrank the files
//	                Terraform does auto-load.
//	rankDefault   — `terraform.tfvars`
//	rankDefaultJSON — `terraform.tfvars.json`
//	rankAuto      — `*.auto.tfvars` / `*.auto.tfvars.json`, applied in lexical order
const (
	rankExplicit = iota
	rankDefault
	rankDefaultJSON
	rankAuto
)

func tfvarsRank(name string) int {
	switch {
	case name == "terraform.tfvars":
		return rankDefault
	case name == "terraform.tfvars.json":
		return rankDefaultJSON
	case strings.HasSuffix(name, ".auto.tfvars"), strings.HasSuffix(name, ".auto.tfvars.json"):
		return rankAuto
	default:
		return rankExplicit
	}
}

func isTfVarsFile(name string) bool {
	return strings.HasSuffix(name, ".tfvars") || strings.HasSuffix(name, ".tfvars.json")
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
	var manifestRecords []Record
	seenModuleKeys := map[string]struct{}{}

	assetType = configurationfiles
	// FIXME: cannot handle relative paths
	stat, err := os.Stat(path)
	if err != nil {
		// os.Stat fails with more than ENOENT: a path whose ancestor is a
		// regular file yields ENOTDIR, an unreadable parent yields EACCES, and
		// a symlink cycle yields ELOOP. All of them return a nil FileInfo, so
		// handling only os.IsNotExist dereferenced nil and panicked the
		// provider on a mistyped scan path.
		return nil, errors.Wrapf(err, "path %q is not a valid file or directory", path)
	}

	if stat.IsDir() {
		var tfvarsFiles []tfvarsCandidate

		walkErr := filepath.WalkDir(path, func(entryPath string, d fs.DirEntry, err error) error {
			if err != nil {
				// A subdirectory we cannot read must not abort discovery of
				// everything else in the tree.
				log.Warn().Err(err).Str("path", entryPath).Msg("skipping unreadable path during terraform discovery")
				if d != nil && d.IsDir() {
					return fs.SkipDir
				}
				return nil
			}

			rel, relErr := filepath.Rel(path, entryPath)
			if relErr != nil {
				rel = entryPath
			}
			rel = filepath.ToSlash(rel)

			// Skip the vendored module cache. `modules.json` inside it is read
			// first, below, because it is the manifest describing what was
			// vendored.
			if hasPathSegment(rel, dotTerraformDir) {
				if d.IsDir() {
					// The manifest lives at .terraform/modules/modules.json, so
					// the cache still has to be walked unless the user opted
					// out; only its *.tf files are skipped.
					if !includeDotTerraform {
						return fs.SkipDir
					}
					return nil
				}

				if includeDotTerraform && strings.HasSuffix(rel, ".terraform/modules/modules.json") {
					manifest, mErr := ParseTerraformModuleManifest(entryPath)
					switch {
					case errors.Is(mErr, os.ErrNotExist):
						log.Debug().Str("path", entryPath).Msg("no terraform module manifest found")
					case mErr != nil:
						log.Warn().Err(mErr).Str("path", entryPath).Msg("could not parse terraform module manifest")
					default:
						// A monorepo has one module cache per stack. Merging
						// their records keeps every stack's modules visible;
						// overwriting reported only the last one walked.
						for _, record := range manifest.Records {
							if _, seen := seenModuleKeys[record.Key]; seen {
								continue
							}
							seenModuleKeys[record.Key] = struct{}{}
							manifestRecords = append(manifestRecords, record)
						}
					}
				}

				// Never parse configuration out of the vendored cache.
				return nil
			}

			// Skip example configurations vendored inside cached modules.
			if MODULE_EXAMPLES.MatchString(rel) {
				log.Debug().Str("path", entryPath).Msg("ignoring terraform module example")
				return nil
			}

			if d.IsDir() {
				return nil
			}

			if isTfVarsFile(entryPath) {
				// Collected rather than applied here: applying in walk order
				// let `terraform.tfvars` override an `*.auto.tfvars` that
				// Terraform itself ranks higher, silently inverting the
				// effective value of a variable.
				tfvarsFiles = append(tfvarsFiles, tfvarsCandidate{
					path: entryPath,
					rank: tfvarsRank(filepath.Base(entryPath)),
				})
				return nil
			}

			log.Debug().Str("path", entryPath).Msg("parsing hcl file")
			if err := loader.ParseHclFile(entryPath); err != nil {
				// A single unparseable file must not truncate the scan. The
				// walk used to abort here and the abort was then discarded, so
				// every file ordered after the broken one silently vanished
				// from a scan that reported success.
				log.Warn().Err(err).Str("path", entryPath).Msg("could not parse hcl file; skipping")
			}
			return nil
		})
		if walkErr != nil {
			return nil, errors.Wrap(walkErr, "could not walk terraform configuration")
		}

		sort.SliceStable(tfvarsFiles, func(i, j int) bool {
			if tfvarsFiles[i].rank != tfvarsFiles[j].rank {
				return tfvarsFiles[i].rank < tfvarsFiles[j].rank
			}
			return tfvarsFiles[i].path < tfvarsFiles[j].path
		})
		for _, candidate := range tfvarsFiles {
			if err := ReadTfVarsFromFile(candidate.path, tfVars); err != nil {
				log.Warn().Err(err).Str("path", candidate.path).Msg("could not parse tfvars file; skipping")
			}
		}
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

	var modulesManifest *ModuleManifest
	if len(manifestRecords) > 0 {
		modulesManifest = &ModuleManifest{Records: manifestRecords}
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
	path, closer, err := plugin.NewGitClone(asset)
	if err != nil {
		return nil, err
	}
	conn, err := newHclConnection(id, path, asset)
	if err != nil {
		// The clone succeeded but the connection did not; without this the
		// whole checkout stays behind in the temp dir on every failed scan.
		closer()
		return nil, err
	}
	conn.closer = closer
	return conn, nil
}
