// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/operationalinsights/armoperationalinsights/v3"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/securityinsights/armsecurityinsights"
	"github.com/rs/zerolog/log"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/azure/connection"
	"go.mondoo.com/mql/v13/types"
)

func initAzureSubscriptionSentinelService(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 0 {
		return args, nil, nil
	}

	conn, ok := runtime.Connection.(*connection.AzureConnection)
	if !ok {
		return nil, nil, errors.New("invalid connection provided, it is not an Azure connection")
	}
	args["subscriptionId"] = llx.StringData(conn.SubId())

	return args, nil, nil
}

func (a *mqlAzureSubscriptionSentinelService) id() (string, error) {
	return "azure.subscription.sentinelService/" + a.SubscriptionId.Data, nil
}

func (a *mqlAzureSubscriptionSentinelServiceWorkspace) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionSentinelServiceAlertRule) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionSentinelService) workspaces() ([]any, error) {
	conn, ok := a.MqlRuntime.Connection.(*connection.AzureConnection)
	if !ok {
		return nil, errors.New("invalid connection provided, it is not an Azure connection")
	}

	subId := a.SubscriptionId.Data
	ctx := context.Background()

	wsClient, err := armoperationalinsights.NewWorkspacesClient(subId, conn.Token(), &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}
	onboardClient, err := armsecurityinsights.NewSentinelOnboardingStatesClient(subId, conn.Token(), &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	res := []any{}
	wsPager := wsClient.NewListPager(nil)
	for wsPager.More() {
		page, err := wsPager.NextPage(ctx)
		if err != nil {
			var respErr *azcore.ResponseError
			if errors.As(err, &respErr) && respErr.StatusCode == http.StatusForbidden {
				log.Warn().Err(err).Msg("could not list Log Analytics workspaces due to access denied")
				return res, nil
			}
			return nil, err
		}
		for _, ws := range page.Value {
			if ws == nil || ws.ID == nil {
				continue
			}
			parsed, err := ParseResourceID(*ws.ID)
			if err != nil {
				continue
			}
			workspaceName := ""
			if ws.Name != nil {
				workspaceName = *ws.Name
			}
			onboarded, err := sentinelOnboarded(ctx, onboardClient, parsed.ResourceGroup, workspaceName)
			if err != nil {
				return nil, err
			}
			if !onboarded {
				continue
			}
			mqlWs, err := CreateResource(a.MqlRuntime, "azure.subscription.sentinelService.workspace", map[string]*llx.RawData{
				"id":             llx.StringDataPtr(ws.ID),
				"name":           llx.StringData(workspaceName),
				"resourceGroup":  llx.StringData(parsed.ResourceGroup),
				"subscriptionId": llx.StringData(parsed.SubscriptionID),
			})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlWs)
		}
	}
	return res, nil
}

// sentinelOnboarded reports whether Microsoft Sentinel is onboarded on
// the given Log Analytics workspace. Returns false (without erroring)
// on 404, which is the response when the SecurityInsights solution has
// never been onboarded on the workspace.
func sentinelOnboarded(ctx context.Context, client *armsecurityinsights.SentinelOnboardingStatesClient, resourceGroup, workspaceName string) (bool, error) {
	resp, err := client.List(ctx, resourceGroup, workspaceName, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) {
			if respErr.StatusCode == http.StatusNotFound {
				return false, nil
			}
			if respErr.StatusCode == http.StatusForbidden {
				log.Debug().Str("workspace", workspaceName).Err(err).Msg("could not read Sentinel onboarding state due to access denied")
				return false, nil
			}
		}
		return false, err
	}
	return len(resp.Value) > 0, nil
}

func (a *mqlAzureSubscriptionSentinelServiceWorkspace) workspace() (*mqlAzureSubscriptionMonitorServiceWorkspace, error) {
	mqlWs, err := NewResource(a.MqlRuntime, "azure.subscription.monitorService.workspace",
		map[string]*llx.RawData{"id": llx.StringData(a.Id.Data)})
	if err != nil {
		return nil, err
	}
	return mqlWs.(*mqlAzureSubscriptionMonitorServiceWorkspace), nil
}

func (a *mqlAzureSubscriptionSentinelServiceWorkspace) alertRules() ([]any, error) {
	conn, ok := a.MqlRuntime.Connection.(*connection.AzureConnection)
	if !ok {
		return nil, errors.New("invalid connection provided, it is not an Azure connection")
	}

	client, err := armsecurityinsights.NewAlertRulesClient(a.SubscriptionId.Data, conn.Token(), &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	pager := client.NewListPager(a.ResourceGroup.Data, a.Name.Data, nil)
	res := []any{}
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			var respErr *azcore.ResponseError
			if errors.As(err, &respErr) && respErr.StatusCode == http.StatusForbidden {
				log.Warn().Err(err).Msg("could not list Sentinel alert rules due to access denied")
				return res, nil
			}
			return nil, err
		}
		for _, raw := range page.Value {
			if raw == nil {
				continue
			}
			mqlRule, err := sentinelAlertRuleToMql(a.MqlRuntime, raw)
			if err != nil {
				return nil, err
			}
			res = append(res, mqlRule)
		}
	}
	return res, nil
}

func sentinelAlertRuleToMql(runtime *plugin.Runtime, raw armsecurityinsights.AlertRuleClassification) (*mqlAzureSubscriptionSentinelServiceAlertRule, error) {
	base := raw.GetAlertRule()
	var kind, displayName, description, severity string
	var enabled bool
	tactics := []any{}

	if base.Kind != nil {
		kind = string(*base.Kind)
	}

	var propsDict any
	switch r := raw.(type) {
	case *armsecurityinsights.ScheduledAlertRule:
		if p := r.Properties; p != nil {
			if p.DisplayName != nil {
				displayName = *p.DisplayName
			}
			if p.Enabled != nil {
				enabled = *p.Enabled
			}
			if p.Description != nil {
				description = *p.Description
			}
			if p.Severity != nil {
				severity = string(*p.Severity)
			}
			for _, t := range p.Tactics {
				if t != nil {
					tactics = append(tactics, string(*t))
				}
			}
			d, err := convert.JsonToDict(p)
			if err != nil {
				return nil, err
			}
			propsDict = d
		}
	case *armsecurityinsights.FusionAlertRule:
		if p := r.Properties; p != nil {
			if p.DisplayName != nil {
				displayName = *p.DisplayName
			}
			if p.Enabled != nil {
				enabled = *p.Enabled
			}
			if p.Description != nil {
				description = *p.Description
			}
			if p.Severity != nil {
				severity = string(*p.Severity)
			}
			for _, t := range p.Tactics {
				if t != nil {
					tactics = append(tactics, string(*t))
				}
			}
			d, err := convert.JsonToDict(p)
			if err != nil {
				return nil, err
			}
			propsDict = d
		}
	case *armsecurityinsights.MicrosoftSecurityIncidentCreationAlertRule:
		if p := r.Properties; p != nil {
			if p.DisplayName != nil {
				displayName = *p.DisplayName
			}
			if p.Enabled != nil {
				enabled = *p.Enabled
			}
			if p.Description != nil {
				description = *p.Description
			}
			d, err := convert.JsonToDict(p)
			if err != nil {
				return nil, err
			}
			propsDict = d
		}
	default:
		// Kinds without a dedicated concrete type in the SDK
		// (e.g. NRT, MLBehaviorAnalytics, ThreatIntelligence)
		// deserialize as *armsecurityinsights.AlertRule, which
		// carries only kind/id/name. Marshal the raw payload so
		// kind-specific fields remain queryable via `properties`.
		d, err := convert.JsonToDict(raw)
		if err != nil {
			return nil, err
		}
		propsDict = d
	}

	res, err := CreateResource(runtime, "azure.subscription.sentinelService.alertRule", map[string]*llx.RawData{
		"id":          llx.StringDataPtr(base.ID),
		"name":        llx.StringDataPtr(base.Name),
		"kind":        llx.StringData(kind),
		"enabled":     llx.BoolData(enabled),
		"displayName": llx.StringData(displayName),
		"severity":    llx.StringData(severity),
		"tactics":     llx.ArrayData(tactics, types.String),
		"description": llx.StringData(description),
		"properties":  llx.DictData(propsDict),
	})
	if err != nil {
		return nil, err
	}

	sysData, err := convert.JsonToDict(base.SystemData)
	if err != nil {
		return nil, err
	}
	res.(*mqlAzureSubscriptionSentinelServiceAlertRule).cacheSystemData = sysData

	return res.(*mqlAzureSubscriptionSentinelServiceAlertRule), nil
}

type mqlAzureSubscriptionSentinelServiceAlertRuleInternal struct {
	cacheSystemData any
}

func (a *mqlAzureSubscriptionSentinelServiceAlertRule) systemMetadata() (*mqlAzureSubscriptionSystemData, error) {
	return systemMetadataFromRaw(a.MqlRuntime, a.Id.Data, a.cacheSystemData, &a.SystemMetadata)
}

func (a *mqlAzureSubscriptionSentinelServiceWorkspace) dataConnectors() ([]any, error) {
	conn, ok := a.MqlRuntime.Connection.(*connection.AzureConnection)
	if !ok {
		return nil, errors.New("invalid connection provided, it is not an Azure connection")
	}

	client, err := armsecurityinsights.NewDataConnectorsClient(a.SubscriptionId.Data, conn.Token(), &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	pager := client.NewListPager(a.ResourceGroup.Data, a.Name.Data, nil)
	res := []any{}
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			var respErr *azcore.ResponseError
			if errors.As(err, &respErr) && respErr.StatusCode == http.StatusForbidden {
				log.Warn().Err(err).Msg("could not list Sentinel data connectors due to access denied")
				return res, nil
			}
			return nil, err
		}
		for _, raw := range page.Value {
			if raw == nil {
				continue
			}
			base := raw.GetDataConnector()
			entry := map[string]any{}
			if base.Name != nil {
				entry["name"] = *base.Name
			}
			if base.Kind != nil {
				entry["kind"] = string(*base.Kind)
			}
			if base.ID != nil {
				entry["id"] = *base.ID
			}
			propsDict, err := convert.JsonToDict(raw)
			if err != nil {
				return nil, err
			}
			entry["properties"] = propsDict
			res = append(res, entry)
		}
	}
	return res, nil
}
