package resources_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResource_DNS(t *testing.T) {
	res := testQuery(t, "dns(\"mondoo.io\").mx")
	assert.NotEmpty(t, res)
}
