package awsec2ebs

import (
	"testing"

	"gotest.tools/assert"
)

func TestParseInstanceId(t *testing.T) {
	path := "account/185972265011/region/us-east-1/instance/i-07f67838ada5879af"
	id, err := ParseInstanceId(path)
	assert.NilError(t, err)
	assert.Equal(t, id.Account, "185972265011")
	assert.Equal(t, id.Region, "us-east-1")
	assert.Equal(t, id.Id, "i-07f67838ada5879af")
}
