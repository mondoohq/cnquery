// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package opcua

import "go.mondoo.com/cnquery/motor/platform"

func (p *Provider) Identifier() (string, error) {
	return "//platformid.api.mondoo.app/runtime/opcua/" + p.id, nil
}

func (p *Provider) PlatformInfo() (*platform.Platform, error) {
	return &platform.Platform{
		Name:    "opcua",
		Title:   "OPC UA",
		Runtime: p.Runtime(),
		Kind:    p.Kind(),
		Family:  []string{},
	}, nil
}
