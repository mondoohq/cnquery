// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package packages

import (
	"bufio"
	"fmt"
	"github.com/package-url/packageurl-go"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/inventory"
	cpe2 "go.mondoo.com/cnquery/v10/providers/os/resources/cpe"
	"go.mondoo.com/cnquery/v10/providers/os/resources/purl"
	"io"
	"strings"

	"go.mondoo.com/cnquery/v10/providers/os/connection/shared"
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

		cpe, _ := cpe2.NewPackage2Cpe(record[1], record[1], record[2], "", pf.Arch)
		// Fileset, Level, PtfID, State, Type, Description, EFIXLocked
		pkgs = append(pkgs, Package{
			Name:        record[1],
			Version:     record[2],
			Description: strings.TrimSpace(record[6]),
			Format:      AixPkgFormat,
			PUrl:        purl.NewPackageUrl(pf, record[1], record[2], "", "", packageurl.TypeGeneric),
			CPE:         cpe,
		})

	}
	return pkgs, nil
}

type AixPkgManager struct {
	conn     shared.Connection
	platform *inventory.Platform
}

func (f *AixPkgManager) Name() string {
	return "AIX Package Manager"
}

func (f *AixPkgManager) Format() string {
	return AixPkgFormat
}

func (f *AixPkgManager) List() ([]Package, error) {
	cmd, err := f.conn.RunCommand("lslpp -cl ")
	if err != nil {
		return nil, fmt.Errorf("could not read freebsd package list")
	}

	return parseAixPackages(f.platform, cmd.Stdout)
}

func (f *AixPkgManager) Available() (map[string]PackageUpdate, error) {
	return map[string]PackageUpdate{}, nil
}
