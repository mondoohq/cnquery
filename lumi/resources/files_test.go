package resources_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResource_FilesFind(t *testing.T) {
	res := testQuery(t, "files.find(from: '/etc').list")
	assert.NotEmpty(t, res)
	testResultsErrors(t, res)
	assert.Equal(t, 2, len(res[0].Data.Value.([]interface{})))
}
