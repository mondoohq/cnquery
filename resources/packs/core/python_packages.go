package core

import (
	"regexp"
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/spf13/afero"

	"go.mondoo.com/cnquery/motor/providers/os"
	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/resources/packs/core/packages"
)

func (p *mqlPythonPackages) GetList() ([]interface{}, error) {
	// find suitable package manager
	osProvider, isOSProvider := p.MotorRuntime.Motor.Provider.(os.OperatingSystemProvider)
	if !isOSProvider {
		return nil, errors.New("python package manager is not supported for platform")
	}

	// retrieve all system packages
	pyPkgs, err := listPythonPackagesFromFS(osProvider)
	if err != nil {
		return nil, err
	}

	// create MQL package os for each package
	pkgs := make([]interface{}, len(pyPkgs))
	namedMap := map[string]Package{}
	for i, pyPkg := range pyPkgs {

		pkg, err := p.MotorRuntime.CreateResource("package",
			"name", pyPkg.Name,
			"version", pyPkg.Version,
			"available", "",
			"epoch", "", // TODO: support Epoch
			"arch", pyPkg.Arch,
			"status", pyPkg.Status,
			"description", pyPkg.Description,
			"format", pyPkg.Format,
			"installed", true,
			"origin", pyPkg.Origin,
		)
		if err != nil {
			return nil, err
		}

		pkgs[i] = pkg
		namedMap[pyPkg.Name] = pkg.(Package)
	}

	p.Cache.Store("_map", &resources.CacheEntry{Data: namedMap})

	// return the packages as new entries
	return pkgs, nil
}

func listPythonPackagesFromFS(osProvider os.OperatingSystemProvider) ([]packages.Package, error) {
	fs := osProvider.FS()

	pythonVersions, err := availablePythonVersions(fs)
	if err != nil {
		return nil, err
	}
	searchPaths := []string{"/usr/local/lib", "/usr/lib"}
	allPackages := []packages.Package{}

	for _, version := range pythonVersions {
		for _, searchPath := range searchPaths {
			p := strings.Join([]string{searchPath, version, "site-packages"}, "/")
			f, err := fs.Open(p)
			if err != nil {
				if errors.Is(err, afero.ErrFileNotFound) {
					continue
				}
				return nil, err
			}
			packagesInDir, err := listPythonPackagesInDir(f)
			if err != nil {
				return nil, err
			}
			allPackages = append(allPackages, packagesInDir...)
		}
	}

	return allPackages, nil
}

func listPythonPackagesInDir(f afero.File) ([]packages.Package, error) {
	r := regexp.MustCompile("^([a-z|A-Z|0-9|_]+)-(([0-9]+)\\.[0-9]+(\\.[0-9]+)?)(-py[0-9]+\\.[0-9]+)?.(dist-info|egg-info)$")
	files, err := f.Readdirnames(-1)
	if err != nil {
		return nil, err
	}
	pkgs := []packages.Package{}
	for _, fi := range files {
		matches := r.FindStringSubmatch(fi)
		if len(matches) == 0 {
			continue
		}

		name := matches[1]
		version := matches[2]
		pkgs = append(pkgs, packages.Package{
			Name:    name,
			Version: version,
			Format:  "PyPI",
		})

	}
	return pkgs, nil
}

func availablePythonVersions(fs afero.Fs) ([]string, error) {
	searchPaths := []string{"/usr/local/lib", "/usr/lib"}
	pythonVersions := []string{}
	pythonVersionRegexp := regexp.MustCompile("^python[0-9]+\\.[0-9]+$")
	for _, searchPath := range searchPaths {
		f, err := fs.Open(searchPath)
		if err != nil {
			if errors.Is(err, afero.ErrFileNotFound) {
				continue
			}
			return nil, err
		}

		versions, err := matchFiles(f, pythonVersionRegexp)
		if err != nil {
			return nil, err
		}
		pythonVersions = append(pythonVersions, versions...)
	}
	return pythonVersions, nil
}

func matchFiles(f afero.File, r *regexp.Regexp) ([]string, error) {
	files, err := f.Readdirnames(-1)
	if err != nil {
		return nil, err
	}
	matches := []string{}
	for _, fi := range files {
		if r.MatchString(fi) {
			matches = append(matches, fi)
		}
	}
	return matches, nil
}

func (p *mqlPythonPackages) id() (string, error) {
	return "pythonPackages", nil
}
