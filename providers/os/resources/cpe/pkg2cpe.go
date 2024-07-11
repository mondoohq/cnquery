// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package cpe

import (
	"errors"
	"fmt"
	"strings"

	"github.com/facebookincubator/nvdtools/wfn"
)

func NewPackage2Cpe(vendor, name, version, release, arch string) ([]string, error) {
	cpes := []string{}
	vendor = strings.ToLower(vendor)
	name = strings.ToLower(name)
	version = strings.ToLower(version)
	release = strings.ToLower(release)
	arch = strings.ToLower(arch)

	var err error
	for n, addr := range map[string]*string{
		"vendor":  &vendor,
		"name":    &name,
		"version": &version,
		"release": &release,
		"arch":    &arch,
	} {
		if *addr, err = wfn.WFNize(*addr); err != nil {
			return cpes, fmt.Errorf("couldn't wfnize %s %q: %v", n, *addr, err)
		}
	}

	if name == "" {
		return cpes, errors.New("name is empty")
	}
	if version == "" {
		return cpes, errors.New("version is empty")
	}

	attr := wfn.Attributes{}
	attr.Part = "a"
	attr.Vendor = vendor
	attr.Product = name
	attr.Version = version
	attr.Update = release
	attr.TargetHW = arch

	cpes = append(cpes, attr.BindToFmtString())

	// TODO: modify different fields to have a higher chance of matching
	// nested for loops with funcs to modify different fields

	// Modify the CPE to later have a higher chance of matching
	// e.g. "microsoft_corporation" -> "microsoft"
	if strings.HasSuffix(attr.Vendor, "_corporation") {
		attr.Vendor = strings.TrimSuffix(attr.Vendor, "_corporation")
		cpes = append(cpes, attr.BindToFmtString())
	}

	if strings.HasSuffix(attr.Product, attr.Version) {
		attr.Product = strings.TrimSuffix(attr.Product, attr.Version)
		cpes = append(cpes, attr.BindToFmtString())
	}

	versionParts := strings.Split(attr.Version, ".")
	if len(versionParts) > 3 {
		attr.Version = strings.Join(versionParts[:3], ".")
		cpes = append(cpes, attr.BindToFmtString())
	}

	return cpes, nil
}
