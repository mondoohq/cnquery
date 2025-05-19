// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package awsebs

import (
	"context"
	"regexp"
	"slices"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v11/providers/os/connection/shared"
	"go.mondoo.com/cnquery/v11/providers/os/id/awsec2"
	"go.mondoo.com/cnquery/v11/providers/os/id/machineid"
)

type Identity struct {
	InstanceMachineID string
	// difficult to get for EBS volumes, but if we can detect it, we will return it
	InstanceID  string
	PlatformIDs []string
}
type InstanceIdentifier interface {
	Identify() (Identity, error)
	RawMetadata() (any, error)
}

func Resolve(conn shared.Connection, pf *inventory.Platform) (InstanceIdentifier, error) {
	return &ebsMetadata{conn, pf}, nil
}

type ebsMetadata struct {
	conn     shared.Connection
	platform *inventory.Platform
}

func (m *ebsMetadata) RawMetadata() (any, error) {
	// Try to use AWS apis to detect network information first, if that doesn't work,
	// fallback to accessing the mounted volume, which is more difficult
	if instance, ok := m.fetchEC2Instance(); ok {
		mdata := map[string]any{}
		if privateDNS := convert.ToValue(instance.PrivateDnsName); privateDNS != "" {
			mdata["hostname"] = privateDNS
		}
		if publicDNS := convert.ToValue(instance.PublicDnsName); publicDNS != "" {
			mdata["public-hostname"] = publicDNS
		}
		if vpcID := convert.ToValue(instance.VpcId); vpcID != "" {
			mdata["vpc-id"] = vpcID
		}

		macs := map[string]*macDetails{}
		for _, nInterface := range instance.NetworkInterfaces {
			mac := convert.ToValue(nInterface.MacAddress)
			if mac == "" {
				continue
			}

			md, exist := macs[mac]
			if !exist {
				md = &macDetails{
					MAC:        mac,
					LocalIPv4s: convert.ToValue(nInterface.PrivateIpAddress),
				}
				macs[mac] = md
			}

			if nInterface.Association != nil {
				// We may have found a public ip
				md.PublicIPv4s = convert.ToValue(nInterface.Association.PublicIp)
			}
		}

		mdata["network"] = map[string]any{"interfaces": macs}
		return mdata, nil
	}

	switch {
	case m.platform.IsFamily(inventory.FAMILY_UNIX):
		return m.unixMetadata()
	case m.platform.IsFamily(inventory.FAMILY_WINDOWS):
		return m.windowsMetadata()
	default:
		return nil, errors.New("your platform is not supported by aws metadata identifier resource")
	}
}

func (m *ebsMetadata) Identify() (Identity, error) {
	log.Debug().Msg("getting ebs device identity")
	identity := Identity{}

	guid, err := machineid.MachineId(m.conn, m.platform)
	if err != nil {
		return identity, errors.Wrap(err, "unable to identify platform metadata")
	}

	identity.InstanceMachineID = guid

	// if we get execute by our serverless offering, we will have an injected platform-id
	if platformID, instanceID, ok := m.extractInjectedPlatformID(); ok {
		identity.InstanceID = instanceID.Id
		identity.PlatformIDs = []string{"//platformid.api.mondoo.app/runtime/aws/accounts/" + instanceID.Account, platformID}
	} else if id, ok := m.getInstanceID(); ok {
		// if we couldn't detect the injected platform-id, try to get information from the volume itself
		identity.InstanceID = id
		identity.PlatformIDs = []string{
			"//platformid.api.mondoo.app/machineid/" + guid,
			"//platformid.api.mondoo.app/aws/ebs/instances/" + id,
		}
	}

	return identity, nil
}

var instanceIdRE = regexp.MustCompile(`i-[0-9a-f]{17}`)

// TODO @afiune add windows locations
func (m *ebsMetadata) getInstanceID() (string, bool) {
	// list of files inside the mounted volume that might contain the id of the instance
	locations := []string{
		"/var/lib/cloud/data/instance-id",
		"/var/lib/amazon/ssm/runtimeconfig/identity_config.json",
		"/var/log/cloud-init.log",
	}
	for _, loc := range locations {
		data, err := afero.ReadFile(m.conn.FileSystem(), loc)
		if err != nil {
			log.Debug().Str("file", loc).Msg("instance id not found in mounted volume")
			continue
		}

		if match := instanceIdRE.FindString(string(data)); match != "" {
			return match, true
		}
	}

	return "", false
}

func (m *ebsMetadata) extractInjectedPlatformID() (string, *awsec2.MondooInstanceId, bool) {
	if asset := m.conn.Asset(); asset != nil {
		connections := asset.GetConnections()
		index := slices.IndexFunc(connections, func(c *inventory.Config) bool {
			if c != nil && c.Type == shared.Type_Device.String() {
				return true
			}
			return false
		})
		// @afiune we can't use `device.PlatformIdInject` because of a cyclic dep
		// TODO move it to a shared package
		platformInjected, ok := connections[index].Options["inject-platform-ids"]
		if ok {
			instanceId, err := awsec2.ParseMondooInstanceID(platformInjected)
			if err == nil {
				return platformInjected, instanceId, true
			}
		}
	}
	return "", nil, false
}

func (m *ebsMetadata) fetchEC2Instance() (*ec2types.Instance, bool) {
	id, ok := m.getInstanceID()
	if !ok || id == "" {
		log.Debug().Msg("no instance id found")
		return nil, false
	}

	cfg, err := awsConfig(m.conn)
	if err != nil {
		log.Debug().Err(err).Msg("unable to create aws client")
		return nil, false
	}

	ec2svc := ec2.NewFromConfig(cfg)
	ctx := context.Background()
	filters := []ec2types.Filter{
		{
			Name:   aws.String("instance-id"),
			Values: []string{id},
		},
	}
	resp, err := ec2svc.DescribeInstances(ctx, &ec2.DescribeInstancesInput{Filters: filters})
	if err != nil {
		log.Debug().Err(err).Msg("unable to describe instances")
		return nil, false
	}

	for _, r := range resp.Reservations {
		for _, i := range r.Instances {
			if convert.ToValue(i.InstanceId) == id {
				return &i, true
			}
		}
	}
	return nil, false
}

// awsConfig looks at the connection to see if it has additional options that need
// to be used to create an AWS configuration.
func awsConfig(conn shared.Connection) (aws.Config, error) {
	awsConfigOptions := []func(*config.LoadOptions) error{}

	if asset := conn.Asset(); asset != nil && len(asset.Connections) != 0 {
		for key, value := range asset.Connections[0].Options {
			switch key {
			case "region":
			case "profile":
				awsConfigOptions = append(awsConfigOptions, config.WithSharedConfigProfile(value))
			}
		}
	}
	region, err := afero.ReadFile(conn.FileSystem(), "/etc/dnf/vars/awsregion")
	if err == nil {
		awsConfigOptions = append(awsConfigOptions, config.WithRegion(string(region)))
	}

	return config.LoadDefaultConfig(context.Background(), awsConfigOptions...)
}
