// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"go.mondoo.com/cnquery/v10/llx"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v10/providers/azure/connection"
	"go.mondoo.com/cnquery/v10/types"

	azureres "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
)

func (a *mqlAzureSubscription) resources() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)

	ctx := context.Background()
	token := conn.Token()
	subId := a.SubscriptionId.Data

	client, err := azureres.NewClient(subId, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}
	expand := "createdTime,changedTime,provisioningState"
	pager := client.NewListPager(&azureres.ClientListOptions{Expand: &expand})
	res := []interface{}{}
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, resource := range page.Value {
			// NOTE: properties not not properly filled, therefore you would need to ask each individual resource:
			// https://docs.microsoft.com/en-us/rest/api/resources/resources/getbyid
			// In order to make it happen you need to support each individual type and their api version. Therefore
			// we should not support that via the resource api but instead make sure those properties are properly
			// exposed by the typed resources
			sku, err := convert.JsonToDict(resource.SKU)
			if err != nil {
				return nil, err
			}

			plan, err := convert.JsonToDict(resource.Plan)
			if err != nil {
				return nil, err
			}

			identity, err := convert.JsonToDict(resource.Identity)
			if err != nil {
				return nil, err
			}
			mqlAzure, err := CreateResource(a.MqlRuntime, "azure.subscription.resource",
				map[string]*llx.RawData{
					"id":                llx.StringDataPtr(resource.ID),
					"name":              llx.StringDataPtr(resource.Name),
					"kind":              llx.StringDataPtr(resource.Kind),
					"location":          llx.StringDataPtr(resource.Location),
					"tags":              llx.MapData(convert.PtrMapStrToInterface(resource.Tags), types.String),
					"type":              llx.StringDataPtr(resource.Type),
					"managedBy":         llx.StringDataPtr(resource.ManagedBy),
					"sku":               llx.DictData(sku),
					"plan":              llx.DictData(plan),
					"identity":          llx.DictData(identity),
					"provisioningState": llx.StringDataPtr(resource.ProvisioningState),
					"createdTime":       llx.TimeDataPtr(resource.CreatedTime),
					"changedTime":       llx.TimeDataPtr(resource.ChangedTime),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlAzure)
		}
	}
	return res, nil
}

func (a *mqlAzureSubscriptionResource) id() (string, error) {
	return a.Id.Data, nil
}
