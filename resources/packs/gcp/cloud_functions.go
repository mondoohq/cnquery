package gcp

import (
	"context"
	"fmt"

	functions "cloud.google.com/go/functions/apiv1"
	"cloud.google.com/go/functions/apiv1/functionspb"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

func (g *mqlGcpProject) GetCloudFunctions() ([]interface{}, error) {
	projectId, err := g.Id()
	if err != nil {
		return nil, err
	}

	provider, err := gcpProvider(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	creds, err := provider.Credentials(functions.DefaultAuthScopes()...)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()

	cloudFuncSvc, err := functions.NewCloudFunctionsClient(ctx, option.WithCredentials(creds))
	if err != nil {
		return nil, err
	}
	defer cloudFuncSvc.Close()

	it := cloudFuncSvc.ListFunctions(ctx, &functionspb.ListFunctionsRequest{Parent: fmt.Sprintf("projects/%s/locations/-", projectId)})
	var cloudFunctions []interface{}
	for {
		f, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		mqlCloudFuncs, err := g.MotorRuntime.CreateResource("gcp.project.cloudFunction",
			"projectId", projectId,
			"name", parseResourceName(f.Name),
		)
		if err != nil {
			return nil, err
		}
		cloudFunctions = append(cloudFunctions, mqlCloudFuncs)
	}
	return cloudFunctions, nil
}

func (g *mqlGcpProjectCloudFunction) id() (string, error) {
	projectId, err := g.ProjectId()
	if err != nil {
		return "", err
	}
	name, err := g.Name()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/%s", projectId, name), nil
}
