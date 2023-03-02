package python

import (
	"bufio"
	"errors"
	"fmt"
	"net/textproto"
	"os"
	"path/filepath"
	"runtime"

	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"

	motoros "go.mondoo.com/cnquery/motor/providers/os"
	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/resources/packs/core"
	"go.mondoo.com/cnquery/resources/packs/python/info"
)

var Registry = info.Registry

type pythonDirectory struct {
	path   string
	addLib bool
}

var pythonDirectories = []pythonDirectory{
	{
		path: "/usr/local/lib/python*",
	},
	{
		path: "/usr/lib/python*",
	},
	{
		path: "/opt/homebrew/lib/python*",
	},
	{
		// surprisingly, this is handled in a case-sensitive way in go (the filepath.Match() glob/pattern matching)
		path: "C:/Python*",
		// true because in Windows the 'site-packages' dir lives in a path like:
		// C:\Python3.11\Lib\site-packages
		addLib: true,
	},
}

var pythonDirectoriesDarwin = []string{
	"/System/Library/Frameworks/Python.framework/Versions",
	"/Library/Developer/CommandLineTools/Library/Frameworks/Python3.framework/Versions",
}

func init() {
	Init(Registry)
}

func (k *mqlPython) init(args *resources.Args) (*resources.Args, Python, error) {
	if x, ok := (*args)["path"]; ok {
		_, ok := x.(string)
		if !ok {
			return nil, nil, errors.New("Wrong type for 'path' in python initialization, it must be a string")
		}
	} else {
		// empty path means search through default locations
		(*args)["path"] = ""
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

	pyPath, err := k.Path()
	if err != nil {
		return nil, err
	}
	if pyPath != "" {
		// only search the specific path provided (if it was provided)
		allResults = gatherPackages(afs, pyPath)
	} else {
		// search through default locations
		searchFunctions := []func(*afero.Afero) ([]pythonPackageDetails, error){
			genericSearch,
			darwinSearch,
		}

		for _, sFunc := range searchFunctions {
			results, err := sFunc(afs)
			if err != nil {
				log.Error().Err(err).Msg("error while searching for python packages")
				return nil, err
			}
			allResults = append(allResults, results...)
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
	f, err := motorRuntime.CreateResource("file", "path", ppd.file)
	if err != nil {
		log.Error().Err(err).Msg("error while creating file resource for python package resource")
		return nil, err
	}

	r, err := motorRuntime.CreateResource("python.package",
		"id", ppd.file,
		"name", ppd.name,
		"version", ppd.version,
		"author", ppd.author,
		"summary", ppd.summary,
		"license", ppd.license,
		"file", f.(core.File),
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
	file    string
	license string
	author  string
	summary string
	version string
}

func gatherPackages(afs *afero.Afero, pythonPackagePath string) (allResults []pythonPackageDetails) {
	fileList, err := afs.ReadDir(pythonPackagePath)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Warn().Err(err).Str("dir", pythonPackagePath).Msg("unable to open directory")
		}
		return
	}
	for _, dEntry := range fileList {
		// handle the case where the directory entry is
		// a .egg-type file with metadata directly available
		packagePayload := dEntry.Name()

		// in the event the directory entry is itself another directory
		// go into each directory looking for our parsable payload
		// (ie. METADATA and PKG-INFO files)
		if dEntry.IsDir() {
			pythonPackageDir := filepath.Join(pythonPackagePath, packagePayload)
			packageDirFiles, err := afs.ReadDir(pythonPackageDir)
			if err != nil {
				log.Warn().Err(err).Str("dir", pythonPackageDir).Msg("error while walking through files in directory")
				return
			}

			foundMeta := false
			for _, packageFile := range packageDirFiles {
				if packageFile.Name() == "METADATA" || packageFile.Name() == "PKG-INFO" {
					// use the METADATA / PKG-INFO file as our source of python package info
					packagePayload = filepath.Join(dEntry.Name(), packageFile.Name())
					foundMeta = true
					break
				}
			}
			if !foundMeta {
				// nothing to process (happens when we've traversed a directory
				// containing the actual python source files)
				continue
			}

		}

		pythonPackageFilepath := filepath.Join(pythonPackagePath, packagePayload)
		ppd, err := parseMIME(afs, pythonPackageFilepath)
		if err != nil {
			continue
		}

		allResults = append(allResults, *ppd)
	}

	return
}

func searchForPythonPackages(afs *afero.Afero, path string) []pythonPackageDetails {
	allResults := []pythonPackageDetails{}

	packageDirs := []string{"site-packages", "dist-packages"}
	for _, packageDir := range packageDirs {
		pythonPackageDir := filepath.Join(path, packageDir)
		allResults = append(allResults, gatherPackages(afs, pythonPackageDir)...)
	}

	return allResults
}

func parseMIME(afs *afero.Afero, pythonMIMEFilepath string) (*pythonPackageDetails, error) {
	f, err := afs.Open(pythonMIMEFilepath)
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
		file:    pythonMIMEFilepath,
	}, nil
}

func genericSearch(afs *afero.Afero) ([]pythonPackageDetails, error) {
	allResults := []pythonPackageDetails{}

	// Look through each potential location for the existence of a matching python* directory
	for _, pyDir := range pythonDirectories {
		parentDir := filepath.Dir(pyDir.path)

		fileList, err := afs.ReadDir(parentDir)
		if err != nil {
			if !os.IsNotExist(err) {
				log.Warn().Err(err).Str("dir", parentDir).Msg("unable to read directory")
			}
			continue
		}

		for _, dEntry := range fileList {
			base := filepath.Base(pyDir.path)
			matched, err := filepath.Match(base, dEntry.Name())
			if err != nil {
				return nil, err
			}
			if matched {
				matchedPath := filepath.Join(parentDir, dEntry.Name())
				log.Debug().Str("filepath", matchedPath).Msg("found matching python path")

				if pyDir.addLib {
					matchedPath = filepath.Join(matchedPath, "lib")
				}

				results := searchForPythonPackages(afs, matchedPath)
				allResults = append(allResults, results...)
			}
		}
	}
	return allResults, nil
}

// darwinSearch has custom handling for the specific way that darwin
// can structure the paths holding python packages
func darwinSearch(afs *afero.Afero) ([]pythonPackageDetails, error) {
	allResults := []pythonPackageDetails{}

	if runtime.GOOS != "darwin" {
		return allResults, nil
	}

	for _, pyPath := range pythonDirectoriesDarwin {

		fileList, err := afs.ReadDir(pyPath)
		if err != nil {
			if !os.IsNotExist(err) {
				log.Warn().Err(err).Str("dir", pyPath).Msg("unable to read directory")
			}
			continue
		}

		for _, aFile := range fileList {
			// want to not double-search the case where the files look like:
			// 3.9
			// Current -> 3.9
			// FIXME: doesn't work with AFS (we actually want an Lstat() call here)
			// fStat, err := afs.Stat(filepath.Join(pyPath, aFile.Name()))
			// if err != nil {
			// 	log.Warn().Err(err).Str("file", aFile.Name()).Msg("error trying to stat file")
			// 	continue
			// }
			// if fStat.Mode()&os.ModeSymlink != 0 {
			// 	// ignore symlinks (basically the Current -> 3.9 symlink) so that
			// 	// we don't process the same set of packages twice
			// 	continue
			// }
			if aFile.Name() == "Current" {
				continue
			}

			pythonPackagePath := filepath.Join(pyPath, aFile.Name(), "lib")
			fileList, err := afs.ReadDir(pythonPackagePath)
			if err != nil {
				log.Warn().Err(err).Str("path", pythonPackagePath).Msg("failed to read directory")
				continue
			}
			for _, oneFile := range fileList {
				// if we run into a directory name that starts with "python"
				// then we have a candidate to search through
				match, err := filepath.Match("python*", oneFile.Name())
				if err != nil {
					log.Error().Err(err).Msg("unexpected error while checking for python file pattern")
					continue
				}
				if match {
					matchedPath := filepath.Join(pythonPackagePath, oneFile.Name())
					log.Debug().Str("filepath", matchedPath).Msg("found matching python path")
					results := searchForPythonPackages(afs, matchedPath)
					allResults = append(allResults, results...)
				}
			}
		}
	}
	return allResults, nil
}
