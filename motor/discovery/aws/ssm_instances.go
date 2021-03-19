package aws

import (
	"context"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"

	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/ssm/types"
	"go.mondoo.io/mondoo/lumi/library/jobpool"
	"go.mondoo.io/mondoo/motor/asset"
	"go.mondoo.io/mondoo/motor/motorid/awsec2"
	"go.mondoo.io/mondoo/motor/platform"
	"go.mondoo.io/mondoo/motor/transports"
	aws_transport "go.mondoo.io/mondoo/motor/transports/aws"
)

func NewSSMManagedInstancesDiscovery(cfg aws.Config) (*SSMManagedInstances, error) {
	return &SSMManagedInstances{config: cfg}, nil
}

type SSMManagedInstances struct {
	config        aws.Config
	FilterOptions ec2InstancesFilters
}

func (ssmi *SSMManagedInstances) Name() string {
	return "AWS SSM Discover"
}

func (ssmi *SSMManagedInstances) List() ([]*asset.Asset, error) {
	identityResp, err := aws_transport.CheckIam(ssmi.config)
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
		return regions, nil
	}
	for _, region := range res.Regions {
		regions = append(regions, *region.RegionName)
	}
	return regions, nil
}

func (ssmi *SSMManagedInstances) getInstances(account string, ec2InstancesFilters ec2InstancesFilters) []*jobpool.Job {
	var tasks = make([]*jobpool.Job, 0)
	var err error

	regions := ec2InstancesFilters.regions
	if len(regions) == 0 {
		// user did not include a region filter, fetch em all
		regions, err = ssmi.getRegions()
		if err != nil {
			return []*jobpool.Job{&jobpool.Job{Err: err}} // return the error
		}
	}
	log.Debug().Msgf("regions being called for ec2 instance list are: %v", regions)
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
			if len(ec2InstancesFilters.instanceIds) > 0 {
				input.Filters = append(input.Filters, types.InstanceInformationStringFilter{Key: aws.String("InstanceIds"), Values: ec2InstancesFilters.instanceIds})
				log.Debug().Msgf("filtering by instance ids %v", ec2InstancesFilters.instanceIds)
			}
			if len(ec2InstancesFilters.tags) > 0 {
				for k, v := range ec2InstancesFilters.tags {
					input.Filters = append(input.Filters, types.InstanceInformationStringFilter{Key: &k, Values: []string{v}})
					log.Debug().Msgf("filtering by tag %s:%s", k, v)
				}
			}
			isssmresp, err := ssmsvc.DescribeInstanceInformation(ctx, input)
			if err != nil {
				return nil, errors.Wrap(err, "could not gather ssm information")
			}

			log.Debug().Str("account", account).Str("region", clonedConfig.Region).Int("instance count", len(isssmresp.InstanceInformationList)).Msg("found ec2 ssm instances")

			for i := range isssmresp.InstanceInformationList {
				instance := isssmresp.InstanceInformationList[i]
				res = append(res, ssmInstanceToAsset(account, region, instance, clonedConfig))
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
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

func ssmInstanceToAsset(account string, region string, instance types.InstanceInformation, clonedConfig aws.Config) *asset.Asset {

	connections := []*transports.TransportConfig{}

	connections = append(connections, &transports.TransportConfig{
		Backend: transports.TransportBackend_CONNECTION_AWS_SSM_RUN_COMMAND,
		Host:    *instance.InstanceId,
	})
	asset := &asset.Asset{
		PlatformIDs: []string{awsec2.MondooInstanceID(account, region, *instance.InstanceId)},
		Name:        *instance.InstanceId,
		Platform: &platform.Platform{
			Kind:    transports.Kind_KIND_VIRTUAL_MACHINE,
			Runtime: transports.RUNTIME_AWS_SSM_MANAGED,
		},

		Connections: connections,
		State:       mapSmmManagedPingStateCode(instance.PingStatus),
		Labels:      make(map[string]string),
	}

	ec2svc := ec2.NewFromConfig(clonedConfig)
	tagresp, err := ec2svc.DescribeTags(context.Background(), &ec2.DescribeTagsInput{
		Filters: []ec2types.Filter{
			{Name: aws.String("resource-id"),
				Values: []string{*instance.InstanceId}},
		},
	})

	asset.Labels["ssm.aws.mondoo.app/platform"] = string(instance.PlatformType)

	if err != nil {
		log.Warn().Err(err).Msg("could not gather ssm instance tag information")
	} else if tagresp != nil {
		for j := range tagresp.Tags {
			tag := tagresp.Tags[j]
			if tag.Key != nil {
				key := *tag.Key
				value := ""
				if tag.Value != nil {
					value = *tag.Value
				}
				asset.Labels[key] = value
			}
		}
	}

	// fetch aws specific metadata
	asset.Labels["mondoo.app/region"] = region
	if instance.InstanceId != nil {
		asset.Labels["mondoo.app/instance"] = *instance.InstanceId
	}
	if instance.IPAddress != nil {
		asset.Labels["mondoo.app/public-ip"] = *instance.IPAddress
	}

	return asset
}
