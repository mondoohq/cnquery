package gcp_test

import (
	"bytes"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.io/mondoo/motor/transports/gcp"
)

func TestParseGcloudConfig(t *testing.T) {

	data, err := ioutil.ReadFile("./testdata/gcloud_config.json")
	require.NoError(t, err)

	config, err := gcp.ParseGcloudConfig(bytes.NewReader(data))
	require.NoError(t, err)

	assert.Equal(t, "mondoo-abc-12345", config.Configuration.Properties.Core.Project)
}
