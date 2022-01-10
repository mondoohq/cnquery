package resources

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/acm/types"
	"github.com/stretchr/testify/assert"
)

func TestCertTagsToMapTags(t *testing.T) {
	certTags := []types.Tag{
		{Key: aws.String("k"), Value: aws.String("v")},
		{Key: aws.String("key"), Value: aws.String("val")},
	}
	assert.Equal(t, certTagsToMapTags(certTags), map[string]interface{}{"k": "v", "key": "val"})
	certTags = []types.Tag{
		{Key: aws.String("key"), Value: nil},
	}
	m := make(map[string]interface{})
	assert.Equal(t, certTagsToMapTags(certTags), m)
	certTags = []types.Tag{}
	assert.Equal(t, certTagsToMapTags(certTags), m)
}
