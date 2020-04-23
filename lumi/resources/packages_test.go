package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResource_Packages(t *testing.T) {
	res := testQuery(t, "packages")
	assert.NotEmpty(t, res)
}
