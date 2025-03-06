// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package cloud

import (
	"encoding/json"
	"errors"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v11/providers/os/connection/shared"
	"go.mondoo.com/cnquery/v11/providers/os/id/awsec2"
)

const AWS Provider = "aws"

type aws struct {
	conn shared.Connection
}

func (a *aws) Provider() Provider {
	return AWS
}

func (a *aws) Instance() (*InstanceMetadata, error) {
	mdsvc, err := awsec2.Resolve(a.conn, a.conn.Asset().GetPlatform())
	if err != nil {
		log.Debug().Err(err).Msg("os.cloud.aws> failed to get metadata resolver")
		return nil, err
	}
	metadata, err := mdsvc.RawMetadata()
	if err != nil {
		log.Debug().Err(err).Msg("os.cloud.aws> failed to get raw metadata")
		return nil, err
	}
	if metadata == nil {
		log.Debug().Msg("os.cloud.aws> no metadata found")
		return nil, errors.New("no metadata")
	}

	instanceMd := InstanceMetadata{Metadata: metadata}

	m, ok := metadata.(map[string]any)
	if !ok {
		return &instanceMd, errors.New("unexpected raw metadata")
	}

	if value, ok := m["public-hostname"]; ok {
		instanceMd.PublicHostname = value.(string)
	}
	if value, ok := m["hostname"]; ok {
		instanceMd.PrivateHostname = value.(string)
	}

	if value, ok := m["network"]; ok {
		byteData, err := json.Marshal(value)
		if err != nil {
			return &instanceMd, err
		}

		var network AWSNetwork
		if err := json.Unmarshal(byteData, &network); err != nil {
			return &instanceMd, err
		}

		// all network interfaces
		instanceMd.PublicIpv4 = make([]Ipv4Address, 0)
		instanceMd.PrivateIpv4 = make([]Ipv4Address, 0)
		for mac, details := range network.Interfaces.Macs {
			ignored := true

			if ip, ok := details.PublicIP(); ok {
				instanceMd.PublicIpv4 = append(instanceMd.PublicIpv4, ip)
				ignored = false
			}

			if ip, ok := details.PrivateIP(); ok {
				instanceMd.PrivateIpv4 = append(instanceMd.PrivateIpv4, ip)
				ignored = false
			}

			if ignored {
				log.Debug().
					Str("mac", mac).
					Interface("interface_details", details).
					Msg("no valid public or private ipaddress, skipping")
			}
		}

	}

	return &instanceMd, nil
}

// AWSNetwork structure for AWS
type AWSNetwork struct {
	Interfaces AWSInterfaces `json:"interfaces"`
}

// AWSInterfaces structure for AWS
type AWSInterfaces struct {
	Macs map[string]MacDetails `json:"macs"`
}

// MacDetails structure
type MacDetails struct {
	DeviceNumber        int64  `json:"device-number"`
	InterfaceID         string `json:"interface-id"`
	IPv4Associations    string `json:"ipv4-associations"`
	LocalHostname       string `json:"local-hostname"`
	LocalIPv4s          string `json:"local-ipv4s"`
	Mac                 string `json:"mac"`
	OwnerID             int64  `json:"owner-id"`
	PublicHostname      string `json:"public-hostname"`
	PublicIPv4s         string `json:"public-ipv4s"`
	SecurityGroupIDs    string `json:"security-group-ids"`
	SecurityGroups      string `json:"security-groups"`
	SubnetID            string `json:"subnet-id"`
	SubnetIPv4CIDRBlock string `json:"subnet-ipv4-cidr-block"`
	VPCID               string `json:"vpc-id"`
	VPCIPv4CIDRBlock    string `json:"vpc-ipv4-cidr-block"`
	VPCIPv4CIDRBlocks   string `json:"vpc-ipv4-cidr-blocks"`
}

// PublicIP detects if the network interface has a public ip address,
// if so it initializes an Ipv4Address struct and return true.
func (d MacDetails) PublicIP() (Ipv4Address, bool) {
	return Ipv4Address{IP: d.PublicIPv4s}, d.PublicIPv4s != ""
}

// PrivateIP detects if the network interface has a private ip address,
// if so it initializes an Ipv4Address structure and return true.
func (d MacDetails) PrivateIP() (Ipv4Address, bool) {
	// Note that AWS has two IP ranges, the VPC (`VPCIPv4CIDRBlock`) and the
	// Subnet (`SubnetIPv4CIDRBlock`), we use the logical segment since there
	// are cases where the subnet might have additional configuration like ACLs,
	// route tables, etc. that we can't detect from within the os
	return NewIpv4WithSubnet(d.LocalIPv4s, d.SubnetIPv4CIDRBlock), d.LocalIPv4s != ""
}
