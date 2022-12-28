package gcp

import (
	"context"
	"fmt"
	"sync"

	kms "cloud.google.com/go/kms/apiv1"
	"cloud.google.com/go/kms/apiv1/kmspb"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/resources"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/genproto/googleapis/cloud/location"
)

func (g *mqlGcpProjectKms) id() (string, error) {
	return "gcp.project.kms", nil
}

func (g *mqlGcpProjectKms) init(args *resources.Args) (*resources.Args, GcpProjectKms, error) {
	if len(*args) > 0 {
		return args, nil, nil
	}

	provider, err := gcpProvider(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, nil, err
	}

	projectId := provider.ResourceID()
	(*args)["projectId"] = projectId

	return args, nil, nil
}

func (g *mqlGcpProject) GetKms() (interface{}, error) {
	projectId, err := g.Id()
	if err != nil {
		return nil, err
	}

	return g.MotorRuntime.CreateResource("gcp.project.kms",
		"projectId", projectId,
	)
}

func (g *mqlGcpProjectKmsKeyring) id() (string, error) {
	id, err := g.Id()
	if err != nil {
		return "", err
	}
	return id, nil
}

func (g *mqlGcpProjectKms) GetLocations() ([]interface{}, error) {
	projectId, err := g.ProjectId()
	if err != nil {
		return nil, err
	}

	provider, err := gcpProvider(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	creds, err := provider.Credentials(kms.DefaultAuthScopes()...)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()

	kmsSvc, err := kms.NewKeyManagementClient(ctx, option.WithCredentials(creds))
	if err != nil {
		return nil, err
	}

	var locations []interface{}
	it := kmsSvc.ListLocations(ctx, &location.ListLocationsRequest{Name: fmt.Sprintf("projects/%s", projectId)})
	for {
		l, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		locations = append(locations, l.LocationId)
	}
	return locations, nil
}

func (g *mqlGcpProjectKms) GetKeyrings() ([]interface{}, error) {
	projectId, err := g.ProjectId()
	if err != nil {
		return nil, err
	}

	locations, err := g.Locations()
	if err != nil {
		return nil, err
	}

	provider, err := gcpProvider(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	creds, err := provider.Credentials(kms.DefaultAuthScopes()...)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()

	kmsSvc, err := kms.NewKeyManagementClient(ctx, option.WithCredentials(creds))
	if err != nil {
		return nil, err
	}

	var keyrings []interface{}
	var wg sync.WaitGroup
	wg.Add(len(locations))
	mux := &sync.Mutex{}

	for _, location := range locations {
		go func(svc *kms.KeyManagementClient, project string, location string) {
			defer wg.Done()
			it := kmsSvc.ListKeyRings(ctx,
				&kmspb.ListKeyRingsRequest{Parent: fmt.Sprintf("projects/%s/locations/%s", projectId, location)})
			for {
				k, err := it.Next()
				if err == iterator.Done {
					break
				}
				if err != nil {
					log.Error().Err(err)
					return
				}

				created := k.CreateTime.AsTime()
				mqlKeyring, err := g.MotorRuntime.CreateResource("gcp.project.kms.keyring",
					"id", k.Name,
					"projectId", projectId,
					"name", k.Name,
					"created", &created,
					"location", location,
				)
				if err != nil {
					log.Error().Err(err)
					return
				}
				mux.Lock()
				keyrings = append(keyrings, mqlKeyring)
				mux.Unlock()
			}
		}(kmsSvc, projectId, location.(string))
	}
	wg.Wait()
	return keyrings, nil
}
