package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	motor "go.mondoo.io/mondoo/motor/motoros"
	"go.mondoo.io/mondoo/motor/motoros/mock"
	"go.mondoo.io/mondoo/motor/motoros/types"
)

func TestParseOpenbsdServicesRunning(t *testing.T) {
	mock, err := mock.NewFromToml(&types.Endpoint{Backend: "mock", Path: "./testdata/openbsd6.toml"})
	if err != nil {
		t.Fatal(err)
	}

	m, err := motor.New(mock)
	require.NoError(t, err)
	openbsd := OpenBsdRcctlServiceManager{motor: m}

	// iterate over services and check if they are running
	services, err := openbsd.List()

	assert.Nil(t, err)
	assert.Equal(t, 70, len(services), "detected the right amount of services")
}
