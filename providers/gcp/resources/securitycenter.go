// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"

	securitycenter "cloud.google.com/go/securitycenter/apiv1"
	sccpb "cloud.google.com/go/securitycenter/apiv1/securitycenterpb"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/gcp/connection"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/protobuf/types/known/structpb"
)

func (g *mqlGcpSccSource) id() (string, error) {
	return g.Name.Data, g.Name.Error
}

func (g *mqlGcpSccFinding) id() (string, error) {
	return g.Name.Data, g.Name.Error
}

func (g *mqlGcpSccNotificationConfig) id() (string, error) {
	return g.Name.Data, g.Name.Error
}

func (g *mqlGcpSccMuteConfig) id() (string, error) {
	return g.Name.Data, g.Name.Error
}

func (g *mqlGcpSccBigQueryExport) id() (string, error) {
	return g.Name.Data, g.Name.Error
}

func newSCCClient(conn *connection.GcpConnection) (*securitycenter.Client, error) {
	creds, err := conn.Credentials(securitycenter.DefaultAuthScopes()...)
	if err != nil {
		return nil, err
	}
	return securitycenter.NewClient(context.Background(), option.WithCredentials(creds))
}

// listSCCSources lists Security Command Center sources for a given parent.
// parent should be "organizations/{id}".
func listSCCSources(runtime *plugin.Runtime, conn *connection.GcpConnection, parent string) ([]any, error) {
	client, err := newSCCClient(conn)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	it := client.ListSources(context.Background(), &sccpb.ListSourcesRequest{
		Parent: parent,
	})

	var res []any
	for {
		source, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		mqlSource, err := CreateResource(runtime, "gcp.scc.source", map[string]*llx.RawData{
			"name":          llx.StringData(source.Name),
			"displayName":   llx.StringData(source.DisplayName),
			"description":   llx.StringData(source.Description),
			"canonicalName": llx.StringData(source.CanonicalName),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlSource)
	}

	return res, nil
}

// listSCCFindings lists Security Command Center findings for a given parent.
// parent should be "organizations/{id}" or "projects/{id}".
func listSCCFindings(runtime *plugin.Runtime, conn *connection.GcpConnection, parent string) ([]any, error) {
	client, err := newSCCClient(conn)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	it := client.ListFindings(context.Background(), &sccpb.ListFindingsRequest{
		Parent:   parent + "/sources/-",
		Filter:   `state="ACTIVE"`,
		PageSize: 1000,
	})

	var res []any
	for {
		result, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		f := result.Finding

		sourceProps, err := sourcePropertiesToDict(f.SourceProperties)
		if err != nil {
			return nil, err
		}

		var marks map[string]any
		if f.SecurityMarks != nil {
			marks = make(map[string]any, len(f.SecurityMarks.Marks))
			for k, v := range f.SecurityMarks.Marks {
				marks[k] = v
			}
		}

		mqlFinding, err := CreateResource(runtime, "gcp.scc.finding", map[string]*llx.RawData{
			"name":             llx.StringData(f.Name),
			"parent":           llx.StringData(f.Parent),
			"category":         llx.StringData(f.Category),
			"externalUri":      llx.StringData(f.ExternalUri),
			"sourceProperties": llx.DictData(sourceProps),
			"securityMarks":    llx.DictData(marks),
			"eventTime":        llx.TimeDataPtr(timestampAsTimePtr(f.EventTime)),
			"createTime":       llx.TimeDataPtr(timestampAsTimePtr(f.CreateTime)),
			"severity":         llx.StringData(f.Severity.String()),
			"mute":             llx.StringData(f.Mute.String()),
			"findingClass":     llx.StringData(f.FindingClass.String()),
			"state":            llx.StringData(f.State.String()),
			"resourceName":     llx.StringData(f.ResourceName),
			"chokepoint":       llx.DictData(chokepointToDict(f.Chokepoint)),
			"externalExposure": llx.DictData(externalExposureToDict(f.ExternalExposure)),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlFinding)
	}

	return res, nil
}

// sourcePropertiesToDict converts source properties (map[string]*structpb.Value) to map[string]any.
func sourcePropertiesToDict(props map[string]*structpb.Value) (map[string]any, error) {
	if props == nil {
		return nil, nil
	}
	result := make(map[string]any, len(props))
	for k, v := range props {
		result[k] = v.AsInterface()
	}
	return result, nil
}

// chokepointToDict converts a Chokepoint protobuf to a dict.
func chokepointToDict(cp *sccpb.Chokepoint) map[string]any {
	if cp == nil {
		return nil
	}
	return map[string]any{
		"relatedFindings": cp.RelatedFindings,
	}
}

// externalExposureToDict converts an ExternalExposure protobuf to a dict.
func externalExposureToDict(ee *sccpb.ExternalExposure) map[string]any {
	if ee == nil {
		return nil
	}
	return map[string]any{
		"privateIpAddress":           ee.PrivateIpAddress,
		"privatePort":                ee.PrivatePort,
		"exposedService":             ee.ExposedService,
		"publicIpAddress":            ee.PublicIpAddress,
		"publicPort":                 ee.PublicPort,
		"exposedEndpoint":            ee.ExposedEndpoint,
		"loadBalancerFirewallPolicy": ee.LoadBalancerFirewallPolicy,
		"serviceFirewallPolicy":      ee.ServiceFirewallPolicy,
		"forwardingRule":             ee.ForwardingRule,
		"backendService":             ee.BackendService,
		"instanceGroup":              ee.InstanceGroup,
		"networkEndpointGroup":       ee.NetworkEndpointGroup,
	}
}

// listSCCNotificationConfigs lists SCC notification configs for a given parent.
func listSCCNotificationConfigs(runtime *plugin.Runtime, conn *connection.GcpConnection, parent string) ([]any, error) {
	client, err := newSCCClient(conn)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	it := client.ListNotificationConfigs(context.Background(), &sccpb.ListNotificationConfigsRequest{
		Parent: parent,
	})

	var res []any
	for {
		nc, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		filter := ""
		if sc := nc.GetStreamingConfig(); sc != nil {
			filter = sc.Filter
		}

		mqlNC, err := CreateResource(runtime, "gcp.scc.notificationConfig", map[string]*llx.RawData{
			"name":           llx.StringData(nc.Name),
			"description":    llx.StringData(nc.Description),
			"pubsubTopic":    llx.StringData(nc.PubsubTopic),
			"serviceAccount": llx.StringData(nc.ServiceAccount),
			"filter":         llx.StringData(filter),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlNC)
	}

	return res, nil
}

// listSCCMuteConfigs lists SCC mute configs for a given parent.
func listSCCMuteConfigs(runtime *plugin.Runtime, conn *connection.GcpConnection, parent string) ([]any, error) {
	client, err := newSCCClient(conn)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	it := client.ListMuteConfigs(context.Background(), &sccpb.ListMuteConfigsRequest{
		Parent: parent,
	})

	var res []any
	for {
		mc, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		mqlMC, err := CreateResource(runtime, "gcp.scc.muteConfig", map[string]*llx.RawData{
			"name":             llx.StringData(mc.Name),
			"displayName":      llx.StringData(mc.DisplayName),
			"description":      llx.StringData(mc.Description),
			"filter":           llx.StringData(mc.Filter),
			"createTime":       llx.TimeDataPtr(timestampAsTimePtr(mc.CreateTime)),
			"updateTime":       llx.TimeDataPtr(timestampAsTimePtr(mc.UpdateTime)),
			"mostRecentEditor": llx.StringData(mc.MostRecentEditor),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlMC)
	}

	return res, nil
}

// listSCCBigQueryExports lists SCC BigQuery export configs for a given parent.
func listSCCBigQueryExports(runtime *plugin.Runtime, conn *connection.GcpConnection, parent string) ([]any, error) {
	client, err := newSCCClient(conn)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	it := client.ListBigQueryExports(context.Background(), &sccpb.ListBigQueryExportsRequest{
		Parent: parent,
	})

	var res []any
	for {
		bqe, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		mqlBQE, err := CreateResource(runtime, "gcp.scc.bigQueryExport", map[string]*llx.RawData{
			"name":             llx.StringData(bqe.Name),
			"description":      llx.StringData(bqe.Description),
			"dataset":          llx.StringData(bqe.Dataset),
			"filter":           llx.StringData(bqe.Filter),
			"createTime":       llx.TimeDataPtr(timestampAsTimePtr(bqe.CreateTime)),
			"updateTime":       llx.TimeDataPtr(timestampAsTimePtr(bqe.UpdateTime)),
			"mostRecentEditor": llx.StringData(bqe.MostRecentEditor),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlBQE)
	}

	return res, nil
}

// Organization-level methods
// Note: org-level SCC methods do not check isServiceEnabled because the Security
// Command Center API is enabled at the project level, not the organization level.
// Organization-scoped queries work as long as the caller has the appropriate IAM
// permissions on the org.

func (g *mqlGcpOrganization) sccParent() (string, *connection.GcpConnection, error) {
	if g.Id.Error != nil {
		return "", nil, g.Id.Error
	}
	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)
	return "organizations/" + g.Id.Data, conn, nil
}

func (g *mqlGcpOrganization) sccSources() ([]any, error) {
	parent, conn, err := g.sccParent()
	if err != nil {
		return nil, err
	}
	return listSCCSources(g.MqlRuntime, conn, parent)
}

func (g *mqlGcpOrganization) sccFindings() ([]any, error) {
	parent, conn, err := g.sccParent()
	if err != nil {
		return nil, err
	}
	return listSCCFindings(g.MqlRuntime, conn, parent)
}

func (g *mqlGcpOrganization) sccNotificationConfigs() ([]any, error) {
	parent, conn, err := g.sccParent()
	if err != nil {
		return nil, err
	}
	return listSCCNotificationConfigs(g.MqlRuntime, conn, parent)
}

func (g *mqlGcpOrganization) sccMuteConfigs() ([]any, error) {
	parent, conn, err := g.sccParent()
	if err != nil {
		return nil, err
	}
	return listSCCMuteConfigs(g.MqlRuntime, conn, parent)
}

func (g *mqlGcpOrganization) sccBigQueryExports() ([]any, error) {
	parent, conn, err := g.sccParent()
	if err != nil {
		return nil, err
	}
	return listSCCBigQueryExports(g.MqlRuntime, conn, parent)
}

// Project-level method

func (g *mqlGcpProject) sccFindings() ([]any, error) {
	if g.Id.Error != nil {
		return nil, g.Id.Error
	}
	projectId := g.Id.Data

	serviceEnabled, err := g.isServiceEnabled(service_securitycenter)
	if err != nil {
		return nil, err
	}
	if !serviceEnabled {
		return nil, nil
	}

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)
	return listSCCFindings(g.MqlRuntime, conn, "projects/"+projectId)
}
