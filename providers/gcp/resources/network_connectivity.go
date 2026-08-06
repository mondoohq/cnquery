// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"fmt"
	"sync"

	networkconnectivity "cloud.google.com/go/networkconnectivity/apiv1"
	"cloud.google.com/go/networkconnectivity/apiv1/networkconnectivitypb"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/gcp/connection"
	"go.mondoo.com/mql/v13/types"
	googleoauth "golang.org/x/oauth2/google"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

type mqlGcpProjectNetworkConnectivityServiceInternal struct {
	serviceEnabled bool
	serviceOnce    sync.Once
	serviceErr     error
}

func (g *mqlGcpProject) networkConnectivity() (*mqlGcpProjectNetworkConnectivityService, error) {
	if g.Id.Error != nil {
		return nil, g.Id.Error
	}
	projectId := g.Id.Data

	res, err := CreateResource(g.MqlRuntime, "gcp.project.networkConnectivityService", map[string]*llx.RawData{
		"projectId": llx.StringData(projectId),
	})
	if err != nil {
		return nil, err
	}

	serviceEnabled, err := g.isServiceEnabled(service_networkconnectivity)
	if err != nil {
		return nil, err
	}

	svc := res.(*mqlGcpProjectNetworkConnectivityService)
	svc.serviceEnabled = serviceEnabled
	if !serviceEnabled {
		log.Debug().Str("service", service_networkconnectivity).Msg("gcp service is not enabled, skipping")
	}

	return svc, nil
}

func initGcpProjectNetworkConnectivityService(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 0 {
		return args, nil, nil
	}

	conn, ok := runtime.Connection.(*connection.GcpConnection)
	if !ok {
		return nil, nil, errors.New("invalid connection provided, it is not a GCP connection")
	}

	if args == nil {
		args = make(map[string]*llx.RawData)
	}
	args["projectId"] = llx.StringData(conn.ResourceID())
	return args, nil, nil
}

func (g *mqlGcpProjectNetworkConnectivityService) id() (string, error) {
	if g.ProjectId.Error != nil {
		return "", g.ProjectId.Error
	}
	return fmt.Sprintf("%s/gcp.project.networkConnectivityService", g.ProjectId.Data), nil
}

// isEnabled resolves the service-enabled gate lazily so every construction path
// agrees. Without it the Go zero value false would make both collections return
// an empty list with no error whenever the resource is not reached through the
// gcp.project accessor, which reads as "there is no connectivity fabric here".
func (g *mqlGcpProjectNetworkConnectivityService) isEnabled() (bool, error) {
	g.serviceOnce.Do(func() {
		if g.serviceEnabled {
			return // already set by the parent accessor
		}
		if g.ProjectId.Error != nil {
			g.serviceErr = g.ProjectId.Error
			return
		}
		proj, err := CreateResource(g.MqlRuntime, "gcp.project", map[string]*llx.RawData{
			"id": llx.StringData(g.ProjectId.Data),
		})
		if err != nil {
			g.serviceErr = err
			return
		}
		g.serviceEnabled, g.serviceErr = proj.(*mqlGcpProject).isServiceEnabled(service_networkconnectivity)
	})
	return g.serviceEnabled, g.serviceErr
}

// networkConnectivityCredentials returns credentials scoped for the Network
// Connectivity API. Only the credentials are shared, not the client: the
// permissions extractor tracks client variables per function, so a helper
// returning the client would hide the constructor and drop the permissions from
// the manifest.
func (g *mqlGcpProjectNetworkConnectivityService) networkConnectivityCredentials() (*googleoauth.Credentials, error) {
	conn, ok := g.MqlRuntime.Connection.(*connection.GcpConnection)
	if !ok {
		return nil, errors.New("invalid connection provided, it is not a GCP connection")
	}
	return conn.Credentials(networkconnectivity.DefaultAuthScopes()...)
}

func (g *mqlGcpProjectNetworkConnectivityServiceHub) id() (string, error) {
	return g.Name.Data, g.Name.Error
}

func (g *mqlGcpProjectNetworkConnectivityServiceSpoke) id() (string, error) {
	return g.Name.Data, g.Name.Error
}

// hubs lists the project's connectivity hubs. Hubs are global resources.
func (g *mqlGcpProjectNetworkConnectivityService) hubs() ([]any, error) {
	enabled, err := g.isEnabled()
	if err != nil {
		return nil, err
	}
	if !enabled {
		return nil, nil
	}
	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	creds, err := g.networkConnectivityCredentials()
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	client, err := networkconnectivity.NewHubClient(ctx, option.WithCredentials(creds), connection.GRPCClientTraceOption())
	if err != nil {
		return nil, err
	}
	defer client.Close()

	it := client.ListHubs(ctx, &networkconnectivitypb.ListHubsRequest{
		Parent: fmt.Sprintf("projects/%s/locations/global", projectId),
	})

	res := []any{}
	for {
		hub, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			if isGRPCSkippable(err) {
				log.Warn().Err(err).Str("project", projectId).Msg("could not list Network Connectivity hubs")
				return nil, nil
			}
			return nil, err
		}

		routingVpcs, err := convert.JsonToDictSlice(hub.GetRoutingVpcs())
		if err != nil {
			return nil, err
		}
		spokeSummary, err := protoToDict(hub.GetSpokeSummary())
		if err != nil {
			return nil, err
		}

		mqlHub, err := CreateResource(g.MqlRuntime, "gcp.project.networkConnectivityService.hub", map[string]*llx.RawData{
			"name":         llx.StringData(hub.GetName()),
			"description":  llx.StringData(hub.GetDescription()),
			"uniqueId":     llx.StringData(hub.GetUniqueId()),
			"state":        llx.StringData(hub.GetState().String()),
			"policyMode":   llx.StringData(hub.GetPolicyMode().String()),
			"routingVpcs":  llx.ArrayData(routingVpcs, types.Dict),
			"routeTables":  llx.ArrayData(convert.SliceAnyToInterface(hub.GetRouteTables()), types.String),
			"spokeSummary": llx.DictData(spokeSummary),
			"labels":       llx.MapData(convert.MapToInterfaceMap(hub.GetLabels()), types.String),
			"createTime":   llx.TimeDataPtr(timestampAsTimePtr(hub.GetCreateTime())),
			"updateTime":   llx.TimeDataPtr(timestampAsTimePtr(hub.GetUpdateTime())),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlHub)
	}

	return res, nil
}

// spokes lists the project's spokes. The "-" location wildcard collects spokes
// from every region in one call, which matters because a spoke lives in the
// region of the resource it attaches, not in global.
func (g *mqlGcpProjectNetworkConnectivityService) spokes() ([]any, error) {
	enabled, err := g.isEnabled()
	if err != nil {
		return nil, err
	}
	if !enabled {
		return nil, nil
	}
	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	creds, err := g.networkConnectivityCredentials()
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	client, err := networkconnectivity.NewHubClient(ctx, option.WithCredentials(creds), connection.GRPCClientTraceOption())
	if err != nil {
		return nil, err
	}
	defer client.Close()

	it := client.ListSpokes(ctx, &networkconnectivitypb.ListSpokesRequest{
		Parent: fmt.Sprintf("projects/%s/locations/-", projectId),
	})

	res := []any{}
	for {
		spoke, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			if isGRPCSkippable(err) {
				log.Warn().Err(err).Str("project", projectId).Msg("could not list Network Connectivity spokes")
				return nil, nil
			}
			return nil, err
		}

		reasons, err := convert.JsonToDictSlice(spoke.GetReasons())
		if err != nil {
			return nil, err
		}
		linkedVpcNetwork, err := protoToDict(spoke.GetLinkedVpcNetwork())
		if err != nil {
			return nil, err
		}
		linkedVpnTunnels, err := protoToDict(spoke.GetLinkedVpnTunnels())
		if err != nil {
			return nil, err
		}
		linkedInterconnectAttachments, err := protoToDict(spoke.GetLinkedInterconnectAttachments())
		if err != nil {
			return nil, err
		}
		linkedRouterApplianceInstances, err := protoToDict(spoke.GetLinkedRouterApplianceInstances())
		if err != nil {
			return nil, err
		}

		mqlSpoke, err := CreateResource(g.MqlRuntime, "gcp.project.networkConnectivityService.spoke", map[string]*llx.RawData{
			"name":                           llx.StringData(spoke.GetName()),
			"description":                    llx.StringData(spoke.GetDescription()),
			"hub":                            llx.StringData(spoke.GetHub()),
			"group":                          llx.StringData(spoke.GetGroup()),
			"spokeType":                      llx.StringData(spoke.GetSpokeType().String()),
			"state":                          llx.StringData(spoke.GetState().String()),
			"reasons":                        llx.ArrayData(reasons, types.Dict),
			"linkedVpcNetwork":               llx.DictData(linkedVpcNetwork),
			"linkedVpnTunnels":               llx.DictData(linkedVpnTunnels),
			"linkedInterconnectAttachments":  llx.DictData(linkedInterconnectAttachments),
			"linkedRouterApplianceInstances": llx.DictData(linkedRouterApplianceInstances),
			"uniqueId":                       llx.StringData(spoke.GetUniqueId()),
			"labels":                         llx.MapData(convert.MapToInterfaceMap(spoke.GetLabels()), types.String),
			"createTime":                     llx.TimeDataPtr(timestampAsTimePtr(spoke.GetCreateTime())),
			"updateTime":                     llx.TimeDataPtr(timestampAsTimePtr(spoke.GetUpdateTime())),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlSpoke)
	}

	return res, nil
}
