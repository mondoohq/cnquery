// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	armdeployments "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armdeployments/v2"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/azure/connection"
	"go.mondoo.com/mql/v13/types"
)

type mqlAzureSubscriptionDeploymentInternal struct {
	cacheSystemData any
}

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
	// Most fields live under Properties, which can be nil. Collect them into
	// locals (zero-valued when absent) so the resource map is built once below.
	var (
		provisioningState string
		mode              string
		timestamp         *time.Time
		duration          *string
		correlationID     *string
		templateHash      *string
		templateLink      *string
		parametersLink    *string
		parameters        any
		outputs           any
		deploymentErr     any
	)
	providers := []any{}
	outputResources := []any{}

	if props := deployment.Properties; props != nil {
		if props.ProvisioningState != nil {
			provisioningState = string(*props.ProvisioningState)
		}
		if props.Mode != nil {
			mode = string(*props.Mode)
		}
		timestamp = props.Timestamp
		duration = props.Duration
		correlationID = props.CorrelationID
		templateHash = props.TemplateHash
		if props.TemplateLink != nil {
			templateLink = props.TemplateLink.URI
		}
		if props.ParametersLink != nil {
			parametersLink = props.ParametersLink.URI
		}

		var err error
		if parameters, err = convert.JsonToDict(props.Parameters); err != nil {
			return nil, err
		}
		if outputs, err = convert.JsonToDict(props.Outputs); err != nil {
			return nil, err
		}
		if providers, err = convert.JsonToDictSlice(props.Providers); err != nil {
			return nil, err
		}
		for _, r := range props.OutputResources {
			if r != nil && r.ID != nil {
				outputResources = append(outputResources, *r.ID)
			}
		}
		if props.Error != nil {
			if deploymentErr, err = convert.JsonToDict(props.Error); err != nil {
				return nil, err
			}
		}
	}

	res, err := CreateResource(runtime, "azure.subscription.deployment", map[string]*llx.RawData{
		"id":                llx.StringDataPtr(deployment.ID),
		"name":              llx.StringDataPtr(deployment.Name),
		"type":              llx.StringDataPtr(deployment.Type),
		"location":          llx.StringDataPtr(deployment.Location),
		"tags":              llx.MapData(convert.PtrMapStrToInterface(deployment.Tags), types.String),
		"provisioningState": llx.StringData(provisioningState),
		"timestamp":         llx.TimeDataPtr(timestamp),
		"duration":          llx.StringDataPtr(duration),
		"correlationId":     llx.StringDataPtr(correlationID),
		"mode":              llx.StringData(mode),
		"templateHash":      llx.StringDataPtr(templateHash),
		"templateLink":      llx.StringDataPtr(templateLink),
		"parametersLink":    llx.StringDataPtr(parametersLink),
		"parameters":        llx.DictData(parameters),
		"outputs":           llx.DictData(outputs),
		"providers":         llx.ArrayData(providers, types.Dict),
		"outputResources":   llx.ArrayData(outputResources, types.String),
		"error":             llx.DictData(deploymentErr),
	})
	if err != nil {
		return nil, err
	}
	mqlDeployment := res.(*mqlAzureSubscriptionDeployment)
	sysData, err := convert.JsonToDict(deployment.SystemData)
	if err != nil {
		return nil, err
	}
	mqlDeployment.cacheSystemData = sysData
	return mqlDeployment, nil
}
