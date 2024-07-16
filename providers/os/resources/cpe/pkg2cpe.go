// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package cpe

import (
	"errors"
	"fmt"
	"regexp"
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

	// Remove epoch when present
	// Otherwise the WFNize will only use the epoch as the version
	epochRegex := regexp.MustCompile(`^\d+:(.*)$`)
	if matches := epochRegex.FindStringSubmatch(version); len(matches) > 1 {
		version = matches[1]
	}

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

	genericMutationAttr := attr
	// Modify the CPE to later have a higher chance of matching
	for _, mutation := range genericCPEVendorMutations {
		vendorMutationAttr := mutation(genericMutationAttr)
		if vendorMutationAttr != nil {
			cpes = append(cpes, vendorMutationAttr.BindToFmtString())
		} else {
			vendorMutationAttr = &genericMutationAttr
		}
		for _, mutation := range genericCPEProductMutations {
			productMutationAttr := mutation(*vendorMutationAttr)
			if productMutationAttr != nil {
				cpes = append(cpes, productMutationAttr.BindToFmtString())
			} else {
				productMutationAttr = vendorMutationAttr
			}
			for _, mutation := range genericCPEVersionMutations {
				versionMutationAttr := mutation(*productMutationAttr)
				if versionMutationAttr != nil {
					cpes = append(cpes, versionMutationAttr.BindToFmtString())
				}
			}
		}
	}

	return cpes, nil
}

var genericCPEProductMutations = []func(attr wfn.Attributes) *wfn.Attributes{
	func(attr wfn.Attributes) *wfn.Attributes {
		if strings.HasSuffix(attr.Product, attr.Version) {
			attr.Product = strings.TrimSuffix(attr.Product, attr.Version)
			attr.Product = strings.TrimSuffix(attr.Product, "_")
			return &attr
		}
		return nil
	},
}

var genericCPEVendorMutations = []func(attr wfn.Attributes) *wfn.Attributes{
	// e.g. "microsoft_corporation" -> "microsoft"
	// e.g. "nextgencorporation" -> "nextgen"
	func(attr wfn.Attributes) *wfn.Attributes {
		if strings.HasSuffix(attr.Vendor, "corporation") {
			attr.Vendor = strings.TrimSuffix(attr.Vendor, "corporation")
			attr.Vendor = strings.TrimSuffix(attr.Vendor, "_")
			return &attr
		}
		return nil
	},
}

var genericCPEVersionMutations = []func(attr wfn.Attributes) *wfn.Attributes{
	func(attr wfn.Attributes) *wfn.Attributes {
		versionParts := strings.Split(attr.Version, ".")
		if len(versionParts) > 3 {
			attr.Version = strings.Join(versionParts[:3], ".")
			attr.Version = strings.TrimSuffix(attr.Version, "\\")
			return &attr
		}
		return nil
	},
	func(attr wfn.Attributes) *wfn.Attributes {
		versionParts := strings.Split(attr.Version, "-")
		if len(versionParts) > 1 {
			attr.Version = versionParts[0]
			attr.Version = strings.TrimSuffix(attr.Version, "\\")
			return &attr
		}
		return nil
	},
}
