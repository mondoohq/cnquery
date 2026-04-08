// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eks"
	ekstypes "github.com/aws/aws-sdk-go-v2/service/eks/types"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/mql/v13/providers/aws/connection"
)

func (a *mqlAwsEks) id() (string, error) {
	return ResourceAwsEks, nil
}

func (a *mqlAwsEks) clusters() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getClusters(conn), 5)
	poolOfJobs.Run()

	// check for errors
	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	// get all the results
	for i := range poolOfJobs.Jobs {
		res = append(res, poolOfJobs.Jobs[i].Result.([]any)...)
	}

	return res, nil
}

func (a *mqlAwsEks) getClusters(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}} // return the error
	}
	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("eks>getClusters>calling aws with region %s", region)

			svc := conn.Eks(region)
			ctx := context.Background()
			res := []any{}

			paginator := eks.NewListClustersPaginator(svc, &eks.ListClustersInput{})
			for paginator.HasMorePages() {
				page, err := paginator.NextPage(ctx)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS API")
						return res, nil
					}
					return nil, err
				}

				for _, clusterName := range page.Clusters {
					clusterArn := fmt.Sprintf("arn:aws:eks:%s:%s:cluster/%s", region, conn.AccountId(), clusterName)

					// If tag filters are active, we need to describe the cluster to check tags.
					// Cache the response to avoid a redundant call in fetchDetail().
					var cachedDescribe *ekstypes.Cluster
					if conn.Filters.General.HasTags() {
						descResp, err := svc.DescribeCluster(ctx, &eks.DescribeClusterInput{Name: aws.String(clusterName)})
						if err != nil {
							return nil, err
						}
						if descResp == nil || descResp.Cluster == nil {
							continue
						}
						if conn.Filters.General.IsFilteredOutByTags(descResp.Cluster.Tags) {
							log.Debug().Str("cluster", clusterName).Msg("skipping eks cluster due to filters")
							continue
						}
						cachedDescribe = descResp.Cluster
						// Use the real ARN from the API (handles partitions correctly)
						if descResp.Cluster.Arn != nil {
							clusterArn = *descResp.Cluster.Arn
						}
					}

					args := map[string]*llx.RawData{
						"name":   llx.StringData(clusterName),
						"arn":    llx.StringData(clusterArn),
						"region": llx.StringData(region),
					}

					mqlCluster, err := CreateResource(a.MqlRuntime, ResourceAwsEksCluster, args)
					if err != nil {
						return nil, err
					}
					// If we already described the cluster for tag filtering, cache it
					// to avoid a redundant DescribeCluster call in fetchDetail()
					if cachedDescribe != nil {
						cast := mqlCluster.(*mqlAwsEksCluster)
						if err := cast.populateFromDescribe(cachedDescribe); err != nil {
							return nil, err
						}
					}
					res = append(res, mqlCluster)
				}
			}

			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

type mqlAwsEksClusterInternal struct {
	fetched  bool
	fetchErr error
	lock     sync.Mutex
}

func (a *mqlAwsEksCluster) fetchDetail() error {
	if a.fetched {
		return a.fetchErr
	}
	a.lock.Lock()
	defer a.lock.Unlock()
	if a.fetched {
		return a.fetchErr
	}

	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	ctx := context.Background()
	svc := conn.Eks(a.Region.Data)

	descResp, err := svc.DescribeCluster(ctx, &eks.DescribeClusterInput{
		Name: aws.String(a.Name.Data),
	})
	if err != nil {
		a.fetched = true
		a.fetchErr = err
		return err
	}
	if err := a.populateFromDescribe(descResp.Cluster); err != nil {
		a.fetched = true
		a.fetchErr = err
		return err
	}
	return nil
}

// populateFromDescribe sets all computed fields from a DescribeCluster response.
// Called from fetchDetail() and also from getClusters() when tag filtering is active
// (to avoid a redundant DescribeCluster call).
func (a *mqlAwsEksCluster) populateFromDescribe(cluster *ekstypes.Cluster) error {
	a.Tags = plugin.TValue[map[string]any]{Data: toInterfaceMap(cluster.Tags), State: plugin.StateIsSet}
	a.Endpoint = plugin.TValue[string]{Data: convert.ToValue(cluster.Endpoint), State: plugin.StateIsSet}
	a.Version = plugin.TValue[string]{Data: convert.ToValue(cluster.Version), State: plugin.StateIsSet}
	a.PlatformVersion = plugin.TValue[string]{Data: convert.ToValue(cluster.PlatformVersion), State: plugin.StateIsSet}
	a.Status = plugin.TValue[string]{Data: string(cluster.Status), State: plugin.StateIsSet}

	encryptionConfig, _ := convert.JsonToDictSlice(cluster.EncryptionConfig)
	a.EncryptionConfig = plugin.TValue[[]any]{Data: encryptionConfig, State: plugin.StateIsSet}

	logging, _ := convert.JsonToDict(cluster.Logging)
	a.Logging = plugin.TValue[any]{Data: logging, State: plugin.StateIsSet}

	kubernetesNetworkConfig, _ := convert.JsonToDict(cluster.KubernetesNetworkConfig)
	a.NetworkConfig = plugin.TValue[any]{Data: kubernetesNetworkConfig, State: plugin.StateIsSet}

	vpcConfig, _ := convert.JsonToDict(cluster.ResourcesVpcConfig)
	a.ResourcesVpcConfig = plugin.TValue[any]{Data: vpcConfig, State: plugin.StateIsSet}

	a.CreatedAt = plugin.TValue[*time.Time]{Data: cluster.CreatedAt, State: plugin.StateIsSet}

	supportType := ""
	if cluster.UpgradePolicy != nil {
		supportType = string(cluster.UpgradePolicy.SupportType)
	}
	a.SupportType = plugin.TValue[string]{Data: supportType, State: plugin.StateIsSet}

	authMode := ""
	if cluster.AccessConfig != nil {
		authMode = string(cluster.AccessConfig.AuthenticationMode)
	}
	a.AuthenticationMode = plugin.TValue[string]{Data: authMode, State: plugin.StateIsSet}

	if cluster.DeletionProtection != nil {
		a.DeletionProtection = plugin.TValue[bool]{Data: *cluster.DeletionProtection, State: plugin.StateIsSet}
	} else {
		a.DeletionProtection = plugin.TValue[bool]{State: plugin.StateIsSet | plugin.StateIsNull}
	}

	var endpointPublicAccess, endpointPrivateAccess bool
	publicAccessCidrs := []any{}
	if cluster.ResourcesVpcConfig != nil {
		endpointPublicAccess = cluster.ResourcesVpcConfig.EndpointPublicAccess
		endpointPrivateAccess = cluster.ResourcesVpcConfig.EndpointPrivateAccess
		for _, cidr := range cluster.ResourcesVpcConfig.PublicAccessCidrs {
			publicAccessCidrs = append(publicAccessCidrs, cidr)
		}
	}
	a.EndpointPublicAccess = plugin.TValue[bool]{Data: endpointPublicAccess, State: plugin.StateIsSet}
	a.EndpointPrivateAccess = plugin.TValue[bool]{Data: endpointPrivateAccess, State: plugin.StateIsSet}
	a.PublicAccessCidrs = plugin.TValue[[]any]{Data: publicAccessCidrs, State: plugin.StateIsSet}

	if cluster.RoleArn != nil {
		mqlIam, err := NewResource(a.MqlRuntime, ResourceAwsIamRole,
			map[string]*llx.RawData{"arn": llx.StringDataPtr(cluster.RoleArn)},
		)
		if err != nil {
			return err
		}
		a.IamRole = plugin.TValue[*mqlAwsIamRole]{Data: mqlIam.(*mqlAwsIamRole), State: plugin.StateIsSet}
	} else {
		a.IamRole = plugin.TValue[*mqlAwsIamRole]{State: plugin.StateIsSet | plugin.StateIsNull}
	}

	a.fetched = true
	return nil
}

func (a *mqlAwsEksCluster) tags() (map[string]any, error) {
	return nil, a.fetchDetail()
}

func (a *mqlAwsEksCluster) endpoint() (string, error) {
	return "", a.fetchDetail()
}

func (a *mqlAwsEksCluster) version() (string, error) {
	return "", a.fetchDetail()
}

func (a *mqlAwsEksCluster) platformVersion() (string, error) {
	return "", a.fetchDetail()
}

func (a *mqlAwsEksCluster) status() (string, error) {
	return "", a.fetchDetail()
}

func (a *mqlAwsEksCluster) encryptionConfig() ([]any, error) {
	return nil, a.fetchDetail()
}

func (a *mqlAwsEksCluster) logging() (map[string]any, error) {
	return nil, a.fetchDetail()
}

func (a *mqlAwsEksCluster) networkConfig() (map[string]any, error) {
	return nil, a.fetchDetail()
}

func (a *mqlAwsEksCluster) resourcesVpcConfig() (map[string]any, error) {
	return nil, a.fetchDetail()
}

func (a *mqlAwsEksCluster) createdAt() (*time.Time, error) {
	return nil, a.fetchDetail()
}

func (a *mqlAwsEksCluster) iamRole() (*mqlAwsIamRole, error) {
	if err := a.fetchDetail(); err != nil {
		return nil, err
	}
	return a.IamRole.Data, nil
}

func (a *mqlAwsEksCluster) supportType() (string, error) {
	return "", a.fetchDetail()
}

func (a *mqlAwsEksCluster) authenticationMode() (string, error) {
	return "", a.fetchDetail()
}

func (a *mqlAwsEksCluster) deletionProtection() (bool, error) {
	return false, a.fetchDetail()
}

func (a *mqlAwsEksCluster) endpointPublicAccess() (bool, error) {
	return false, a.fetchDetail()
}

func (a *mqlAwsEksCluster) endpointPrivateAccess() (bool, error) {
	return false, a.fetchDetail()
}

func (a *mqlAwsEksCluster) publicAccessCidrs() ([]any, error) {
	return nil, a.fetchDetail()
}

func initAwsEksCluster(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}

	if len(args) == 0 {
		if ids := getAssetIdentifier(runtime); ids != nil {
			args["name"] = llx.StringData(ids.name)
			args["arn"] = llx.StringData(ids.arn)
		}
	}

	if args["arn"] == nil {
		return nil, nil, errors.New("arn required to fetch eks cluster")
	}

	// load all eks clusters
	obj, err := CreateResource(runtime, ResourceAwsEks, map[string]*llx.RawData{})
	if err != nil {
		return nil, nil, err
	}

	eks := obj.(*mqlAwsEks)
	rawResources := eks.GetClusters()

	arnVal := args["arn"].Value.(string)
	for _, rawResource := range rawResources.Data {
		cluster := rawResource.(*mqlAwsEksCluster)
		if cluster.Arn.Data == arnVal {
			return args, cluster, nil
		}
	}
	return nil, nil, errors.New("eks cluster does not exist")
}

func (a *mqlAwsEksCluster) id() (string, error) {
	return a.Arn.Data, nil
}

func (a *mqlAwsEksCluster) nodeGroups() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	regionVal := a.Region.Data
	log.Debug().Msgf("eks>getNodegroups>calling aws with region %s", regionVal)

	svc := conn.Eks(regionVal)
	ctx := context.Background()
	res := []any{}

	paginator := eks.NewListNodegroupsPaginator(svc, &eks.ListNodegroupsInput{ClusterName: aws.String(a.Name.Data)})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			if Is400AccessDeniedError(err) {
				log.Warn().Str("region", regionVal).Msg("error accessing region for AWS API")
				return res, nil
			}
			return nil, err
		}

		for i := range page.Nodegroups {
			nodegroup := page.Nodegroups[i]
			args := map[string]*llx.RawData{
				"name":   llx.StringData(nodegroup),
				"region": llx.StringData(regionVal),
			}

			mqlNg, err := CreateResource(a.MqlRuntime, ResourceAwsEksNodegroup, args)
			if err != nil {
				return nil, err
			}
			mqlNg.(*mqlAwsEksNodegroup).clusterName = a.Name.Data
			mqlNg.(*mqlAwsEksNodegroup).region = regionVal
			res = append(res, mqlNg)
		}
	}
	return res, nil
}

type mqlAwsEksNodegroupInternal struct {
	details     *ekstypes.Nodegroup
	fetched     bool
	region      string
	lock        sync.Mutex
	clusterName string
}

func (a *mqlAwsEksNodegroup) id() (string, error) {
	return a.Arn.Data, nil
}

func (a *mqlAwsEksNodegroup) autoscalingGroups() ([]any, error) {
	ng, err := a.fetchDetails()
	if err != nil {
		return nil, err
	}
	if ng.Resources == nil || ng.Resources.AutoScalingGroups == nil {
		return nil, nil
	}
	res := []any{}
	for i := range ng.Resources.AutoScalingGroups {
		ag := ng.Resources.AutoScalingGroups[i]
		mqlAg, err := NewResource(a.MqlRuntime, ResourceAwsAutoscalingGroup,
			map[string]*llx.RawData{
				"name":   llx.StringDataPtr(ag.Name),
				"region": llx.StringData(a.region),
			})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlAg)
	}

	return res, nil
}

func (a *mqlAwsEksNodegroup) fetchDetails() (*ekstypes.Nodegroup, error) {
	if a.fetched {
		return a.details, nil
	}
	a.lock.Lock()
	defer a.lock.Unlock()
	if a.fetched {
		return a.details, nil
	}
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	ctx := context.Background()
	svc := conn.Eks(a.region)
	desc, err := svc.DescribeNodegroup(ctx, &eks.DescribeNodegroupInput{NodegroupName: aws.String(a.Name.Data), ClusterName: aws.String(a.clusterName)})
	if err != nil {
		return nil, err
	}
	a.details = desc.Nodegroup
	a.fetched = true
	return desc.Nodegroup, nil
}

func (a *mqlAwsEksNodegroup) arn() (string, error) {
	ng, err := a.fetchDetails()
	if err != nil {
		return "", err
	}
	return *ng.NodegroupArn, nil
}

func (a *mqlAwsEksNodegroup) capacityType() (string, error) {
	ng, err := a.fetchDetails()
	if err != nil {
		return "", err
	}
	return string(ng.CapacityType), nil
}

func (a *mqlAwsEksNodegroup) status() (string, error) {
	ng, err := a.fetchDetails()
	if err != nil {
		return "", err
	}
	return string(ng.Status), nil
}

func (a *mqlAwsEksNodegroup) amiType() (string, error) {
	ng, err := a.fetchDetails()
	if err != nil {
		return "", err
	}
	return string(ng.AmiType), nil
}

func (a *mqlAwsEksNodegroup) diskSize() (int64, error) {
	ng, err := a.fetchDetails()
	if err != nil {
		return 0, err
	}
	if ng.DiskSize == nil {
		a.DiskSize.State = plugin.StateIsNull | plugin.StateIsSet
		return 0, nil
	}
	return int64(*ng.DiskSize), nil
}

func (a *mqlAwsEksNodegroup) createdAt() (*time.Time, error) {
	ng, err := a.fetchDetails()
	if err != nil {
		return nil, err
	}
	return ng.CreatedAt, nil
}

func (a *mqlAwsEksNodegroup) modifiedAt() (*time.Time, error) {
	ng, err := a.fetchDetails()
	if err != nil {
		return nil, err
	}
	return ng.ModifiedAt, nil
}

func (a *mqlAwsEksNodegroup) scalingConfig() (map[string]any, error) {
	ng, err := a.fetchDetails()
	if err != nil {
		return nil, err
	}
	return convert.JsonToDict(ng.ScalingConfig)
}

func (a *mqlAwsEksNodegroup) warmPoolConfig() (map[string]any, error) {
	ng, err := a.fetchDetails()
	if err != nil {
		return nil, err
	}
	if ng.WarmPoolConfig == nil {
		return nil, nil
	}
	return convert.JsonToDict(ng.WarmPoolConfig)
}

func (a *mqlAwsEksNodegroup) instanceTypes() ([]any, error) {
	ng, err := a.fetchDetails()
	if err != nil {
		return nil, err
	}
	s := []any{}
	for i := range ng.InstanceTypes {
		s = append(s, ng.InstanceTypes[i])
	}
	return s, nil
}

func (a *mqlAwsEksNodegroup) labels() (map[string]any, error) {
	ng, err := a.fetchDetails()
	if err != nil {
		return nil, err
	}
	new := make(map[string]any)
	for k, v := range ng.Labels {
		new[k] = v
	}
	return new, nil
}

func (a *mqlAwsEksNodegroup) tags() (map[string]any, error) {
	ng, err := a.fetchDetails()
	if err != nil {
		return nil, err
	}
	new := make(map[string]any)
	for k, v := range ng.Tags {
		new[k] = v
	}
	return new, nil
}

func (a *mqlAwsEksNodegroup) nodeRole() (*mqlAwsIamRole, error) {
	ng, err := a.fetchDetails()
	if err != nil {
		return nil, err
	}
	if ng.NodeRole == nil {
		a.NodeRole.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	mqlIam, err := NewResource(a.MqlRuntime, ResourceAwsIamRole,
		map[string]*llx.RawData{
			"arn": llx.StringDataPtr(ng.NodeRole),
		})
	if err != nil {
		return nil, err
	}
	return mqlIam.(*mqlAwsIamRole), nil
}

// AwsEksAddons
func (a *mqlAwsEksCluster) addons() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	regionVal := a.Region.Data
	log.Debug().Msgf("eks>getAddons>calling aws with region %s", regionVal)

	svc := conn.Eks(regionVal)
	ctx := context.Background()
	res := []any{}

	paginator := eks.NewListAddonsPaginator(svc, &eks.ListAddonsInput{ClusterName: aws.String(a.Name.Data)})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			if Is400AccessDeniedError(err) {
				log.Warn().Str("region", regionVal).Msg("error accessing region for AWS API")
				return res, nil
			}
			return nil, err
		}

		for i := range page.Addons {
			addon := page.Addons[i]
			args := map[string]*llx.RawData{
				"__id":   llx.StringData(fmt.Sprintf("%s/%s/%s", ResourceAwsEksAddon, a.Name.Data, addon)),
				"name":   llx.StringData(addon),
				"region": llx.StringData(regionVal),
			}

			mqlNg, err := CreateResource(a.MqlRuntime, ResourceAwsEksAddon, args)
			if err != nil {
				return nil, err
			}
			mqlNg.(*mqlAwsEksAddon).clusterName = a.Name.Data
			mqlNg.(*mqlAwsEksAddon).region = regionVal
			res = append(res, mqlNg)
		}
	}
	return res, nil
}

type mqlAwsEksAddonInternal struct {
	details     *ekstypes.Addon
	fetched     bool
	region      string
	lock        sync.Mutex
	clusterName string
}

func (a *mqlAwsEksAddon) fetchDetails() (*ekstypes.Addon, error) {
	if a.fetched {
		return a.details, nil
	}
	a.lock.Lock()
	defer a.lock.Unlock()
	if a.fetched {
		return a.details, nil
	}
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	ctx := context.Background()
	svc := conn.Eks(a.region)
	desc, err := svc.DescribeAddon(ctx, &eks.DescribeAddonInput{AddonName: aws.String(a.Name.Data), ClusterName: aws.String(a.clusterName)})
	if err != nil {
		return nil, err
	}
	a.details = desc.Addon
	a.fetched = true
	return desc.Addon, nil
}

func (a *mqlAwsEksAddon) arn() (string, error) {
	ao, err := a.fetchDetails()
	if err != nil {
		return "", err
	}
	if ao.AddonArn == nil {
		return "", nil
	}
	return *ao.AddonArn, nil
}

func (a *mqlAwsEksAddon) status() (string, error) {
	ao, err := a.fetchDetails()
	if err != nil {
		return "", err
	}
	return string(ao.Status), nil
}

func (a *mqlAwsEksAddon) createdAt() (*time.Time, error) {
	ao, err := a.fetchDetails()
	if err != nil {
		return nil, err
	}
	return ao.CreatedAt, nil
}

func (a *mqlAwsEksAddon) modifiedAt() (*time.Time, error) {
	ao, err := a.fetchDetails()
	if err != nil {
		return nil, err
	}
	return ao.ModifiedAt, nil
}

func (a *mqlAwsEksAddon) tags() (map[string]any, error) {
	ao, err := a.fetchDetails()
	if err != nil {
		return nil, err
	}
	new := make(map[string]any)
	for k, v := range ao.Tags {
		new[k] = v
	}
	return new, nil
}

func (a *mqlAwsEksAddon) addonVersion() (string, error) {
	ao, err := a.fetchDetails()
	if err != nil {
		return "", err
	}
	if ao.AddonVersion == nil {
		return "", nil
	}
	return *ao.AddonVersion, nil
}

func (a *mqlAwsEksAddon) publisher() (string, error) {
	ao, err := a.fetchDetails()
	if err != nil {
		return "", err
	}
	if ao.Publisher == nil {
		return "", nil
	}
	return *ao.Publisher, nil
}

func (a *mqlAwsEksAddon) owner() (string, error) {
	ao, err := a.fetchDetails()
	if err != nil {
		return "", err
	}
	if ao.Owner == nil {
		return "", nil
	}
	return *ao.Owner, nil
}

func (a *mqlAwsEksAddon) configurationValues() (string, error) {
	ao, err := a.fetchDetails()
	if err != nil {
		return "", err
	}
	if ao.ConfigurationValues == nil {
		return "", nil
	}
	return *ao.ConfigurationValues, nil
}
