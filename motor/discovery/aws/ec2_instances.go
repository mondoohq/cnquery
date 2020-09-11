package aws

import (
	"context"
	"regexp"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/cockroachdb/errors"
	"go.mondoo.io/mondoo/motor/asset"
	"go.mondoo.io/mondoo/motor/motorid/awsec2"
	"go.mondoo.io/mondoo/motor/platform"
	"go.mondoo.io/mondoo/motor/transports"
	aws_transport "go.mondoo.io/mondoo/motor/transports/aws"

	"github.com/rs/zerolog/log"
)

func NewEc2Discovery(cfg aws.Config) (*Ec2Instances, error) {
	return &Ec2Instances{config: cfg}, nil
}

type Ec2Instances struct {
	config              aws.Config
	InstanceSSHUsername string
}

func (ec2i *Ec2Instances) List() ([]*asset.Asset, error) {
	ctx := context.Background()
	ec2svc := ec2.New(ec2i.config)

	identity, err := aws_transport.CheckIam(ec2i.config)
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

	instances := []*asset.Asset{}
	for i := range resp.Reservations {
		reservation := resp.Reservations[i]
		for j := range reservation.Instances {
			instance := reservation.Instances[j]

			connections := []*transports.TransportConfig{}

			// add ssh and ssm run command if the transports.Connectionsm
			// connections = append(connections, &transports.TransportConfig{
			// 	Backend: transports.TransportConfigBackend_CONNECTION_AWS_SSM_RUN_COMMAND,
			// 	Host:    *instance.InstanceId,
			// })

			if instance.PublicIpAddress != nil {
				connections = append(connections, &transports.TransportConfig{
					Backend: transports.TransportBackend_CONNECTION_SSH,
					User:    ec2i.InstanceSSHUsername,
					Host:    *instance.PublicIpAddress,
				})
			}

			asset := &asset.Asset{
				ReferenceIDs: []string{awsec2.MondooInstanceID(account, ec2i.config.Region, *instance.InstanceId)},
				Name:         *instance.InstanceId,
				Platform: &platform.Platform{
					Kind:    transports.Kind_KIND_VIRTUAL_MACHINE,
					Runtime: transports.RUNTIME_AWS_EC2,
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
				asset.Labels["mondoo.app/ami-id"] = *instance.ImageId
			}

			instances = append(instances, asset)
		}
	}

	log.Debug().Int("instances", len(instances)).Msg("found ec2 instances")
	return instances, nil
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

func mapEc2InstanceStateCode(state *ec2.InstanceState) asset.State {
	if state == nil {
		return asset.State_STATE_UNKNOWN
	}
	switch *state.Code {
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
		log.Warn().Str("state", state.String()).Msg("unknown ec2 state")
		return asset.State_STATE_UNKNOWN
	}
}
