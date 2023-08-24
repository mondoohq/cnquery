// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"github.com/packethost/packngo"
	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/providers/equinix/connection"
	"time"
)

// "2021-03-03T11:13:46Z"
func parseEquinixTime(timestamp string) (time.Time, error) {
	return time.Parse("2006-01-02T15:04:05Z", timestamp)
}

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

	pm, err := convert.JsonToDict(p.PaymentMethod)
	if err != nil {
		return nil, nil, err
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
	args["paymentMethod"] = llx.DictData(pm)
	args["createdAt"] = llx.TimeData(created)
	args["updatedAt"] = llx.TimeData(updated)
	return args, nil, nil
}

func (r *mqlEquinixMetalProject) organization() (*mqlEquinixMetalOrganization, error) {
	conn := r.MqlRuntime.Connection.(*connection.EquinixConnection)
	c := conn.Client()

	// NOTE: if we are going to support multiple projects, we need to change this logic
	project := conn.Project()

	// we need to list the organization to circumvent the get issue
	// if we request the project and try to access the org, it only returns the url
	// its similar to https://github.com/packethost/packngo/issues/245
	var org *packngo.Organization
	orgs, _, err := c.Organizations.List(nil)
	if err != nil {
		return nil, err
	}

	for i := range orgs {
		o := orgs[i]
		if o.URL == project.Organization.URL {
			org = &o
			break
		}
	}

	if org == nil {
		return nil, errors.New("could not retrieve the organization: " + project.Organization.URL)
	}

	created, _ := parseEquinixTime(org.Created)
	updated, _ := parseEquinixTime(org.Updated)
	address, _ := convert.JsonToDict(org.Address)

	res, err := CreateResource(r.MqlRuntime, "equinix.metal.organization", map[string]*llx.RawData{
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

func (r *mqlEquinixMetalProject) users() ([]interface{}, error) {
	conn := r.MqlRuntime.Connection.(*connection.EquinixConnection)
	c := conn.Client()

	// NOTE: if we are going to support multiple projects, we need to change this logic
	project := conn.Project()

	// NOTE: circumvent the API, since project user only includes url of the user
	userMap := map[string]packngo.User{}
	users, _, err := c.Users.List(nil)
	if err != nil {
		return nil, err
	}

	for i := range users {
		user := users[i]
		userMap[user.URL] = user
	}

	// now iterate over the user urls of the project
	res := []interface{}{}
	for i := range project.Users {
		usr := project.Users[i]
		fetchedUserData, ok := userMap[usr.URL]
		if !ok {
			return nil, errors.New("could not retrieve information for user: " + usr.URL)
		}

		created, _ := parseEquinixTime(fetchedUserData.Created)
		updated, _ := parseEquinixTime(fetchedUserData.Updated)

		var twitter, facebook, linkedin string
		if fetchedUserData.SocialAccounts != nil {
			twitter = fetchedUserData.SocialAccounts.Twitter
			linkedin = fetchedUserData.SocialAccounts.LinkedIn
			// TODO: let's update the used fields here, I'm not sure which ones are needed (dom)
		}

		mqlEquinixSshKey, err := CreateResource(r.MqlRuntime, "equinix.metal.user", map[string]*llx.RawData{
			"url":           llx.StringData(fetchedUserData.URL),
			"id":            llx.StringData(fetchedUserData.ID),
			"firstName":     llx.StringData(fetchedUserData.FirstName),
			"lastName":      llx.StringData(fetchedUserData.LastName),
			"fullName":      llx.StringData(fetchedUserData.FullName),
			"email":         llx.StringData(fetchedUserData.Email),
			"phoneNumber":   llx.StringData(fetchedUserData.PhoneNumber),
			"twitter":       llx.StringData(twitter),
			"facebook":      llx.StringData(facebook),
			"linkedin":      llx.StringData(linkedin),
			"timezone":      llx.StringData(fetchedUserData.TimeZone),
			"twoFactorAuth": llx.StringData(fetchedUserData.TwoFactorAuth),
			"avatarUrl":     llx.StringData(fetchedUserData.AvatarURL),
			"createdAt":     llx.TimeData(created),
			"updatedAt":     llx.TimeData(updated),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlEquinixSshKey)
	}

	return res, nil
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

func (r *mqlEquinixMetalOrganization) id() (string, error) {
	return r.Url.Data, r.Url.Error
}

func (r *mqlEquinixMetalUser) id() (string, error) {
	return r.Url.Data, r.Url.Error
}

func (r *mqlEquinixMetalSshkey) id() (string, error) {
	return r.Url.Data, r.Url.Error
}

func (r *mqlEquinixMetalDevice) id() (string, error) {
	return r.Url.Data, r.Url.Error
}
