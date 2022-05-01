package aws

import (
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/ssm/types"
	"go.mondoo.io/mondoo/motor/discovery/common"
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
	StateLabel          string = "mondoo.com`/instance-state"
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

func ssmInstanceToBasicInstanceInfo(instance types.InstanceInformation, region string, account string) basicInstanceInfo {
	return basicInstanceInfo{
		InstanceId:    instance.InstanceId,
		IPAddress:     instance.IPAddress,
		Region:        region,
		SSMPingStatus: string(instance.PingStatus),
		PlatformType:  string(instance.PlatformType),
		AccountId:     account,
	}
}

func ec2InstanceToBasicInstanceInfo(instance ec2types.Instance, region string, account string) basicInstanceInfo {
	return basicInstanceInfo{
		InstanceId:    instance.InstanceId,
		IPAddress:     instance.PublicIpAddress,
		Region:        region,
		PublicDnsName: instance.PublicDnsName,
		ImageId:       instance.ImageId,
		PlatformType:  string(instance.Platform),
		State:         string(instance.State.Name),
		AccountId:     account,
	}
}
