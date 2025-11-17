// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package packages

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/rs/zerolog/log"

	"go.mondoo.com/cnquery/v12/providers/os/connection/shared"
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

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		parts := strings.Split(line, "\t")
		if len(parts) != 5 {
			log.Debug().Msgf("skipping invalid freebsd package line: %s", line)
			continue
		}

		pkgs = append(pkgs, Package{
			Name:        parts[0],
			Version:     parts[1],
			Description: parts[2],
			Arch:        parts[3],
			Origin:      parts[4],
			Format:      FreebsdPkgFormat,
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return pkgs, nil
}

type FreeBSDPkgManager struct {
	conn shared.Connection
}

func (f *FreeBSDPkgManager) Name() string {
	return "FreeBSD Package Manager"
}

func (f *FreeBSDPkgManager) Format() string {
	return FreebsdPkgFormat
}

func (f *FreeBSDPkgManager) List() ([]Package, error) {
	cmd, err := f.conn.RunCommand("pkg query -a '%n\\t%v\\t%c\\t%q\\t%o'")
	if err != nil {
		return nil, fmt.Errorf("could not read freebsd package list")
	}

	return ParseFreeBSDPackages(cmd.Stdout)
}

func (f *FreeBSDPkgManager) Available() (map[string]PackageUpdate, error) {
	return map[string]PackageUpdate{}, nil
}

func (mpm *FreeBSDPkgManager) Files(name string, version string, arch string) ([]FileRecord, error) {
	// not yet implemented
	return nil, nil
}
