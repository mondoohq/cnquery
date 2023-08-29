// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/providers/azure/connection"
	"go.mondoo.com/cnquery/types"

	azureres "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
)

func (a *mqlAzureSubscription) resources() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)

	ctx := context.Background()
	token := conn.Token()
	subId := a.SubscriptionId.Data

	client, err := azureres.NewClient(subId, token, &arm.ClientOptions{})
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
					"id":                llx.StringData(convert.ToString(resource.ID)),
					"name":              llx.StringData(convert.ToString(resource.Name)),
					"kind":              llx.StringData(convert.ToString(resource.Kind)),
					"location":          llx.StringData(convert.ToString(resource.Location)),
					"tags":              llx.MapData(convert.PtrMapStrToInterface(resource.Tags), types.String),
					"type":              llx.StringData(convert.ToString(resource.Type)),
					"managedBy":         llx.StringData(convert.ToString(resource.ManagedBy)),
					"sku":               llx.DictData(sku),
					"plan":              llx.DictData(plan),
					"identity":          llx.DictData(identity),
					"provisioningState": llx.StringData(convert.ToString(resource.ProvisioningState)),
					"createdTime":       llx.TimeData(*resource.CreatedTime),
					"changedTime":       llx.TimeData(*resource.ChangedTime),
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
