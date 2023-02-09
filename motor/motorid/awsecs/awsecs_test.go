package awsecsid

import (
	"testing"

	"gotest.tools/assert"
)

func TestParseECSContainerId(t *testing.T) {
	path := "//platformid.api.mondoo.app/runtime/aws/ecs/v1/accounts/185972265011/regions/us-east-1/container/vjtest/f088b38d61ac45d6a946b5aebbe7197a/314e35e0-2d0a-4408-b37e-16063461d73a"
	id, err := ParseMondooECSContainerId(path)
	assert.NilError(t, err)
	assert.Equal(t, id.Account, "185972265011")
	assert.Equal(t, id.Region, "us-east-1")
	assert.Equal(t, id.Id, "vjtest/f088b38d61ac45d6a946b5aebbe7197a/314e35e0-2d0a-4408-b37e-16063461d73a")
}
