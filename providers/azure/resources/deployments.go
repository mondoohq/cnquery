// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	armdeployments "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armdeployments"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/azure/connection"
	"go.mondoo.com/mql/v13/types"
)

func (a *mqlAzureSubscriptionDeployment) id() (string, error) {
	return a.Id.Data, nil
}

// deployments lists Azure Resource Manager deployments scoped to the
// subscription itself (subscription-scoped templates). Resource-group-scoped
// deployments are reached through azure.subscription.resourcegroup.deployments.
func (a *mqlAzureSubscription) deployments() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	client, err := armdeployments.NewDeploymentsClient(a.SubscriptionId.Data, conn.Token(), &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	pager := client.NewListAtSubscriptionScopePager(nil)
	res := []any{}
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, deployment := range page.Value {
			mqlDeployment, err := newMqlAzureDeployment(a.MqlRuntime, deployment)
			if err != nil {
				return nil, err
			}
			res = append(res, mqlDeployment)
		}
	}
	return res, nil
}

func (a *mqlAzureSubscriptionResourcegroup) deployments() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	subId, err := extractSubscriptionID(a.Id.Data)
	if err != nil {
		return nil, err
	}
	client, err := armdeployments.NewDeploymentsClient(subId, conn.Token(), &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	pager := client.NewListByResourceGroupPager(a.Name.Data, nil)
	res := []any{}
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, deployment := range page.Value {
			mqlDeployment, err := newMqlAzureDeployment(a.MqlRuntime, deployment)
			if err != nil {
				return nil, err
			}
			res = append(res, mqlDeployment)
		}
	}
	return res, nil
}

func newMqlAzureDeployment(runtime *plugin.Runtime, deployment *armdeployments.DeploymentExtended) (*mqlAzureSubscriptionDeployment, error) {
	args := map[string]*llx.RawData{
		"id":                llx.StringDataPtr(deployment.ID),
		"name":              llx.StringDataPtr(deployment.Name),
		"type":              llx.StringDataPtr(deployment.Type),
		"location":          llx.StringDataPtr(deployment.Location),
		"tags":              llx.MapData(convert.PtrMapStrToInterface(deployment.Tags), types.String),
		"provisioningState": llx.StringData(""),
		"timestamp":         llx.TimeDataPtr(nil),
		"duration":          llx.StringData(""),
		"correlationId":     llx.StringData(""),
		"mode":              llx.StringData(""),
		"templateHash":      llx.StringData(""),
		"templateLink":      llx.StringData(""),
		"parametersLink":    llx.StringData(""),
		"parameters":        llx.DictData(nil),
		"outputs":           llx.DictData(nil),
		"providers":         llx.ArrayData([]any{}, types.Dict),
		"outputResources":   llx.ArrayData([]any{}, types.String),
		"error":             llx.DictData(nil),
	}

	if props := deployment.Properties; props != nil {
		if props.ProvisioningState != nil {
			args["provisioningState"] = llx.StringData(string(*props.ProvisioningState))
		}
		args["timestamp"] = llx.TimeDataPtr(props.Timestamp)
		args["duration"] = llx.StringDataPtr(props.Duration)
		args["correlationId"] = llx.StringDataPtr(props.CorrelationID)
		if props.Mode != nil {
			args["mode"] = llx.StringData(string(*props.Mode))
		}
		args["templateHash"] = llx.StringDataPtr(props.TemplateHash)
		if props.TemplateLink != nil {
			args["templateLink"] = llx.StringDataPtr(props.TemplateLink.URI)
		}
		if props.ParametersLink != nil {
			args["parametersLink"] = llx.StringDataPtr(props.ParametersLink.URI)
		}

		parameters, err := convert.JsonToDict(props.Parameters)
		if err != nil {
			return nil, err
		}
		args["parameters"] = llx.DictData(parameters)

		outputs, err := convert.JsonToDict(props.Outputs)
		if err != nil {
			return nil, err
		}
		args["outputs"] = llx.DictData(outputs)

		providers, err := convert.JsonToDictSlice(props.Providers)
		if err != nil {
			return nil, err
		}
		args["providers"] = llx.ArrayData(providers, types.Dict)

		outputResources := []any{}
		for _, r := range props.OutputResources {
			if r != nil && r.ID != nil {
				outputResources = append(outputResources, *r.ID)
			}
		}
		args["outputResources"] = llx.ArrayData(outputResources, types.String)

		if props.Error != nil {
			deploymentErr, err := convert.JsonToDict(props.Error)
			if err != nil {
				return nil, err
			}
			args["error"] = llx.DictData(deploymentErr)
		}
	}

	res, err := CreateResource(runtime, "azure.subscription.deployment", args)
	if err != nil {
		return nil, err
	}
	return res.(*mqlAzureSubscriptionDeployment), nil
}
