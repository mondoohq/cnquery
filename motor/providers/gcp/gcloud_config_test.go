package gcp_test

import (
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/motor/providers/gcp"
)

func TestParseGcloudConfig(t *testing.T) {
	data, err := os.ReadFile("./testdata/gcloud_config.json")
	require.NoError(t, err)

	config, err := gcp.ParseGcloudConfig(bytes.NewReader(data))
	require.NoError(t, err)

	assert.Equal(t, "mondoo-abc-12345", config.Configuration.Properties.Core.Project)
}
