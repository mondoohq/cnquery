// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package packages

import (
	"bufio"
	"fmt"
	"github.com/package-url/packageurl-go"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v10/providers/os/resources/purl"
	"io"
	"regexp"

	"github.com/cockroachdb/errors"
	"go.mondoo.com/cnquery/v10/providers/os/connection/shared"
)

const (
	PacmanPkgFormat = "pacman"
)

var PACMAN_REGEX = regexp.MustCompile(`^([\w-]*)\s([\w\d-+.:]+)$`)

func ParsePacmanPackages(pf *inventory.Platform, input io.Reader) []Package {
	pkgs := []Package{}
	scanner := bufio.NewScanner(input)
	for scanner.Scan() {
		line := scanner.Text()
		m := PACMAN_REGEX.FindStringSubmatch(line)
		if m != nil {
			name := m[1]
			version := m[2]
			pkgs = append(pkgs, Package{
				Name:    name,
				Version: version,
				Format:  PacmanPkgFormat,
				PUrl:    purl.NewPackageUrl(pf, name, version, "", "", packageurl.TypeAlpm),
			})
		}
	}
	return pkgs
}

// Arch, Manjaro
type PacmanPkgManager struct {
	conn     shared.Connection
	platform *inventory.Platform
}

func (ppm *PacmanPkgManager) Name() string {
	return "Pacman Package Manager"
}

func (ppm *PacmanPkgManager) Format() string {
	return PacmanPkgFormat
}

func (ppm *PacmanPkgManager) List() ([]Package, error) {
	cmd, err := ppm.conn.RunCommand("pacman -Q")
	if err != nil {
		return nil, fmt.Errorf("could not read package list")
	}

	return ParsePacmanPackages(ppm.platform, cmd.Stdout), nil
}

func (ppm *PacmanPkgManager) Available() (map[string]PackageUpdate, error) {
	return nil, errors.New("Available() not implemented for pacman")
}
