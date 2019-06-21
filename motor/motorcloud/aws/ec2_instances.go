package aws

import (
	"context"
	"regexp"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/pkg/errors"
	"go.mondoo.io/mondoo/nexus/assets"

	"github.com/rs/zerolog/log"
)

func NewEc2Discovery(cfg aws.Config) (*Ec2Instances, error) {
	return &Ec2Instances{config: cfg}, nil
}

type Ec2Instances struct {
	config aws.Config
}

func (ec2i *Ec2Instances) List() ([]*assets.Asset, error) {
	ctx := context.Background()
	ec2svc := ec2.New(ec2i.config)

	identity, err := CheckIam(ec2i.config)
	if err != nil {
		return nil, err
	}

	account := *identity.Account

	log.Debug().Str("region", ec2i.config.Region).Msg("search ec2 instances")
	req := ec2svc.DescribeInstancesRequest(&ec2.DescribeInstancesInput{})
	resp, err := req.Send(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to describe instances, %s", ec2i.config.Region)
	}

	instances := []*assets.Asset{}
	for i := range resp.Reservations {
		reservation := resp.Reservations[i]
		for j := range reservation.Instances {
			instance := reservation.Instances[j]

			connections := []*assets.Connection{}

			// add ssh and ssm run command if the node is part of ssm
			connections = append(connections, &assets.Connection{
				Backend: assets.ConnectionBackend_CONNECTION_AWS_SSM_RUN_COMMAND,
				Host:    *instance.InstanceId,
			})

			if instance.PublicIpAddress != nil {
				connections = append(connections, &assets.Connection{
					Backend: assets.ConnectionBackend_CONNECTION_SSH,
					Host:    "ec2-user@" + *instance.PublicIpAddress,
				})
			}

			asset := &assets.Asset{
				ReferenceIDs: []string{MondooEc2InstanceID(account, ec2i.config.Region, *instance.InstanceId)},
				Name:         *instance.InstanceId,
				Platform: &assets.Platform{
					Kind:    assets.Kind_KIND_VIRTUAL_MACHINE,
					Runtime: "aws ec2",
				},
				Connections: connections,
				State:       mapEc2InstanceStateCode(instance.State),
				Labels:      make(map[string]string),
			}

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

			// fetch aws specific metadata
			asset.Labels["mondoo.app/region"] = ec2i.config.Region
			if instance.InstanceId != nil {
				asset.Labels["mondoo.app/instance"] = *instance.InstanceId
			}
			if instance.PublicDnsName != nil {
				asset.Labels["mondoo.app/public-dns-name"] = *instance.PublicDnsName
			}
			if instance.PublicIpAddress != nil {
				asset.Labels["mondoo.app/public-ip"] = *instance.PublicIpAddress
			}
			if instance.ImageId != nil {
				asset.Labels["mondoo.app/image-id"] = *instance.ImageId
			}

			instances = append(instances, asset)
		}
	}

	log.Debug().Int("instances", len(instances)).Msg("found ec2 instances")
	return instances, nil
}

// aws://ec2/v1/accounts/{account}/regions/{region}/instances/{instanceid}
func MondooEc2InstanceID(account string, region string, instanceid string) string {
	return "//platformid.api.mondoo.app/runtime/aws/ec2/v1/accounts/" + account + "/regions/" + region + "/instances/" + instanceid
}

type awsec2id struct {
	Account  string
	Region   string
	Instance string
}

func ParseEc2ReferenceID(uri string) *awsec2id {
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

func mapEc2InstanceStateCode(state *ec2.InstanceState) assets.State {
	if state == nil {
		return assets.State_STATE_UNKNOWN
	}
	switch *state.Code {
	case 16:
		return assets.State_STATE_RUNNING
	case 0:
		return assets.State_STATE_PENDING
	case 32:
		return assets.State_STATE_STOPPING
	case 64:
		return assets.State_STATE_STOPPING
	case 80:
		return assets.State_STATE_STOPPED
	case 48:
		return assets.State_STATE_TERMINATED
	default:
		log.Warn().Str("state", state.String()).Msg("unknown ec2 state")
		return assets.State_STATE_UNKNOWN
	}
}
