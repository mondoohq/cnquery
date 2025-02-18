// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"

	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v11/providers/tailscale/connection"
)

func (r *mqlTailscale) id() (string, error) {
	return r.Tailnet.Data, nil
}

func initTailscale(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	conn := runtime.Connection.(*connection.TailscaleConnection)
	tailnet, set := connection.GetTailnet(conn.Conf)
	if !set {
		// When no tailnet was specified, we will be using the default tailnet of the
		// authentication method being used to make API calls. Tailscale recommend this
		// option for most users. (https://tailscale.com/api)
		//
		// NOTE that today, we cannot make an API call to get the actual tailnet
		tailnet = "default"
	}
	mqlResource, err := CreateResource(runtime, "tailscale",
		map[string]*llx.RawData{
			"tailnet": llx.StringData(tailnet),
		})
	if err != nil {
		return args, nil, err
	}
	return args, mqlResource, nil
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

func (t *mqlTailscale) users() ([]any, error) {
	conn := t.MqlRuntime.Connection.(*connection.TailscaleConnection)
	// TODO we can do filter here for user type and role
	users, err := conn.Client().Users().List(context.Background(), nil, nil)
	if err != nil {
		return nil, err
	}

	var resources []interface{}
	for _, user := range users {
		resource, err := createTailscaleUserResource(t.MqlRuntime, &user)
		if err != nil {
			return nil, err
		}
		resources = append(resources, resource)
	}

	return resources, nil
}

func (t *mqlTailscale) nameservers() ([]any, error) {
	conn := t.MqlRuntime.Connection.(*connection.TailscaleConnection)
	nameservers, err := conn.Client().DNS().Nameservers(context.Background())
	return convert.SliceAnyToInterface(nameservers), err
}
