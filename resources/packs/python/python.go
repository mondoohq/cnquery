package python

import (
	"bufio"
	"errors"
	"fmt"
	"net/textproto"
	"os"
	"path/filepath"

	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"

	//"go.mondoo.com/cnquery/resources/packs/core"

	motoros "go.mondoo.com/cnquery/motor/providers/os"
	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/resources/packs/python/info"
)

var Registry = info.Registry

var pythonDirectories = []string{
	"/usr/local/lib/python*",
	"/usr/lib/python*",
	"/opt/homebrew/lib/python*",
}

func init() {
	Init(Registry)
}

func (k *mqlPython) init(args *resources.Args) (*resources.Args, Python, error) {
	if x, ok := (*args)["path"]; ok {
		path, ok := x.(string)
		if !ok {
			return nil, nil, errors.New("Wrong type for 'path' in python initialization, it must be a string")
		}

		f, err := k.MotorRuntime.CreateResource("file", "path", path)
		if err != nil {
			return nil, nil, err
		}
		(*args)["file"] = f
	}
	return args, nil, nil
}

func (k *mqlPython) id() (string, error) {
	return "python", nil
}

func (k *mqlPython) GetPackages() ([]interface{}, error) {
	allResults := []pythonPackageDetails{}

	provider, ok := k.MotorRuntime.Motor.Provider.(motoros.OperatingSystemProvider)
	if !ok {
		return nil, fmt.Errorf("provider is not an operating system provider")
	}

	afs := &afero.Afero{Fs: provider.FS()}

	// Look through each potential location for the existence of a matching python* directory
	for _, pyPath := range pythonDirectories {
		parentDir := filepath.Dir(pyPath)

		fileList, err := afs.ReadDir(parentDir)
		if err != nil {
			if !os.IsNotExist(err) {
				log.Warn().Err(err).Str("dir", parentDir).Msg("unable to read directory")
			}
			continue
		}
		for _, dEntry := range fileList {
			base := filepath.Base(pyPath)
			matched, err := filepath.Match(base, dEntry.Name())
			if err != nil {
				return nil, err
			}
			if matched {
				matchedPath := filepath.Join(parentDir, dEntry.Name())
				log.Debug().Str("filepath", matchedPath).Msg("found matching python path")
				results, err := k.gatherFoundPackages(afs, matchedPath)
				if err != nil {
					return nil, err
				}
				allResults = append(allResults, results...)
			}
		}
	}

	resp := []interface{}{}

	for _, result := range allResults {
		r, err := pythonPackageDetailsToResource(k.MotorRuntime, result)
		if err != nil {
			continue
		}
		resp = append(resp, r)
	}

	return resp, nil
}

func pythonPackageDetailsToResource(motorRuntime *resources.Runtime, ppd pythonPackageDetails) (resources.ResourceType, error) {
	r, err := motorRuntime.CreateResource("python.package",
		"id", ppd.path,
		"name", ppd.name,
		"version", ppd.version,
		"author", ppd.author,
		"summary", ppd.summary,
		"license", ppd.license,
		"path", ppd.path,
	)
	if err != nil {
		log.Error().AnErr("err", err).Msg("error while creating MQL resource")
		return nil, err
	}
	return r, nil
}

func (k *mqlPython) GetChildren() ([]interface{}, error) {
	return nil, nil
}

type pythonPackageDetails struct {
	name    string
	path    string
	license string
	author  string
	summary string
	version string
}

func (k *mqlPython) gatherFoundPackages(afs *afero.Afero, path string) ([]pythonPackageDetails, error) {
	allResults := []pythonPackageDetails{}

	packageDirs := []string{"site-packages", "dist-packages"}
	for _, packageDir := range packageDirs {
		parentDir := filepath.Join(path, packageDir)
		fileList, err := afs.ReadDir(parentDir)
		if err != nil {
			if !os.IsNotExist(err) {
				log.Warn().Err(err).Str("dir", parentDir).Msg("unable to open directory")
			}
			continue
		}
		for _, dEntry := range fileList {
			packagePayload := dEntry.Name()

			// go into each directory looking for our parsable payload
			if dEntry.IsDir() {
				pythonPackageDir := filepath.Join(parentDir, packagePayload)
				packageDirFiles, err := afs.ReadDir(pythonPackageDir)
				if err != nil {
					log.Warn().Err(err).Str("dir", pythonPackageDir).Msg("error while walking through files in directory")
					continue
				}

				foundMeta := false
				for _, packageFile := range packageDirFiles {
					if packageFile.Name() == "METADATA" || packageFile.Name() == "PKG-INFO" {
						packagePayload = filepath.Join(dEntry.Name(), packageFile.Name())
						foundMeta = true
						break
					}
				}
				if !foundMeta {
					// nothing to process...move along
					continue
				}

			}

			pythonPackageFilepath := filepath.Join(parentDir, packagePayload)
			ppd, err := parseMIME(pythonPackageFilepath)
			if err != nil {
				continue
			}

			allResults = append(allResults, *ppd)
		}
	}

	return allResults, nil
}

func parseMIME(pythonMIMEFilepath string) (*pythonPackageDetails, error) {
	f, err := os.Open(pythonMIMEFilepath)
	if err != nil {
		log.Warn().Err(err).Msg("error opening python metadata file")
		return nil, err
	}
	defer f.Close()

	textReader := textproto.NewReader(bufio.NewReader(f))
	mimeData, err := textReader.ReadMIMEHeader()
	if err != nil {
		return nil, fmt.Errorf("error reading MIME data: %s", err)
	}

	// TODO: deal with dependencies
	// deps := mimeData.Values("Requires-Dist")

	return &pythonPackageDetails{
		name:    mimeData.Get("Name"),
		summary: mimeData.Get("Summary"),
		author:  mimeData.Get("Author"),
		license: mimeData.Get("License"),
		version: mimeData.Get("Version"),
		path:    pythonMIMEFilepath,
	}, nil
}
