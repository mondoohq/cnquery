package aws

import (
	"testing"

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
