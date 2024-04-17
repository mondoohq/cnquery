// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package purl

import (
	"github.com/package-url/packageurl-go"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v11/providers/os/detector"
	"sort"
	"strings"
)

const (
	QualifierArch   = "arch"
	QualifierDistro = "distro"
	QualifierEpoch  = "epoch"
)

// NewQualifiers creates a new Qualifiers slice from a map of key/value pairs.
// see https://github.com/package-url/purl-spec/blob/master/PURL-TYPES.rst for more information
func NewQualifiers(qualifier map[string]string) packageurl.Qualifiers {
	// Create a slice for the keys to sort them
	keys := make([]string, 0, len(qualifier))
	for k := range qualifier {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Create the list of Qualifiers
	list := make(packageurl.Qualifiers, 0, len(keys))
	for _, k := range keys {
		val := qualifier[k]
		if val != "" {
			list = append(list, packageurl.Qualifier{
				Key:   k,
				Value: val,
			})
		}
	}

	return list
}

// NewPackageUrl creates a new package url for a given platform, name, version, arch, epoch and purlType
// see https://github.com/package-url/purl-spec/blob/master/PURL-TYPES.rst for more information
func NewPackageUrl(pf *inventory.Platform, name string, version string, arch string, epoch string, purlType string) string {
	qualifiers := map[string]string{}
	if arch != "" {
		qualifiers[QualifierArch] = arch
	}

	if epoch != "" && epoch != "0" {
		qualifiers[QualifierEpoch] = epoch
	}

	namespace := pf.Name
	if pf.Labels != nil && pf.Labels[detector.LabelDistroID] != "" {
		namespace = pf.Labels[detector.LabelDistroID]
	}

	// generate distro qualifier
	distroQualifiers := []string{}
	distroQualifiers = append(distroQualifiers, namespace)
	if pf.Version != "" {
		distroQualifiers = append(distroQualifiers, pf.Version)
	} else if pf.Build != "" {
		distroQualifiers = append(distroQualifiers, pf.Build)
	}
	qualifiers[QualifierDistro] = strings.Join(distroQualifiers, "-")

	return packageurl.NewPackageURL(
		purlType,
		namespace,
		name,
		version,
		NewQualifiers(qualifiers),
		"",
	).ToString()
}
