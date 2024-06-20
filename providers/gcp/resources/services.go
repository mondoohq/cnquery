// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"strings"

	serviceusage "cloud.google.com/go/serviceusage/apiv1"
	"cloud.google.com/go/serviceusage/apiv1/serviceusagepb"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers/gcp/connection"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

const (
	service_compute             = "compute.googleapis.com"
	service_gke                 = "container.googleapis.com"
	service_bigquery            = "bigquery.googleapis.com"
	service_essential_contacts  = "essentialcontacts.googleapis.com"
	service_dns                 = "dns.googleapis.com"
	service_accessapproval      = "accessapproval.googleapis.com"
	service_apikeys             = "apikeys.googleapis.com"
	service_dataproc            = "dataproc.googleapis.com"
	service_binaryauthorization = "binaryauthorization.googleapis.com"
)

func serviceName(name string) string {
	entries := strings.Split(name, "/")
	return entries[len(entries)-1]
}

func (g *mqlGcpProject) services() ([]interface{}, error) {
	return g.fetchServices("")
}

// fetches the gcp services with a filter, e.g. "state:ENABLED"
func (g *mqlGcpProject) fetchServices(filter string) ([]interface{}, error) {
	if g.Id.Error != nil {
		return nil, g.Id.Error
	}
	projectId := g.Id.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)

	credentials, err := conn.Credentials(serviceusage.DefaultAuthScopes()...)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	c, err := serviceusage.NewClient(ctx, option.WithCredentials(credentials))
	if err != nil {
		log.Info().Err(err).Msg("could not create client")
		return nil, err
	}

	// projects/123/services/serviceusage.googleapis.com
	//service, err := c.GetService(ctx, &serviceusagepb.GetServiceRequest{
	//	Name: name,
	//})
	//service.Config.Title

	it := c.ListServices(ctx, &serviceusagepb.ListServicesRequest{
		Parent:   `projects/` + projectId,
		Filter:   filter,
		PageSize: 200,
	})

	res := []interface{}{}
	for {
		item, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		title := ""
		if item.Config != nil {
			title = item.Config.Title
		}

		mqlService, err := CreateResource(g.MqlRuntime, "gcp.service", map[string]*llx.RawData{
			"projectId":  llx.StringData(projectId),
			"name":       llx.StringData(serviceName(item.Name)),
			"parentName": llx.StringData(item.Parent),
			"state":      llx.StringData(item.State.String()),
			"title":      llx.StringData(title),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlService)
	}

	return res, nil
}

func (g *mqlGcpService) id() (string, error) {
	if g.Name.Error != nil {
		return "", g.Name.Error
	}
	name := g.Name.Data
	parent := g.GetParentName()
	if parent.Error != nil {
		return "", parent.Error
	}

	return "gcp.service/" + parent.Data + "/" + name, nil
}

func initGcpService(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}

	nameRaw := args["name"]
	if nameRaw == nil {
		return args, nil, nil
	}
	name := nameRaw.Value.(string)

	conn, ok := runtime.Connection.(*connection.GcpConnection)
	if !ok {
		return nil, nil, errors.New("invalid connection provided, it is not a GCP connection")
	}
	credentials, err := conn.Credentials(serviceusage.DefaultAuthScopes()...)
	if err != nil {
		return nil, nil, err
	}

	var projectId string
	projectIdRaw := args["projectId"]
	if projectIdRaw != nil {
		projectId = projectIdRaw.Value.(string)
	} else {
		projectId = conn.ResourceID()
	}

	ctx := context.Background()
	c, err := serviceusage.NewClient(ctx, option.WithCredentials(credentials))
	if err != nil {
		return nil, nil, err
	}

	// name is constructed `projects/123/services/serviceusage.googleapis.com`
	item, err := c.GetService(context.Background(), &serviceusagepb.GetServiceRequest{
		Name: `projects/` + projectId + "/services/" + name,
	})
	if err != nil {
		return nil, nil, err
	}

	args["projectId"] = llx.StringData(projectId)
	args["name"] = llx.StringData(serviceName(item.Name))
	args["parentName"] = llx.StringData(item.Parent)
	args["state"] = llx.StringData(item.State.String())

	title := ""
	if item.Config != nil {
		title = item.Config.Title
	}
	args["title"] = llx.StringData(title)

	return args, nil, nil
}

func (g *mqlGcpService) enabled() (bool, error) {
	if g.State.Error != nil {
		return false, g.State.Error
	}
	state := g.State.Data
	return state == "ENABLED", nil
}
