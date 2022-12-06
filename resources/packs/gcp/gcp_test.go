package gcp

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestZoneParser(t *testing.T) {
	zone := "https://www.googleapis.com/compute/v1/projects/example-123456/zones/us-central1-a"

	res, err := parseZone(zone)
	require.NoError(t, err)
	assert.Equal(t, "example-123456", res.ProjectID)
	assert.Equal(t, "us-central1-a", res.Name)
}
