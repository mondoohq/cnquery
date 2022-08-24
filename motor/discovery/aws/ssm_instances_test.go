package aws

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.com/cnquery/motor/asset"
)

func TestAssetHasLabels(t *testing.T) {
	a := &asset.Asset{Name: "1", Labels: map[string]string{"test": "1", "othertag": "2"}}
	assert.Equal(t, true, assetHasLabels(a, map[string]string{}))
	assert.Equal(t, true, assetHasLabels(a, map[string]string{"othertag": "2"}))
	assert.Equal(t, false, assetHasLabels(a, map[string]string{"sometag": "2"}))
	assert.Equal(t, false, assetHasLabels(a, map[string]string{"othertag": "1"}))
}
