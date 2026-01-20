// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"

	"github.com/microsoftgraph/msgraph-beta-sdk-go/models/security"
	"go.mondoo.com/cnquery/v12/llx"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v12/providers/ms365/connection"
	"go.mondoo.com/cnquery/v12/types"
)

func (r *mqlMicrosoftSecurity) informationProtection() (*mqlMicrosoftSecurityInformationProtection, error) {
	resource, err := CreateResource(r.MqlRuntime, ResourceMicrosoftSecurityInformationProtection, map[string]*llx.RawData{})
	if err != nil {
		return nil, err
	}
	return resource.(*mqlMicrosoftSecurityInformationProtection), nil
}

func (r *mqlMicrosoftSecurityInformationProtection) sensitivityLabels() ([]any, error) {
	conn := r.MqlRuntime.Connection.(*connection.Ms365Connection)
	betaClient, err := conn.BetaGraphClient()
	if err != nil {
		return nil, err
	}
	ctx := context.Background()

	resp, err := betaClient.Security().InformationProtection().SensitivityLabels().Get(ctx, nil)
	if err != nil {
		return nil, transformError(err)
	}

	var res []any
	labels := resp.GetValue()
	for i := range labels {
		label := labels[i]
		mqlResource, err := createSensitivityLabelResource(r.MqlRuntime, label)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlResource)
	}
	return res, nil
}

func createSensitivityLabelResource(runtime *plugin.Runtime, label security.SensitivityLabelable) (plugin.Resource, error) {
	if label == nil {
		return nil, nil
	}

	var contentFormats []any
	if formats := label.GetContentFormats(); formats != nil {
		for _, format := range formats {
			contentFormats = append(contentFormats, format)
		}
	}

	var parentResource plugin.Resource
	if parent := label.GetParent(); parent != nil {
		var err error
		parentResource, err = createSensitivityLabelResource(runtime, parent)
		if err != nil {
			return nil, err
		}
	}

	mqlResource, err := CreateResource(runtime, ResourceMicrosoftSecurityInformationProtectionSensitivityLabel,
		map[string]*llx.RawData{
			"__id":           llx.StringDataPtr(label.GetId()),
			"id":             llx.StringDataPtr(label.GetId()),
			"name":           llx.StringDataPtr(label.GetName()),
			"description":    llx.StringDataPtr(label.GetDescription()),
			"toolTip":        llx.StringDataPtr(label.GetTooltip()),
			"color":          llx.StringDataPtr(label.GetColor()),
			"contentFormats": llx.ArrayData(contentFormats, types.String),
			"isAppliable":    llx.BoolDataPtr(label.GetIsAppliable()),
			"hasProtection":  llx.BoolDataPtr(label.GetHasProtection()),
			"isActive":       llx.BoolDataPtr(label.GetIsActive()),
			"sensitivity":    llx.IntDataPtr(label.GetSensitivity()),
			"parent":         llx.ResourceData(parentResource, ResourceMicrosoftSecurityInformationProtectionSensitivityLabel),
		})
	if err != nil {
		return nil, err
	}

	return mqlResource, nil
}

func (r *mqlMicrosoftSecurityInformationProtectionSensitivityLabel) id() (string, error) {
	return r.Id.Data, nil
}
