// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package vcd

import (
	"fmt"
	"strconv"

	"go.mondoo.com/cnquery/v9/motor/platform"
)

func (p *Provider) PlatformInfo() (*platform.Platform, error) {
	vcdVersion, err := p.client.Client.GetVcdFullVersion()
	if err != nil {
		return nil, err
	}

	digits := vcdVersion.Version.Segments()

	return &platform.Platform{
		Name:    "vcd",
		Title:   "VMware Cloud Director " + p.host,
		Version: fmt.Sprintf("%d.%d.%d", digits[0], digits[1], digits[2]),
		Build:   strconv.Itoa(digits[3]),
		Runtime: p.Runtime(),
		Kind:    p.Kind(),
		Labels: map[string]string{
			"vcd.vmware.com/api-version": p.client.Client.APIVersion,
		},
	}, nil
}

func (p *Provider) Identifier() (string, error) {
	return "//platformid.api.mondoo.app/runtime/vcd/host/" + p.host, nil
}
