package aws

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	ssmtypes "github.com/aws/aws-sdk-go-v2/service/ssm/types"

	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/stretchr/testify/assert"
)

func TestAddAWSMetadataLabels(t *testing.T) {
	labels := map[string]string{"a": "1", "b": "2"}
	id := "i-90348u304"
	instance := basicInstanceInfo{
		ImageId:       nil,
		SSMPingStatus: "",
		InstanceId:    &id,
		Region:        "us-east-1",
	}
	expected := map[string]string{"a": "1", "b": "2", RegionLabel: "us-east-1", InstanceLabel: id}
	assert.Equal(t, addAWSMetadataLabels(labels, instance), expected)
}

func TestEc2InstanceToBasicInstanceInfo(t *testing.T) {
	i := ec2types.Instance{
		InstanceId:      aws.String("i-0000"),
		PublicIpAddress: aws.String("172.154.32.10"),
		Platform:        "windows",
		State:           &ec2types.InstanceState{Name: ec2types.InstanceStateNameTerminated},
	}
	expected := basicInstanceInfo{
		InstanceId:   aws.String("i-0000"),
		IPAddress:    aws.String("172.154.32.10"),
		Region:       "us-east-1",
		PlatformType: "windows",
		State:        "terminated",
	}
	assert.Equal(t, expected, ec2InstanceToBasicInstanceInfo(i, "us-east-1"))
}

func TestSSMInstanceToBasicInstanceInfo(t *testing.T) {
	i := ssmtypes.InstanceInformation{
		InstanceId: aws.String("i-0000"),
		IPAddress:  aws.String("172.154.32.10"),
		PingStatus: ssmtypes.PingStatusOnline,
	}
	expected := basicInstanceInfo{
		InstanceId:    aws.String("i-0000"),
		IPAddress:     aws.String("172.154.32.10"),
		Region:        "us-east-1",
		SSMPingStatus: "Online",
	}

	assert.Equal(t, expected, ssmInstanceToBasicInstanceInfo(i, "us-east-1"))

}
