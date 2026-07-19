// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"strconv"
	"time"

	ecsclient "github.com/alibabacloud-go/ecs-20140526/v6/client"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/alicloud/connection"
	"go.mondoo.com/mql/v13/types"
)

// ecsTimeLayouts are the timestamp formats returned by the ECS API. Alibaba
// Cloud usually returns minute-precision UTC timestamps (2017-12-10T04:04Z),
// but some fields carry seconds, so both are attempted.
var ecsTimeLayouts = []string{
	"2006-01-02T15:04Z",
	time.RFC3339,
	"2006-01-02T15:04:05Z",
}

// parseEcsTime parses an Alibaba Cloud RFC3339-style timestamp, returning nil
// on a nil pointer, an empty string, or an unparseable value.
func parseEcsTime(s *string) *time.Time {
	if s == nil || *s == "" {
		return nil
	}
	for _, layout := range ecsTimeLayouts {
		if t, err := time.Parse(layout, *s); err == nil {
			return &t
		}
	}
	return nil
}

// strPtrSlice dereferences a slice of string pointers, skipping nil elements.
func strPtrSlice(in []*string) []string {
	out := make([]string, 0, len(in))
	for _, s := range in {
		if s != nil {
			out = append(out, *s)
		}
	}
	return out
}

// ecsTagsToMap converts a repeated TagKey/TagValue list into a string map.
func ecsTagsToMap(tags []*ecsclient.DescribeInstancesResponseBodyInstancesInstanceTagsTag) map[string]any {
	out := map[string]any{}
	for _, t := range tags {
		if t == nil || t.TagKey == nil {
			continue
		}
		var v string
		if t.TagValue != nil {
			v = *t.TagValue
		}
		out[*t.TagKey] = v
	}
	return out
}

func (r *mqlAlicloudEcs) id() (string, error) {
	return "alicloud.ecs", nil
}

// ---------------------------------------------------------------------------
// alicloud.ecs.instance
// ---------------------------------------------------------------------------

func (r *mqlAlicloudEcs) instances() ([]any, error) {
	conn := r.MqlRuntime.Connection.(*connection.AlicloudConnection)
	regions, err := conn.GetRegions()
	if err != nil {
		return nil, err
	}

	res := []any{}
	for _, region := range regions {
		instances, err := ecsInstancesInRegion(r.MqlRuntime, conn, region)
		if err != nil {
			return nil, err
		}
		res = append(res, instances...)
	}
	return res, nil
}

// ecsInstancesInRegion lists every instance in one region and maps each to an
// alicloud.ecs.instance resource.
func ecsInstancesInRegion(runtime *plugin.Runtime, conn *connection.AlicloudConnection, region string) ([]any, error) {
	client, err := conn.EcsClient(region)
	if err != nil {
		return nil, err
	}

	res := []any{}
	req := &ecsclient.DescribeInstancesRequest{
		RegionId:   &region,
		MaxResults: int32Ptr(100),
	}
	for {
		resp, err := client.DescribeInstances(req)
		if err != nil {
			return nil, err
		}
		if resp == nil || resp.Body == nil || resp.Body.Instances == nil {
			break
		}

		for _, inst := range resp.Body.Instances.Instance {
			if inst == nil {
				continue
			}
			mqlInst, err := newMqlEcsInstance(runtime, region, inst)
			if err != nil {
				return nil, err
			}
			res = append(res, mqlInst)
		}

		if resp.Body.NextToken == nil || *resp.Body.NextToken == "" {
			break
		}
		req.NextToken = resp.Body.NextToken
	}
	return res, nil
}

// newMqlEcsInstance maps one DescribeInstances item to a resource.
func newMqlEcsInstance(runtime *plugin.Runtime, region string, inst *ecsclient.DescribeInstancesResponseBodyInstancesInstance) (*mqlAlicloudEcsInstance, error) {
	var vpcId, vswitchId *string
	var privateIps []string
	if inst.VpcAttributes != nil {
		vpcId = inst.VpcAttributes.VpcId
		vswitchId = inst.VpcAttributes.VSwitchId
		if inst.VpcAttributes.PrivateIpAddress != nil {
			privateIps = strPtrSlice(inst.VpcAttributes.PrivateIpAddress.IpAddress)
		}
	}

	securityGroupIds := []string{}
	if inst.SecurityGroupIds != nil {
		securityGroupIds = strPtrSlice(inst.SecurityGroupIds.SecurityGroupId)
	}

	publicIps := []string{}
	if inst.PublicIpAddress != nil {
		publicIps = strPtrSlice(inst.PublicIpAddress.IpAddress)
	}

	var eip *string
	if inst.EipAddress != nil {
		eip = inst.EipAddress.IpAddress
	}

	var tags []*ecsclient.DescribeInstancesResponseBodyInstancesInstanceTagsTag
	if inst.Tags != nil {
		tags = inst.Tags.Tag
	}

	instanceId := ""
	if inst.InstanceId != nil {
		instanceId = *inst.InstanceId
	}

	// SpotPriceLimit is a *float32 and has no pointer helper, so it is mapped
	// to a float value or an explicit null.
	spotPriceLimit := llx.NilData
	if inst.SpotPriceLimit != nil {
		spotPriceLimit = llx.FloatData(float64(*inst.SpotPriceLimit))
	}

	args := map[string]*llx.RawData{
		"__id":                    llx.StringData(region + "/" + instanceId),
		"instanceId":              llx.StringDataPtr(inst.InstanceId),
		"instanceName":            llx.StringDataPtr(inst.InstanceName),
		"description":             llx.StringDataPtr(inst.Description),
		"status":                  llx.StringDataPtr(inst.Status),
		"instanceType":            llx.StringDataPtr(inst.InstanceType),
		"instanceTypeFamily":      llx.StringDataPtr(inst.InstanceTypeFamily),
		"imageId":                 llx.StringDataPtr(inst.ImageId),
		"regionId":                llx.StringData(region),
		"zoneId":                  llx.StringDataPtr(inst.ZoneId),
		"cpu":                     llx.IntDataPtr(inst.Cpu),
		"memory":                  llx.IntDataPtr(inst.Memory),
		"osName":                  llx.StringDataPtr(inst.OSName),
		"osNameEn":                llx.StringDataPtr(inst.OSNameEn),
		"osType":                  llx.StringDataPtr(inst.OSType),
		"hostName":                llx.StringDataPtr(inst.HostName),
		"serialNumber":            llx.StringDataPtr(inst.SerialNumber),
		"instanceChargeType":      llx.StringDataPtr(inst.InstanceChargeType),
		"spotStrategy":            llx.StringDataPtr(inst.SpotStrategy),
		"spotPriceLimit":          spotPriceLimit,
		"spotDuration":            llx.IntDataPtr(inst.SpotDuration),
		"internetChargeType":      llx.StringDataPtr(inst.InternetChargeType),
		"internetMaxBandwidthIn":  llx.IntDataPtr(inst.InternetMaxBandwidthIn),
		"internetMaxBandwidthOut": llx.IntDataPtr(inst.InternetMaxBandwidthOut),
		"creationTime":            llx.TimeDataPtr(parseEcsTime(inst.CreationTime)),
		"startTime":               llx.TimeDataPtr(parseEcsTime(inst.StartTime)),
		"expiredTime":             llx.TimeDataPtr(parseEcsTime(inst.ExpiredTime)),
		"autoReleaseTime":         llx.TimeDataPtr(parseEcsTime(inst.AutoReleaseTime)),
		"stoppedMode":             llx.StringDataPtr(inst.StoppedMode),
		"deploymentSetId":         llx.StringDataPtr(inst.DeploymentSetId),
		"keyPairName":             llx.StringDataPtr(inst.KeyPairName),
		"deletionProtection":      llx.BoolDataPtr(inst.DeletionProtection),
		"ioOptimized":             llx.BoolDataPtr(inst.IoOptimized),
		"gpuAmount":               llx.IntDataPtr(inst.GPUAmount),
		"gpuSpec":                 llx.StringDataPtr(inst.GPUSpec),
		"creditSpecification":     llx.StringDataPtr(inst.CreditSpecification),
		"deviceAvailable":         llx.BoolDataPtr(inst.DeviceAvailable),
		"localStorageAmount":      llx.IntDataPtr(inst.LocalStorageAmount),
		"localStorageCapacity":    llx.IntDataPtr(inst.LocalStorageCapacity),
		"resourceGroupId":         llx.StringDataPtr(inst.ResourceGroupId),
		"networkType":             llx.StringDataPtr(inst.InstanceNetworkType),
		"vpcId":                   llx.StringDataPtr(vpcId),
		"vswitchId":               llx.StringDataPtr(vswitchId),
		"securityGroupIds":        llx.ArrayData(llx.TArr2Raw(securityGroupIds), types.String),
		"privateIpAddresses":      llx.ArrayData(llx.TArr2Raw(privateIps), types.String),
		"publicIpAddresses":       llx.ArrayData(llx.TArr2Raw(publicIps), types.String),
		"eipAddress":              llx.StringDataPtr(eip),
		"tags":                    llx.MapData(ecsTagsToMap(tags), types.String),
	}

	resource, err := CreateResource(runtime, "alicloud.ecs.instance", args)
	if err != nil {
		return nil, err
	}
	return resource.(*mqlAlicloudEcsInstance), nil
}

func (r *mqlAlicloudEcsInstance) id() (string, error) {
	return r.RegionId.Data + "/" + r.InstanceId.Data, nil
}

// securityGroups resolves the security groups applied to this instance by
// listing the groups in the instance's region and matching by ID.
func (r *mqlAlicloudEcsInstance) securityGroups() ([]any, error) {
	wanted := map[string]struct{}{}
	for _, id := range r.SecurityGroupIds.Data {
		if s, ok := id.(string); ok {
			wanted[s] = struct{}{}
		}
	}
	if len(wanted) == 0 {
		return []any{}, nil
	}

	conn := r.MqlRuntime.Connection.(*connection.AlicloudConnection)
	groups, err := ecsSecurityGroupsInRegion(r.MqlRuntime, conn, r.RegionId.Data)
	if err != nil {
		return nil, err
	}

	res := []any{}
	for _, g := range groups {
		sg, ok := g.(*mqlAlicloudEcsSecuritygroup)
		if !ok {
			continue
		}
		if _, match := wanted[sg.SecurityGroupId.Data]; match {
			res = append(res, sg)
		}
	}
	return res, nil
}

// ---------------------------------------------------------------------------
// alicloud.ecs.disk
// ---------------------------------------------------------------------------

func ecsDiskTagsToMap(tags []*ecsclient.DescribeDisksResponseBodyDisksDiskTagsTag) map[string]any {
	out := map[string]any{}
	for _, t := range tags {
		if t == nil || t.TagKey == nil {
			continue
		}
		var v string
		if t.TagValue != nil {
			v = *t.TagValue
		}
		out[*t.TagKey] = v
	}
	return out
}

func (r *mqlAlicloudEcs) disks() ([]any, error) {
	conn := r.MqlRuntime.Connection.(*connection.AlicloudConnection)
	regions, err := conn.GetRegions()
	if err != nil {
		return nil, err
	}

	res := []any{}
	for _, region := range regions {
		client, err := conn.EcsClient(region)
		if err != nil {
			return nil, err
		}

		req := &ecsclient.DescribeDisksRequest{
			RegionId:   &region,
			MaxResults: int32Ptr(100),
		}
		for {
			resp, err := client.DescribeDisks(req)
			if err != nil {
				return nil, err
			}
			if resp == nil || resp.Body == nil || resp.Body.Disks == nil {
				break
			}

			for _, disk := range resp.Body.Disks.Disk {
				if disk == nil {
					continue
				}
				var tags []*ecsclient.DescribeDisksResponseBodyDisksDiskTagsTag
				if disk.Tags != nil {
					tags = disk.Tags.Tag
				}
				diskId := ""
				if disk.DiskId != nil {
					diskId = *disk.DiskId
				}
				resource, err := CreateResource(r.MqlRuntime, "alicloud.ecs.disk", map[string]*llx.RawData{
					"__id":               llx.StringData(region + "/" + diskId),
					"diskId":             llx.StringDataPtr(disk.DiskId),
					"diskName":           llx.StringDataPtr(disk.DiskName),
					"description":        llx.StringDataPtr(disk.Description),
					"type":               llx.StringDataPtr(disk.Type),
					"category":           llx.StringDataPtr(disk.Category),
					"size":               llx.IntDataPtr(disk.Size),
					"status":             llx.StringDataPtr(disk.Status),
					"encrypted":          llx.BoolDataPtr(disk.Encrypted),
					"kmsKeyId":           llx.StringDataPtr(disk.KMSKeyId),
					"instanceId":         llx.StringDataPtr(disk.InstanceId),
					"device":             llx.StringDataPtr(disk.Device),
					"deleteWithInstance": llx.BoolDataPtr(disk.DeleteWithInstance),
					"deleteAutoSnapshot": llx.BoolDataPtr(disk.DeleteAutoSnapshot),
					"enableAutoSnapshot": llx.BoolDataPtr(disk.EnableAutoSnapshot),
					"portable":           llx.BoolDataPtr(disk.Portable),
					"zoneId":             llx.StringDataPtr(disk.ZoneId),
					"regionId":           llx.StringData(region),
					"creationTime":       llx.TimeDataPtr(parseEcsTime(disk.CreationTime)),
					"attachedTime":       llx.TimeDataPtr(parseEcsTime(disk.AttachedTime)),
					"detachedTime":       llx.TimeDataPtr(parseEcsTime(disk.DetachedTime)),
					"performanceLevel":   llx.StringDataPtr(disk.PerformanceLevel),
					"diskChargeType":     llx.StringDataPtr(disk.DiskChargeType),
					"tags":               llx.MapData(ecsDiskTagsToMap(tags), types.String),
				})
				if err != nil {
					return nil, err
				}
				res = append(res, resource)
			}

			if resp.Body.NextToken == nil || *resp.Body.NextToken == "" {
				break
			}
			req.NextToken = resp.Body.NextToken
		}
	}
	return res, nil
}

func (r *mqlAlicloudEcsDisk) id() (string, error) {
	return r.RegionId.Data + "/" + r.DiskId.Data, nil
}

// ---------------------------------------------------------------------------
// alicloud.ecs.image
// ---------------------------------------------------------------------------

func (r *mqlAlicloudEcs) images() ([]any, error) {
	conn := r.MqlRuntime.Connection.(*connection.AlicloudConnection)
	regions, err := conn.GetRegions()
	if err != nil {
		return nil, err
	}

	res := []any{}
	for _, region := range regions {
		client, err := conn.EcsClient(region)
		if err != nil {
			return nil, err
		}

		pageNumber := int32(1)
		pageSize := int32(50)
		for {
			req := &ecsclient.DescribeImagesRequest{
				RegionId:   &region,
				PageNumber: &pageNumber,
				PageSize:   &pageSize,
			}
			resp, err := client.DescribeImages(req)
			if err != nil {
				return nil, err
			}
			if resp == nil || resp.Body == nil || resp.Body.Images == nil {
				break
			}

			count := 0
			for _, img := range resp.Body.Images.Image {
				if img == nil {
					continue
				}
				count++
				imageId := ""
				if img.ImageId != nil {
					imageId = *img.ImageId
				}
				resource, err := CreateResource(r.MqlRuntime, "alicloud.ecs.image", map[string]*llx.RawData{
					"__id":               llx.StringData(region + "/" + imageId),
					"imageId":            llx.StringDataPtr(img.ImageId),
					"imageName":          llx.StringDataPtr(img.ImageName),
					"description":        llx.StringDataPtr(img.Description),
					"osName":             llx.StringDataPtr(img.OSName),
					"osType":             llx.StringDataPtr(img.OSType),
					"architecture":       llx.StringDataPtr(img.Architecture),
					"size":               llx.IntDataPtr(img.Size),
					"imageOwnerAlias":    llx.StringDataPtr(img.ImageOwnerAlias),
					"status":             llx.StringDataPtr(img.Status),
					"isPublic":           llx.BoolDataPtr(img.IsPublic),
					"isSelfShared":       llx.StringDataPtr(img.IsSelfShared),
					"isSupportCloudinit": llx.BoolDataPtr(img.IsSupportCloudinit),
					"platform":           llx.StringDataPtr(img.Platform),
					"creationTime":       llx.TimeDataPtr(parseEcsTime(img.CreationTime)),
					"imageVersion":       llx.StringDataPtr(img.ImageVersion),
					"usage":              llx.StringDataPtr(img.Usage),
					"regionId":           llx.StringData(region),
				})
				if err != nil {
					return nil, err
				}
				res = append(res, resource)
			}

			total := int32(0)
			if resp.Body.TotalCount != nil {
				total = *resp.Body.TotalCount
			}
			if count == 0 || pageNumber*pageSize >= total {
				break
			}
			pageNumber++
		}
	}
	return res, nil
}

func (r *mqlAlicloudEcsImage) id() (string, error) {
	return r.RegionId.Data + "/" + r.ImageId.Data, nil
}

// ---------------------------------------------------------------------------
// alicloud.ecs.keypair
// ---------------------------------------------------------------------------

func ecsKeyPairTagsToMap(tags []*ecsclient.DescribeKeyPairsResponseBodyKeyPairsKeyPairTagsTag) map[string]any {
	out := map[string]any{}
	for _, t := range tags {
		if t == nil || t.TagKey == nil {
			continue
		}
		var v string
		if t.TagValue != nil {
			v = *t.TagValue
		}
		out[*t.TagKey] = v
	}
	return out
}

func (r *mqlAlicloudEcs) keyPairs() ([]any, error) {
	conn := r.MqlRuntime.Connection.(*connection.AlicloudConnection)
	regions, err := conn.GetRegions()
	if err != nil {
		return nil, err
	}

	res := []any{}
	for _, region := range regions {
		client, err := conn.EcsClient(region)
		if err != nil {
			return nil, err
		}

		pageNumber := int32(1)
		pageSize := int32(50)
		for {
			req := &ecsclient.DescribeKeyPairsRequest{
				RegionId:   &region,
				PageNumber: &pageNumber,
				PageSize:   &pageSize,
			}
			resp, err := client.DescribeKeyPairs(req)
			if err != nil {
				return nil, err
			}
			if resp == nil || resp.Body == nil || resp.Body.KeyPairs == nil {
				break
			}

			count := 0
			for _, kp := range resp.Body.KeyPairs.KeyPair {
				if kp == nil {
					continue
				}
				count++
				var tags []*ecsclient.DescribeKeyPairsResponseBodyKeyPairsKeyPairTagsTag
				if kp.Tags != nil {
					tags = kp.Tags.Tag
				}
				name := ""
				if kp.KeyPairName != nil {
					name = *kp.KeyPairName
				}
				resource, err := CreateResource(r.MqlRuntime, "alicloud.ecs.keypair", map[string]*llx.RawData{
					"__id":               llx.StringData(region + "/" + name),
					"keyPairName":        llx.StringDataPtr(kp.KeyPairName),
					"keyPairFingerPrint": llx.StringDataPtr(kp.KeyPairFingerPrint),
					"creationTime":       llx.TimeDataPtr(parseEcsTime(kp.CreationTime)),
					"resourceGroupId":    llx.StringDataPtr(kp.ResourceGroupId),
					"regionId":           llx.StringData(region),
					"tags":               llx.MapData(ecsKeyPairTagsToMap(tags), types.String),
				})
				if err != nil {
					return nil, err
				}
				res = append(res, resource)
			}

			total := int32(0)
			if resp.Body.TotalCount != nil {
				total = *resp.Body.TotalCount
			}
			if count == 0 || pageNumber*pageSize >= total {
				break
			}
			pageNumber++
		}
	}
	return res, nil
}

func (r *mqlAlicloudEcsKeypair) id() (string, error) {
	return r.RegionId.Data + "/" + r.KeyPairName.Data, nil
}

// ---------------------------------------------------------------------------
// alicloud.ecs.securitygroup
// ---------------------------------------------------------------------------

func ecsSecurityGroupTagsToMap(tags []*ecsclient.DescribeSecurityGroupsResponseBodySecurityGroupsSecurityGroupTagsTag) map[string]any {
	out := map[string]any{}
	for _, t := range tags {
		if t == nil || t.TagKey == nil {
			continue
		}
		var v string
		if t.TagValue != nil {
			v = *t.TagValue
		}
		out[*t.TagKey] = v
	}
	return out
}

func (r *mqlAlicloudEcs) securityGroups() ([]any, error) {
	conn := r.MqlRuntime.Connection.(*connection.AlicloudConnection)
	regions, err := conn.GetRegions()
	if err != nil {
		return nil, err
	}

	res := []any{}
	for _, region := range regions {
		groups, err := ecsSecurityGroupsInRegion(r.MqlRuntime, conn, region)
		if err != nil {
			return nil, err
		}
		res = append(res, groups...)
	}
	return res, nil
}

// ecsSecurityGroupsInRegion lists every security group in one region and maps
// each to an alicloud.ecs.securitygroup resource.
func ecsSecurityGroupsInRegion(runtime *plugin.Runtime, conn *connection.AlicloudConnection, region string) ([]any, error) {
	client, err := conn.EcsClient(region)
	if err != nil {
		return nil, err
	}

	res := []any{}
	req := &ecsclient.DescribeSecurityGroupsRequest{
		RegionId:   &region,
		MaxResults: int32Ptr(100),
	}
	for {
		resp, err := client.DescribeSecurityGroups(req)
		if err != nil {
			return nil, err
		}
		if resp == nil || resp.Body == nil || resp.Body.SecurityGroups == nil {
			break
		}

		for _, sg := range resp.Body.SecurityGroups.SecurityGroup {
			if sg == nil {
				continue
			}
			var tags []*ecsclient.DescribeSecurityGroupsResponseBodySecurityGroupsSecurityGroupTagsTag
			if sg.Tags != nil {
				tags = sg.Tags.Tag
			}
			sgId := ""
			if sg.SecurityGroupId != nil {
				sgId = *sg.SecurityGroupId
			}
			resource, err := CreateResource(runtime, "alicloud.ecs.securitygroup", map[string]*llx.RawData{
				"__id":              llx.StringData(region + "/" + sgId),
				"securityGroupId":   llx.StringDataPtr(sg.SecurityGroupId),
				"securityGroupName": llx.StringDataPtr(sg.SecurityGroupName),
				"description":       llx.StringDataPtr(sg.Description),
				"vpcId":             llx.StringDataPtr(sg.VpcId),
				"securityGroupType": llx.StringDataPtr(sg.SecurityGroupType),
				"creationTime":      llx.TimeDataPtr(parseEcsTime(sg.CreationTime)),
				"regionId":          llx.StringData(region),
				"resourceGroupId":   llx.StringDataPtr(sg.ResourceGroupId),
				"tags":              llx.MapData(ecsSecurityGroupTagsToMap(tags), types.String),
			})
			if err != nil {
				return nil, err
			}
			res = append(res, resource)
		}

		if resp.Body.NextToken == nil || *resp.Body.NextToken == "" {
			break
		}
		req.NextToken = resp.Body.NextToken
	}
	return res, nil
}

func (r *mqlAlicloudEcsSecuritygroup) id() (string, error) {
	return r.RegionId.Data + "/" + r.SecurityGroupId.Data, nil
}

// instances resolves the instances the security group is applied to by listing
// the instances in the group's region and matching on their security group IDs.
func (r *mqlAlicloudEcsSecuritygroup) instances() ([]any, error) {
	conn := r.MqlRuntime.Connection.(*connection.AlicloudConnection)
	instances, err := ecsInstancesInRegion(r.MqlRuntime, conn, r.RegionId.Data)
	if err != nil {
		return nil, err
	}

	sgId := r.SecurityGroupId.Data
	res := []any{}
	for _, i := range instances {
		inst, ok := i.(*mqlAlicloudEcsInstance)
		if !ok {
			continue
		}
		for _, id := range inst.SecurityGroupIds.Data {
			if s, ok := id.(string); ok && s == sgId {
				res = append(res, inst)
				break
			}
		}
	}
	return res, nil
}

// permissions fetches the inbound and outbound rules of the security group via
// DescribeSecurityGroupAttribute, mapping each rule to a permission resource.
func (r *mqlAlicloudEcsSecuritygroup) permissions() ([]any, error) {
	conn := r.MqlRuntime.Connection.(*connection.AlicloudConnection)
	region := r.RegionId.Data
	client, err := conn.EcsClient(region)
	if err != nil {
		return nil, err
	}

	sgId := r.SecurityGroupId.Data
	res := []any{}
	idx := 0
	req := &ecsclient.DescribeSecurityGroupAttributeRequest{
		RegionId:        &region,
		SecurityGroupId: &sgId,
		MaxResults:      int32Ptr(100),
	}
	for {
		resp, err := client.DescribeSecurityGroupAttribute(req)
		if err != nil {
			return nil, err
		}
		if resp == nil || resp.Body == nil || resp.Body.Permissions == nil {
			break
		}

		for _, p := range resp.Body.Permissions.Permission {
			if p == nil {
				continue
			}
			direction := ""
			if p.Direction != nil {
				direction = *p.Direction
			}
			resource, err := CreateResource(r.MqlRuntime, "alicloud.ecs.securitygroup.permission", map[string]*llx.RawData{
				"__id":                llx.StringData(sgId + "/" + direction + "/" + strconv.Itoa(idx)),
				"securityGroupRuleId": llx.StringDataPtr(p.SecurityGroupRuleId),
				"direction":           llx.StringDataPtr(p.Direction),
				"policy":              llx.StringDataPtr(p.Policy),
				"priority":            llx.StringDataPtr(p.Priority),
				"ipProtocol":          llx.StringDataPtr(p.IpProtocol),
				"nicType":             llx.StringDataPtr(p.NicType),
				"portRange":           llx.StringDataPtr(p.PortRange),
				"sourcePortRange":     llx.StringDataPtr(p.SourcePortRange),
				"sourceCidrIp":        llx.StringDataPtr(p.SourceCidrIp),
				"sourceGroupId":       llx.StringDataPtr(p.SourceGroupId),
				"sourcePrefixListId":  llx.StringDataPtr(p.SourcePrefixListId),
				"ipv6SourceCidrIp":    llx.StringDataPtr(p.Ipv6SourceCidrIp),
				"destCidrIp":          llx.StringDataPtr(p.DestCidrIp),
				"destGroupId":         llx.StringDataPtr(p.DestGroupId),
				"destPrefixListId":    llx.StringDataPtr(p.DestPrefixListId),
				"ipv6DestCidrIp":      llx.StringDataPtr(p.Ipv6DestCidrIp),
				"description":         llx.StringDataPtr(p.Description),
				"createTime":          llx.TimeDataPtr(parseEcsTime(p.CreateTime)),
			})
			if err != nil {
				return nil, err
			}
			res = append(res, resource)
			idx++
		}

		if resp.Body.NextToken == nil || *resp.Body.NextToken == "" {
			break
		}
		req.NextToken = resp.Body.NextToken
	}
	return res, nil
}

func (r *mqlAlicloudEcsSecuritygroupPermission) id() (string, error) {
	return r.__id, nil
}

// int32Ptr returns a pointer to an int32 literal for request paging fields.
func int32Ptr(v int32) *int32 {
	return &v
}
