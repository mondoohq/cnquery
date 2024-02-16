// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package cpe

import (
	"errors"
	"fmt"
	"strings"

	"github.com/facebookincubator/nvdtools/wfn"
)

func NewPackage2Cpe(vendor, name, version, release, arch string) (string, error) {
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
			return "", fmt.Errorf("couldn't wfnize %s %q: %v", n, *addr, err)
		}
	}

	if name == "" {
		return "", errors.New("name is empty")
	}
	if version == "" {
		return "", errors.New("version is empty")
	}

	attr := wfn.Attributes{}
	attr.Part = "a"
	attr.Vendor = vendor
	attr.Product = name
	attr.Version = version
	attr.Update = release
	attr.TargetHW = arch

	return attr.BindToFmtString(), nil
}
