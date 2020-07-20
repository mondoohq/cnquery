package packages

import (
	"bufio"
	"fmt"
	"io"
	"regexp"

	"go.mondoo.io/mondoo/motor"
)

var (
	OPKG_REGEX = regexp.MustCompile(`^([\w\d\-]+)\s-\s([\w\d\-\.]+)$`)
)

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
				Format:  "opkg",
			})
		}
	}
	return pkgs
}

type OpkgPkgManager struct {
	motor *motor.Motor
}

func (opkg *OpkgPkgManager) Name() string {
	return "Opkg Package Manager"
}

func (opkg *OpkgPkgManager) Format() string {
	return "opkg"
}

func (opkg *OpkgPkgManager) List() ([]Package, error) {
	cmd, err := opkg.motor.Transport.RunCommand("opkg list")
	if err != nil {
		return nil, fmt.Errorf("could not read package list")
	}

	return ParseOpkgPackages(cmd.Stdout), nil
}

func (opkg *OpkgPkgManager) Available() (map[string]PackageUpdate, error) {
	return map[string]PackageUpdate{}, nil
}
