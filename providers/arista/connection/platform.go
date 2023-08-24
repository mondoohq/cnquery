// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package connection

func (c *AristaConnection) Identifier() (string, error) {
	v, err := c.GetVersion()
	if err != nil {
		return "", err
	}

	return "//platformid.api.mondoo.app/runtime/arista/serial/" + v.SerialNumber + "/systemmac/" + v.SystemMacAddress, nil
}
