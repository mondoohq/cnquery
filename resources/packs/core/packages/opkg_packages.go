package packages

import (
	"bufio"
	"fmt"
	"io"
	"regexp"

	"go.mondoo.com/cnquery/motor/providers/os"
)

const (
	OpkgPkgFormat = "opkg"
)

var OPKG_REGEX = regexp.MustCompile(`^([\w\d\-]+)\s-\s([\w\d\-\.]+)$`)

// parse opkg package list
func ParseOpkgPackages(input io.Reader) []Package {
	pkgs := []Package{}
	scanner := bufio.NewScanner(input)
	for scanner.Scan() {
		line := scanner.Text()
		m := OPKG_REGEX.FindStringSubmatch(line)
		if m != nil {
			pkgs = append(pkgs, Package{
				Name:    m[1],
				Version: m[2],
				Format:  OpkgPkgFormat,
			})
		}
	}
	return pkgs
}

type OpkgPkgManager struct {
	provider os.OperatingSystemProvider
}

func (opkg *OpkgPkgManager) Name() string {
	return "Opkg Package Manager"
}

func (opkg *OpkgPkgManager) Format() string {
	return OpkgPkgFormat
}

func (opkg *OpkgPkgManager) List() ([]Package, error) {
	cmd, err := opkg.provider.RunCommand("opkg list")
	if err != nil {
		return nil, fmt.Errorf("could not read package list")
	}

	return ParseOpkgPackages(cmd.Stdout), nil
}

func (opkg *OpkgPkgManager) Available() (map[string]PackageUpdate, error) {
	return map[string]PackageUpdate{}, nil
}
