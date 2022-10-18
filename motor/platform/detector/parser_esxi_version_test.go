package detector_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/motor/platform/detector"
)

func TestEsxiVersionParser(t *testing.T) {
	m, err := detector.ParseEsxiRelease("VMware ESXi 6.7.0 build-13006603")
	require.NoError(t, err)
	assert.Equal(t, "6.7.0 build-13006603", m)

	m, err = detector.ParseEsxiRelease("VMware ESXi 6.7.0 build-13006603\n")
	require.NoError(t, err)
	assert.Equal(t, "6.7.0 build-13006603", m)
}
