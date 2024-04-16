// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"

	"github.com/packethost/packngo"
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v11/providers/equinix/connection"
)

func initEquinixMetalOrganization(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}

	// fetch the default project from connection
	conn := runtime.Connection.(*connection.EquinixConnection)
	org := conn.Organization()
	if org == nil {
		return nil, nil, errors.New("could not retrieve organization information from provider")
	}

	created, _ := parseEquinixTime(org.Created)
	updated, _ := parseEquinixTime(org.Updated)
	address, _ := convert.JsonToDict(org.Address)

	args["url"] = llx.StringData(org.URL)
	args["id"] = llx.StringData(org.ID)
	args["name"] = llx.StringData(org.Name)
	args["description"] = llx.StringData(org.Description)
	args["website"] = llx.StringData(org.Website)
	args["twitter"] = llx.StringData(org.Twitter)
	args["address"] = llx.DictData(address)
	args["taxId"] = llx.StringData(org.TaxID)
	args["mainPhone"] = llx.StringData(org.MainPhone)
	args["billingPhone"] = llx.StringData(org.BillingPhone)
	args["creditAmount"] = llx.FloatData(org.CreditAmount)
	args["createdAt"] = llx.TimeData(created)
	args["updatedAt"] = llx.TimeData(updated)
	return args, nil, nil
}

func newMqlOrganization(runtime *plugin.Runtime, org *packngo.Organization) (*mqlEquinixMetalOrganization, error) {
	created, _ := parseEquinixTime(org.Created)
	updated, _ := parseEquinixTime(org.Updated)
	address, _ := convert.JsonToDict(org.Address)

	res, err := CreateResource(runtime, "equinix.metal.organization", map[string]*llx.RawData{
		"url":          llx.StringData(org.URL),
		"id":           llx.StringData(org.ID),
		"name":         llx.StringData(org.Name),
		"description":  llx.StringData(org.Description),
		"website":      llx.StringData(org.Website),
		"twitter":      llx.StringData(org.Twitter),
		"address":      llx.DictData(address),
		"taxId":        llx.StringData(org.TaxID),
		"mainPhone":    llx.StringData(org.MainPhone),
		"billingPhone": llx.StringData(org.BillingPhone),
		"creditAmount": llx.FloatData(org.CreditAmount),
		"createdAt":    llx.TimeData(created),
		"updatedAt":    llx.TimeData(updated),
	})
	if err != nil {
		return nil, err
	}

	return res.(*mqlEquinixMetalOrganization), nil
}

func (r *mqlEquinixMetalOrganization) id() (string, error) {
	return r.Url.Data, r.Url.Error
}

func (r *mqlEquinixMetalOrganization) users() ([]interface{}, error) {
	conn := r.MqlRuntime.Connection.(*connection.EquinixConnection)
	c := conn.Client()

	org := conn.Organization()
	members, _, err := c.Members.List(org.ID, &packngo.ListOptions{
		Includes: []string{"user"},
	})
	if err != nil {
		return nil, err
	}

	// now iterate over the user urls of the project
	res := []interface{}{}
	for i := range members {
		usr := members[i].User

		mqlEquinixUser, err := newMqlUser(r.MqlRuntime, &usr)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlEquinixUser)
	}

	return res, nil
}
