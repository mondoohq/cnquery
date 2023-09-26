// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package packages

import (
	"bufio"
	"fmt"
	"go.mondoo.com/cnquery/providers/os/connection/shared"
	"io"
	"strings"
)

const (
	AixPkgFormat = "bff"
)

func parseAixPackages(r io.Reader) ([]Package, error) {
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

		// Fileset, Level, PtfID, State, Type, Description, EFIXLocked
		pkgs = append(pkgs, Package{
			Name:        record[1],
			Version:     record[2],
			Description: strings.TrimSpace(record[6]),
			Format:      AixPkgFormat,
		})

	}
	return pkgs, nil
}

type AixPkgManager struct {
	conn shared.Connection
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

	return parseAixPackages(cmd.Stdout)
}

func (f *AixPkgManager) Available() (map[string]PackageUpdate, error) {
	return map[string]PackageUpdate{}, nil
}
