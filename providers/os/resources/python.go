// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/package-url/packageurl-go"
	"go.mondoo.com/cnquery/v9/providers/os/resources/cpe"
	"io"
	"net/textproto"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
	"go.mondoo.com/cnquery/v9/llx"
	"go.mondoo.com/cnquery/v9/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v9/providers/os/connection/shared"
	"go.mondoo.com/cnquery/v9/types"
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

func (k *mqlPython) id() (string, error) {
	return "python", nil
}

func (k *mqlPython) getAllPackages() ([]pythonPackageDetails, error) {
	allResults := []pythonPackageDetails{}

	conn, ok := k.MqlRuntime.Connection.(shared.Connection)
	if !ok {
		return nil, fmt.Errorf("provider is not an operating system provider")
	}
	afs := &afero.Afero{Fs: conn.FileSystem()}

	if k.Path.Error != nil {
		return nil, k.Path.Error
	}
	pyPath := k.Path.Data
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

	return allResults, nil
}

func (k *mqlPython) packages() ([]interface{}, error) {
	allPyPkgDetails, err := k.getAllPackages()
	if err != nil {
		return nil, err
	}

	// this is the "global" map so that the recursive function calls can keep track of
	// resources already created
	pythonPackageResourceMap := map[string]plugin.Resource{}

	resp := []interface{}{}

	for _, pyPkgDetails := range allPyPkgDetails {
		res, err := pythonPackageDetailsWithDependenciesToResource(k.MqlRuntime, pyPkgDetails, allPyPkgDetails, pythonPackageResourceMap)
		if err != nil {
			log.Error().Err(err).Msg("error while creating resource(s) for python package")
			// we will keep trying to make resources even if a single one failed
			continue
		}
		resp = append(resp, res)
	}

	return resp, nil
}

func pythonPackageDetailsWithDependenciesToResource(runtime *plugin.Runtime, newPyPkgDetails pythonPackageDetails,
	pythonPgkDetailsList []pythonPackageDetails, pythonPackageResourceMap map[string]plugin.Resource,
) (interface{}, error) {
	res := pythonPackageResourceMap[newPyPkgDetails.name]
	if res != nil {
		// already created the pythonPackage resource
		return res, nil
	}

	dependencies := []interface{}{}
	for _, dep := range newPyPkgDetails.dependencies {
		found := false
		var depPyPkgDetails pythonPackageDetails
		for i, pyPkgDetails := range pythonPgkDetailsList {
			if pyPkgDetails.name == dep {
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
			log.Warn().Err(err).Msg("failed to create python packag resource")
			continue
		}
		dependencies = append(dependencies, res)
	}

	// finally create the resource
	r, err := pythonPackageDetailsToResource(runtime, newPyPkgDetails, dependencies)
	if err != nil {
		log.Error().Err(err).Str("resource", newPyPkgDetails.file).Msg("error while creating MQL resource")
		return nil, err
	}

	pythonPackageResourceMap[newPyPkgDetails.name] = r

	return r, nil
}

func pythonPackageDetailsToResource(runtime *plugin.Runtime, ppd pythonPackageDetails, dependencies []interface{}) (plugin.Resource, error) {
	f, err := CreateResource(runtime, "file", map[string]*llx.RawData{
		"path": llx.StringData(ppd.file),
	})
	if err != nil {
		log.Error().Err(err).Msg("error while creating file resource for python package resource")
		return nil, err
	}

	cpes := []interface{}{}
	for i := range ppd.cpes {
		cpe, err := runtime.CreateSharedResource("cpe", map[string]*llx.RawData{
			"uri": llx.StringData(ppd.cpes[i]),
		})
		if err != nil {
			return nil, err
		}
		cpes = append(cpes, cpe)
	}

	r, err := CreateResource(runtime, "python.package", map[string]*llx.RawData{
		"id":           llx.StringData(ppd.file),
		"name":         llx.StringData(ppd.name),
		"version":      llx.StringData(ppd.version),
		"author":       llx.StringData(ppd.author),
		"summary":      llx.StringData(ppd.summary),
		"license":      llx.StringData(ppd.license),
		"file":         llx.ResourceData(f, f.MqlName()),
		"dependencies": llx.ArrayData(dependencies, types.Any),
		"purl":         llx.StringData(ppd.purl),
		"cpes":         llx.ArrayData(cpes, types.Resource("cpe")),
	})
	if err != nil {
		log.Error().AnErr("err", err).Msg("error while creating MQL resource")
		return nil, err
	}
	return r, nil
}

func (k *mqlPython) toplevel() ([]interface{}, error) {
	allPyPkgDetails, err := k.getAllPackages()
	if err != nil {
		return nil, err
	}

	// this is the "global" map so that the recursive function calls can keep track of
	// resources already created
	pythonPackageResourceMap := map[string]plugin.Resource{}

	resp := []interface{}{}

	for _, pyPkgDetails := range allPyPkgDetails {
		if !pyPkgDetails.isLeaf {
			continue
		}

		res, err := pythonPackageDetailsWithDependenciesToResource(k.MqlRuntime, pyPkgDetails, allPyPkgDetails, pythonPackageResourceMap)
		if err != nil {
			log.Error().Err(err).Msg("error while creating resource(s) for python package")
			// we will keep trying to make resources even if a single one failed
			continue
		}
		resp = append(resp, res)
	}

	return resp, nil
}

type pythonPackageDetails struct {
	name         string
	file         string
	license      string
	author       string
	summary      string
	version      string
	dependencies []string
	isLeaf       bool
	purl         string
	cpes         []string
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
		// only process files/directories that might acctually contain
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
		ppd.isLeaf = requestedPackage

		// if the MIME data didn't include dependency information, but there was a requires.txt file available,
		// then use that for dependency info (as pip appears to do)
		if len(ppd.dependencies) == 0 && requiresTxtPath != "" {
			requiresTxtDeps, err := parseRequiresTxtDependencies(afs, filepath.Join(pythonPackagePath, requiresTxtPath))
			if err != nil {
				log.Warn().Err(err).Str("dir", pythonPackageFilepath).Msg("failed to parse requires.txt")
			} else {
				ppd.dependencies = requiresTxtDeps
			}
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

// firstWordRegexp is just trying to catch everything leading up the >, >=, = in a requires.txt
// Example:
//
// nose>=1.2
// Mock>=1.0
// pycryptodome
//
// [crypto]
// pycryptopp>=0.5.12
//
// [cryptography]
// cryptography
//
// would match nose / Mock / pycrptodome / etc

var firstWordRegexp = regexp.MustCompile(`^[a-zA-Z0-9\._-]*`)

func parseRequiresTxtDependencies(afs *afero.Afero, requiresTxtPath string) ([]string, error) {
	f, err := afs.Open(requiresTxtPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	fileScanner := bufio.NewScanner(f)
	fileScanner.Split(bufio.ScanLines)

	depdendencies := []string{}
	for fileScanner.Scan() {
		line := fileScanner.Text()
		if strings.HasPrefix(line, "[") {
			// this means a new optional section of dependencies
			// so stop processing
			break
		}
		matched := firstWordRegexp.FindString(line)
		if matched == "" {
			continue
		}
		depdendencies = append(depdendencies, matched)
	}

	return depdendencies, nil
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
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("error reading MIME data: %s", err)
	}

	deps := extractMimeDeps(mimeData.Values("Requires-Dist"))

	cpes := []string{}
	cpeEntry, err := cpe.NewPackage2Cpe(mimeData.Get("Name")+"_project", mimeData.Get("Name"), mimeData.Get("Version"), "", "")
	if err == nil && cpeEntry != "" {
		cpes = append(cpes, cpeEntry)
	}

	return &pythonPackageDetails{
		name:         mimeData.Get("Name"),
		summary:      mimeData.Get("Summary"),
		author:       mimeData.Get("Author"),
		license:      mimeData.Get("License"),
		version:      mimeData.Get("Version"),
		dependencies: deps,
		file:         pythonMIMEFilepath,
		purl:         newPythonPackageUrl(mimeData.Get("Name"), mimeData.Get("Version"), mimeData.Get("Home-page")),
		cpes:         cpes,
	}, nil
}

func newPythonPackageUrl(name string, version string, homepage string) string {
	// ensure the name is accoring to the PURL spec
	// see https://github.com/package-url/purl-spec/blob/master/PURL-TYPES.rst#pypi
	name = strings.ReplaceAll(name, "_", "-")

	return packageurl.NewPackageURL(
		packageurl.TypePyPi,
		"",
		name,
		version,
		nil,
		"").String()
}

// extractMimeDeps will go through each of the listed dependencies
// from the "Requires-Dist" values, and strip off everything but
// the name of the package/dependency itself
func extractMimeDeps(deps []string) []string {
	parsedDeps := []string{}
	for _, dep := range deps {
		// the semicolon indicates an optional dependency
		if strings.Contains(dep, ";") {
			continue
		}
		parsedDep := strings.Split(dep, " ")
		if len(parsedDep) > 0 {
			parsedDeps = append(parsedDeps, parsedDep[0])
		}
	}
	return parsedDeps
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
