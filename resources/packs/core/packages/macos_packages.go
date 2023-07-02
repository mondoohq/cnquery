package packages

import (
	"fmt"
	"io"
	"io/ioutil"
	"strings"

	"go.mondoo.com/cnquery/motor/providers/os"

	"errors"
	plist "howett.net/plist"
)

const (
	MacosPkgFormat = "macos"
)

// parse macos system version property list
func ParseMacOSPackages(input io.Reader) ([]Package, error) {
	var r io.ReadSeeker
	r, ok := input.(io.ReadSeeker)

	// if the read seaker is not implemented lets cache stdout in-memory
	if !ok {
		packageList, err := ioutil.ReadAll(input)
		if err != nil {
			return nil, err
		}
		r = strings.NewReader(string(packageList))
	}

	type sysProfilerItems struct {
		Name    string `plist:"_name"`
		Version string `plist:"version"`
	}

	type sysProfiler struct {
		Items []sysProfilerItems `plist:"_items"`
	}

	var data []sysProfiler
	decoder := plist.NewDecoder(r)
	err := decoder.Decode(&data)
	if err != nil {
		return nil, err
	}

	if len(data) != 1 {
		return nil, errors.New("format not supported")
	}

	pkgs := make([]Package, len(data[0].Items))
	for i, entry := range data[0].Items {
		pkgs[i].Name = entry.Name
		pkgs[i].Version = entry.Version
		pkgs[i].Format = MacosPkgFormat
	}

	return pkgs, nil
}

// MacOS
type MacOSPkgManager struct {
	provider os.OperatingSystemProvider
}

func (mpm *MacOSPkgManager) Name() string {
	return "macOS Package Manager"
}

func (mpm *MacOSPkgManager) Format() string {
	return MacosPkgFormat
}

func (mpm *MacOSPkgManager) List() ([]Package, error) {
	cmd, err := mpm.provider.RunCommand("system_profiler SPApplicationsDataType -xml")
	if err != nil {
		return nil, fmt.Errorf("could not read package list")
	}

	return ParseMacOSPackages(cmd.Stdout)
}

func (mpm *MacOSPkgManager) Available() (map[string]PackageUpdate, error) {
	return nil, errors.New("cannot determine available packages for macOS")
}
