// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"fmt"

	"go.mondoo.com/cnquery/v10/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v10/providers/gcp/connection"
	"go.mondoo.com/cnquery/v10/types"

	functions "cloud.google.com/go/functions/apiv1"
	"cloud.google.com/go/functions/apiv1/functionspb"
	"go.mondoo.com/cnquery/v10/llx"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

func (g *mqlGcpProject) cloudFunctions() ([]interface{}, error) {
	if g.Id.Error != nil {
		return nil, g.Id.Error
	}
	projectId := g.Id.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)

	creds, err := conn.Credentials(functions.DefaultAuthScopes()...)
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
	type mqlSourceRepository struct {
		Url         string `json:"url"`
		DeployedUrl string `json:"deployedUrl"`
	}
	type mqlHttpsTrigger struct {
		Url           string `json:"url"`
		SecurityLevel string `json:"securityLevel"`
	}
	type mqlFailurePolicy struct {
		Retry string `json:"retry"`
	}
	type mqlEventTrigger struct {
		EventType     string           `json:"eventType"`
		Resource      string           `json:"resource"`
		Service       string           `json:"service"`
		FailurePolicy mqlFailurePolicy `json:"failurePolicy"`
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
			envVar, err := convert.JsonToDict(mqlSecretEnvVar{ProjectId: v.ProjectId, Secret: v.Secret, Version: v.Version})
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
			vol, err := convert.JsonToDict(mqlSecretVolume{MountPath: v.MountPath, ProjectId: v.ProjectId, Secret: v.Secret, Versions: versions})
			if err != nil {
				return nil, err
			}
			secretVolumes = append(secretVolumes, vol)
		}

		var sourceUploadUrl, sourceArchiveUrl string
		var sourceRepository map[string]interface{}
		switch f.SourceCode.(type) {
		case *functionspb.CloudFunction_SourceArchiveUrl:
			sourceArchiveUrl = f.GetSourceArchiveUrl()
		case *functionspb.CloudFunction_SourceRepository:
			pbSourceRepo := f.GetSourceRepository()
			sourceRepository, err = convert.JsonToDict(mqlSourceRepository{Url: pbSourceRepo.Url, DeployedUrl: pbSourceRepo.DeployedUrl})
			if err != nil {
				return nil, err
			}
		case *functionspb.CloudFunction_SourceUploadUrl:
			sourceUploadUrl = f.GetSourceUploadUrl()
		}

		var httpsTrigger, eventTrigger map[string]interface{}
		switch f.Trigger.(type) {
		case *functionspb.CloudFunction_HttpsTrigger:
			pbHttpsTrigger := f.GetHttpsTrigger()
			httpsTrigger, err = convert.JsonToDict(mqlHttpsTrigger{Url: pbHttpsTrigger.Url, SecurityLevel: pbHttpsTrigger.SecurityLevel.String()})
			if err != nil {
				return nil, err
			}
		case *functionspb.CloudFunction_EventTrigger:
			pbEventTrigger := f.GetEventTrigger()
			eventTrigger, err = convert.JsonToDict(mqlEventTrigger{
				EventType:     pbEventTrigger.EventType,
				Resource:      pbEventTrigger.Resource,
				Service:       pbEventTrigger.Service,
				FailurePolicy: mqlFailurePolicy{Retry: pbEventTrigger.FailurePolicy.GetRetry().String()},
			})
			if err != nil {
				return nil, err
			}
		}

		mqlCloudFuncs, err := CreateResource(g.MqlRuntime, "gcp.project.cloudFunction", map[string]*llx.RawData{
			"projectId":           llx.StringData(projectId),
			"name":                llx.StringData(parseResourceName(f.Name)),
			"description":         llx.StringData(f.Description),
			"sourceArchiveUrl":    llx.StringData(sourceArchiveUrl),
			"sourceRepository":    llx.DictData(sourceRepository),
			"sourceUploadUrl":     llx.StringData(sourceUploadUrl),
			"httpsTrigger":        llx.DictData(httpsTrigger),
			"eventTrigger":        llx.DictData(eventTrigger),
			"status":              llx.StringData(f.Status.String()),
			"entryPoint":          llx.StringData(f.EntryPoint),
			"runtime":             llx.StringData(f.Runtime),
			"timeout":             llx.TimeData(llx.DurationToTime(int64(f.Timeout.Seconds))),
			"availableMemoryMb":   llx.IntData(int64(f.AvailableMemoryMb)),
			"serviceAccountEmail": llx.StringData(f.ServiceAccountEmail),
			"updated":             llx.TimeData(f.UpdateTime.AsTime()),
			"versionId":           llx.IntData(f.VersionId),
			"labels":              llx.MapData(convert.MapToInterfaceMap(f.Labels), types.String),
			"envVars":             llx.MapData(convert.MapToInterfaceMap(f.EnvironmentVariables), types.String),
			"buildEnvVars":        llx.MapData(convert.MapToInterfaceMap(f.BuildEnvironmentVariables), types.String),
			"network":             llx.StringData(f.Network),
			"maxInstances":        llx.IntData(int64(f.MaxInstances)),
			"minInstances":        llx.IntData(int64(f.MinInstances)),
			"vpcConnector":        llx.StringData(f.VpcConnector),
			"egressSettings":      llx.StringData(f.VpcConnectorEgressSettings.String()),
			"ingressSettings":     llx.StringData(f.IngressSettings.String()),
			"kmsKeyName":          llx.StringData(f.KmsKeyName),
			"buildWorkerPool":     llx.StringData(f.BuildWorkerPool),
			"buildId":             llx.StringData(f.BuildId),
			"buildName":           llx.StringData(f.BuildName),
			"secretEnvVars":       llx.MapData(secretEnvVars, types.Dict),
			"secretVolumes":       llx.ArrayData(secretVolumes, types.Dict),
			"dockerRepository":    llx.StringData(f.DockerRepository),
			"dockerRegistry":      llx.StringData(f.DockerRegistry.String()),
		})
		if err != nil {
			return nil, err
		}
		cloudFunctions = append(cloudFunctions, mqlCloudFuncs)
	}
	return cloudFunctions, nil
}

func (g *mqlGcpProjectCloudFunction) id() (string, error) {
	if g.ProjectId.Error != nil {
		return "", g.ProjectId.Error
	}
	projectId := g.ProjectId.Data
	if g.Name.Error != nil {
		return "", g.Name.Error
	}
	name := g.Name.Data
	return fmt.Sprintf("%s/%s", projectId, name), nil
}
