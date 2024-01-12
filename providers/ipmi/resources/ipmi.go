// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v10/providers/ipmi/connection"
)

func (r *mqlIpmi) id() (string, error) {
	return "ipmi", nil
}

func (r *mqlIpmi) guid() (string, error) {
	conn := r.MqlRuntime.Connection.(*connection.IpmiConnection)
	client := conn.Client()

	resp, err := client.DeviceGUID()
	if err != nil {
		return "", err
	}
	return resp.GUID, nil
}

func (r *mqlIpmi) deviceID() (map[string]interface{}, error) {
	conn := r.MqlRuntime.Connection.(*connection.IpmiConnection)
	client := conn.Client()

	resp, err := client.DeviceID()
	if err != nil {
		return nil, err
	}

	return convert.JsonToDict(resp)
}

func (r *mqlIpmiChassis) id() (string, error) {
	return "ipmi.chassis", nil
}

func (r *mqlIpmiChassis) status() (map[string]interface{}, error) {
	conn := r.MqlRuntime.Connection.(*connection.IpmiConnection)
	client := conn.Client()

	resp, err := client.ChassisStatus()
	if err != nil {
		return nil, err
	}

	return convert.JsonToDict(resp)
}

func (r *mqlIpmiChassis) systemBootOptions() (map[string]interface{}, error) {
	conn := r.MqlRuntime.Connection.(*connection.IpmiConnection)
	client := conn.Client()

	resp, err := client.ChassisSystemBootOptions()
	if err != nil {
		return nil, err
	}

	return convert.JsonToDict(resp)
}
