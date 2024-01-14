// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"

	"go.mondoo.com/cnquery/v10/llx"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v10/providers/okta/connection"
	"go.mondoo.com/cnquery/v10/providers/okta/resources/sdk"
	"go.mondoo.com/cnquery/v10/types"
	"go.mondoo.com/ranger-rpc"
)

func (o *mqlOktaOrganization) id() (string, error) {
	return "okta/organization/" + o.Id.Data, o.Id.Error
}

func initOktaOrganization(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}

	conn := runtime.Connection.(*connection.OktaConnection)

	ctx := context.Background()
	client := conn.Client()
	settings, _, err := client.OrgSetting.GetOrgSettings(ctx)
	if err != nil {
		return nil, nil, err
	}

	args["id"] = llx.StringData(settings.Id)
	args["companyName"] = llx.StringData(settings.CompanyName)
	args["status"] = llx.StringData(settings.Status)
	args["subdomain"] = llx.StringData(settings.Subdomain)
	args["address1"] = llx.StringData(settings.Address1)
	args["address2"] = llx.StringData(settings.Address2)
	args["city"] = llx.StringData(settings.City)
	args["state"] = llx.StringData(settings.State)
	args["phoneNumber"] = llx.StringData(settings.PhoneNumber)
	args["postalCode"] = llx.StringData(settings.PostalCode)
	args["country"] = llx.StringData(settings.Country)
	args["supportPhoneNumber"] = llx.StringData(settings.SupportPhoneNumber)
	args["website"] = llx.StringData(settings.Website)
	args["endUserSupportHelpURL"] = llx.StringData(settings.EndUserSupportHelpURL)
	args["created"] = llx.TimeDataPtr(settings.Created)
	args["lastUpdated"] = llx.TimeDataPtr(settings.LastUpdated)
	args["expiresAt"] = llx.TimeDataPtr(settings.ExpiresAt)

	return args, nil, nil
}

func (o *mqlOktaOrganization) optOutCommunicationEmails() (bool, error) {
	conn := o.MqlRuntime.Connection.(*connection.OktaConnection)
	client := conn.Client()

	ctx := context.Background()
	settings, _, err := client.OrgSetting.GetOktaCommunicationSettings(ctx)
	if err != nil {
		return false, err
	}

	if settings.OptOutEmailUsers == nil {
		return *settings.OptOutEmailUsers, nil
	}

	return false, nil
}

func (o *mqlOktaOrganization) billingContact() (*mqlOktaUser, error) {
	conn := o.MqlRuntime.Connection.(*connection.OktaConnection)
	client := conn.Client()

	ctx := context.Background()
	contactUser, _, err := client.OrgSetting.GetOrgContactUser(ctx, "BILLING")
	if err != nil {
		return nil, err
	}
	uid := contactUser.UserId

	usr, _, err := client.User.GetUser(
		ctx,
		uid,
	)
	if err != nil {
		return nil, err
	}

	return newMqlOktaUser(o.MqlRuntime, usr)
}

func (o *mqlOktaOrganization) technicalContact() (*mqlOktaUser, error) {
	conn := o.MqlRuntime.Connection.(*connection.OktaConnection)
	client := conn.Client()

	ctx := context.Background()
	contactUser, _, err := client.OrgSetting.GetOrgContactUser(ctx, "TECHNICAL")
	if err != nil {
		return nil, err
	}

	uid := contactUser.UserId

	usr, _, err := client.User.GetUser(
		ctx,
		uid,
	)
	if err != nil {
		return nil, err
	}

	return newMqlOktaUser(o.MqlRuntime, usr)
}

func (o *mqlOktaOrganization) securityNotificationEmails() (interface{}, error) {
	conn := o.MqlRuntime.Connection.(*connection.OktaConnection)
	client := conn.Client()

	ctx := context.Background()
	apiSupplement := &sdk.ApiExtension{
		RequestExecutor: client.CloneRequestExecutor(),
	}

	emails, err := apiSupplement.GetSecurityNotificationEmails(
		ctx,
		conn.OrganizationID(),
		conn.Token(),
		ranger.DefaultHttpClient(),
	)
	if err != nil {
		return nil, err
	}

	return convert.JsonToDict(emails)
}

// threatInsightSettings returns the Threat Insight settings for the organization
func (o *mqlOktaOrganization) threatInsightSettings() (*mqlOktaThreatsConfiguration, error) {
	conn := o.MqlRuntime.Connection.(*connection.OktaConnection)
	client := conn.Client()

	ctx := context.Background()
	config, _, err := client.ThreatInsightConfiguration.GetCurrentConfiguration(ctx)
	if err != nil {
		return nil, err
	}

	excludesZones := []interface{}{}
	for i := range config.ExcludeZones {
		zone, _, err := client.NetworkZone.GetNetworkZone(ctx, config.ExcludeZones[i])
		if err != nil {
			return nil, err
		}
		mqlZone, err := newMqlOktaNetworkZone(o.MqlRuntime, zone)
		if err != nil {
			return nil, err
		}
		excludesZones = append(excludesZones, mqlZone)
	}

	r, err := CreateResource(o.MqlRuntime, "okta.threatsConfiguration", map[string]*llx.RawData{
		"action":       llx.StringData(config.Action),
		"excludeZones": llx.ArrayData(excludesZones, types.Resource("okta.network")),
		"created":      llx.TimeDataPtr(config.Created),
		"lastUpdated":  llx.TimeDataPtr(config.LastUpdated),
	})
	if err != nil {
		return nil, err
	}
	return r.(*mqlOktaThreatsConfiguration), nil
}

func (o *mqlOktaThreatsConfiguration) id() (string, error) {
	conn := o.MqlRuntime.Connection.(*connection.OktaConnection)
	return "okta.threatsConfiguration/" + conn.OrganizationID(), nil
}
