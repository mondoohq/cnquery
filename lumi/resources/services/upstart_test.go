package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	motor "go.mondoo.io/mondoo/motor/motoros"
	"go.mondoo.io/mondoo/motor/motoros/mock"
	"go.mondoo.io/mondoo/motor/motoros/types"
)

func TestParseUpstartServicesRunning(t *testing.T) {
	mock, err := mock.NewFromToml(&types.Endpoint{Backend: "mock", Path: "./testdata/ubuntu1404.toml"})
	if err != nil {
		t.Fatal(err)
	}

	m, err := motor.New(mock)
	require.NoError(t, err)
	upstart := UpstartServiceManager{SysVServiceManager{motor: m}}

	// iterate over services and check if they are running
	services, err := upstart.List()

	assert.Nil(t, err)
	assert.Equal(t, 9, len(services), "detected the right amount of services")
}
