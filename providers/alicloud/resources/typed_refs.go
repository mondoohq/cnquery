// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"fmt"

	vpcclient "github.com/alibabacloud-go/vpc-20160428/v6/client"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/alicloud/connection"
)

// resolveVpcNetwork returns the typed VPC network for a native vpc id within a
// region, or (nil, nil) when vpcID is empty (the caller sets StateIsNull). The
// underlying init reuses an already-listed network from the resource cache and
// otherwise fetches it via DescribeVpcs.
func resolveVpcNetwork(runtime *plugin.Runtime, region, vpcID string) (*mqlAlicloudVpcNetwork, error) {
	if vpcID == "" {
		return nil, nil
	}
	res, err := NewResource(runtime, "alicloud.vpc.network", map[string]*llx.RawData{
		"vpcId":    llx.StringData(vpcID),
		"regionId": llx.StringData(region),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAlicloudVpcNetwork), nil
}

// resolveVpcVswitch is the vSwitch equivalent of resolveVpcNetwork.
func resolveVpcVswitch(runtime *plugin.Runtime, region, vswitchID string) (*mqlAlicloudVpcVswitch, error) {
	if vswitchID == "" {
		return nil, nil
	}
	res, err := NewResource(runtime, "alicloud.vpc.vswitch", map[string]*llx.RawData{
		"vSwitchId": llx.StringData(vswitchID),
		"regionId":  llx.StringData(region),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAlicloudVpcVswitch), nil
}

// initAlicloudVpcNetwork resolves a VPC network by its native vpc id within a
// region. It backs both direct lookups and typed vpc() cross-references, reusing
// the cached instance when the network has already been listed.
func initAlicloudVpcNetwork(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}

	vpcID, err := requiredStringArg(args, "vpcId", "alicloud.vpc.network")
	if err != nil {
		return nil, nil, err
	}
	region, err := requiredStringArg(args, "regionId", "alicloud.vpc.network")
	if err != nil {
		return nil, nil, err
	}

	key := region + "/" + vpcID
	if x, ok := runtime.Resources.Get("alicloud.vpc.network\x00" + key); ok {
		return nil, x, nil
	}

	conn := runtime.Connection.(*connection.AlicloudConnection)
	client, err := conn.VpcClient(region)
	if err != nil {
		return nil, nil, err
	}
	resp, err := client.DescribeVpcs(&vpcclient.DescribeVpcsRequest{
		RegionId: &region,
		VpcId:    &vpcID,
	})
	if err != nil {
		return nil, nil, err
	}
	if resp != nil && resp.Body != nil && resp.Body.Vpcs != nil {
		for _, vpc := range resp.Body.Vpcs.Vpc {
			if vpc == nil || vpc.VpcId == nil || *vpc.VpcId != vpcID {
				continue
			}
			res, err := newVpcNetwork(runtime, region, vpc)
			if err != nil {
				return nil, nil, err
			}
			return nil, res, nil
		}
	}
	return nil, nil, fmt.Errorf("alicloud.vpc.network %q not found in region %q", vpcID, region)
}

// initAlicloudVpcVswitch resolves a vSwitch by its native id within a region. It
// backs both direct lookups and typed vswitch() cross-references, reusing the
// cached instance when the vSwitch has already been listed.
func initAlicloudVpcVswitch(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}

	vswitchID, err := requiredStringArg(args, "vSwitchId", "alicloud.vpc.vswitch")
	if err != nil {
		return nil, nil, err
	}
	region, err := requiredStringArg(args, "regionId", "alicloud.vpc.vswitch")
	if err != nil {
		return nil, nil, err
	}

	key := region + "/" + vswitchID
	if x, ok := runtime.Resources.Get("alicloud.vpc.vswitch\x00" + key); ok {
		return nil, x, nil
	}

	conn := runtime.Connection.(*connection.AlicloudConnection)
	client, err := conn.VpcClient(region)
	if err != nil {
		return nil, nil, err
	}
	resp, err := client.DescribeVSwitches(&vpcclient.DescribeVSwitchesRequest{
		RegionId:  &region,
		VSwitchId: &vswitchID,
	})
	if err != nil {
		return nil, nil, err
	}
	if resp != nil && resp.Body != nil && resp.Body.VSwitches != nil {
		for _, vsw := range resp.Body.VSwitches.VSwitch {
			if vsw == nil || vsw.VSwitchId == nil || *vsw.VSwitchId != vswitchID {
				continue
			}
			res, err := newVpcVswitch(runtime, region, vsw)
			if err != nil {
				return nil, nil, err
			}
			return nil, res, nil
		}
	}
	return nil, nil, fmt.Errorf("alicloud.vpc.vswitch %q not found in region %q", vswitchID, region)
}

// requiredStringArg reads a required non-empty string argument from an init
// args map, returning a descriptive error when it is missing or blank.
func requiredStringArg(args map[string]*llx.RawData, name, resource string) (string, error) {
	raw, ok := args[name]
	if !ok {
		return "", errors.New(resource + " requires a " + name + " to look up")
	}
	v, ok := raw.Value.(string)
	if !ok || v == "" {
		return "", errors.New(resource + " requires a " + name + " to look up")
	}
	return v, nil
}
