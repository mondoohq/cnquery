// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"

	"go.mondoo.com/cnquery/v10/llx"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v10/providers/equinix/connection"
)

func (r *mqlEquinixMetalProject) id() (string, error) {
	return r.Url.Data, r.Url.Error
}

func initEquinixMetalProject(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}

	// fetch the default project from connection
	conn := runtime.Connection.(*connection.EquinixConnection)
	p := conn.Project()
	if p == nil {
		return nil, nil, errors.New("could not retrieve project information from provider")
	}

	created, err := parseEquinixTime(p.Created)
	if err != nil {
		return nil, nil, err
	}
	updated, err := parseEquinixTime(p.Updated)
	if err != nil {
		return nil, nil, err
	}

	args["id"] = llx.StringData(p.ID)
	args["name"] = llx.StringData(p.Name)
	args["url"] = llx.StringData(p.URL)
	args["createdAt"] = llx.TimeData(created)
	args["updatedAt"] = llx.TimeData(updated)
	return args, nil, nil
}

func (r *mqlEquinixMetalProject) organization() (*mqlEquinixMetalOrganization, error) {
	conn := r.MqlRuntime.Connection.(*connection.EquinixConnection)
	org := conn.Organization()

	if org == nil {
		return nil, errors.New("could not retrieve the organization")
	}

	return newMqlOrganization(r.MqlRuntime, org)
}

func (r *mqlEquinixMetalProject) sshKeys() ([]interface{}, error) {
	conn := r.MqlRuntime.Connection.(*connection.EquinixConnection)
	c := conn.Client()

	// NOTE: if we are going to support multiple projects, we need to change this logic
	project := conn.Project()

	keys, _, err := c.SSHKeys.ProjectList(project.ID)
	if err != nil {
		return nil, err
	}

	res := []interface{}{}
	for i := range keys {
		key := keys[i]

		created, _ := parseEquinixTime(key.Created)
		updated, _ := parseEquinixTime(key.Updated)

		mqlEquinixSshKey, err := CreateResource(r.MqlRuntime, "equinix.metal.sshkey", map[string]*llx.RawData{
			"url":         llx.StringData(key.URL),
			"id":          llx.StringData(key.ID),
			"label":       llx.StringData(key.Label),
			"key":         llx.StringData(key.Key),
			"fingerPrint": llx.StringData(key.FingerPrint),
			"createdAt":   llx.TimeData(created),
			"updatedAt":   llx.TimeData(updated),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlEquinixSshKey)
	}

	return res, nil
}

func (r *mqlEquinixMetalProject) devices() ([]interface{}, error) {
	conn := r.MqlRuntime.Connection.(*connection.EquinixConnection)
	c := conn.Client()

	// NOTE: if we are going to support multiple projects, we need to change this logic
	project := conn.Project()

	devices, _, err := c.Devices.List(project.ID, nil)
	if err != nil {
		return nil, err
	}

	res := []interface{}{}
	for i := range devices {
		device := devices[i]

		created, _ := parseEquinixTime(device.Created)
		updated, _ := parseEquinixTime(device.Updated)
		os, _ := convert.JsonToDict(device.OS)

		mqlEquinixDevice, err := CreateResource(r.MqlRuntime, "equinix.metal.device", map[string]*llx.RawData{
			"url":          llx.StringData(device.Href),
			"id":           llx.StringData(device.ID),
			"shortID":      llx.StringData(device.ShortID),
			"hostname":     llx.StringData(device.Hostname),
			"description":  llx.StringDataPtr(device.Description),
			"state":        llx.StringData(device.State),
			"locked":       llx.BoolData(device.Locked),
			"billingCycle": llx.StringData(device.BillingCycle),
			"spotInstance": llx.BoolData(device.SpotInstance),
			"os":           llx.DictData(os),
			"createdAt":    llx.TimeData(created),
			"updatedAt":    llx.TimeData(updated),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlEquinixDevice)
	}

	return res, nil
}

func (r *mqlEquinixMetalSshkey) id() (string, error) {
	return r.Url.Data, r.Url.Error
}

func (r *mqlEquinixMetalDevice) id() (string, error) {
	return r.Url.Data, r.Url.Error
}
