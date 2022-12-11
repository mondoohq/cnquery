package gcp

import (
	"context"

	serviceusage "cloud.google.com/go/serviceusage/apiv1"
	"cloud.google.com/go/serviceusage/apiv1/serviceusagepb"
	"github.com/rs/zerolog/log"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

func (g *mqlGcpProject) GetServices() ([]interface{}, error) {
	projectNumber, err := g.Number()
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
		Parent: `projects/` + projectNumber,
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
			"name", item.Name,
			"parent", item.Parent,
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
	parent, err := g.Parent()
	if err != nil {
		return "", err
	}

	return "gcp.service/" + parent + "/" + name, nil
}
