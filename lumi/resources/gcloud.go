package resources

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/lumi"
	"go.mondoo.io/mondoo/motor/providers"
	gcp_transport "go.mondoo.io/mondoo/motor/providers/gcp"
	"google.golang.org/api/cloudresourcemanager/v1"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/iam/v1"
	"google.golang.org/api/option"
	sqladmin "google.golang.org/api/sqladmin/v1beta4"
	"google.golang.org/api/storage/v1"
)

func gcptransport(t providers.Transport) (*gcp_transport.Provider, error) {
	gt, ok := t.(*gcp_transport.Provider)
	if !ok {
		return nil, errors.New("aws resource is not supported on this transport")
	}
	return gt, nil
}

func (g *lumiGcloudOrganization) id() (string, error) {
	return "gcloud.organization", nil
}

func (g *lumiGcloudOrganization) init(args *lumi.Args) (*lumi.Args, GcloudOrganization, error) {
	if len(*args) > 2 {
		return args, nil, nil
	}

	gt, err := gcptransport(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, nil, err
	}

	client, err := gt.Client(cloudresourcemanager.CloudPlatformReadOnlyScope, iam.CloudPlatformScope, compute.CloudPlatformScope)
	if err != nil {
		return nil, nil, err
	}

	ctx := context.Background()
	svc, err := cloudresourcemanager.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, nil, err
	}

	// determine org from project in transport
	orgId, err := gt.OrganizationID()
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

func (g *lumiGcloudOrganization) GetId() (string, error) {
	// placeholder to convince lumi that this is an optional field
	// should never be called since the data is initialized in init
	return "", errors.New("not implemented")
}

func (g *lumiGcloudOrganization) GetName() (string, error) {
	// placeholder to convince lumi that this is an optional field
	// should never be called since the data is initialized in init
	return "", errors.New("not implemented")
}

func (g *lumiGcloudOrganization) GetLifecycleState() (string, error) {
	// placeholder to convince lumi that this is an optional field
	// should never be called since the data is initialized in init
	return "", errors.New("not implemented")
}

func (g *lumiGcloudOrganization) GetIamPolicy() ([]interface{}, error) {
	gt, err := gcptransport(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	client, err := gt.Client(cloudresourcemanager.CloudPlatformReadOnlyScope, iam.CloudPlatformScope, compute.CloudPlatformScope)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	svc, err := cloudresourcemanager.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}

	// determine org from project in transport
	orgId, err := gt.OrganizationID()
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

		lumiServiceaccount, err := g.MotorRuntime.CreateResource("gcloud.resourcemanager.binding",
			"id", name+"-"+strconv.Itoa(i),
			"role", b.Role,
			"members", strSliceToInterface(b.Members),
		)
		if err != nil {
			return nil, err
		}
		res = append(res, lumiServiceaccount)
	}

	return res, nil
}

func (g *lumiGcloudProject) id() (string, error) {
	return "gcloud.project", nil
}

func (g *lumiGcloudProject) init(args *lumi.Args) (*lumi.Args, GcloudProject, error) {
	if len(*args) > 2 {
		return args, nil, nil
	}

	gt, err := gcptransport(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, nil, err
	}

	client, err := gt.Client(cloudresourcemanager.CloudPlatformReadOnlyScope, iam.CloudPlatformScope, compute.CloudPlatformScope)
	if err != nil {
		return nil, nil, err
	}

	ctx := context.Background()
	svc, err := cloudresourcemanager.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, nil, err
	}

	projectName := gt.ResourceID()
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
	(*args)["labels"] = strMapToInterface(project.Labels)
	// TODO: add organization gcloud.organization
	return args, nil, nil
}

func (g *lumiGcloudProject) GetId() (string, error) {
	// placeholder to convince lumi that this is an optional field
	// should never be called since the data is initialized in init
	return "", errors.New("not implemented")
}

func (g *lumiGcloudProject) GetName() (string, error) {
	// placeholder to convince lumi that this is an optional field
	// should never be called since the data is initialized in init
	return "", errors.New("not implemented")
}

func (g *lumiGcloudProject) GetNumber() (string, error) {
	// placeholder to convince lumi that this is an optional field
	// should never be called since the data is initialized in init
	return "", errors.New("not implemented")
}

func (g *lumiGcloudProject) GetLifecycleState() (string, error) {
	// placeholder to convince lumi that this is an optional field
	// should never be called since the data is initialized in init
	return "", errors.New("not implemented")
}

func (g *lumiGcloudProject) GetCreateTime() (string, error) {
	// placeholder to convince lumi that this is an optional field
	// should never be called since the data is initialized in init
	return "", errors.New("not implemented")
}

func (g *lumiGcloudProject) GetLabels() (map[string]interface{}, error) {
	// placeholder to convince lumi that this is an optional field
	// should never be called since the data is initialized in init
	return nil, errors.New("not implemented")
}

func (g *lumiGcloudProject) GetIamPolicy() ([]interface{}, error) {
	projectId, err := g.Id()
	if err != nil {
		return nil, err
	}

	gt, err := gcptransport(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	client, err := gt.Client(cloudresourcemanager.CloudPlatformReadOnlyScope, iam.CloudPlatformScope, compute.CloudPlatformScope)
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

		lumiServiceaccount, err := g.MotorRuntime.CreateResource("gcloud.resourcemanager.binding",
			"id", projectId+"-"+strconv.Itoa(i),
			"role", b.Role,
			"members", strSliceToInterface(b.Members),
		)
		if err != nil {
			return nil, err
		}
		res = append(res, lumiServiceaccount)
	}

	return res, nil
}

func (g *lumiGcloudResourcemanagerBinding) id() (string, error) {
	return g.Id()
}

func (g *lumiGcloudCompute) id() (string, error) {
	return "gcloud.compute", nil
}

func (g *lumiGcloudCompute) GetInstances() ([]interface{}, error) {
	gt, err := gcptransport(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	client, err := gt.Client(cloudresourcemanager.CloudPlatformReadOnlyScope, iam.CloudPlatformScope, compute.CloudPlatformScope)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	computeSvc, err := compute.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}

	projectName := gt.ResourceID()

	// TODO: iterate over all instances
	// TODO: harmonize instance list with discovery?, at least borrow the parallel execution
	instances, err := computeSvc.Instances.List(projectName, "us-central1-a").Do()
	if err != nil {
		return nil, err
	}

	res := []interface{}{}

	for i := range instances.Items {
		instance := instances.Items[i]

		metadata := map[string]string{}
		for m := range instance.Metadata.Items {
			item := instance.Metadata.Items[m]
			metadata[item.Key] = toString(item.Value)
		}

		lumiServiceAccounts := []interface{}{}
		for i := range instance.ServiceAccounts {
			sa := instance.ServiceAccounts[i]

			lumiServiceaccount, err := g.MotorRuntime.CreateResource("gcloud.compute.serviceaccount",
				"email", sa.Email,
				"scopes", strSliceToInterface(sa.Scopes),
			)
			if err != nil {
				return nil, err
			}
			lumiServiceAccounts = append(lumiServiceAccounts, lumiServiceaccount)
		}

		lumiInstance, err := g.MotorRuntime.CreateResource("gcloud.compute.instance",
			"id", strconv.FormatUint(instance.Id, 10),
			"name", instance.Name,
			"cpuPlatform", instance.CpuPlatform,
			"deletionProtection", instance.DeletionProtection,
			"description", instance.Description,
			"hostname", instance.Hostname,
			"labels", strMapToInterface(instance.Labels),
			"status", instance.Status,
			"statusMessage", instance.StatusMessage,
			"tags", strSliceToInterface(instance.Tags.Items),
			"metadata", strMapToInterface(metadata),
			"serviceAccounts", lumiServiceAccounts,
		)
		if err != nil {
			return nil, err
		}
		res = append(res, lumiInstance)
	}

	return res, nil
}

func (g *lumiGcloudComputeInstance) id() (string, error) {
	return g.Id()
}

func (g *lumiGcloudComputeServiceaccount) id() (string, error) {
	return g.Email()
}

func (g *lumiGcloudStorage) id() (string, error) {
	return "gcloud.storage", nil
}

func (g *lumiGcloudStorage) GetBuckets() ([]interface{}, error) {
	gt, err := gcptransport(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	client, err := gt.Client(cloudresourcemanager.CloudPlatformReadOnlyScope, iam.CloudPlatformScope, storage.CloudPlatformScope)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	storageSvc, err := storage.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}

	projectID := gt.ResourceID()
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

		lumiInstance, err := g.MotorRuntime.CreateResource("gcloud.storage.bucket",
			"id", bucket.Id,
			"name", bucket.Name,
			"kind", bucket.Kind,
			"labels", strMapToInterface(bucket.Labels),
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
		res = append(res, lumiInstance)
	}

	return res, nil
}

func (g *lumiGcloudStorageBucket) id() (string, error) {
	return g.Name()
}

func (g *lumiGcloudStorageBucket) GetIamPolicy() ([]interface{}, error) {
	bucketName, err := g.Name()
	if err != nil {
		return nil, err
	}

	gt, err := gcptransport(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	client, err := gt.Client(cloudresourcemanager.CloudPlatformReadOnlyScope, iam.CloudPlatformScope, storage.CloudPlatformScope)
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

		lumiServiceaccount, err := g.MotorRuntime.CreateResource("gcloud.resourcemanager.binding",
			"id", bucketName+"-"+strconv.Itoa(i),
			"role", b.Role,
			"members", strSliceToInterface(b.Members),
		)
		if err != nil {
			return nil, err
		}
		res = append(res, lumiServiceaccount)
	}

	return res, nil
}

func (g *lumiGcloudSql) id() (string, error) {
	return "gcloud.sql", nil
}

func (g *lumiGcloudSql) GetInstances() ([]interface{}, error) {
	gt, err := gcptransport(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	client, err := gt.Client(cloudresourcemanager.CloudPlatformReadOnlyScope, iam.CloudPlatformScope, sqladmin.CloudPlatformScope)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	sqladminSvc, err := sqladmin.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}

	projectName := gt.ResourceID()
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

		lumiInstance, err := g.MotorRuntime.CreateResource("gcloud.sql.instance",
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
		res = append(res, lumiInstance)
	}

	return res, nil
}

func (g *lumiGcloudSqlInstance) id() (string, error) {
	// TODO: instances are scoped in project
	return g.Name()
}
