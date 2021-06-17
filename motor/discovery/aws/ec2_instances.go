package aws

import (
	"context"
	"regexp"

	"go.mondoo.io/mondoo/lumi/library/jobpool"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"

	"github.com/aws/smithy-go"
	"github.com/cockroachdb/errors"
	"go.mondoo.io/mondoo/motor/asset"
	"go.mondoo.io/mondoo/motor/motorid/awsec2"
	"go.mondoo.io/mondoo/motor/platform"
	"go.mondoo.io/mondoo/motor/transports"
	aws_transport "go.mondoo.io/mondoo/motor/transports/aws"

	"github.com/rs/zerolog/log"
)

func NewEc2Discovery(cfg aws.Config) (*Ec2Instances, error) {
	clone := cfg.Copy()

	// fallback to default region, we run things cross-region anyhow
	if clone.Region == "" {
		clone.Region = "us-east-1"
	}

	return &Ec2Instances{config: cfg}, nil
}

type Ec2Instances struct {
	config                     aws.Config
	InstanceSSHUsername        string
	Insecure                   bool
	FilterOptions              ec2InstancesFilters
	SSMInstancesPlatformIdsMap map[string]*asset.Asset
}

func (ec2i *Ec2Instances) Name() string {
	return "AWS EC2 Discover"
}

func (ec2i *Ec2Instances) getRegions() ([]string, error) {
	// if no cache, get regions using ec2 client (using the ssm list global regions does not give the same list)
	regions := []string{}

	ec2svc := ec2.NewFromConfig(ec2i.config)
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

func (ec2i *Ec2Instances) getInstances(account string, ec2InstancesFilters ec2InstancesFilters) []*jobpool.Job {
	var tasks = make([]*jobpool.Job, 0)
	var err error

	regions := ec2InstancesFilters.regions
	if len(regions) == 0 {
		// user did not include a region filter, fetch em all
		regions, err = ec2i.getRegions()
		if err != nil {
			return []*jobpool.Job{&jobpool.Job{Err: err}} // return the error
		}
	}
	log.Debug().Msgf("regions being called for ec2 instance list are: %v", regions)
	for ri := range regions {
		region := regions[ri]
		f := func() (jobpool.JobResult, error) {
			// get client for region
			clonedConfig := ec2i.config.Copy()
			clonedConfig.Region = region

			// fetch instances
			ec2svc := ec2.NewFromConfig(clonedConfig)
			ctx := context.Background()
			res := []*asset.Asset{}

			input := &ec2.DescribeInstancesInput{}
			if len(ec2i.FilterOptions.instanceIds) > 0 {
				input.InstanceIds = ec2i.FilterOptions.instanceIds
				log.Debug().Msgf("filtering by instance ids %v", input.InstanceIds)
			}
			if len(ec2i.FilterOptions.tags) > 0 {
				for k, v := range ec2i.FilterOptions.tags {
					input.Filters = append(input.Filters, types.Filter{Name: &k, Values: []string{v}})
					log.Debug().Msgf("filtering by tag %s:%s", k, v)
				}
			}
			resp, err := ec2svc.DescribeInstances(ctx, input)
			if err != nil {
				var ae smithy.APIError
				if errors.As(err, &ae) {
					// when filtering for instance ids, we'll get this error in the regions where the instance ids are not found
					if ae.ErrorCode() == "InvalidInstanceID.NotFound" {
						return res, nil
					}
				}
				return nil, errors.Wrapf(err, "failed to describe instances, %s", clonedConfig.Region)
			}
			log.Debug().Str("account", account).Str("region", clonedConfig.Region).Int("instance count", len(resp.Reservations)).Msg("found ec2 instances")

			// resolve all instances
			for i := range resp.Reservations {
				reservation := resp.Reservations[i]
				for j := range reservation.Instances {
					instance := reservation.Instances[j]
					res = append(res, instanceToAsset(account, region, instance, ec2i.InstanceSSHUsername, ec2i.Insecure, ec2i.SSMInstancesPlatformIdsMap))
				}
			}

			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func (ec2i *Ec2Instances) List() ([]*asset.Asset, error) {
	identityResp, err := aws_transport.CheckIam(ec2i.config)
	if err != nil {
		return nil, err
	}

	account := *identityResp.Account

	instances := []*asset.Asset{}
	poolOfJobs := jobpool.CreatePool(ec2i.getInstances(account, ec2i.FilterOptions), 5)
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

func instanceToAsset(account string, region string, instance types.Instance, sshUsername string, insecure bool, ssmInstancesPlatformIdsMap map[string]*asset.Asset) *asset.Asset {

	connections := []*transports.TransportConfig{}

	var connection *transports.TransportConfig
	if instance.PublicIpAddress != nil {
		connection = &transports.TransportConfig{
			Backend:  transports.TransportBackend_CONNECTION_SSH,
			User:     sshUsername,
			Host:     *instance.PublicIpAddress,
			Insecure: insecure,
			Runtime:  transports.RUNTIME_AWS_EC2,
		}
		connections = append(connections, connection)
	}

	asset := &asset.Asset{}
	if ssmAsset, ok := ssmInstancesPlatformIdsMap[awsec2.MondooInstanceID(account, region, *instance.InstanceId)]; ok {
		// instance already discovered via ssm search. only add connections
		ssmAsset.Connections = append(ssmAsset.Connections, connections...)
		ssmAsset.Labels = addAssetLabels(ssmAsset.Labels, instance, region)
	} else {
		asset.PlatformIds = []string{awsec2.MondooInstanceID(account, region, *instance.InstanceId)}
		asset.Name = *instance.InstanceId
		asset.Platform = &platform.Platform{
			Kind:    transports.Kind_KIND_VIRTUAL_MACHINE,
			Runtime: transports.RUNTIME_AWS_EC2,
		}
		asset.Connections = connections
		asset.State = mapEc2InstanceStateCode(instance.State)
		asset.Labels = addAssetLabels(map[string]string{}, instance, region)
		for k := range instance.Tags {
			tag := instance.Tags[k]
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

	return asset
}

const (
	ImageIdLabel string = "mondoo.app/ami-id"
	RegionLabel  string = "mondoo.app/region"
)

func addAssetLabels(labels map[string]string, instance types.Instance, region string) map[string]string {
	// fetch aws specific metadata
	labels[RegionLabel] = region
	if instance.InstanceId != nil {
		labels["mondoo.app/instance"] = *instance.InstanceId
	}
	if instance.PublicDnsName != nil {
		labels["mondoo.app/public-dns-name"] = *instance.PublicDnsName
	}
	if instance.PublicIpAddress != nil {
		labels["mondoo.app/public-ip"] = *instance.PublicIpAddress
	}
	if instance.ImageId != nil {
		labels[ImageIdLabel] = *instance.ImageId
	}
	return labels
}

type awsec2id struct {
	Account  string
	Region   string
	Instance string
}

func ParseEc2PlatformID(uri string) *awsec2id {
	// aws://ec2/v1/accounts/{account}/regions/{region}/instances/{instanceid}
	awsec2 := regexp.MustCompile(`^\/\/platformid.api.mondoo.app\/runtime\/aws\/ec2\/v1\/accounts\/(.*)\/regions\/(.*)\/instances\/(.*)$`)
	m := awsec2.FindStringSubmatch(uri)
	if len(m) == 0 {
		return nil
	}

	return &awsec2id{
		Account:  m[1],
		Region:   m[2],
		Instance: m[3],
	}
}

func mapEc2InstanceStateCode(state *types.InstanceState) asset.State {
	if state == nil {
		return asset.State_STATE_UNKNOWN
	}
	switch state.Code {
	case 16:
		return asset.State_STATE_RUNNING
	case 0:
		return asset.State_STATE_PENDING
	case 32:
		return asset.State_STATE_STOPPING
	case 64:
		return asset.State_STATE_STOPPING
	case 80:
		return asset.State_STATE_STOPPED
	case 48:
		return asset.State_STATE_TERMINATED
	default:
		log.Warn().Str("state", string(state.Name)).Msg("unknown ec2 state")
		return asset.State_STATE_UNKNOWN
	}
}
