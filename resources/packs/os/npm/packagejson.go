package npm

import (
	"encoding/json"
	"io"
	"io/ioutil"

	"go.mondoo.com/cnquery/resources/packs/core/vadvisor"
)

// PackageJson allows parsing the package json file
type PackageJson struct {
	Name            string            `json:"name"`
	Description     string            `json:"description"`
	Version         string            `json:"version"`
	License         string            `jsonn:"license"`
	Dependencies    map[string]string `jsonn:"dependencies"`
	DevDependencies map[string]string `jsonn:"devDependencies"`
}

type PackageJsonLockEntry struct {
	Version string `json:"version"`
	Dev     bool   `json:"dev"`
}

// PackageJsonLock is the struct to represent the package.lock file
type PackageJsonLock struct {
	Name         string                          `json:"name"`
	Version      string                          `json:"version"`
	Dependencies map[string]PackageJsonLockEntry `jsonn:"dependencies"`
}

func ParsePackageJson(r io.Reader) ([]*vadvisor.Package, error) {

	data, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	var packageJson PackageJson
	err = json.Unmarshal(data, &packageJson)
	if err != nil {
		return nil, err
	}

	entries := []*vadvisor.Package{}

	// add own package
	entries = append(entries, &vadvisor.Package{
		Name:      packageJson.Name,
		Version:   packageJson.Version,
		Format:    "npm",
		Namespace: "nodejs",
	})

	// add all dependencies

	for k, v := range packageJson.Dependencies {
		entries = append(entries, &vadvisor.Package{
			Name:      k,
			Version:   v,
			Format:    "npm",
			Namespace: "nodejs",
		})
	}

	return entries, nil
}

func ParsePackageJsonLock(r io.Reader) ([]*vadvisor.Package, error) {

	data, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	var packageJsonLock PackageJsonLock
	err = json.Unmarshal(data, &packageJsonLock)
	if err != nil {
		return nil, err
	}

	entries := []*vadvisor.Package{}

	// add own package
	entries = append(entries, &vadvisor.Package{
		Name:      packageJsonLock.Name,
		Version:   packageJsonLock.Version,
		Format:    "npm",
		Namespace: "nodejs",
	})

	// add all dependencies
	for k, v := range packageJsonLock.Dependencies {
		entries = append(entries, &vadvisor.Package{
			Name:      k,
			Version:   v.Version,
			Format:    "npm",
			Namespace: "nodejs",
		})
	}

	return entries, nil
}
