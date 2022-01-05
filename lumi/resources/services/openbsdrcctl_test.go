package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.io/mondoo/motor"
	"go.mondoo.io/mondoo/motor/transports/mock"
)

func TestParseOpenbsdServicesRunning(t *testing.T) {
	mock, err := mock.NewFromTomlFile("./testdata/openbsd6.toml")
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
