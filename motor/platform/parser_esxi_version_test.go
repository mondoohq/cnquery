package platform_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.io/mondoo/motor/platform"
)

func TestEsxiVersionParser(t *testing.T) {

	m, err := platform.ParseEsxiRelease("VMware ESXi 6.7.0 build-13006603")
	assert.Nil(t, err)

	assert.Equal(t, "6.7.0 build-13006603", m)

	m, err = platform.ParseEsxiRelease("VMware ESXi 6.7.0 build-13006603\n")
	assert.Nil(t, err)

	assert.Equal(t, "6.7.0 build-13006603", m)
}
