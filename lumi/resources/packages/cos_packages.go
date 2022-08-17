package packages

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"

	"go.mondoo.io/mondoo/motor/providers/os"
)

const (
	CosPkgFormat = "cos"
)

type cosPackages struct {
	InstalledPackages []cosPackage `json:"installedPackages"`
}

type cosPackage struct {
	Name          string `json:"name"`
	Version       string `json:"version"`
	Category      string `json:"category"`
	EbuildVersion string `json:"ebuild_version"`
}

type CosPkgManager struct {
	provider os.OperatingSystemProvider
}

func (cpm *CosPkgManager) Name() string {
	return "COS Package Manager"
}

func (cpm *CosPkgManager) Format() string {
	return CosPkgFormat
}

func (cpm *CosPkgManager) List() ([]Package, error) {
	// added as a feature in cos 85
	// https://cloud.google.com/container-optimized-os/docs/release-notes/m85#cos-85-13310-1260-1
	fr, err := cpm.provider.FS().Open("/etc/cos-package-info.json")
	if err != nil {
		return nil, fmt.Errorf("could not read package list")
	}
	defer fr.Close()

	return ParseCosPackages(fr)
}

func (mpm *CosPkgManager) Available() (map[string]PackageUpdate, error) {
	return nil, errors.New("cannot determine available packages for cos")
}

func ParseCosPackages(input io.Reader) ([]Package, error) {
	data, err := ioutil.ReadAll(input)
	if err != nil {
		return nil, err
	}

	// handle case where no packages are installed
	if len(data) == 0 {
		return []Package{}, nil
	}

	cPkgs := cosPackages{}
	err = json.Unmarshal(data, &cPkgs)
	if err != nil {
		return nil, err
	}

	pkgs := make([]Package, len(cPkgs.InstalledPackages))
	for i := range cPkgs.InstalledPackages {
		pkgs[i].Name = cPkgs.InstalledPackages[i].Name
		pkgs[i].Version = cPkgs.InstalledPackages[i].EbuildVersion
		pkgs[i].Format = CosPkgFormat
	}

	return pkgs, nil
}
