package aws

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAssembleEc2InstancesFilters(t *testing.T) {
	opts := make(map[string]string)
	f := AssembleEc2InstancesFilters(opts)
	assert.NotNil(t, f)
	assert.Equal(t, ec2InstancesFilters{}, f)

	opts["instance-ids"] = "1-034345,i-53253"
	f = AssembleEc2InstancesFilters(opts)
	assert.NotNil(t, f)
	assert.Equal(t, ec2InstancesFilters{instanceIds: []string{"1-034345", "i-53253"}}, f)

	opts["regions"] = "eu-west-1"
	f = AssembleEc2InstancesFilters(opts)
	assert.NotNil(t, f)
	assert.Equal(t, []string{"eu-west-1"}, f.regions)

	opts["tags"] = "Name"
	f = AssembleEc2InstancesFilters(opts)
	assert.NotNil(t, f)
	assert.Equal(t, map[string]string{"tag-key": "Name"}, f.tags)

	opts["tags"] = "env=test"
	f = AssembleEc2InstancesFilters(opts)
	assert.NotNil(t, f)
	assert.Equal(t, map[string]string{"tag:env": "test"}, f.tags)
}
