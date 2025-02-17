// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package mvd

import "go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"

//go:generate protoc --proto_path=../../../../:. --go_out=. --go_opt=paths=source_relative --rangerrpc_out=. mvd.proto

// Determine all Cves of all Advisories
func (r *VulnReport) Cves() []*CVE {
	cveMap := map[string]*CVE{}

	for i := range r.Advisories {
		advisory := r.Advisories[i]
		for j := range advisory.Cves {
			cve := advisory.Cves[j]
			cveMap[cve.ID] = cve
		}
	}

	cveList := []*CVE{}
	for _, v := range cveMap {
		cveList = append(cveList, v)
	}
	return cveList
}

// MvdPlatform converts the inventory.Asset that contains the platform to the
// platform object we use for MVD
func NewMvdPlatform(asset *inventory.Asset) *Platform {
	if asset == nil || asset.GetPlatform() == nil {
		return nil
	}
	return &Platform{
		Name:    asset.Platform.Name,
		Release: asset.Platform.Version,
		Build:   asset.Platform.Build,
		Arch:    asset.Platform.Arch,
		Title:   asset.Platform.Title,
		Labels:  asset.Labels,
	}
}
