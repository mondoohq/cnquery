package gcp

import (
	"context"
	"fmt"
	"sync"

	"cloud.google.com/go/longrunning/autogen/longrunningpb"
	run "cloud.google.com/go/run/apiv2"
	runpb "cloud.google.com/go/run/apiv2/runpb"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/resources/packs/core"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

func (g *mqlGcpProjectCloudrunService) id() (string, error) {
	projectId, err := g.ProjectId()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/gcp.project.cloudrunService", projectId), nil
}

func (g *mqlGcpProjectCloudrunService) init(args *resources.Args) (*resources.Args, GcpProjectCloudrunService, error) {
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

	return g.MotorRuntime.CreateResource("gcp.project.cloudrunService",
		"projectId", projectId,
	)
}

func (g *mqlGcpProjectCloudrunServiceOperation) id() (string, error) {
	projectId, err := g.ProjectId()
	if err != nil {
		return "", err
	}
	name, err := g.Name()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("gcp.project.cloudrunService.operation/%s/%s", projectId, name), nil
}

func (g *mqlGcpProjectCloudrunServiceService) id() (string, error) {
	projectId, err := g.ProjectId()
	if err != nil {
		return "", err
	}
	name, err := g.Name()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("gcp.project.cloudrunService.service/%s/%s", projectId, name), nil
}

func (g *mqlGcpProjectCloudrunServiceJob) id() (string, error) {
	projectId, err := g.ProjectId()
	if err != nil {
		return "", err
	}
	name, err := g.Name()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("gcp.project.cloudrunService.job/%s/%s", projectId, name), nil
}

func (g *mqlGcpProjectCloudrunServiceServiceRevisionTemplate) id() (string, error) {
	return g.Id()
}

func (g *mqlGcpProjectCloudrunServiceContainer) id() (string, error) {
	return g.Id()
}

func (g *mqlGcpProjectCloudrunServiceContainerProbe) id() (string, error) {
	return g.Id()
}

func (g *mqlGcpProjectCloudrunServiceCondition) id() (string, error) {
	return g.Id()
}

func (g *mqlGcpProjectCloudrunServiceJobExecutionTemplate) id() (string, error) {
	return g.Id()
}

func (g *mqlGcpProjectCloudrunServiceJobExecutionTemplateTaskTemplate) id() (string, error) {
	return g.Id()
}

func (g *mqlGcpProjectCloudrunService) GetRegions() ([]interface{}, error) {
	provider, err := gcpProvider(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	projectId, err := g.ProjectId()
	if err != nil {
		return nil, err
	}

	client, err := provider.Client(run.DefaultAuthScopes()...)
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

func (g *mqlGcpProjectCloudrunService) GetOperations() ([]interface{}, error) {
	projectId, err := g.ProjectId()
	if err != nil {
		return nil, err
	}

	regions, err := g.Regions()
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
				mqlOp, err := g.MotorRuntime.CreateResource("gcp.project.cloudrunService.operation",
					"projectId", projectId,
					"name", t.Name,
					"done", t.Done,
				)
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

func (g *mqlGcpProjectCloudrunService) GetServices() ([]interface{}, error) {
	projectId, err := g.ProjectId()
	if err != nil {
		return nil, err
	}

	regions, err := g.Regions()
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

				var mqlTemplate resources.ResourceType
				if s.Template != nil {
					var scalingCfg map[string]interface{}
					if s.Template.Scaling != nil {
						scalingCfg, err = core.JsonToDict(mqlRevisionScaling{
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

					templateId := fmt.Sprintf("gcp.project.cloudrunService.service/%s/%s/revisionTemplate", projectId, s.Name)
					mqlContainers, err := mqlContainers(g.MotorRuntime, s.Template.Containers, templateId)
					if err != nil {
						log.Error().Err(err).Send()
					}

					mqlTemplate, err = g.MotorRuntime.CreateResource("gcp.project.cloudrunService.service.revisionTemplate",
						"id", templateId,
						"name", s.Template.Revision,
						"labels", core.StrMapToInterface(s.Template.Labels),
						"annotations", core.StrMapToInterface(s.Template.Annotations),
						"scaling", scalingCfg,
						"vpcAccess", vpcCfg,
						"timeout", core.MqlTime(llx.DurationToTime((s.Template.Timeout.Seconds))),
						"serviceAccount", s.Template.ServiceAccount,
						"containers", mqlContainers,
						"volumes", mqlVolumes(s.Template.Volumes),
						"executionEnvironment", s.Template.ExecutionEnvironment.String(),
						"encryptionKey", s.Template.EncryptionKey,
						"maxInstanceRequestConcurrency", int64(s.Template.MaxInstanceRequestConcurrency),
					)
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

				serviceId := fmt.Sprintf("gcp.project.cloudrunService.service/%s/%s", projectId, s.Name)
				mqlTerminalCondition, err := mqlCondition(g.MotorRuntime, s.TerminalCondition, serviceId, "terminal")
				if err != nil {
					log.Error().Err(err).Send()
				}

				mqlConditions := make([]interface{}, 0, len(s.Conditions))
				for i, c := range s.Conditions {
					mqlCondition, err := mqlCondition(g.MotorRuntime, c, serviceId, fmt.Sprintf("%d", i))
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

				mqlS, err := g.MotorRuntime.CreateResource("gcp.project.cloudrunService.service",
					"projectId", projectId,
					"region", region,
					"name", s.Name,
					"description", s.Description,
					"generation", s.Generation,
					"labels", core.StrMapToInterface(s.Labels),
					"annotations", core.StrMapToInterface(s.Annotations),
					"created", core.MqlTime(s.CreateTime.AsTime()),
					"updated", core.MqlTime(s.UpdateTime.AsTime()),
					"deleted", core.MqlTime(s.DeleteTime.AsTime()),
					"expired", core.MqlTime(s.ExpireTime.AsTime()),
					"creator", s.Creator,
					"lastModifier", s.LastModifier,
					"client", s.Client,
					"clientVersion", s.ClientVersion,
					"ingress", s.Ingress.String(),
					"launchStage", s.LaunchStage.String(),
					"template", mqlTemplate,
					"traffic", mqlTraffic,
					"observedGeneration", s.ObservedGeneration,
					"terminalCondition", mqlTerminalCondition,
					"conditions", mqlConditions,
					"latestReadyRevision", s.LatestReadyRevision,
					"latestCreatedRevision", s.LatestCreatedRevision,
					"trafficStatuses", mqlTrafficStatuses,
					"uri", s.Uri,
					"reconciling", s.Reconciling,
				)
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

func (g *mqlGcpProjectCloudrunService) GetJobs() ([]interface{}, error) {
	projectId, err := g.ProjectId()
	if err != nil {
		return nil, err
	}

	regions, err := g.Regions()
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

				jobId := fmt.Sprintf("gcp.project.cloudrunService.job/%s/%s", projectId, j.Name)
				var mqlTemplate resources.ResourceType
				if j.Template != nil {
					templateId := fmt.Sprintf("%s/executionTemplate", jobId)
					var mqlTaskTemplate resources.ResourceType
					if j.Template.Template != nil {
						vpcAccess, err := mqlVpcAccess(j.Template.Template.VpcAccess)
						if err != nil {
							log.Error().Err(err).Send()
							return
						}

						mqlContainers, err := mqlContainers(g.MotorRuntime, j.Template.Template.Containers, templateId)
						if err != nil {
							log.Error().Err(err).Send()
							return
						}

						mqlTaskTemplate, err = g.MotorRuntime.CreateResource("gcp.project.cloudrunService.job.executionTemplate.taskTemplate",
							"id", fmt.Sprintf("%s/template", templateId),
							"vpcAccess", vpcAccess,
							"timeout", core.MqlTime(llx.DurationToTime((j.Template.Template.Timeout.Seconds))),
							"serviceAccount", j.Template.Template.ServiceAccount,
							"containers", mqlContainers,
							"volumes", mqlVolumes(j.Template.Template.Volumes),
							"executionEnvironment", j.Template.Template.ExecutionEnvironment.String(),
							"encryptionKey", j.Template.Template.EncryptionKey,
							"maxRetries", int64(j.Template.Template.GetMaxRetries()),
						)
						if err != nil {
							log.Error().Err(err).Send()
							return
						}
					}

					mqlTemplate, err = g.MotorRuntime.CreateResource("gcp.project.cloudrunService.job.executionTemplate",
						"id", templateId,
						"labels", core.StrMapToInterface(j.Template.Labels),
						"annotations", core.StrMapToInterface(j.Template.Annotations),
						"parallelism", int64(j.Template.Parallelism),
						"taskCount", int64(j.Template.TaskCount),
						"template", mqlTaskTemplate,
					)
					if err != nil {
						log.Error().Err(err).Send()
						return
					}
				}

				mqlTerminalCondition, err := mqlCondition(g.MotorRuntime, j.TerminalCondition, jobId, "terminal")
				if err != nil {
					log.Error().Err(err).Send()
					return
				}

				mqlConditions := make([]interface{}, 0, len(j.Conditions))
				for i, c := range j.Conditions {
					mqlCondition, err := mqlCondition(g.MotorRuntime, c, jobId, fmt.Sprintf("%d", i))
					if err != nil {
						log.Error().Err(err).Send()
						return
					}
					mqlConditions = append(mqlConditions, mqlCondition)
				}

				mqlJob, err := g.MotorRuntime.CreateResource("gcp.project.cloudrunService.job",
					"projectId", projectId,
					"region", region,
					"name", j.Name,
					"generation", j.Generation,
					"labels", core.StrMapToInterface(j.Labels),
					"annotations", core.StrMapToInterface(j.Annotations),
					"created", core.MqlTime(j.CreateTime.AsTime()),
					"updated", core.MqlTime(j.UpdateTime.AsTime()),
					"deleted", core.MqlTime(j.DeleteTime.AsTime()),
					"expired", core.MqlTime(j.ExpireTime.AsTime()),
					"creator", j.Creator,
					"lastModifier", j.LastModifier,
					"client", j.Client,
					"clientVersion", j.ClientVersion,
					"launchStage", j.LaunchStage.String(),
					"template", mqlTemplate,
					"observedGeneration", j.ObservedGeneration,
					"terminalCondition", mqlTerminalCondition,
					"conditions", mqlConditions,
					"executionCount", int64(j.ExecutionCount),
					"reconciling", j.Reconciling,
				)
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

func mqlContainerProbe(runtime *resources.Runtime, probe *runpb.Probe, containerId string) (resources.ResourceType, error) {
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

	return runtime.CreateResource("gcp.project.cloudrunService.container.probe",
		"id", fmt.Sprintf("%s/livenessProbe", containerId),
		"initialDelaySeconds", probe.InitialDelaySeconds,
		"timeoutSeconds", probe.TimeoutSeconds,
		"periodSeconds", probe.PeriodSeconds,
		"failureThreshold", probe.FailureThreshold,
		"httpGet", mqlHttpGet,
		"tcpSocket", mqlTcpSocket,
	)
}

func mqlCondition(runtime *resources.Runtime, c *runpb.Condition, parentId, suffix string) (resources.ResourceType, error) {
	if c == nil {
		return nil, nil
	}
	return runtime.CreateResource("gcp.project.cloudrunService.condition",
		"id", fmt.Sprintf("%s/condition/%s", parentId, suffix),
		"type", c.Type,
		"state", c.String(),
		"message", c.Message,
		"lastTransitionTime", core.MqlTime(c.LastTransitionTime.AsTime()),
		"severity", c.Severity.String(),
	)
}

func mqlVpcAccess(vpcAccess *runpb.VpcAccess) (map[string]interface{}, error) {
	type mqlVpcAccess struct {
		Connector string `json:"connector"`
		Egress    string `json:"egress"`
	}
	if vpcAccess == nil {
		return nil, nil
	}
	return core.JsonToDict(mqlVpcAccess{
		Connector: vpcAccess.Connector,
		Egress:    vpcAccess.Egress.String(),
	})
}

func mqlContainers(runtime *resources.Runtime, containers []*runpb.Container, templateId string) ([]interface{}, error) {
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
				"limits":  core.StrMapToInterface(c.Resources.Limits),
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

		mqlContainer, err := runtime.CreateResource("gcp.project.cloudrunService.container",
			"id", containerId,
			"name", c.Name,
			"image", c.Image,
			"command", core.StrSliceToInterface(c.Command),
			"args", core.StrSliceToInterface(c.Args),
			"env", mqlEnvs,
			"resources", mqlResources,
			"ports", mqlPorts,
			"volumeMounts", mqlVolumeMounts,
			"workingDir", c.WorkingDir,
			"livenessProbe", mqlLivenessProbe,
			"startupProbe", mqlStartupProbe,
		)
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
