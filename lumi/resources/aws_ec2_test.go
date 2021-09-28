package resources

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/stretchr/testify/assert"
)

func TestEc2TagsToMap(t *testing.T) {
	tags := []types.Tag{{Key: aws.String("1"), Value: aws.String("2")}, {Key: aws.String("3"), Value: aws.String("4")}}
	expected := make(map[string]interface{})
	expected["1"] = "2"
	expected["3"] = "4"
	assert.Equal(t, ec2TagsToMap(tags), expected)
}
