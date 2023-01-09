package gcp

import (
	"context"
	"fmt"

	functions "cloud.google.com/go/functions/apiv1"
	"cloud.google.com/go/functions/apiv1/functionspb"
	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/resources/packs/core"
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

	type mqlSecretEnvVar struct {
		ProjectId string `json:"projectId"`
		Secret    string `json:"secret"`
		Version   string `json:"version"`
	}
	type mqlSecretVolumeVersion struct {
		Version string `json:"version"`
		Path    string `json:"path"`
	}
	type mqlSecretVolume struct {
		MountPath string                   `json:"mountPath"`
		ProjectId string                   `json:"projectId"`
		Secret    string                   `json:"secret"`
		Versions  []mqlSecretVolumeVersion `json:"versions"`
	}

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

		secretEnvVars := make(map[string]interface{})
		for _, v := range f.SecretEnvironmentVariables {
			envVar, err := core.JsonToDict(mqlSecretEnvVar{ProjectId: v.ProjectId, Secret: v.Secret, Version: v.Version})
			if err != nil {
				return nil, err
			}
			secretEnvVars[v.Key] = envVar
		}

		secretVolumes := make([]interface{}, 0, len(f.SecretVolumes))
		for _, v := range f.SecretVolumes {
			versions := make([]mqlSecretVolumeVersion, 0, len(v.Versions))
			for _, vv := range v.Versions {
				versions = append(versions, mqlSecretVolumeVersion{Version: vv.Version, Path: vv.Path})
			}
			vol, err := core.JsonToDict(mqlSecretVolume{MountPath: v.MountPath, ProjectId: v.ProjectId, Secret: v.Secret, Versions: versions})
			if err != nil {
				return nil, err
			}
			secretVolumes = append(secretVolumes, vol)
		}

		mqlCloudFuncs, err := g.MotorRuntime.CreateResource("gcp.project.cloudFunction",
			"projectId", projectId,
			"name", parseResourceName(f.Name),
			"description", f.Description,
			"status", f.Status.String(),
			"entryPoint", f.EntryPoint,
			"runtime", f.Runtime,
			"timeout", core.MqlTime(llx.DurationToTime(int64(f.Timeout.Seconds))),
			"availableMemoryMb", int64(f.AvailableMemoryMb),
			"serviceAccountEmail", f.ServiceAccountEmail,
			"updated", core.MqlTime(f.UpdateTime.AsTime()),
			"versionId", f.VersionId,
			"labels", core.StrMapToInterface(f.Labels),
			"envVars", core.StrMapToInterface(f.EnvironmentVariables),
			"buildEnvVars", core.StrMapToInterface(f.BuildEnvironmentVariables),
			"network", f.Network,
			"maxInstances", int64(f.MaxInstances),
			"minInstances", int64(f.MinInstances),
			"vpcConnector", f.VpcConnector,
			"egressSettings", f.VpcConnectorEgressSettings.String(),
			"ingressSettings", f.IngressSettings.String(),
			"kmsKeyName", f.KmsKeyName,
			"buildWorkerPool", f.BuildWorkerPool,
			"buildId", f.BuildId,
			"buildName", f.BuildName,
			"secretEnvVars", secretEnvVars,
			"secretVolumes", secretVolumes,
			"dockerRepository", f.DockerRepository,
			"dockerRegistry", f.DockerRegistry.String(),
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
