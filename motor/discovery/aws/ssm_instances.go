package aws

import (
	"context"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
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
	config aws.Config
}

func (ssmi *SSMManagedInstances) List() ([]*asset.Asset, error) {
	ctx := context.Background()
	ssmsvc := ssm.New(ssmi.config)

	identity, err := aws_transport.CheckIam(ssmi.config)
	if err != nil {
		return nil, err
	}

	account := *identity.Account

	// check that all instances have ssm agent installed and are reachable
	// it will return only those instances that are active in ssm
	// e.g stopped instances are not reachable
	platformFilter := string(ssm.InstanceInformationFilterKeyPlatformTypes)
	resourceFilter := string(ssm.InstanceInformationFilterKeyResourceType)
	isssmreq := ssmsvc.DescribeInstanceInformationRequest(&ssm.DescribeInstanceInformationInput{
		Filters: []ssm.InstanceInformationStringFilter{
			ssm.InstanceInformationStringFilter{Key: &platformFilter, Values: []string{string(ssm.PlatformTypeLinux), string(ssm.PlatformTypeWindows)}},
			// we only look for managed instanced
			ssm.InstanceInformationStringFilter{Key: &resourceFilter, Values: []string{string(ssm.ResourceTypeManagedInstance)}},
		},
	})
	isssmresp, err := isssmreq.Send(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "could not gather ssm information")
	}

	log.Debug().Msgf("%+v\n", *isssmresp)

	instances := []*asset.Asset{}
	for i := range isssmresp.InstanceInformationList {
		instance := isssmresp.InstanceInformationList[i]

		asset := &asset.Asset{
			ReferenceIDs: []string{awsec2.MondooInstanceID(account, ssmi.config.Region, *instance.InstanceId)},
			Name:         *instance.InstanceId,
			Platform: &platform.Platform{
				Kind:    transports.Kind_KIND_VIRTUAL_MACHINE,
				Runtime: transports.RUNTIME_AWS_SSM_MANAGED,
			},

			// Connections: connections,
			State:  mapSmmManagedPingStateCode(instance.PingStatus),
			Labels: make(map[string]string),
		}

		tagreq := ssmsvc.ListTagsForResourceRequest(&ssm.ListTagsForResourceInput{
			ResourceId:   instance.InstanceId,
			ResourceType: ssm.ResourceTypeForTaggingManagedInstance,
		})

		tagresp, err := tagreq.Send(ctx)
		if err != nil {
			log.Warn().Err(err).Msg("could not gather ssm information")
		} else if tagresp != nil {
			log.Debug().Msgf("%+v\n", *tagresp)

			for j := range tagresp.TagList {
				tag := tagresp.TagList[j]
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
		asset.Labels["mondoo.app/region"] = ssmi.config.Region
		if instance.InstanceId != nil {
			asset.Labels["mondoo.app/instance"] = *instance.InstanceId
		}
		if instance.IPAddress != nil {
			asset.Labels["mondoo.app/public-ip"] = *instance.IPAddress
		}

		instances = append(instances, asset)
	}

	return instances, nil
}

func mapSmmManagedPingStateCode(pingStatus ssm.PingStatus) asset.State {
	switch pingStatus {
	case ssm.PingStatusOnline:
		return asset.State_STATE_RUNNING
	case ssm.PingStatusConnectionLost:
		return asset.State_STATE_PENDING
	case ssm.PingStatusInactive:
		return asset.State_STATE_STOPPED
	default:
		return asset.State_STATE_UNKNOWN
	}
}
