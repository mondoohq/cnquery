package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/motor/providers/mock"
)

func TestParseUpstartServicesRunning(t *testing.T) {
	mock, err := mock.NewFromTomlFile("./testdata/ubuntu1404.toml")
	require.NoError(t, err)

	upstart := UpstartServiceManager{SysVServiceManager{provider: mock}}

	// iterate over services and check if they are running
	services, err := upstart.List()
	require.NoError(t, err)
	assert.Equal(t, 9, len(services), "detected the right amount of services")
}
