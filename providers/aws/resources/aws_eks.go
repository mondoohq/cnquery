// Copyright (c) Mondoo, Inc.
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
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/cnquery/v11/providers/aws/connection"

	"go.mondoo.com/cnquery/v11/types"
)

func (a *mqlAwsEks) id() (string, error) {
	return "aws.eks", nil
}

func (a *mqlAwsEks) clusters() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(a.getClusters(conn), 5)
	poolOfJobs.Run()

	// check for errors
	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	// get all the results
	for i := range poolOfJobs.Jobs {
		res = append(res, poolOfJobs.Jobs[i].Result.([]interface{})...)
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
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("eks>getClusters>calling aws with region %s", regionVal)

			svc := conn.Eks(regionVal)
			ctx := context.Background()
			res := []interface{}{}

			describeClusterRes, err := svc.ListClusters(ctx, &eks.ListClustersInput{})
			if err != nil {
				if Is400AccessDeniedError(err) {
					log.Warn().Str("region", regionVal).Msg("error accessing region for AWS API")
					return res, nil
				}
				return nil, err
			}

			if describeClusterRes == nil {
				return jobpool.JobResult(res), nil
			}

			for i := range describeClusterRes.Clusters {
				clusterName := describeClusterRes.Clusters[i]

				// get cluster details
				log.Debug().Str("cluster", clusterName).Str("region", region).Msg("get info for cluster")
				describeClusterOutput, err := svc.DescribeCluster(ctx, &eks.DescribeClusterInput{
					Name: aws.String(clusterName),
				})
				if err != nil {
					return nil, err
				}

				if describeClusterOutput == nil {
					continue
				}

				cluster := describeClusterOutput.Cluster
				encryptionConfig, _ := convert.JsonToDictSlice(cluster.EncryptionConfig)
				logging, _ := convert.JsonToDict(cluster.Logging)
				kubernetesNetworkConfig, _ := convert.JsonToDict(cluster.KubernetesNetworkConfig)
				vpcConfig, _ := convert.JsonToDict(cluster.ResourcesVpcConfig)

				args := map[string]*llx.RawData{
					"arn":                llx.StringDataPtr(cluster.Arn),
					"authenticationMode": llx.StringData(string(cluster.AccessConfig.AuthenticationMode)),
					"createdAt":          llx.TimeDataPtr(cluster.CreatedAt),
					"encryptionConfig":   llx.ArrayData(encryptionConfig, types.Any),
					"endpoint":           llx.StringDataPtr(cluster.Endpoint),
					"iamRole":            llx.NilData, // set iamRole to nil as default, if iam is not set
					"logging":            llx.MapData(logging, types.Any),
					"name":               llx.StringDataPtr(cluster.Name),
					"networkConfig":      llx.MapData(kubernetesNetworkConfig, types.Any),
					"platformVersion":    llx.StringDataPtr(cluster.PlatformVersion),
					"region":             llx.StringData(regionVal),
					"resourcesVpcConfig": llx.MapData(vpcConfig, types.Any),
					"status":             llx.StringData(string(cluster.Status)),
					"supportType":        llx.StringData(string(cluster.UpgradePolicy.SupportType)),
					"tags":               llx.MapData(strMapToInterface(cluster.Tags), types.String),
					"version":            llx.StringDataPtr(cluster.Version),
				}

				if cluster.RoleArn != nil {
					mqlIam, err := NewResource(a.MqlRuntime, "aws.iam.role",
						map[string]*llx.RawData{"arn": llx.StringDataPtr(cluster.RoleArn)},
					)
					if err != nil {
						return nil, err
					}
					// update the iam setting
					args["iamRole"] = llx.ResourceData(mqlIam, mqlIam.MqlName())
				}

				mqlFilesystem, err := CreateResource(a.MqlRuntime, "aws.eks.cluster", args)
				if err != nil {
					return nil, err
				}
				res = append(res, mqlFilesystem)
			}

			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
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
	obj, err := CreateResource(runtime, "aws.eks", map[string]*llx.RawData{})
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

func (a *mqlAwsEksCluster) nodeGroups() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	regionVal := a.Region.Data
	log.Debug().Msgf("eks>getNodegroups>calling aws with region %s", regionVal)

	svc := conn.Eks(regionVal)
	ctx := context.Background()
	res := []interface{}{}

	nodeGroupsRes, err := svc.ListNodegroups(ctx, &eks.ListNodegroupsInput{ClusterName: aws.String(a.Name.Data)})
	if err != nil {
		if Is400AccessDeniedError(err) {
			log.Warn().Str("region", regionVal).Msg("error accessing region for AWS API")
			return res, nil
		}
		return nil, err
	}

	if nodeGroupsRes == nil {
		return nil, nil
	}

	for i := range nodeGroupsRes.Nodegroups {
		nodegroup := nodeGroupsRes.Nodegroups[i]
		args := map[string]*llx.RawData{
			"name":   llx.StringData(nodegroup),
			"region": llx.StringData(regionVal),
		}

		mqlNg, err := CreateResource(a.MqlRuntime, "aws.eks.nodegroup", args)
		if err != nil {
			return nil, err
		}
		mqlNg.(*mqlAwsEksNodegroup).clusterName = a.Name.Data
		mqlNg.(*mqlAwsEksNodegroup).region = regionVal
		res = append(res, mqlNg)
	}
	return res, nil
}

type mqlAwsEksNodegroupInternal struct {
	details     *ekstypes.Nodegroup
	region      string
	lock        sync.Mutex
	clusterName string
}

func (a *mqlAwsEksNodegroup) id() (string, error) {
	return a.Arn.Data, nil
}

func (a *mqlAwsEksNodegroup) autoscalingGroups() ([]interface{}, error) {
	ng, err := a.fetchDetails()
	if err != nil {
		return nil, err
	}
	if ng.Resources == nil || ng.Resources.AutoScalingGroups == nil {
		return nil, nil
	}
	res := []interface{}{}
	for i := range ng.Resources.AutoScalingGroups {
		ag := ng.Resources.AutoScalingGroups[i]
		mqlAg, err := NewResource(a.MqlRuntime, "aws.autoscaling.group",
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
	if a.details != nil {
		return a.details, nil
	}
	a.lock.Lock()
	defer a.lock.Unlock()
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	ctx := context.Background()
	svc := conn.Eks(a.region)
	desc, err := svc.DescribeNodegroup(ctx, &eks.DescribeNodegroupInput{NodegroupName: aws.String(a.Name.Data), ClusterName: aws.String(a.clusterName)})
	if err != nil {
		return nil, err
	}
	a.details = desc.Nodegroup
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

func (a *mqlAwsEksNodegroup) scalingConfig() (map[string]interface{}, error) {
	ng, err := a.fetchDetails()
	if err != nil {
		return nil, err
	}
	return convert.JsonToDict(ng.ScalingConfig)
}

func (a *mqlAwsEksNodegroup) instanceTypes() ([]interface{}, error) {
	ng, err := a.fetchDetails()
	if err != nil {
		return nil, err
	}
	s := []interface{}{}
	for i := range ng.InstanceTypes {
		s = append(s, ng.InstanceTypes[i])
	}
	return s, nil
}

func (a *mqlAwsEksNodegroup) labels() (map[string]interface{}, error) {
	ng, err := a.fetchDetails()
	if err != nil {
		return nil, err
	}
	new := make(map[string]interface{})
	for k, v := range ng.Labels {
		new[k] = v
	}
	return new, nil
}

func (a *mqlAwsEksNodegroup) tags() (map[string]interface{}, error) {
	ng, err := a.fetchDetails()
	if err != nil {
		return nil, err
	}
	new := make(map[string]interface{})
	for k, v := range ng.Labels {
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
	mqlIam, err := NewResource(a.MqlRuntime, "aws.iam.role",
		map[string]*llx.RawData{
			"arn": llx.StringDataPtr(ng.NodeRole),
		})
	if err != nil {
		return nil, err
	}
	return mqlIam.(*mqlAwsIamRole), nil
}

// AwsEksAddons
func (a *mqlAwsEksCluster) addons() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	regionVal := a.Region.Data
	log.Debug().Msgf("eks>getAddons>calling aws with region %s", regionVal)

	svc := conn.Eks(regionVal)
	ctx := context.Background()
	res := []interface{}{}

	addonsRes, err := svc.ListAddons(ctx, &eks.ListAddonsInput{ClusterName: aws.String(a.Name.Data)})
	if err != nil {
		if Is400AccessDeniedError(err) {
			log.Warn().Str("region", regionVal).Msg("error accessing region for AWS API")
			return res, nil
		}
		return nil, err
	}

	if addonsRes == nil {
		return nil, nil
	}

	for i := range addonsRes.Addons {
		addon := addonsRes.Addons[i]
		args := map[string]*llx.RawData{
			"__id": llx.StringData(fmt.Sprintf("aws.eks.addon/%s/%s", a.Name.Data, addon)),
			"name": llx.StringData(addon),
		}

		mqlNg, err := CreateResource(a.MqlRuntime, "aws.eks.addon", args)
		if err != nil {
			return nil, err
		}
		mqlNg.(*mqlAwsEksAddon).clusterName = a.Name.Data
		mqlNg.(*mqlAwsEksAddon).region = regionVal
		res = append(res, mqlNg)
	}
	return res, nil
}

type mqlAwsEksAddonInternal struct {
	details     *ekstypes.Addon
	region      string
	lock        sync.Mutex
	clusterName string
}

func (a *mqlAwsEksAddon) fetchDetails() (*ekstypes.Addon, error) {
	if a.details != nil {
		return a.details, nil
	}
	a.lock.Lock()
	defer a.lock.Unlock()
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	ctx := context.Background()
	svc := conn.Eks(a.region)
	desc, err := svc.DescribeAddon(ctx, &eks.DescribeAddonInput{AddonName: aws.String(a.Name.Data), ClusterName: aws.String(a.clusterName)})
	if err != nil {
		return nil, err
	}
	a.details = desc.Addon
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

func (a *mqlAwsEksAddon) tags() (map[string]interface{}, error) {
	ao, err := a.fetchDetails()
	if err != nil {
		return nil, err
	}
	new := make(map[string]interface{})
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
