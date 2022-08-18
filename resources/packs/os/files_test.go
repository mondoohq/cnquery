package os_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.io/mondoo/resources/packs/testutils"
)

func TestResource_FilesFind(t *testing.T) {
	res := x.TestQuery(t, "files.find(from: '/etc').list")
	assert.NotEmpty(t, res)
	testutils.TestNoResultErrors(t, res)
	assert.Equal(t, 2, len(res[0].Data.Value.([]interface{})))
}
