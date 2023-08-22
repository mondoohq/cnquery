// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package oci

import "go.mondoo.com/cnquery/motor/platform"

func (p *Provider) Identifier() (string, error) {
	return "//platformid.api.mondoo.app/runtime/oci/" + p.id, nil
}

func (p *Provider) PlatformInfo() (*platform.Platform, error) {
	return &platform.Platform{
		Name:    "oci",
		Title:   "Oracle Cloud Infrastructure",
		Runtime: p.Runtime(),
		Kind:    p.Kind(),
		Family:  []string{"oci"},
	}, nil
}
