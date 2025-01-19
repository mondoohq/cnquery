// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v11/providers/os/connection/shared"
	"go.mondoo.com/cnquery/v11/providers/os/resources/languages/python"
	"go.mondoo.com/cnquery/v11/providers/os/resources/languages/python/requirements"
	"go.mondoo.com/cnquery/v11/providers/os/resources/languages/python/wheelegg"
	"go.mondoo.com/cnquery/v11/types"
)

var (
	pythonDirectories = []string{
		// Linux
		"/usr/local/lib/python*",
		"/usr/local/lib64/python*",
		"/usr/lib/python*",
		"/usr/lib64/python*",
		// Windows: surprisingly, this is handled in a case-sensitive way in go (the filepath.Match() glob/pattern matching)
		"C:\\Python*\\Lib",
		// macOS
		"/opt/homebrew/lib/python*",
		"/System/Library/Frameworks/Python.framework/Versions/*/lib/python*",
		// we use 3.x to exclude the macOS Current symlink
		"/Library/Developer/CommandLineTools/Library/Frameworks/Python3.framework/Versions/3.*/lib/python*",
	}
)

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
	fs := conn.FileSystem()
	afs := &afero.Afero{Fs: conn.FileSystem()}

	if r.Path.Error != nil {
		return nil, r.Path.Error
	}
	pyPath := r.Path.Data
	if pyPath != "" {
		// only search the specific path provided (if it was provided)
		allResults = gatherPackages(afs, pyPath)
		return allResults, nil
	} else {
		return collectPythonPackagesInPaths(r.MqlRuntime, fs, pythonDirectories)
	}
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

func parseRequiresTxtDependencies(afs *afero.Afero, requiresTxtPath string) ([]string, error) {
	f, err := afs.Open(requiresTxtPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return requirements.ParseRequiresTxtDependencies(f)
}

func parseMIME(afs *afero.Afero, pythonMIMEFilepath string) (*python.PackageDetails, error) {
	f, err := afs.Open(pythonMIMEFilepath)
	if err != nil {
		log.Warn().Err(err).Msg("error opening python metadata file")
		return nil, err
	}
	defer f.Close()

	return wheelegg.ParseMIME(f, pythonMIMEFilepath)
}

func collectPythonPackagesInPaths(runtime *plugin.Runtime, fs afero.Fs, paths []string) ([]python.PackageDetails, error) {
	allResults := []python.PackageDetails{}
	afs := &afero.Afero{Fs: fs}

	for _, pattern := range paths {
		log.Debug().Str("path", pattern).Msg("searching for npm packages")
		m, err := afero.Glob(fs, pattern)
		if err != nil {
			log.Debug().Err(err).Str("path", pattern).Msg("could not search for npm packages")
			// nothing to do, we just ignore it
		}
		for _, walkPath := range m {
			log.Debug().Str("filepath", walkPath).Msg("found matching python path")
			packageDirs := []string{"site-packages", "dist-packages"}
			for _, packageDir := range packageDirs {
				pythonPackageDir := filepath.Join(walkPath, packageDir)
				allResults = append(allResults, gatherPackages(afs, pythonPackageDir)...)
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

	pkg, err := wheelegg.ParseMIME(strings.NewReader(file.Data.Content.Data), file.Data.Path.Data)
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
