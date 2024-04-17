// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v11/providers/os/connection/shared"
	"go.mondoo.com/cnquery/v11/providers/os/resources/python"
	"go.mondoo.com/cnquery/v11/types"
)

type pythonDirectory struct {
	path   string
	addLib bool
}

var pythonDirectories = []pythonDirectory{
	{
		path: "/usr/local/lib/python*",
	},
	{
		path: "/usr/local/lib64/python*",
	},
	{
		path: "/usr/lib/python*",
	},
	{
		path: "/usr/lib64/python*",
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

func initPython(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if x, ok := args["path"]; ok {
		_, ok := x.Value.(string)
		if !ok {
			return nil, nil, errors.New("Wrong type for 'path' in python initialization, it must be a string")
		}
	} else {
		// empty path means search through default locations
		args["path"] = llx.StringData("")
	}

	return args, nil, nil
}

func (r *mqlPython) id() (string, error) {
	return "python", nil
}

func (r *mqlPython) packages() ([]interface{}, error) {
	allPyPkgDetails, err := r.getAllPackages()
	if err != nil {
		return nil, err
	}

	// this is the "global" map so that the recursive function calls can keep track of
	// resources already created
	pythonPackageResourceMap := map[string]plugin.Resource{}

	resp := []interface{}{}

	for _, pyPkgDetails := range allPyPkgDetails {
		res, err := pythonPackageDetailsWithDependenciesToResource(r.MqlRuntime, pyPkgDetails, allPyPkgDetails, pythonPackageResourceMap)
		if err != nil {
			log.Error().Err(err).Msg("error while creating resource(s) for python package")
			// we will keep trying to make resources even if a single one failed
			continue
		}
		resp = append(resp, res)
	}

	return resp, nil
}

func (r *mqlPython) toplevel() ([]interface{}, error) {
	allPyPkgDetails, err := r.getAllPackages()
	if err != nil {
		return nil, err
	}

	// this is the "global" map so that the recursive function calls can keep track of
	// resources already created
	pythonPackageResourceMap := map[string]plugin.Resource{}

	resp := []interface{}{}

	for _, pyPkgDetails := range allPyPkgDetails {
		if !pyPkgDetails.IsLeaf {
			continue
		}

		res, err := pythonPackageDetailsWithDependenciesToResource(r.MqlRuntime, pyPkgDetails, allPyPkgDetails, pythonPackageResourceMap)
		if err != nil {
			log.Error().Err(err).Msg("error while creating resource(s) for python package")
			// we will keep trying to make resources even if a single one failed
			continue
		}
		resp = append(resp, res)
	}

	return resp, nil
}

func (r *mqlPython) getAllPackages() ([]python.PackageDetails, error) {
	allResults := []python.PackageDetails{}

	conn, ok := r.MqlRuntime.Connection.(shared.Connection)
	if !ok {
		return nil, fmt.Errorf("provider is not an operating system provider")
	}
	afs := &afero.Afero{Fs: conn.FileSystem()}

	if r.Path.Error != nil {
		return nil, r.Path.Error
	}
	pyPath := r.Path.Data
	if pyPath != "" {
		// only search the specific path provided (if it was provided)
		allResults = gatherPackages(afs, pyPath)
	} else {
		// search through default locations
		searchFunctions := []func(*afero.Afero) ([]python.PackageDetails, error){
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

	return allResults, nil
}

func pythonPackageDetailsWithDependenciesToResource(
	runtime *plugin.Runtime,
	newPyPkgDetails python.PackageDetails,
	pythonPgkDetailsList []python.PackageDetails,
	pythonPackageResourceMap map[string]plugin.Resource,
) (interface{}, error) {
	res := pythonPackageResourceMap[newPyPkgDetails.Name]
	if res != nil {
		// already created the pythonPackage resource
		return res, nil
	}

	dependencies := []interface{}{}
	for _, dep := range newPyPkgDetails.Dependencies {
		found := false
		var depPyPkgDetails python.PackageDetails
		for i, pyPkgDetails := range pythonPgkDetailsList {
			if pyPkgDetails.Name == dep {
				depPyPkgDetails = pythonPgkDetailsList[i]
				found = true
				break
			}
		}
		if !found {
			// can't create a resource for something we didn't discover ¯\_(ツ)_/¯
			continue
		}
		res, err := pythonPackageDetailsWithDependenciesToResource(runtime, depPyPkgDetails, pythonPgkDetailsList, pythonPackageResourceMap)
		if err != nil {
			log.Warn().Err(err).Msg("failed to create python package resource")
			continue
		}
		dependencies = append(dependencies, res)
	}

	// finally create the resource
	r, err := newMqlPythonPackage(runtime, newPyPkgDetails, dependencies)
	if err != nil {
		log.Error().Err(err).Str("resource", newPyPkgDetails.File).Msg("error while creating MQL resource")
		return nil, err
	}

	// name is not guaranteed to be unique, so we use the file path as the key
	pythonPackageResourceMap[newPyPkgDetails.File] = r

	return r, nil
}

func gatherPackages(afs *afero.Afero, pythonPackagePath string) (allResults []python.PackageDetails) {
	fileList, err := afs.ReadDir(pythonPackagePath)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Warn().Err(err).Str("dir", pythonPackagePath).Msg("unable to open directory")
		}
		return
	}
	for _, dEntry := range fileList {
		// only process files/directories that might actually contain
		// the data we're looking for
		if !strings.HasSuffix(dEntry.Name(), ".dist-info") &&
			!strings.HasSuffix(dEntry.Name(), ".egg-info") {
			continue
		}

		// There is the possibility that the .egg-info entry is a file
		// (not a directory) that we can directly process.
		packagePayload := dEntry.Name()

		// requestedPackage just marks whether we found the empty REQUESTED file
		// to indicate a child/leaf package
		requestedPackage := false

		requiresTxtPath := ""

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
				}
				if packageFile.Name() == "REQUESTED" {
					requestedPackage = true
				}
				if packageFile.Name() == "requires.txt" {
					requiresTxtPath = filepath.Join(dEntry.Name(), packageFile.Name())
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
		ppd.IsLeaf = requestedPackage

		// if the MIME data didn't include dependency information, but there was a requires.txt file available,
		// then use that for dependency info (as pip appears to do)
		if len(ppd.Dependencies) == 0 && requiresTxtPath != "" {
			requiresTxtDeps, err := parseRequiresTxtDependencies(afs, filepath.Join(pythonPackagePath, requiresTxtPath))
			if err != nil {
				log.Warn().Err(err).Str("dir", pythonPackageFilepath).Msg("failed to parse requires.txt")
			} else {
				ppd.Dependencies = requiresTxtDeps
			}
		}

		allResults = append(allResults, *ppd)
	}

	return
}

func searchForPythonPackages(afs *afero.Afero, path string) []python.PackageDetails {
	allResults := []python.PackageDetails{}

	packageDirs := []string{"site-packages", "dist-packages"}
	for _, packageDir := range packageDirs {
		pythonPackageDir := filepath.Join(path, packageDir)
		allResults = append(allResults, gatherPackages(afs, pythonPackageDir)...)
	}

	return allResults
}

func parseRequiresTxtDependencies(afs *afero.Afero, requiresTxtPath string) ([]string, error) {
	f, err := afs.Open(requiresTxtPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return python.ParseRequiresTxtDependencies(f)
}

func parseMIME(afs *afero.Afero, pythonMIMEFilepath string) (*python.PackageDetails, error) {
	f, err := afs.Open(pythonMIMEFilepath)
	if err != nil {
		log.Warn().Err(err).Msg("error opening python metadata file")
		return nil, err
	}
	defer f.Close()

	return python.ParseMIME(f, pythonMIMEFilepath)
}

func genericSearch(afs *afero.Afero) ([]python.PackageDetails, error) {
	allResults := []python.PackageDetails{}

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
func darwinSearch(afs *afero.Afero) ([]python.PackageDetails, error) {
	allResults := []python.PackageDetails{}

	// TODO: this does not work properly, we need to use the connection here to determine if we are running on a
	// local connection
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

func newMqlPythonPackage(runtime *plugin.Runtime, ppd python.PackageDetails, dependencies []interface{}) (plugin.Resource, error) {
	f, err := CreateResource(runtime, "file", map[string]*llx.RawData{
		"path": llx.StringData(ppd.File),
	})
	if err != nil {
		log.Error().Err(err).Msg("error while creating file resource for python package resource")
		return nil, err
	}

	cpes := []interface{}{}
	for i := range ppd.Cpes {
		cpe, err := runtime.CreateSharedResource("cpe", map[string]*llx.RawData{
			"uri": llx.StringData(ppd.Cpes[i]),
		})
		if err != nil {
			return nil, err
		}
		cpes = append(cpes, cpe)
	}

	r, err := CreateResource(runtime, "python.package", map[string]*llx.RawData{
		"id":           llx.StringData(ppd.File),
		"name":         llx.StringData(ppd.Name),
		"version":      llx.StringData(ppd.Version),
		"author":       llx.StringData(ppd.Author),
		"authorEmail":  llx.StringData(ppd.AuthorEmail),
		"summary":      llx.StringData(ppd.Summary),
		"license":      llx.StringData(ppd.License),
		"file":         llx.ResourceData(f, f.MqlName()),
		"dependencies": llx.ArrayData(dependencies, types.Any),
		"purl":         llx.StringData(ppd.Purl),
		"cpes":         llx.ArrayData(cpes, types.Resource("cpe")),
	})
	if err != nil {
		log.Error().AnErr("err", err).Msg("error while creating MQL resource")
		return nil, err
	}
	return r, nil
}

func (r *mqlPythonPackage) id() (string, error) {
	return r.Id.Data, nil
}

func initPythonPackage(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 1 {
		return args, nil, nil
	}
	if x, ok := args["path"]; ok {
		path, ok := x.Value.(string)
		if !ok {
			return nil, nil, errors.New("Wrong type for 'path' in python.package initialization, it must be a string")
		}

		file, err := CreateResource(runtime, "file", map[string]*llx.RawData{
			"path": llx.StringData(path),
		})
		if err != nil {
			return nil, nil, err
		}
		args["id"] = llx.StringData(path)
		args["file"] = llx.ResourceData(file, "file")

		delete(args, "path")
	}
	return args, nil, nil
}

func (r *mqlPythonPackage) name() (string, error) {
	err := r.populateData()
	if err != nil {
		return "", err
	}
	return r.Name.Data, nil
}

func (r *mqlPythonPackage) version() (string, error) {
	err := r.populateData()
	if err != nil {
		return "", err
	}
	return r.Version.Data, nil
}

func (r *mqlPythonPackage) license() (string, error) {
	err := r.populateData()
	if err != nil {
		return "", err
	}
	return r.License.Data, nil
}

func (r *mqlPythonPackage) author() (string, error) {
	err := r.populateData()
	if err != nil {
		return "", err
	}
	return r.Author.Data, nil
}

func (r *mqlPythonPackage) authorEmail() (string, error) {
	err := r.populateData()
	if err != nil {
		return "", err
	}
	return r.AuthorEmail.Data, nil
}

func (r *mqlPythonPackage) summary() (string, error) {
	err := r.populateData()
	if err != nil {
		return "", err
	}
	return r.Summary.Data, nil
}

func (r *mqlPythonPackage) purl() (string, error) {
	err := r.populateData()
	if err != nil {
		return "", err
	}
	return r.Purl.Data, nil
}

func (r *mqlPythonPackage) cpes() ([]interface{}, error) {
	err := r.populateData()
	if err != nil {
		return nil, err
	}
	return r.Cpes.Data, nil
}

func (r *mqlPythonPackage) dependencies() ([]interface{}, error) {
	err := r.populateData()
	if err != nil {
		return nil, err
	}
	return r.Dependencies.Data, nil
}

func (r *mqlPythonPackage) populateData() error {
	file := r.GetFile()
	if file.Error != nil {
		return file.Error
	}

	if file.Data == nil || file.Data.Path.Data == "" {
		return fmt.Errorf("file path is empty")
	}

	pkg, err := python.ParseMIME(strings.NewReader(file.Data.Content.Data), file.Data.Path.Data)
	if err != nil {
		return fmt.Errorf("error parsing python package data: %s", err)
	}

	r.Name = plugin.TValue[string]{Data: pkg.Name, State: plugin.StateIsSet}
	r.Version = plugin.TValue[string]{Data: pkg.Version, State: plugin.StateIsSet}
	r.Author = plugin.TValue[string]{Data: pkg.Author, State: plugin.StateIsSet}
	r.AuthorEmail = plugin.TValue[string]{Data: pkg.AuthorEmail, State: plugin.StateIsSet}
	r.Summary = plugin.TValue[string]{Data: pkg.Summary, State: plugin.StateIsSet}
	r.License = plugin.TValue[string]{Data: pkg.License, State: plugin.StateIsSet}
	r.Dependencies = plugin.TValue[[]interface{}]{Data: convert.SliceAnyToInterface(pkg.Dependencies), State: plugin.StateIsSet}

	cpes := []interface{}{}
	for i := range pkg.Cpes {
		cpe, err := r.MqlRuntime.CreateSharedResource("cpe", map[string]*llx.RawData{
			"uri": llx.StringData(pkg.Cpes[i]),
		})
		if err != nil {
			return err
		}
		cpes = append(cpes, cpe)
	}

	r.Cpes = plugin.TValue[[]interface{}]{Data: cpes, State: plugin.StateIsSet}
	r.Purl = plugin.TValue[string]{Data: pkg.Purl, State: plugin.StateIsSet}
	return nil
}
