// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"fmt"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v11/providers/os/connection/shared"
	"go.mondoo.com/cnquery/v11/providers/os/resources/cloud"
	"go.mondoo.com/cnquery/v11/types"
)

func initCloud(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	conn := runtime.Connection.(shared.Connection)
	log.Debug().Msg("os.cloud> init")
	osCloud, err := cloud.Resolve(conn)
	if err != nil {
		return args, nil, err
	}
	raw, err := CreateResource(runtime, "cloud", map[string]*llx.RawData{
		"provider": llx.StringData(string(osCloud.Provider())),
	})
	if err != nil {
		return args, nil, err
	}

	return args, raw.(*mqlCloud), nil
}

func (c *mqlCloud) id() (string, error) {
	return c.GetProvider().Data, nil
}

func (c *mqlCloud) instance() (*mqlCloudInstance, error) {
	obj, err := NewResource(c.MqlRuntime, "cloud.instance", nil)
	if err != nil {
		return nil, err
	}
	return obj.(*mqlCloudInstance), nil
}

type mqlCloudInstanceInternal struct {
	instanceMd *cloud.InstanceMetadata
}

func initCloudInstance(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	conn := runtime.Connection.(shared.Connection)
	osCloud, err := cloud.Resolve(conn)
	if err != nil {
		return args, nil, err
	}

	raw, err := CreateResource(runtime, "cloud.instance", nil)
	if err != nil {
		return args, nil, err
	}
	cloudInstance := raw.(*mqlCloudInstance)
	cloudInstance.instanceMd, err = osCloud.Instance()
	return args, cloudInstance, err
}

func (i *mqlCloudInstance) id() (string, error) {
	if i.instanceMd != nil {
		return fmt.Sprintf("cloud.instance/%s", i.instanceMd.PublicHostname), nil
	}
	return "", nil
}

func (i *mqlCloudInstance) publicHostname() (value string, err error) {
	if i.instanceMd != nil {
		value = i.instanceMd.PublicHostname
	}
	return
}

func (i *mqlCloudInstance) privateHostname() (value string, err error) {
	if i.instanceMd != nil {
		value = i.instanceMd.PrivateHostname
	}
	return
}

func (i *mqlCloudInstance) privateIpv4() (value []interface{}, err error) {
	if i.instanceMd != nil {
		var resource plugin.Resource
		for _, ipaddress := range i.instanceMd.PrivateIpv4 {
			resource, err = NewResource(i.MqlRuntime, "ipv4Address", map[string]*llx.RawData{
				"__id":      llx.StringData(ipaddress.IP),
				"ip":        {Type: types.IP, Value: ipaddress.IP},
				"subnet":    llx.StringData(ipaddress.Subnet),
				"cidr":      llx.StringData(ipaddress.CIDR),
				"broadcast": llx.StringData(ipaddress.Broadcast),
				"gateway":   llx.StringData(ipaddress.Gateway),
			})
			if err != nil {
				return
			}
			value = append(value, resource)
		}
	}
	return
}

func (i *mqlCloudInstance) publicIpv4() (value []interface{}, err error) {
	if i.instanceMd != nil {
		var resource plugin.Resource
		for _, ipaddress := range i.instanceMd.PublicIpv4 {
			resource, err = NewResource(i.MqlRuntime, "ipv4Address", map[string]*llx.RawData{
				"__id":      llx.StringData(ipaddress.IP),
				"ip":        {Type: types.IP, Value: ipaddress.IP},
				"subnet":    llx.StringData(ipaddress.Subnet),
				"cidr":      llx.StringData(ipaddress.CIDR),
				"broadcast": llx.StringData(ipaddress.Broadcast),
				"gateway":   llx.StringData(ipaddress.Gateway),
			})
			if err != nil {
				return
			}
			value = append(value, resource)
		}
	}
	return
}

func (i *mqlCloudInstance) metadata() (value interface{}, err error) {
	if i.instanceMd != nil {
		value, err = convert.JsonToDict(i.instanceMd.Metadata)
	}
	return
}
