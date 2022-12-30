package gcp

import (
	"context"
	"fmt"

	"cloud.google.com/go/longrunning/autogen/longrunningpb"
	run "cloud.google.com/go/run/apiv2"
	"go.mondoo.com/cnquery/resources"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

func (g *mqlGcpProjectCloudrun) id() (string, error) {
	return "gcp.project.cloudrun", nil
}

func (g *mqlGcpProjectCloudrun) init(args *resources.Args) (*resources.Args, GcpProjectCloudrun, error) {
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

func (g *mqlGcpProject) GetCloudrun() (interface{}, error) {
	projectId, err := g.Id()
	if err != nil {
		return nil, err
	}

	return g.MotorRuntime.CreateResource("gcp.project.cloudrun",
		"projectId", projectId,
	)
}

func (g *mqlGcpProjectCloudrunOperation) id() (string, error) {
	id, err := g.Id()
	if err != nil {
		return "", err
	}
	return id, nil
}

func (g *mqlGcpProjectCloudrun) GetOperations() ([]interface{}, error) {
	projectId, err := g.ProjectId()
	if err != nil {
		return nil, err
	}

	provider, err := gcpProvider(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	creds, err := provider.Credentials(run.DefaultAuthScopes()...)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()

	runSvc, err := run.NewServicesClient(ctx, option.WithCredentials(creds))
	if err != nil {
		return nil, err
	}
	defer runSvc.Close()

	var operations []interface{}

	it := runSvc.ListOperations(ctx, &longrunningpb.ListOperationsRequest{Name: fmt.Sprintf("projects/%s", projectId)})
	for {
		t, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		mqlTopic, err := g.MotorRuntime.CreateResource("gcp.project.cloudrun.operation",
			"id", t.Name,
			"projectId", projectId,
			"name", t.Name,
		)
		if err != nil {
			return nil, err
		}
		operations = append(operations, mqlTopic)
	}

	return operations, nil
}
