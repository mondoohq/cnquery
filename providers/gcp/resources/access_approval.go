// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"fmt"

	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v11/providers/gcp/connection"
	"go.mondoo.com/cnquery/v11/types"

	accessapproval "cloud.google.com/go/accessapproval/apiv1"
	accessapprovalpb "cloud.google.com/go/accessapproval/apiv1/accessapprovalpb"
	"google.golang.org/api/option"
)

func (g *mqlGcpOrganization) accessApprovalSettings() (*mqlGcpAccessApprovalSettings, error) {
	if g.Id.Error != nil {
		return nil, g.Id.Error
	}
	id := g.Id.Data

	return accessApprovalSettings(g.MqlRuntime, fmt.Sprintf("organizations/%s/accessApprovalSettings", id))
}

func (g *mqlGcpProject) accessApprovalSettings() (*mqlGcpAccessApprovalSettings, error) {
	if g.Id.Error != nil {
		return nil, g.Id.Error
	}
	id := g.Id.Data

	serviceEnabled, err := g.isServiceEnabled(service_accessapproval)
	if err != nil {
		return nil, err
	}
	if !serviceEnabled {
		g.AccessApprovalSettings.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}

	return accessApprovalSettings(g.MqlRuntime, fmt.Sprintf("projects/%s/accessApprovalSettings", id))
}

func (g *mqlGcpAccessApprovalSettings) id() (string, error) {
	return g.ResourcePath.Data, g.ResourcePath.Error
}

func accessApprovalSettings(runtime *plugin.Runtime, settingsName string) (*mqlGcpAccessApprovalSettings, error) {
	conn := runtime.Connection.(*connection.GcpConnection)
	credentials, err := conn.Credentials(accessapproval.DefaultAuthScopes()...)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	c, err := accessapproval.NewClient(ctx, option.WithCredentials(credentials))
	if err != nil {
		return nil, err
	}
	defer c.Close()

	settings, err := c.GetAccessApprovalSettings(ctx, &accessapprovalpb.GetAccessApprovalSettingsMessage{
		Name: settingsName,
	})
	if err != nil {
		return nil, err
	}

	mqlEnrolledServices := make([]interface{}, 0, len(settings.EnrolledServices))
	for _, s := range settings.EnrolledServices {
		mqlEnrolledServices = append(mqlEnrolledServices, map[string]interface{}{
			"cloudProduct":    s.CloudProduct,
			"enrollmentLevel": s.EnrollmentLevel.String(),
		})
	}

	res, err := CreateResource(runtime, "gcp.accessApprovalSettings", map[string]*llx.RawData{
		"resourcePath":                llx.StringData(settings.Name),
		"notificationEmails":          llx.ArrayData(convert.SliceAnyToInterface(settings.NotificationEmails), types.String),
		"enrolledServices":            llx.ArrayData(mqlEnrolledServices, types.Dict),
		"enrolledAncestor":            llx.BoolData(settings.EnrolledAncestor),
		"activeKeyVersion":            llx.StringData(settings.ActiveKeyVersion),
		"ancestorHasActiveKeyVersion": llx.BoolData(settings.AncestorHasActiveKeyVersion),
		"invalidKeyVersion":           llx.BoolData(settings.InvalidKeyVersion),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlGcpAccessApprovalSettings), nil
}
