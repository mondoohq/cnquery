package gcp

import (
	"context"
	"errors"
	"strconv"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/motor/providers"
	gcp_provider "go.mondoo.com/cnquery/motor/providers/gcp"
	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/resources/packs/core"
	"google.golang.org/api/cloudresourcemanager/v1"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/iam/v1"
	"google.golang.org/api/option"
	sqladmin "google.golang.org/api/sqladmin/v1beta4"
	"google.golang.org/api/storage/v1"
)

func gcpProvider(t providers.Instance) (*gcp_provider.Provider, error) {
	provider, ok := t.(*gcp_provider.Provider)
	if !ok {
		return nil, errors.New("aws resource is not supported on this transport")
	}
	return provider, nil
}

func (g *mqlGcloudOrganization) id() (string, error) {
	return "gcloud.organization", nil
}

func (g *mqlGcloudOrganization) init(args *resources.Args) (*resources.Args, GcloudOrganization, error) {
	if len(*args) > 2 {
		return args, nil, nil
	}

	provider, err := gcpProvider(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, nil, err
	}

	client, err := provider.Client(cloudresourcemanager.CloudPlatformReadOnlyScope, iam.CloudPlatformScope, compute.CloudPlatformScope)
	if err != nil {
		return nil, nil, err
	}

	ctx := context.Background()
	svc, err := cloudresourcemanager.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, nil, err
	}

	// determine org from project in transport
	orgId, err := provider.OrganizationID()
	if err != nil {
		log.Error().Err(err).Msg("could not determine organization id")
		return nil, nil, err
	}

	name := "organizations/" + orgId
	org, err := svc.Organizations.Get(name).Do()
	if err != nil {
		return nil, nil, err
	}

	(*args)["id"] = org.Name
	(*args)["name"] = org.DisplayName
	(*args)["lifecycleState"] = org.LifecycleState

	return args, nil, nil
}

func (g *mqlGcloudOrganization) GetId() (string, error) {
	// placeholder to convince MQL that this is an optional field
	// should never be called since the data is initialized in init
	return "", errors.New("not implemented")
}

func (g *mqlGcloudOrganization) GetName() (string, error) {
	// placeholder to convince MQL that this is an optional field
	// should never be called since the data is initialized in init
	return "", errors.New("not implemented")
}

func (g *mqlGcloudOrganization) GetLifecycleState() (string, error) {
	// placeholder to convince MQL that this is an optional field
	// should never be called since the data is initialized in init
	return "", errors.New("not implemented")
}

func (g *mqlGcloudOrganization) GetIamPolicy() ([]interface{}, error) {
	provider, err := gcpProvider(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	client, err := provider.Client(cloudresourcemanager.CloudPlatformReadOnlyScope, iam.CloudPlatformScope, compute.CloudPlatformScope)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	svc, err := cloudresourcemanager.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}

	// determine org from project in transport
	orgId, err := provider.OrganizationID()
	if err != nil {
		return nil, err
	}

	name := "organizations/" + orgId
	orgpolicy, err := svc.Organizations.GetIamPolicy(name, &cloudresourcemanager.GetIamPolicyRequest{}).Do()
	if err != nil {
		return nil, err
	}

	res := []interface{}{}
	for i := range orgpolicy.Bindings {
		b := orgpolicy.Bindings[i]

		mqlServiceaccount, err := g.MotorRuntime.CreateResource("gcloud.resourcemanager.binding",
			"id", name+"-"+strconv.Itoa(i),
			"role", b.Role,
			"members", core.StrSliceToInterface(b.Members),
		)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlServiceaccount)
	}

	return res, nil
}

func (g *mqlGcloudProject) id() (string, error) {
	return "gcloud.project", nil
}

func (g *mqlGcloudProject) init(args *resources.Args) (*resources.Args, GcloudProject, error) {
	if len(*args) > 2 {
		return args, nil, nil
	}

	provider, err := gcpProvider(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, nil, err
	}

	client, err := provider.Client(cloudresourcemanager.CloudPlatformReadOnlyScope, iam.CloudPlatformScope, compute.CloudPlatformScope)
	if err != nil {
		return nil, nil, err
	}

	ctx := context.Background()
	svc, err := cloudresourcemanager.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, nil, err
	}

	projectName := provider.ResourceID()
	project, err := svc.Projects.Get(projectName).Do()
	if err != nil {
		return nil, nil, err
	}

	(*args)["id"] = project.ProjectId
	(*args)["name"] = project.Name
	(*args)["number"] = strconv.FormatInt(project.ProjectNumber, 10)
	(*args)["lifecycleState"] = project.LifecycleState
	var createTime *time.Time
	parsedTime, err := time.Parse(time.RFC3339, project.CreateTime)
	if err != nil {
		return nil, nil, errors.New("could not parse gcloud.project create time: " + project.CreateTime)
	} else {
		createTime = &parsedTime
	}
	(*args)["createTime"] = createTime
	(*args)["labels"] = core.StrMapToInterface(project.Labels)
	// TODO: add organization gcloud.organization
	return args, nil, nil
}

func (g *mqlGcloudProject) GetId() (string, error) {
	// placeholder to convince MQL that this is an optional field
	// should never be called since the data is initialized in init
	return "", errors.New("not implemented")
}

func (g *mqlGcloudProject) GetName() (string, error) {
	// placeholder to convince MQL that this is an optional field
	// should never be called since the data is initialized in init
	return "", errors.New("not implemented")
}

func (g *mqlGcloudProject) GetNumber() (string, error) {
	// placeholder to convince MQL that this is an optional field
	// should never be called since the data is initialized in init
	return "", errors.New("not implemented")
}

func (g *mqlGcloudProject) GetLifecycleState() (string, error) {
	// placeholder to convince MQL that this is an optional field
	// should never be called since the data is initialized in init
	return "", errors.New("not implemented")
}

func (g *mqlGcloudProject) GetCreateTime() (string, error) {
	// placeholder to convince MQL that this is an optional field
	// should never be called since the data is initialized in init
	return "", errors.New("not implemented")
}

func (g *mqlGcloudProject) GetLabels() (map[string]interface{}, error) {
	// placeholder to convince MQL that this is an optional field
	// should never be called since the data is initialized in init
	return nil, errors.New("not implemented")
}

func (g *mqlGcloudProject) GetIamPolicy() ([]interface{}, error) {
	projectId, err := g.Id()
	if err != nil {
		return nil, err
	}

	provider, err := gcpProvider(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	client, err := provider.Client(cloudresourcemanager.CloudPlatformReadOnlyScope, iam.CloudPlatformScope, compute.CloudPlatformScope)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	svc, err := cloudresourcemanager.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}

	policy, err := svc.Projects.GetIamPolicy(projectId, &cloudresourcemanager.GetIamPolicyRequest{}).Do()
	if err != nil {
		return nil, err
	}

	res := []interface{}{}
	for i := range policy.Bindings {
		b := policy.Bindings[i]

		mqlServiceaccount, err := g.MotorRuntime.CreateResource("gcloud.resourcemanager.binding",
			"id", projectId+"-"+strconv.Itoa(i),
			"role", b.Role,
			"members", core.StrSliceToInterface(b.Members),
		)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlServiceaccount)
	}

	return res, nil
}

func (g *mqlGcloudResourcemanagerBinding) id() (string, error) {
	return g.Id()
}

func (g *mqlGcloudCompute) id() (string, error) {
	return "gcloud.compute", nil
}

func (g *mqlGcloudCompute) GetInstances() ([]interface{}, error) {
	provider, err := gcpProvider(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	client, err := provider.Client(cloudresourcemanager.CloudPlatformReadOnlyScope, iam.CloudPlatformScope, compute.CloudPlatformScope)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	computeSvc, err := compute.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}

	projectName := provider.ResourceID()

	var wg sync.WaitGroup
	zones, err := computeSvc.Zones.List(projectName).Do()
	if err != nil {
		return nil, err
	}
	res := []interface{}{}
	wg.Add(len(zones.Items))
	mux := &sync.Mutex{}

	// TODO:harmonize instance list with discovery?
	for _, z := range zones.Items {
		go func(svc *compute.Service, project string, zoneName string) {
			instances, err := computeSvc.Instances.List(projectName, zoneName).Do()
			if err == nil {
				mux.Lock()
				for i := range instances.Items {
					instance := instances.Items[i]

					metadata := map[string]string{}
					for m := range instance.Metadata.Items {
						item := instance.Metadata.Items[m]
						metadata[item.Key] = core.ToString(item.Value)
					}

					mqlServiceAccounts := []interface{}{}
					for i := range instance.ServiceAccounts {
						sa := instance.ServiceAccounts[i]

						mqlServiceaccount, err := g.MotorRuntime.CreateResource("gcloud.compute.serviceaccount",
							"email", sa.Email,
							"scopes", core.StrSliceToInterface(sa.Scopes),
						)
						if err == nil {
							mqlServiceAccounts = append(mqlServiceAccounts, mqlServiceaccount)
						}
					}

					mqlInstance, err := g.MotorRuntime.CreateResource("gcloud.compute.instance",
						"id", strconv.FormatUint(instance.Id, 10),
						"name", instance.Name,
						"cpuPlatform", instance.CpuPlatform,
						"deletionProtection", instance.DeletionProtection,
						"description", instance.Description,
						"hostname", instance.Hostname,
						"labels", core.StrMapToInterface(instance.Labels),
						"status", instance.Status,
						"statusMessage", instance.StatusMessage,
						"tags", core.StrSliceToInterface(instance.Tags.Items),
						"metadata", core.StrMapToInterface(metadata),
						"serviceAccounts", mqlServiceAccounts,
					)
					if err == nil {
						res = append(res, mqlInstance)
					}
				}
				mux.Unlock()
			}
			wg.Done()
		}(computeSvc, projectName, z.Name)
	}

	wg.Wait()
	return res, nil
}

func (g *mqlGcloudComputeInstance) id() (string, error) {
	return g.Id()
}

func (g *mqlGcloudComputeServiceaccount) id() (string, error) {
	return g.Email()
}

func (g *mqlGcloudStorage) id() (string, error) {
	return "gcloud.storage", nil
}

func (g *mqlGcloudStorage) GetBuckets() ([]interface{}, error) {
	provider, err := gcpProvider(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	client, err := provider.Client(cloudresourcemanager.CloudPlatformReadOnlyScope, iam.CloudPlatformScope, storage.CloudPlatformScope)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	storageSvc, err := storage.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}

	projectID := provider.ResourceID()
	buckets, err := storageSvc.Buckets.List(projectID).Do()
	if err != nil {
		return nil, err
	}

	res := []interface{}{}

	for i := range buckets.Items {
		bucket := buckets.Items[i]

		var created *time.Time
		// parse created and updated time properly "2019-06-12T21:14:13.190Z"
		parsedCreated, err := time.Parse(time.RFC3339, bucket.TimeCreated)
		if err != nil {
			return nil, err
		}
		created = &parsedCreated

		var updated *time.Time
		parsedUpdated, err := time.Parse(time.RFC3339, bucket.Updated)
		if err != nil {
			return nil, err
		}
		updated = &parsedUpdated

		iamConfigurationDict := map[string]interface{}{}

		if bucket.IamConfiguration != nil {
			iamConfiguration := bucket.IamConfiguration

			if iamConfiguration.BucketPolicyOnly != nil {
				var parsedLockTime time.Time
				if iamConfiguration.BucketPolicyOnly.LockedTime != "" {
					parsedLockTime, err = time.Parse(time.RFC3339, iamConfiguration.BucketPolicyOnly.LockedTime)
					if err != nil {
						return nil, err
					}
				}

				iamConfigurationDict["BucketPolicyOnly"] = map[string]interface{}{
					"enabled":    iamConfiguration.BucketPolicyOnly.Enabled,
					"lockedTime": parsedLockTime,
				}
			}

			if iamConfiguration.UniformBucketLevelAccess != nil {
				var parsedLockTime time.Time
				if iamConfiguration.UniformBucketLevelAccess.LockedTime != "" {
					parsedLockTime, err = time.Parse(time.RFC3339, iamConfiguration.UniformBucketLevelAccess.LockedTime)
					if err != nil {
						return nil, err
					}
				}

				iamConfigurationDict["UniformBucketLevelAccess"] = map[string]interface{}{
					"enabled":    iamConfiguration.UniformBucketLevelAccess.Enabled,
					"lockedTime": parsedLockTime,
				}
			}
		}

		mqlInstance, err := g.MotorRuntime.CreateResource("gcloud.storage.bucket",
			"id", bucket.Id,
			"name", bucket.Name,
			"kind", bucket.Kind,
			"labels", core.StrMapToInterface(bucket.Labels),
			"location", bucket.Location,
			"locationType", bucket.LocationType,
			"projectNumber", strconv.FormatUint(bucket.ProjectNumber, 10),
			"storageClass", bucket.StorageClass,
			"created", created,
			"updated", updated,
			"iamConfiguration", iamConfigurationDict,
		)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlInstance)
	}

	return res, nil
}

func (g *mqlGcloudStorageBucket) id() (string, error) {
	return g.Name()
}

func (g *mqlGcloudStorageBucket) GetIamPolicy() ([]interface{}, error) {
	bucketName, err := g.Name()
	if err != nil {
		return nil, err
	}

	provider, err := gcpProvider(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	client, err := provider.Client(cloudresourcemanager.CloudPlatformReadOnlyScope, iam.CloudPlatformScope, storage.CloudPlatformScope)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	storeSvc, err := storage.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}

	policy, err := storeSvc.Buckets.GetIamPolicy(bucketName).Do()
	if err != nil {
		return nil, err
	}

	res := []interface{}{}
	for i := range policy.Bindings {
		b := policy.Bindings[i]

		mqlServiceaccount, err := g.MotorRuntime.CreateResource("gcloud.resourcemanager.binding",
			"id", bucketName+"-"+strconv.Itoa(i),
			"role", b.Role,
			"members", core.StrSliceToInterface(b.Members),
		)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlServiceaccount)
	}

	return res, nil
}

func (g *mqlGcloudSql) id() (string, error) {
	return "gcloud.sql", nil
}

func (g *mqlGcloudSql) GetInstances() ([]interface{}, error) {
	provider, err := gcpProvider(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	client, err := provider.Client(cloudresourcemanager.CloudPlatformReadOnlyScope, iam.CloudPlatformScope, sqladmin.CloudPlatformScope)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	sqladminSvc, err := sqladmin.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}

	projectName := provider.ResourceID()
	sqlinstances, err := sqladminSvc.Instances.List(projectName).Do()
	if err != nil {
		return nil, err
	}

	res := []interface{}{}

	for i := range sqlinstances.Items {
		instance := sqlinstances.Items[i]

		settingsDict := map[string]interface{}{}
		if instance.Settings != nil {
			settings := instance.Settings
			if settings.DatabaseFlags != nil {
				dbFlags := map[string]interface{}{}
				for di := range settings.DatabaseFlags {
					flag := settings.DatabaseFlags[di]
					dbFlags[flag.Name] = flag.Value
				}
				settingsDict["databaseFlags"] = dbFlags
			}

			if settings.IpConfiguration != nil {
				ipConfig := map[string]interface{}{}

				ipConfig["ipv4Enabled"] = settings.IpConfiguration.Ipv4Enabled
				ipConfig["requireSsl"] = settings.IpConfiguration.RequireSsl
				ipConfig["privateNetwork"] = settings.IpConfiguration.PrivateNetwork

				authorizedNetworks := []interface{}{}
				for ani := range settings.IpConfiguration.AuthorizedNetworks {
					aclEntry := settings.IpConfiguration.AuthorizedNetworks[ani]

					authorizedNetworks = append(authorizedNetworks, map[string]interface{}{
						"name":           aclEntry.Name,
						"value":          aclEntry.Value,
						"kind":           aclEntry.Kind,
						"expirationTime": aclEntry.ExpirationTime,
					})
				}
				ipConfig["authorizedNetworks"] = authorizedNetworks

				settingsDict["ipConfiguration"] = ipConfig
			}

			// TODO: handle all other database settings
		}

		mqlInstance, err := g.MotorRuntime.CreateResource("gcloud.sql.instance",
			"name", instance.Name,
			"backendType", instance.BackendType,
			"connectionName", instance.ConnectionName,
			"databaseVersion", instance.DatabaseVersion,
			"gceZone", instance.GceZone,
			"instanceType", instance.InstanceType,
			"kind", instance.Kind,
			"currentDiskSize", instance.CurrentDiskSize,
			"maxDiskSize", instance.MaxDiskSize,
			"state", instance.State,
			// ref project
			"project", instance.Project,
			"region", instance.Region,
			"serviceAccountEmailAddress", instance.ServiceAccountEmailAddress,
			"settings", settingsDict,
		)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlInstance)
	}

	return res, nil
}

func (g *mqlGcloudSqlInstance) id() (string, error) {
	// TODO: instances are scoped in project
	return g.Name()
}
