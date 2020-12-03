package aws_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"go.mondoo.io/mondoo/motor/discovery/aws"
)

func TestParseParseEc2PlatformId(t *testing.T) {
	uri := "//platformid.api.mondoo.app/runtime/aws/ec2/v1/accounts/675173580680/regions/eu-west-1/instances/i-0e11b0762369fbefa"

	p := aws.ParseEc2PlatformID(uri)
	assert.NotNil(t, p)
	assert.Equal(t, "675173580680", p.Account)
	assert.Equal(t, "eu-west-1", p.Region)
	assert.Equal(t, "i-0e11b0762369fbefa", p.Instance)
}
