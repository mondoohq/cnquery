package aws

import (
	"context"
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"

	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/ssm/types"
	"go.mondoo.com/cnquery/motor/asset"
	"go.mondoo.com/cnquery/motor/motorid/awsec2"
	"go.mondoo.com/cnquery/motor/platform"
	"go.mondoo.com/cnquery/motor/providers"
	aws_provider "go.mondoo.com/cnquery/motor/providers/aws"
	"go.mondoo.com/cnquery/motor/vault"
	"go.mondoo.com/cnquery/resources/library/jobpool"
)

func NewSSMManagedInstancesDiscovery(cfg aws.Config) (*SSMManagedInstances, error) {
	return &SSMManagedInstances{config: cfg}, nil
}

type SSMManagedInstances struct {
	config        aws.Config
	FilterOptions Ec2InstancesFilters
	profile       string
}

func (ssmi *SSMManagedInstances) Name() string {
	return "AWS SSM Discover"
}

func (ssmi *SSMManagedInstances) List() ([]*asset.Asset, error) {
	identityResp, err := aws_provider.CheckIam(ssmi.config)
	if err != nil {
		return nil, err
	}

	account := *identityResp.Account

	instances := []*asset.Asset{}
	poolOfJobs := jobpool.CreatePool(ssmi.getInstances(account, ssmi.FilterOptions), 5)
	poolOfJobs.Run()

	// check for errors
	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	// get all the results
	for i := range poolOfJobs.Jobs {
		instances = append(instances, poolOfJobs.Jobs[i].Result.([]*asset.Asset)...)
	}

	return instances, nil
}

func (ssmi *SSMManagedInstances) getRegions() ([]string, error) {
	regions := []string{}

	ec2svc := ec2.NewFromConfig(ssmi.config)
	ctx := context.Background()

	res, err := ec2svc.DescribeRegions(ctx, &ec2.DescribeRegionsInput{})
	if err != nil {
		return regions, err
	}
	for _, region := range res.Regions {
		regions = append(regions, *region.RegionName)
	}
	return regions, nil
}

func (ssmi *SSMManagedInstances) getInstances(account string, ec2InstancesFilters Ec2InstancesFilters) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	var err error

	regions := ec2InstancesFilters.Regions
	if len(regions) == 0 {
		// user did not include a region filter, fetch em all
		regions, err = ssmi.getRegions()
		if err != nil {
			return []*jobpool.Job{{Err: err}} // return the error
		}
	}
	log.Debug().Msgf("regions being called for ec2 ssm instance list are: %v", regions)
	for ri := range regions {
		region := regions[ri]
		f := func() (jobpool.JobResult, error) {
			// get client for region
			clonedConfig := ssmi.config.Copy()
			clonedConfig.Region = region
			res := []*asset.Asset{}
			ssmsvc := ssm.NewFromConfig(clonedConfig)
			ctx := context.Background()

			// check that all instances have ssm agent installed and are reachable
			// it will return only those instances that are active in ssm
			// e.g stopped instances are not reachable
			input := &ssm.DescribeInstanceInformationInput{
				Filters: []types.InstanceInformationStringFilter{},
			}
			if len(ec2InstancesFilters.InstanceIds) > 0 {
				input.Filters = append(input.Filters, types.InstanceInformationStringFilter{Key: aws.String("InstanceIds"), Values: ec2InstancesFilters.InstanceIds})
				log.Debug().Interface("instance ids", ec2InstancesFilters.InstanceIds).Msgf("filtering")
			}
			// NOTE: AWS does not support filtering by tags for this api call
			nextToken := aws.String("no_token_to_start_with")
			ssminstances := make([]types.InstanceInformation, 0)
			for nextToken != nil {
				isssmresp, err := ssmsvc.DescribeInstanceInformation(ctx, input)
				if err != nil {
					return nil, errors.Wrap(err, "could not gather ssm information")
				}
				nextToken = isssmresp.NextToken
				if isssmresp.NextToken != nil {
					input.NextToken = nextToken
				}
				ssminstances = append(ssminstances, isssmresp.InstanceInformationList...)
			}

			log.Debug().Str("account", account).Str("region", clonedConfig.Region).Int("instance count", len(ssminstances)).Msg("found ec2 ssm instances")
			// the aws tags get a prefix to them so we can build the right map here by prepending the same value to each tag we're searching for
			tagsToFilter := map[string]string{}
			for k, v := range ec2InstancesFilters.Tags {
				tagsToFilter[ImportedFromAWSTagKeyPrefix+k] = v
			}
			for _, instance := range ssminstances {
				a := ssmInstanceToAsset(ssmInstanceInfo{
					account:  account,
					region:   region,
					instance: instance,
					profile:  ssmi.profile,
				}, clonedConfig)

				if len(tagsToFilter) > 0 {
					if !assetHasLabels(a, tagsToFilter) {
						continue
					}
				}
				res = append(res, a)
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func assetHasLabels(a *asset.Asset, labels map[string]string) bool {
	if len(labels) == 0 {
		return true
	}
	for k, v := range labels {
		if a.Labels[k] == v {
			return true
		}
	}
	return false
}

func mapSmmManagedPingStateCode(pingStatus types.PingStatus) asset.State {
	switch pingStatus {
	case types.PingStatusOnline:
		return asset.State_STATE_RUNNING
	case types.PingStatusConnectionLost:
		return asset.State_STATE_PENDING
	case types.PingStatusInactive:
		return asset.State_STATE_STOPPED
	default:
		return asset.State_STATE_UNKNOWN
	}
}

type ssmInstanceInfo struct {
	account      string
	region       string
	instance     types.InstanceInformation
	insecure     bool
	passInLabels map[string]string
	profile      string
}

func ssmInstanceToAsset(instanceInfo ssmInstanceInfo, cfg aws.Config) *asset.Asset {
	creds := []*vault.Credential{
		{
			User: getProbableUsernameFromSSMPlatformName(strings.ToLower(*instanceInfo.instance.PlatformName)),
		},
	}
	if strings.HasPrefix(*instanceInfo.instance.InstanceId, "i-") {
		creds[0].Type = vault.CredentialType_aws_ec2_ssm_session // this will only work for ec2 instances
	}
	host := *instanceInfo.instance.InstanceId
	if instanceInfo.instance.IPAddress != nil && *instanceInfo.instance.IPAddress != "" {
		host = *instanceInfo.instance.IPAddress
	}
	asset := &asset.Asset{
		PlatformIds: []string{awsec2.MondooInstanceID(instanceInfo.account, instanceInfo.region, *instanceInfo.instance.InstanceId)},
		Name:        *instanceInfo.instance.InstanceId,
		Platform: &platform.Platform{
			Kind:    providers.Kind_KIND_VIRTUAL_MACHINE,
			Runtime: providers.RUNTIME_AWS_SSM_MANAGED,
		},
		Connections: []*providers.Config{{
			Backend:     providers.ProviderType_SSH,
			Host:        host,
			Insecure:    true,
			Runtime:     providers.RUNTIME_AWS_EC2,
			Credentials: creds,
			Options: map[string]string{
				"region":  instanceInfo.region,
				"profile": instanceInfo.profile,
			},
		}},
		State:  mapSmmManagedPingStateCode(instanceInfo.instance.PingStatus),
		Labels: make(map[string]string),
	}

	// fetch and add labels from the instance
	ec2svc := ec2.NewFromConfig(cfg)
	tagresp, err := ec2svc.DescribeTags(context.Background(), &ec2.DescribeTagsInput{
		Filters: []ec2types.Filter{
			{
				Name:   aws.String("resource-id"),
				Values: []string{*instanceInfo.instance.InstanceId},
			},
		},
	})
	if err != nil {
		log.Warn().Err(err).Msg("could not gather ssm instance tag information")
	} else if tagresp != nil {
		for j := range tagresp.Tags {
			tag := tagresp.Tags[j]
			if tag.Key != nil {
				key := ImportedFromAWSTagKeyPrefix + *tag.Key
				value := ""
				if tag.Value != nil {
					value = *tag.Value
				}
				asset.Labels[key] = value
			}
		}
	}
	// add AWS metadata labels
	asset.Labels = addAWSMetadataLabels(asset.Labels, ssmInstanceToBasicInstanceInfo(instanceInfo))
	if label, ok := asset.Labels[ImportedFromAWSTagKeyPrefix+AWSNameLabel]; ok {
		asset.Name = label
	}
	return asset
}

func getProbableUsernameFromSSMPlatformName(name string) string {
	if strings.HasPrefix(name, "centos") {
		return "centos"
	}
	if strings.HasPrefix(name, "ubuntu") {
		return "ubuntu"
	}
	return "ec2-user"
}
