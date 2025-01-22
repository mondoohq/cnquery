// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"

	"go.mondoo.com/cnquery/v11/providers/tailscale/connection"
)

func (r *mqlTailscale) id() (string, error) {
	// TODO need to set the tailnet
	return "tailscale", nil
}

func (t *mqlTailscale) devices() ([]any, error) {
	conn := t.MqlRuntime.Connection.(*connection.TailscaleConnection)
	devices, err := conn.Client().Devices().List(context.Background())
	if err != nil {
		return nil, err
	}

	var resources []interface{}
	for _, device := range devices {
		resource, err := createTailscaleDeviceResource(t.MqlRuntime, &device)
		if err != nil {
			return nil, err
		}
		resources = append(resources, resource)
	}

	return resources, nil
}
