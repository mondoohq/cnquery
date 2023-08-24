// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"strings"

	serviceusage "cloud.google.com/go/serviceusage/apiv1"
	"cloud.google.com/go/serviceusage/apiv1/serviceusagepb"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/resources"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

func serviceName(name string) string {
	entries := strings.Split(name, "/")
	return entries[len(entries)-1]
}

func (g *mqlGcpProject) GetServices() ([]interface{}, error) {
	projectId, err := g.Id()
	if err != nil {
		return nil, err
	}

	provider, err := gcpProvider(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	credentials, err := provider.Credentials(serviceusage.DefaultAuthScopes()...)
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
		Parent: `projects/` + projectId,
		// Filter:   "state:ENABLED",
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

		mqlService, err := g.MotorRuntime.CreateResource("gcp.service",
			"projectId", projectId,
			"name", serviceName(item.Name),
			"parentName", item.Parent,
			"state", item.State.String(),
			"title", title,
		)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlService)
	}

	return res, nil
}

func (g *mqlGcpService) id() (string, error) {
	name, err := g.Name()
	if err != nil {
		return "", err
	}
	parent, err := g.ParentName()
	if err != nil {
		return "", err
	}

	return "gcp.service/" + parent + "/" + name, nil
}

func (g *mqlGcpService) init(args *resources.Args) (*resources.Args, GcpService, error) {
	if len(*args) > 2 {
		return args, nil, nil
	}

	nameRaw := (*args)["name"]
	if nameRaw == nil {
		return args, nil, nil
	}
	name := nameRaw.(string)

	provider, err := gcpProvider(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, nil, err
	}

	credentials, err := provider.Credentials(serviceusage.DefaultAuthScopes()...)
	if err != nil {
		return nil, nil, err
	}

	var projectId string
	projectIdRaw := (*args)["projectId"]
	if projectIdRaw != nil {
		projectId = projectIdRaw.(string)
	} else {
		projectId = provider.ResourceID()
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

	(*args)["projectId"] = projectId
	(*args)["name"] = serviceName(item.Name)
	(*args)["parentName"] = item.Parent
	(*args)["state"] = item.State.String()

	title := ""
	if item.Config != nil {
		title = item.Config.Title
	}
	(*args)["title"] = title

	return args, nil, nil
}

func (g *mqlGcpService) GetEnabled() (bool, error) {
	state, err := g.State()
	if err != nil {
		return false, err
	}
	return state == "ENABLED", nil
}
