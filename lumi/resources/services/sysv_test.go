package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.io/mondoo/motor"
	"go.mondoo.io/mondoo/motor/transports/mock"
)

func TestParseSysvServices(t *testing.T) {
	mock, err := mock.NewFromTomlFile("./testdata/amzn1.toml")
	if err != nil {
		t.Fatal(err)
	}

	m, err := motor.New(mock)
	require.NoError(t, err)
	sysv := SysVServiceManager{motor: m}

	services, err := sysv.services()
	require.NoError(t, err)
	assert.Equal(t, 4, len(services), "detected the right amount of services")
}

func TestParseSysvServicesRunlevel(t *testing.T) {
	mock, err := mock.NewFromTomlFile("./testdata/amzn1.toml")
	if err != nil {
		t.Fatal(err)
	}

	m, err := motor.New(mock)
	require.NoError(t, err)
	sysv := SysVServiceManager{motor: m}

	level, err := sysv.serviceRunLevel()

	assert.Nil(t, err)
	assert.Equal(t, 3, len(level), "detected the right amount of services")

	assert.Equal(t, 4, len(level["sshd"]))
}

func TestParseSysvServicesRunning(t *testing.T) {
	mock, err := mock.NewFromTomlFile("./testdata/amzn1.toml")
	if err != nil {
		t.Fatal(err)
	}

	m, err := motor.New(mock)
	require.NoError(t, err)
	sysv := SysVServiceManager{motor: m}

	// iterate over services and check if they are running
	running, err := sysv.running([]string{"sshd", "ntpd", "acpid"})

	assert.Nil(t, err)
	assert.Equal(t, 3, len(running), "detected the right amount of services")
	assert.Equal(t, false, running["acpid"])
	assert.Equal(t, true, running["sshd"])
}
