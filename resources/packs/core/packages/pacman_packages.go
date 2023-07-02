package packages

import (
	"bufio"
	"fmt"
	"io"
	"regexp"

	"go.mondoo.com/cnquery/motor/providers/os"

	"errors"
)

const (
	PacmanPkgFormat = "pacman"
)

var PACMAN_REGEX = regexp.MustCompile(`^([\w-]*)\s([\w\d-+.:]+)$`)

func ParsePacmanPackages(input io.Reader) []Package {
	pkgs := []Package{}
	scanner := bufio.NewScanner(input)
	for scanner.Scan() {
		line := scanner.Text()
		m := PACMAN_REGEX.FindStringSubmatch(line)
		if m != nil {
			pkgs = append(pkgs, Package{
				Name:    m[1],
				Version: m[2],
				Format:  PacmanPkgFormat,
			})
		}
	}
	return pkgs
}

// Arch, Manjaro
type PacmanPkgManager struct {
	provider os.OperatingSystemProvider
}

func (ppm *PacmanPkgManager) Name() string {
	return "Pacman Package Manager"
}

func (ppm *PacmanPkgManager) Format() string {
	return PacmanPkgFormat
}

func (ppm *PacmanPkgManager) List() ([]Package, error) {
	cmd, err := ppm.provider.RunCommand("pacman -Q")
	if err != nil {
		return nil, fmt.Errorf("could not read package list")
	}

	return ParsePacmanPackages(cmd.Stdout), nil
}

func (ppm *PacmanPkgManager) Available() (map[string]PackageUpdate, error) {
	return nil, errors.New("Available() not implemented for pacman")
}
