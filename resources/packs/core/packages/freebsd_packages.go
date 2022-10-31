package packages

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"

	"go.mondoo.com/cnquery/motor/providers/os"
)

const (
	FreebsdPkgFormat = "freebsd"
)

type FreeBSDPackage struct {
	Maintainer string
	Name       string
	Comment    string
	Desc       string
	Version    string
	Origin     string
	Arch       string
}

func ParseFreeBSDPackages(r io.Reader) ([]Package, error) {
	pkgs := []Package{}

	// the raw list does not return a valid json slice, therefore
	// we need to read and parse each line individually
	// https://github.com/freebsd/pkg/issues/1287
	reader := bufio.NewReader(r)
	for {
		line, err := reader.ReadString('\n')
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}

		var freeBSDPkg FreeBSDPackage
		err = json.Unmarshal([]byte(line), &freeBSDPkg)
		if err != nil {
			return nil, err
		}

		pkgs = append(pkgs, Package{
			Name:        freeBSDPkg.Name,
			Version:     freeBSDPkg.Version,
			Description: freeBSDPkg.Desc,
			Arch:        freeBSDPkg.Arch,
			Origin:      freeBSDPkg.Origin,
			Format:      FreebsdPkgFormat,
		})
	}

	return pkgs, nil
}

type FreeBSDPkgManager struct {
	provider os.OperatingSystemProvider
}

func (f *FreeBSDPkgManager) Name() string {
	return "FreeBSD Package Manager"
}

func (f *FreeBSDPkgManager) Format() string {
	return FreebsdPkgFormat
}

func (f *FreeBSDPkgManager) List() ([]Package, error) {
	cmd, err := f.provider.RunCommand("pkg info --raw --raw-format json-compact --all")
	if err != nil {
		return nil, fmt.Errorf("could not read freebsd package list")
	}

	return ParseFreeBSDPackages(cmd.Stdout)
}

func (f *FreeBSDPkgManager) Available() (map[string]PackageUpdate, error) {
	return map[string]PackageUpdate{}, nil
}
