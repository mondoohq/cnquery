// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package packages

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"go.mondoo.com/cnquery/v12/providers/os/connection/shared"
)

const (
	GentooPkgFormat = "gentoo"
)

type GentooPackage struct {
	Name    string
	Version string
}

func ParseGentooPackages(r io.Reader) ([]Package, error) {
	pkgs := []Package{}

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		// Split by colon delimiter
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		name := strings.TrimSpace(parts[0])
		version := strings.TrimSpace(parts[1])

		pkgs = append(pkgs, Package{
			Name:    name,
			Version: version,
			Format:  GentooPkgFormat,
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return pkgs, nil
}

type GentooPkgManager struct {
	conn shared.Connection
}

func (f *GentooPkgManager) Name() string {
	return "Gentoo Package Manager"
}

func (f *GentooPkgManager) Format() string {
	return GentooPkgFormat
}

func (f *GentooPkgManager) List() ([]Package, error) {
	cmd, err := f.conn.RunCommand("qlist -Iv --format '%{CATEGORY}/%{PN}:%{PVR}'")
	if err != nil {
		return nil, fmt.Errorf("could not read gentoo package list from qlist")
	}

	return ParseGentooPackages(cmd.Stdout)
}

func (f *GentooPkgManager) Available() (map[string]PackageUpdate, error) {
	return map[string]PackageUpdate{}, nil
}

func (mpm *GentooPkgManager) Files(name string, version string, arch string) ([]FileRecord, error) {
	// not yet implemented
	return nil, nil
}
