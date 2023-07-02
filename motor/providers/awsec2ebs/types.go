package awsec2ebs

import (
	"path"
	"regexp"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go/aws"
	"errors"
)

type InstanceId struct {
	Id             string
	Region         string
	Name           string
	Account        string
	Zone           string
	MarketplaceImg bool
}

func NewInstanceId(account string, region string, id string) (*InstanceId, error) {
	if region == "" || id == "" || account == "" {
		return nil, errors.New("invalid instance id. account, region and instance id required.")
	}
	return &InstanceId{Account: account, Region: region, Id: id}, nil
}

func (s *InstanceId) String() string {
	// e.g. account/999000999000/region/us-east-2/instance/i-0989478343232
	return path.Join("account", s.Account, "region", s.Region, "instance", s.Id)
}

func ParseInstanceId(path string) (*InstanceId, error) {
	if !IsValidInstanceId(path) {
		return nil, errors.New("invalid instance id. expected account/<id>/region/<region-val>/instance/<instance-id>")
	}
	keyValues := strings.Split(path, "/")
	if len(keyValues) != 6 {
		return nil, errors.New("invalid instance id. expected account/<id>/region/<region-val>/instance/<instance-id>")
	}
	return NewInstanceId(keyValues[1], keyValues[3], keyValues[5])
}

var VALID_INSTANCE_ID = regexp.MustCompile(`^account/\d{12}/region\/(us(-gov)?|ap|ca|cn|eu|sa)-(central|(north|south)?(east|west)?)-\d\/instance\/.+$`)

func IsValidInstanceId(path string) bool {
	return VALID_INSTANCE_ID.MatchString(path)
}

type SnapshotId struct {
	Id      string
	Region  string
	Account string
}

type VolumeInfo struct {
	Id          string
	Region      string
	Account     string
	IsAvailable bool
	Tags        map[string]string
}

func resourceTags(resourceType types.ResourceType, instanceId string) []types.TagSpecification {
	return []types.TagSpecification{
		{
			ResourceType: resourceType,
			Tags: []types.Tag{
				{Key: aws.String("createdBy"), Value: aws.String("Mondoo")},
				{Key: aws.String("Created By"), Value: aws.String("Mondoo")},
				{Key: aws.String("Created From Instance"), Value: aws.String(instanceId)},
			},
		},
	}
}

const (
	EBSTargetInstance = "instance"
	EBSTargetVolume   = "volume"
	EBSTargetSnapshot = "snapshot"
)

type EbsTransportTarget struct {
	Account string
	Region  string
	Id      string
	Type    string
}

func ParseEbsTransportUrl(path string) (*EbsTransportTarget, error) {
	keyValues := strings.Split(path, "/")
	if len(keyValues) != 6 {
		return nil, errors.New("invalid id. expected account/<id>/region/<region-val>/{instance, volume, or snapshot}/<id>")
	}

	var itemType string
	switch keyValues[4] {
	case "volume":
		itemType = EBSTargetVolume
	case "snapshot":
		itemType = EBSTargetSnapshot
	default:
		itemType = EBSTargetInstance
	}

	return &EbsTransportTarget{Account: keyValues[1], Region: keyValues[3], Id: keyValues[5], Type: itemType}, nil
}
