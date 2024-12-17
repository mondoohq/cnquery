// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package packages

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
	cpe2 "go.mondoo.com/cnquery/v11/providers/os/resources/cpe"
	"go.mondoo.com/cnquery/v11/providers/os/resources/purl"

	"go.mondoo.com/cnquery/v11/providers/os/connection/shared"
)

const (
	AixPkgFormat = "bff"
)

func parseAixPackages(pf *inventory.Platform, r io.Reader) ([]Package, error) {
	pkgs := []Package{}

	scanner := bufio.NewScanner(r)
	i := 0
	for scanner.Scan() {
		line := scanner.Text()
		i++

		if i == 1 {
			continue
		}

		record := strings.Split(line, ":")

		cpes, _ := cpe2.NewPackage2Cpe(record[1], record[1], record[2], "", pf.Arch)
		// Fileset, Level, PtfID, State, Type, Description, EFIXLocked
		pkgs = append(pkgs, Package{
			Name:        record[1],
			Version:     record[2],
			Description: strings.TrimSpace(record[6]),
			Format:      AixPkgFormat,
			PUrl: purl.NewPackageURL(
				pf, purl.TypeGeneric, record[1], record[2], purl.WithNamespace(pf.Name),
			).String(),
			CPEs: cpes,
		})

	}
	return pkgs, nil
}

type AixPkgManager struct {
	conn     shared.Connection
	platform *inventory.Platform
}

func (a *AixPkgManager) Name() string {
	return "AIX Package Manager"
}

func (a *AixPkgManager) Format() string {
	return AixPkgFormat
}

func (a *AixPkgManager) List() ([]Package, error) {
	cmd, err := a.conn.RunCommand("lslpp -cl ")
	if err != nil {
		return nil, fmt.Errorf("could not read freebsd package list")
	}

	return parseAixPackages(a.platform, cmd.Stdout)
}

func (a *AixPkgManager) Available() (map[string]PackageUpdate, error) {
	return map[string]PackageUpdate{}, nil
}

func (a *AixPkgManager) Files(name string, version string, arch string) ([]FileRecord, error) {
	// not yet implemented
	return nil, nil
}
