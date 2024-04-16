// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"fmt"
	"sync"

	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v11/providers/gcp/connection"
	"go.mondoo.com/cnquery/v11/types"

	"cloud.google.com/go/longrunning/autogen/longrunningpb"
	run "cloud.google.com/go/run/apiv2"
	runpb "cloud.google.com/go/run/apiv2/runpb"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v11/llx"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

func (g *mqlGcpProjectCloudRunService) id() (string, error) {
	if g.ProjectId.Error != nil {
		return "", g.ProjectId.Error
	}
	projectId := g.ProjectId.Data
	return fmt.Sprintf("%s/gcp.project.cloudRunService", projectId), nil
}

func initGcpProjectCloudRunService(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 0 {
		return args, nil, nil
	}

	conn := runtime.Connection.(*connection.GcpConnection)

	projectId := conn.ResourceID()
	args["projectId"] = llx.StringData(projectId)

	return args, nil, nil
}

func (g *mqlGcpProject) cloudRun() (*mqlGcpProjectCloudRunService, error) {
	if g.Id.Error != nil {
		return nil, g.Id.Error
	}
	projectId := g.Id.Data

	res, err := CreateResource(g.MqlRuntime, "gcp.project.cloudRunService", map[string]*llx.RawData{
		"projectId": llx.StringData(projectId),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlGcpProjectCloudRunService), nil
}

func (g *mqlGcpProjectCloudRunServiceOperation) id() (string, error) {
	if g.ProjectId.Error != nil {
		return "", g.ProjectId.Error
	}
	projectId := g.ProjectId.Data
	if g.Name.Error != nil {
		return "", g.Name.Error
	}
	name := g.Name.Data
	return fmt.Sprintf("gcp.project.cloudRunService.operation/%s/%s", projectId, name), nil
}

func (g *mqlGcpProjectCloudRunServiceService) id() (string, error) {
	if g.Id.Error != nil {
		return "", g.Id.Error
	}
	id := g.Id.Data
	return fmt.Sprintf("gcp.project.cloudRunService.service/%s", id), nil
}

func (g *mqlGcpProjectCloudRunServiceJob) id() (string, error) {
	if g.Id.Error != nil {
		return "", g.Id.Error
	}
	id := g.Id.Data
	return fmt.Sprintf("gcp.project.cloudRunService.job/%s", id), nil
}

func (g *mqlGcpProjectCloudRunServiceServiceRevisionTemplate) id() (string, error) {
	return g.Id.Data, g.Id.Error
}

func (g *mqlGcpProjectCloudRunServiceContainer) id() (string, error) {
	return g.Id.Data, g.Id.Error
}

func (g *mqlGcpProjectCloudRunServiceContainerProbe) id() (string, error) {
	return g.Id.Data, g.Id.Error
}

func (g *mqlGcpProjectCloudRunServiceCondition) id() (string, error) {
	return g.Id.Data, g.Id.Error
}

func (g *mqlGcpProjectCloudRunServiceJobExecutionTemplate) id() (string, error) {
	return g.Id.Data, g.Id.Error
}

func (g *mqlGcpProjectCloudRunServiceJobExecutionTemplateTaskTemplate) id() (string, error) {
	return g.Id.Data, g.Id.Error
}

func (g *mqlGcpProjectCloudRunService) regions() ([]interface{}, error) {
	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)

	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	client, err := conn.Client(run.DefaultAuthScopes()...)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	computeSvc, err := compute.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}

	regions, err := computeSvc.Regions.List(projectId).Do()
	if err != nil {
		return nil, err
	}

	regionNames := make([]interface{}, 0, len(regions.Items))
	for _, region := range regions.Items {
		regionNames = append(regionNames, region.Name)
	}
	return regionNames, nil
}

func (g *mqlGcpProjectCloudRunService) operations() ([]interface{}, error) {
	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	if g.Regions.Error != nil {
		return nil, g.Regions.Error
	}
	regions := g.Regions.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)

	creds, err := conn.Credentials(run.DefaultAuthScopes()...)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()

	runSvc, err := run.NewServicesClient(ctx, option.WithCredentials(creds))
	if err != nil {
		return nil, err
	}
	defer runSvc.Close()

	var wg sync.WaitGroup
	var operations []interface{}
	wg.Add(len(regions))
	mux := &sync.Mutex{}
	for _, region := range regions {
		go func(region string) {
			defer wg.Done()
			it := runSvc.ListOperations(ctx, &longrunningpb.ListOperationsRequest{Name: fmt.Sprintf("projects/%s/locations/%s", projectId, region)})
			for {
				t, err := it.Next()
				if err == iterator.Done {
					break
				}
				if err != nil {
					log.Error().Err(err).Send()
				}
				mqlOp, err := CreateResource(g.MqlRuntime, "gcp.project.cloudRunService.operation", map[string]*llx.RawData{
					"projectId": llx.StringData(projectId),
					"name":      llx.StringData(t.Name),
					"done":      llx.BoolData(t.Done),
				})
				if err != nil {
					log.Error().Err(err).Send()
				}
				mux.Lock()
				operations = append(operations, mqlOp)
				mux.Unlock()
			}
		}(region.(string))
	}
	wg.Wait()
	return operations, nil
}

func (g *mqlGcpProjectCloudRunService) services() ([]interface{}, error) {
	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	if g.Regions.Error != nil {
		return nil, g.Regions.Error
	}
	regions := g.Regions.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)

	creds, err := conn.Credentials(run.DefaultAuthScopes()...)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()

	runSvc, err := run.NewServicesClient(ctx, option.WithCredentials(creds))
	if err != nil {
		return nil, err
	}
	defer runSvc.Close()

	type mqlRevisionScaling struct {
		MinInstanceCount int32 `json:"minInstanceCount"`
		MaxInstanceCount int32 `json:"maxInstanceCount"`
	}

	var wg sync.WaitGroup
	var services []interface{}
	wg.Add(len(regions))
	mux := &sync.Mutex{}
	for _, region := range regions {
		go func(region string) {
			defer wg.Done()
			it := runSvc.ListServices(ctx, &runpb.ListServicesRequest{Parent: fmt.Sprintf("projects/%s/locations/%s", projectId, region)})
			for {
				s, err := it.Next()
				if err == iterator.Done {
					break
				}
				if err != nil {
					log.Error().Err(err).Send()
				}

				var mqlTemplate plugin.Resource
				if s.Template != nil {
					var scalingCfg map[string]interface{}
					if s.Template.Scaling != nil {
						scalingCfg, err = convert.JsonToDict(mqlRevisionScaling{
							MinInstanceCount: s.Template.Scaling.MinInstanceCount,
							MaxInstanceCount: s.Template.Scaling.MaxInstanceCount,
						})
						if err != nil {
							log.Error().Err(err).Send()
						}
					}

					vpcCfg, err := mqlVpcAccess(s.Template.VpcAccess)
					if err != nil {
						log.Error().Err(err).Send()
					}

					templateId := fmt.Sprintf("gcp.project.cloudRunService.service/%s/%s/revisionTemplate", projectId, s.Name)
					mqlContainers, err := mqlContainers(g.MqlRuntime, s.Template.Containers, templateId)
					if err != nil {
						log.Error().Err(err).Send()
					}

					mqlTemplate, err = CreateResource(g.MqlRuntime, "gcp.project.cloudRunService.service.revisionTemplate", map[string]*llx.RawData{
						"id":                            llx.StringData(templateId),
						"projectId":                     llx.StringData(projectId),
						"name":                          llx.StringData(s.Template.Revision),
						"labels":                        llx.MapData(convert.MapToInterfaceMap(s.Template.Labels), types.String),
						"annotations":                   llx.MapData(convert.MapToInterfaceMap(s.Template.Annotations), types.String),
						"scaling":                       llx.DictData(scalingCfg),
						"vpcAccess":                     llx.DictData(vpcCfg),
						"timeout":                       llx.TimeData(llx.DurationToTime((s.Template.Timeout.Seconds))),
						"serviceAccountEmail":           llx.StringData(s.Template.ServiceAccount),
						"containers":                    llx.ArrayData(mqlContainers, "gcp.project.cloudRunService.container"),
						"volumes":                       llx.ArrayData(mqlVolumes(s.Template.Volumes), types.Dict),
						"executionEnvironment":          llx.StringData(s.Template.ExecutionEnvironment.String()),
						"encryptionKey":                 llx.StringData(s.Template.EncryptionKey),
						"maxInstanceRequestConcurrency": llx.IntData(int64(s.Template.MaxInstanceRequestConcurrency)),
					})
					if err != nil {
						log.Error().Err(err).Send()
					}
				}

				mqlTraffic := make([]interface{}, 0, len(s.Traffic))
				for _, t := range s.Traffic {
					mqlTraffic = append(mqlTraffic, map[string]interface{}{
						"type":     t.Type.String(),
						"revision": t.Revision,
						"percent":  t.Percent,
						"tag":      t.Tag,
					})
				}

				mqlTerminalCondition, err := mqlCondition(g.MqlRuntime, s.TerminalCondition, s.Name, "terminal")
				if err != nil {
					log.Error().Err(err).Send()
				}

				mqlConditions := make([]interface{}, 0, len(s.Conditions))
				for i, c := range s.Conditions {
					mqlCondition, err := mqlCondition(g.MqlRuntime, c, s.Name, fmt.Sprintf("%d", i))
					if err != nil {
						log.Error().Err(err).Send()
					}
					mqlConditions = append(mqlConditions, mqlCondition)
				}

				mqlTrafficStatuses := make([]interface{}, 0, len(s.TrafficStatuses))
				for _, t := range s.TrafficStatuses {
					mqlTrafficStatuses = append(mqlTrafficStatuses, map[string]interface{}{
						"type":     t.Type.String(),
						"revision": t.Revision,
						"percent":  t.Percent,
						"tag":      t.Tag,
						"uri":      t.Uri,
					})
				}

				mqlS, err := CreateResource(g.MqlRuntime, "gcp.project.cloudRunService.service", map[string]*llx.RawData{
					"id":                    llx.StringData(s.Name),
					"projectId":             llx.StringData(projectId),
					"region":                llx.StringData(region),
					"name":                  llx.StringData(parseResourceName(s.Name)),
					"description":           llx.StringData(s.Description),
					"generation":            llx.IntData(s.Generation),
					"labels":                llx.MapData(convert.MapToInterfaceMap(s.Labels), types.String),
					"annotations":           llx.MapData(convert.MapToInterfaceMap(s.Annotations), types.String),
					"created":               llx.TimeData(s.CreateTime.AsTime()),
					"updated":               llx.TimeData(s.UpdateTime.AsTime()),
					"deleted":               llx.TimeData(s.DeleteTime.AsTime()),
					"expired":               llx.TimeData(s.ExpireTime.AsTime()),
					"creator":               llx.StringData(s.Creator),
					"lastModifier":          llx.StringData(s.LastModifier),
					"ingress":               llx.StringData(s.Ingress.String()),
					"launchStage":           llx.StringData(s.LaunchStage.String()),
					"template":              llx.ResourceData(mqlTemplate, "gcp.project.cloudRunService.service.revisionTemplate"),
					"traffic":               llx.ArrayData(mqlTraffic, types.Dict),
					"observedGeneration":    llx.IntData(s.ObservedGeneration),
					"terminalCondition":     llx.ResourceData(mqlTerminalCondition, "gcp.project.cloudRunService.condition"),
					"conditions":            llx.ArrayData(mqlConditions, types.Resource("gcp.project.cloudRunService.condition")),
					"latestReadyRevision":   llx.StringData(s.LatestReadyRevision),
					"latestCreatedRevision": llx.StringData(s.LatestCreatedRevision),
					"trafficStatuses":       llx.ArrayData(mqlTrafficStatuses, types.Dict),
					"uri":                   llx.StringData(s.Uri),
					"reconciling":           llx.BoolData(s.Reconciling),
				})
				if err != nil {
					log.Error().Err(err).Send()
				}
				mux.Lock()
				services = append(services, mqlS)
				mux.Unlock()
			}
		}(region.(string))
	}
	wg.Wait()
	return services, nil
}

func (g *mqlGcpProjectCloudRunServiceServiceRevisionTemplate) serviceAccount() (*mqlGcpProjectIamServiceServiceAccount, error) {
	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	if g.ServiceAccountEmail.Error != nil {
		return nil, g.ServiceAccountEmail.Error
	}
	email := g.ServiceAccountEmail.Data

	res, err := CreateResource(g.MqlRuntime, "gcp.project.iamService.serviceAccount", map[string]*llx.RawData{
		"projectId": llx.StringData(projectId),
		"email":     llx.StringData(email),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlGcpProjectIamServiceServiceAccount), nil
}

func (g *mqlGcpProjectCloudRunServiceJobExecutionTemplateTaskTemplate) serviceAccount() (*mqlGcpProjectIamServiceServiceAccount, error) {
	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	if g.ServiceAccountEmail.Error != nil {
		return nil, g.ServiceAccountEmail.Error
	}
	email := g.ServiceAccountEmail.Data

	res, err := CreateResource(g.MqlRuntime, "gcp.project.iamService.serviceAccount", map[string]*llx.RawData{
		"projectId": llx.StringData(projectId),
		"email":     llx.StringData(email),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlGcpProjectIamServiceServiceAccount), nil
}

func (g *mqlGcpProjectCloudRunService) jobs() ([]interface{}, error) {
	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	if g.Regions.Error != nil {
		return nil, g.Regions.Error
	}
	regions := g.Regions.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)

	creds, err := conn.Credentials(run.DefaultAuthScopes()...)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	runSvc, err := run.NewJobsClient(ctx, option.WithCredentials(creds))
	if err != nil {
		return nil, err
	}
	defer runSvc.Close()

	var wg sync.WaitGroup
	var jobs []interface{}
	wg.Add(len(regions))
	mux := &sync.Mutex{}
	for _, region := range regions {
		go func(region string) {
			defer wg.Done()
			it := runSvc.ListJobs(ctx, &runpb.ListJobsRequest{Parent: fmt.Sprintf("projects/%s/locations/%s", projectId, region)})
			for {
				j, err := it.Next()
				if err == iterator.Done {
					break
				}
				if err != nil {
					log.Error().Err(err).Send()
					return
				}

				var mqlTemplate plugin.Resource
				if j.Template != nil {
					templateId := fmt.Sprintf("%s/executionTemplate", j.Name)
					var mqlTaskTemplate plugin.Resource
					if j.Template.Template != nil {
						vpcAccess, err := mqlVpcAccess(j.Template.Template.VpcAccess)
						if err != nil {
							log.Error().Err(err).Send()
							return
						}

						mqlContainers, err := mqlContainers(g.MqlRuntime, j.Template.Template.Containers, templateId)
						if err != nil {
							log.Error().Err(err).Send()
							return
						}

						mqlTaskTemplate, err = CreateResource(g.MqlRuntime, "gcp.project.cloudRunService.job.executionTemplate.taskTemplate", map[string]*llx.RawData{
							"id":                   llx.StringData(fmt.Sprintf("%s/template", templateId)),
							"projectId":            llx.StringData(projectId),
							"vpcAccess":            llx.DictData(vpcAccess),
							"timeout":              llx.TimeData(llx.DurationToTime((j.Template.Template.Timeout.Seconds))),
							"serviceAccountEmail":  llx.StringData(j.Template.Template.ServiceAccount),
							"containers":           llx.ArrayData(mqlContainers, types.Resource("gcp.project.cloudRunService.container")),
							"volumes":              llx.ArrayData(mqlVolumes(j.Template.Template.Volumes), types.Dict),
							"executionEnvironment": llx.StringData(j.Template.Template.ExecutionEnvironment.String()),
							"encryptionKey":        llx.StringData(j.Template.Template.EncryptionKey),
							"maxRetries":           llx.IntData(int64(j.Template.Template.GetMaxRetries())),
						})
						if err != nil {
							log.Error().Err(err).Send()
							return
						}
					}

					mqlTemplate, err = CreateResource(g.MqlRuntime, "gcp.project.cloudRunService.job.executionTemplate", map[string]*llx.RawData{
						"id":          llx.StringData(templateId),
						"labels":      llx.MapData(convert.MapToInterfaceMap(j.Template.Labels), types.String),
						"annotations": llx.MapData(convert.MapToInterfaceMap(j.Template.Annotations), types.String),
						"parallelism": llx.IntData(int64(j.Template.Parallelism)),
						"taskCount":   llx.IntData(int64(j.Template.TaskCount)),
						"template":    llx.ResourceData(mqlTaskTemplate, "gcp.project.cloudRunService.job.executionTemplate.taskTemplate"),
					})
					if err != nil {
						log.Error().Err(err).Send()
						return
					}
				}

				mqlTerminalCondition, err := mqlCondition(g.MqlRuntime, j.TerminalCondition, j.Name, "terminal")
				if err != nil {
					log.Error().Err(err).Send()
					return
				}

				mqlConditions := make([]interface{}, 0, len(j.Conditions))
				for i, c := range j.Conditions {
					mqlCondition, err := mqlCondition(g.MqlRuntime, c, j.Name, fmt.Sprintf("%d", i))
					if err != nil {
						log.Error().Err(err).Send()
						return
					}
					mqlConditions = append(mqlConditions, mqlCondition)
				}

				mqlJob, err := CreateResource(g.MqlRuntime, "gcp.project.cloudRunService.job", map[string]*llx.RawData{
					"id":                 llx.StringData(j.Name),
					"projectId":          llx.StringData(projectId),
					"region":             llx.StringData(region),
					"name":               llx.StringData(parseResourceName(j.Name)),
					"generation":         llx.IntData(j.Generation),
					"labels":             llx.MapData(convert.MapToInterfaceMap(j.Labels), types.String),
					"annotations":        llx.MapData(convert.MapToInterfaceMap(j.Annotations), types.String),
					"created":            llx.TimeData(j.CreateTime.AsTime()),
					"updated":            llx.TimeData(j.UpdateTime.AsTime()),
					"deleted":            llx.TimeData(j.DeleteTime.AsTime()),
					"expired":            llx.TimeData(j.ExpireTime.AsTime()),
					"creator":            llx.StringData(j.Creator),
					"lastModifier":       llx.StringData(j.LastModifier),
					"client":             llx.StringData(j.Client),
					"clientVersion":      llx.StringData(j.ClientVersion),
					"launchStage":        llx.StringData(j.LaunchStage.String()),
					"template":           llx.ResourceData(mqlTemplate, "gcp.project.cloudRunService.job.executionTemplate"),
					"observedGeneration": llx.IntData(j.ObservedGeneration),
					"terminalCondition":  llx.ResourceData(mqlTerminalCondition, "gcp.project.cloudRunService.condition"),
					"conditions":         llx.ArrayData(mqlConditions, types.Resource("gcp.project.cloudRunService.condition")),
					"executionCount":     llx.IntData(int64(j.ExecutionCount)),
					"reconciling":        llx.BoolData(j.Reconciling),
				})
				if err != nil {
					log.Error().Err(err).Send()
					return
				}
				mux.Lock()
				jobs = append(jobs, mqlJob)
				mux.Unlock()
			}
		}(region.(string))
	}
	wg.Wait()
	return jobs, nil
}

func mqlContainerProbe(runtime *plugin.Runtime, probe *runpb.Probe, containerId string) (plugin.Resource, error) {
	if probe == nil {
		return nil, nil
	}
	var mqlHttpGet map[string]interface{}
	if httpGet := probe.GetHttpGet(); httpGet != nil {
		mqlHttpHeaders := make([]interface{}, 0, len(httpGet.HttpHeaders))
		for _, h := range httpGet.HttpHeaders {
			mqlHttpHeaders = append(mqlHttpHeaders, map[string]interface{}{
				"name":  h.Name,
				"value": h.Value,
			})
		}
		mqlHttpGet = map[string]interface{}{
			"path":        httpGet.Path,
			"httpHeaders": mqlHttpHeaders,
		}
	}

	var mqlTcpSocket map[string]interface{}
	if tcpSocket := probe.GetTcpSocket(); tcpSocket != nil {
		mqlTcpSocket = map[string]interface{}{
			"port": tcpSocket.Port,
		}
	}

	return CreateResource(runtime, "gcp.project.cloudRunService.container.probe", map[string]*llx.RawData{
		"id":                  llx.StringData(fmt.Sprintf("%s/livenessProbe", containerId)),
		"initialDelaySeconds": llx.IntData(int64(probe.InitialDelaySeconds)),
		"timeoutSeconds":      llx.IntData(int64(probe.TimeoutSeconds)),
		"periodSeconds":       llx.IntData(int64(probe.PeriodSeconds)),
		"failureThreshold":    llx.IntData(int64(probe.FailureThreshold)),
		"httpGet":             llx.DictData(mqlHttpGet),
		"tcpSocket":           llx.DictData(mqlTcpSocket),
	})
}

func mqlCondition(runtime *plugin.Runtime, c *runpb.Condition, parentId, suffix string) (plugin.Resource, error) {
	if c == nil {
		return nil, nil
	}
	return CreateResource(runtime, "gcp.project.cloudRunService.condition", map[string]*llx.RawData{
		"id":                 llx.StringData(fmt.Sprintf("%s/condition/%s", parentId, suffix)),
		"type":               llx.StringData(c.Type),
		"state":              llx.StringData(c.String()),
		"message":            llx.StringData(c.Message),
		"lastTransitionTime": llx.TimeData(c.LastTransitionTime.AsTime()),
		"severity":           llx.StringData(c.Severity.String()),
	})
}

func mqlVpcAccess(vpcAccess *runpb.VpcAccess) (map[string]interface{}, error) {
	type mqlVpcAccess struct {
		Connector string `json:"connector"`
		Egress    string `json:"egress"`
	}
	if vpcAccess == nil {
		return nil, nil
	}
	return convert.JsonToDict(mqlVpcAccess{
		Connector: vpcAccess.Connector,
		Egress:    vpcAccess.Egress.String(),
	})
}

func mqlContainers(runtime *plugin.Runtime, containers []*runpb.Container, templateId string) ([]interface{}, error) {
	mqlContainers := make([]interface{}, 0, len(containers))
	for _, c := range containers {
		mqlEnvs := make([]interface{}, 0, len(c.Env))
		for _, e := range c.Env {
			valueSource := e.GetValueSource()
			var mqlValueSource map[string]interface{}
			if valueSource != nil {
				mqlValueSource = map[string]interface{}{
					"secretKeyRef": map[string]interface{}{
						"secret":  valueSource.SecretKeyRef.Secret,
						"version": valueSource.SecretKeyRef.Version,
					},
				}
			}
			mqlEnvs = append(mqlEnvs, map[string]interface{}{
				"name":        e.Name,
				"value":       e.GetValue(),
				"valueSource": mqlValueSource,
			})
		}

		var mqlResources map[string]interface{}
		if c.Resources != nil {
			mqlResources = map[string]interface{}{
				"limits":  convert.MapToInterfaceMap(c.Resources.Limits),
				"cpuIdle": c.Resources.CpuIdle,
			}
		}

		mqlPorts := make([]interface{}, 0, len(c.Ports))
		for _, p := range c.Ports {
			mqlPorts = append(mqlPorts, map[string]interface{}{
				"name":          p.Name,
				"containerPort": p.ContainerPort,
			})
		}

		mqlVolumeMounts := make([]interface{}, 0, len(c.Ports))
		for _, v := range c.VolumeMounts {
			mqlVolumeMounts = append(mqlVolumeMounts, map[string]interface{}{
				"name":      v.Name,
				"mountPath": v.MountPath,
			})
		}

		containerId := fmt.Sprintf("%s/container/%s", templateId, c.Name)
		mqlLivenessProbe, err := mqlContainerProbe(runtime, c.LivenessProbe, containerId)
		if err != nil {
			return nil, err
		}

		mqlStartupProbe, err := mqlContainerProbe(runtime, c.StartupProbe, containerId)
		if err != nil {
			return nil, err
		}

		mqlContainer, err := CreateResource(runtime, "gcp.project.cloudRunService.container", map[string]*llx.RawData{
			"id":            llx.StringData(containerId),
			"name":          llx.StringData(c.Name),
			"image":         llx.StringData(c.Image),
			"command":       llx.ArrayData(convert.SliceAnyToInterface(c.Command), types.String),
			"args":          llx.ArrayData(convert.SliceAnyToInterface(c.Args), types.String),
			"env":           llx.ArrayData(mqlEnvs, types.Dict),
			"resources":     llx.DictData(mqlResources),
			"ports":         llx.ArrayData(mqlPorts, types.Dict),
			"volumeMounts":  llx.ArrayData(mqlVolumeMounts, types.Dict),
			"workingDir":    llx.StringData(c.WorkingDir),
			"livenessProbe": llx.ResourceData(mqlLivenessProbe, "gcp.project.cloudRunService.container.probe"),
			"startupProbe":  llx.ResourceData(mqlStartupProbe, "gcp.project.cloudRunService.container.probe"),
		})
		if err != nil {
			return nil, err
		}
		mqlContainers = append(mqlContainers, mqlContainer)
	}
	return mqlContainers, nil
}

func mqlVolumes(volumes []*runpb.Volume) []interface{} {
	mqlVolumes := make([]interface{}, 0, len(volumes))
	for _, v := range volumes {
		mqlVolumes = append(mqlVolumes, map[string]interface{}{
			"name": v.Name,
		})
	}
	return mqlVolumes
}
