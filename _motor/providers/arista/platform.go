// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package arista

func (t *Provider) Identifier() (string, error) {
	v, err := t.GetVersion()
	if err != nil {
		return "", err
	}

	return "//platformid.api.mondoo.app/runtime/arista/serial/" + v.SerialNumber + "/systemmac/" + v.SystemMacAddress, nil
}
