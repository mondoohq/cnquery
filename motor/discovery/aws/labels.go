package aws

import (
	"go.mondoo.com/cnquery/motor/discovery/common"
)

const ImportedFromAWSTagKeyPrefix = "aws.tag/"

const (
	ImageIdLabel        string = "mondoo.com/ami-id"
	RegionLabel         string = "mondoo.com/region"
	IntegrationMrnLabel string = "mondoo.com/integration-mrn"
	SSMPingLabel        string = "mondoo.com/ssm-connection"
	InstanceLabel       string = "mondoo.com/instance"
	EBSScanLabel        string = "mondoo.com/ebs-volume-scan"
	PlatformLabel       string = "mondoo.com/platform"
	StateLabel          string = "mondoo.com/instance-state"
)

func addAWSMetadataLabels(assetLabels map[string]string, instance basicInstanceInfo) map[string]string {
	assetLabels[RegionLabel] = instance.Region
	if instance.InstanceId != nil {
		assetLabels[InstanceLabel] = *instance.InstanceId
	}
	if instance.IPAddress != nil {
		assetLabels[common.IPLabel] = *instance.IPAddress
	}
	if instance.PublicDnsName != nil {
		assetLabels[common.DNSLabel] = *instance.PublicDnsName
	}
	if instance.ImageId != nil {
		assetLabels[ImageIdLabel] = *instance.ImageId
	}
	if instance.PlatformType != "" {
		assetLabels[PlatformLabel] = instance.PlatformType
	}
	if instance.SSMPingStatus != "" {
		assetLabels[SSMPingLabel] = instance.SSMPingStatus
	}
	if instance.State != "" {
		assetLabels[StateLabel] = instance.State
	}
	if instance.AccountId != "" {
		assetLabels[common.ParentId] = instance.AccountId
	}
	return assetLabels
}

type basicInstanceInfo struct {
	InstanceId    *string
	IPAddress     *string
	Region        string
	PublicDnsName *string
	ImageId       *string
	SSMPingStatus string
	PlatformType  string
	State         string
	AccountId     string
}

func ssmInstanceToBasicInstanceInfo(instanceInfo ssmInstanceInfo) basicInstanceInfo {
	return basicInstanceInfo{
		InstanceId:    instanceInfo.instance.InstanceId,
		IPAddress:     instanceInfo.instance.IPAddress,
		Region:        instanceInfo.region,
		SSMPingStatus: string(instanceInfo.instance.PingStatus),
		PlatformType:  string(instanceInfo.instance.PlatformType),
		AccountId:     instanceInfo.account,
	}
}

func ec2InstanceToBasicInstanceInfo(instanceInfo instanceInfo) basicInstanceInfo {
	return basicInstanceInfo{
		InstanceId:    instanceInfo.instance.InstanceId,
		IPAddress:     instanceInfo.instance.PublicIpAddress,
		Region:        instanceInfo.region,
		PublicDnsName: instanceInfo.instance.PublicDnsName,
		ImageId:       instanceInfo.instance.ImageId,
		PlatformType:  string(instanceInfo.instance.Platform),
		State:         string(instanceInfo.instance.State.Name),
		AccountId:     instanceInfo.account,
	}
}
