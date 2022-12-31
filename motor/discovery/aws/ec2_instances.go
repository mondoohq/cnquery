package aws

import (
	"context"
	"regexp"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/smithy-go"
	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/motor/asset"
	"go.mondoo.com/cnquery/motor/motorid/awsec2"
	"go.mondoo.com/cnquery/motor/platform"
	"go.mondoo.com/cnquery/motor/providers"
	aws_provider "go.mondoo.com/cnquery/motor/providers/aws"
	"go.mondoo.com/cnquery/motor/vault"
	"go.mondoo.com/cnquery/resources/library/jobpool"
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
	config        aws.Config
	Insecure      bool
	FilterOptions Ec2InstancesFilters
	Labels        map[string]string
	profile       string
}

type Ec2InstanceState int

const (
	Running Ec2InstanceState = iota
	All
)

type Ec2InstancesFilters struct {
	InstanceIds   []string
	Tags          map[string]string
	Regions       []string
	InstanceState Ec2InstanceState
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
		return regions, err
	}
	for _, region := range res.Regions {
		regions = append(regions, *region.RegionName)
	}
	return regions, nil
}

func (ec2i *Ec2Instances) getInstances(account string, ec2InstancesFilters Ec2InstancesFilters) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	var err error

	regions := ec2InstancesFilters.Regions
	if len(regions) == 0 {
		// user did not include a region filter, fetch em all
		regions, err = ec2i.getRegions()
		if err != nil {
			return []*jobpool.Job{{Err: err}} // return the error
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
			reservs := make([]types.Reservation, 0)
			if len(ec2i.FilterOptions.InstanceIds) > 0 {
				input.InstanceIds = ec2i.FilterOptions.InstanceIds
				log.Debug().Msgf("filtering by instance ids %v", input.InstanceIds)
			}
			if len(ec2i.FilterOptions.Tags) > 0 {
				for k, v := range ec2i.FilterOptions.Tags {
					input.Filters = append(input.Filters, types.Filter{Name: aws.String("tag:" + k), Values: []string{v}})
					log.Debug().Msgf("filtering by tag %s:%s", k, v)
				}
			}
			if ec2i.FilterOptions.InstanceState == 0 {
				input.Filters = append(input.Filters, types.Filter{Name: aws.String("instance-state-name"), Values: []string{"running"}})
				log.Debug().Msg("filtering aws ec2 instances by running state")
			}
			nextToken := aws.String("no_token_to_start_with")
			for nextToken != nil {
				resp, err := ec2svc.DescribeInstances(ctx, input)
				if err != nil {
					var ae smithy.APIError
					if errors.As(err, &ae) {
						// when filtering for instance ids, we'll get this error in the regions where the instance ids are not found
						if ae.ErrorCode() == "InvalidInstanceID.NotFound" {
							return res, nil
						}
						return nil, errors.Wrapf(err, "failed to describe instances, %s", clonedConfig.Region)
					}
				}
				nextToken = resp.NextToken
				if resp.NextToken != nil {
					input.NextToken = nextToken
				}
				reservs = append(reservs, resp.Reservations...)
			}

			log.Debug().Str("account", account).Str("region", clonedConfig.Region).Int("instance count", len(reservs)).Msg("found ec2 instances")

			// resolve all ec2 instances
			for i := range reservs {
				reservation := reservs[i]
				for j := range reservation.Instances {
					instance := reservation.Instances[j]
					res = append(res, instanceToAsset(instanceInfo{
						account:      account,
						region:       region,
						instance:     instance,
						insecure:     ec2i.Insecure,
						passInLabels: ec2i.Labels,
						profile:      ec2i.profile,
						imageName:    getImageName(ctx, *instance.ImageId, region, clonedConfig),
					},
					))
				}
			}

			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func (ec2i *Ec2Instances) List() ([]*asset.Asset, error) {
	identityResp, err := aws_provider.CheckIam(ec2i.config)
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

type instanceInfo struct {
	account      string
	region       string
	instance     types.Instance
	insecure     bool
	passInLabels map[string]string
	profile      string
	imageName    string
}

func instanceToAsset(instanceInfo instanceInfo) *asset.Asset {
	asset := &asset.Asset{
		PlatformIds: []string{awsec2.MondooInstanceID(instanceInfo.account, instanceInfo.region, *instanceInfo.instance.InstanceId)},
		Connections: []*providers.Config{},
		Labels:      make(map[string]string),
		IdDetector:  []string{"awsec2"},
		Name:        *instanceInfo.instance.InstanceId,
		Platform: &platform.Platform{
			Kind:    providers.Kind_KIND_VIRTUAL_MACHINE,
			Runtime: providers.RUNTIME_AWS_EC2,
		},
		State: mapEc2InstanceStateCode(instanceInfo.instance.State),
	}

	// if there is a public ip, we assume ssh is an option
	if instanceInfo.instance.PublicIpAddress != nil {
		asset.Connections = append(asset.Connections, &providers.Config{
			Backend:  providers.ProviderType_SSH,
			Host:     *instanceInfo.instance.PublicIpAddress,
			Insecure: instanceInfo.insecure,
			Runtime:  providers.RUNTIME_AWS_EC2,
			Credentials: []*vault.Credential{
				{
					Type: vault.CredentialType_aws_ec2_instance_connect,
					User: getProbableUsernameFromImageName(strings.ToLower(instanceInfo.imageName)),
				},
			},
			Options: map[string]string{
				"region":  instanceInfo.region,
				"profile": instanceInfo.profile,
			},
		})
	}

	// add labels from the instance
	for k := range instanceInfo.instance.Tags {
		tag := instanceInfo.instance.Tags[k]
		if tag.Key != nil {
			key := ImportedFromAWSTagKeyPrefix + *tag.Key
			value := ""
			if tag.Value != nil {
				value = *tag.Value
			}
			asset.Labels[key] = value
		}
	}
	// add passed in labels
	for k, v := range instanceInfo.passInLabels {
		asset.Labels[k] = v
	}
	// add AWS metadata labels
	asset.Labels = addAWSMetadataLabels(asset.Labels, ec2InstanceToBasicInstanceInfo(instanceInfo))
	if label, ok := asset.Labels[ImportedFromAWSTagKeyPrefix+AWSNameLabel]; ok {
		asset.Name = label
	}
	return asset
}

const AWSNameLabel = "Name"

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
	switch code := *state.Code; code {
	case 16:
		return asset.State_STATE_RUNNING
	case 0:
		return asset.State_STATE_PENDING
	case 32: // 32 is shutting down, which is the step before terminated, assume terminated if we get shutting down
		return asset.State_STATE_TERMINATED
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

func InstanceIsInRunningOrStoppedState(state *types.InstanceState) bool {
	// instance state 16 == running, 80 == stopped
	if state == nil {
		return false
	}
	return *state.Code == 16 || *state.Code == 80
}

func InstanceIsInRunningState(state *types.InstanceState) bool {
	// instance state 16 == running
	if state == nil {
		return false
	}
	return *state.Code == 16
}

func getImageName(ctx context.Context, imageId string, region string, cfg aws.Config) string {
	cfgCopy := cfg
	cfgCopy.Region = region
	svc := ec2.NewFromConfig(cfgCopy)
	images, err := svc.DescribeImages(ctx, &ec2.DescribeImagesInput{ImageIds: []string{imageId}})
	if err == nil {
		if len(images.Images) > 0 {
			// we only gave it one image id
			imageName := images.Images[0].Name
			if imageName != nil {
				return *imageName
			}
		}
	}
	return ""
}

func getProbableUsernameFromImageName(name string) string {
	if strings.Contains(name, "centos") {
		return "centos"
	}
	if strings.Contains(name, "ubuntu") {
		return "ubuntu"
	}
	return "ec2-user"
}
