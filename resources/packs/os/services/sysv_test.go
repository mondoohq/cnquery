package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/motor/providers/mock"
)

func TestParseSysvServices(t *testing.T) {
	mock, err := mock.NewFromTomlFile("./testdata/amzn1.toml")
	require.NoError(t, err)

	sysv := SysVServiceManager{provider: mock}
	services, err := sysv.services()
	require.NoError(t, err)
	assert.Equal(t, 4, len(services), "detected the right amount of services")
}

func TestParseSysvServicesRunlevel(t *testing.T) {
	mock, err := mock.NewFromTomlFile("./testdata/amzn1.toml")
	require.NoError(t, err)

	sysv := SysVServiceManager{provider: mock}
	level, err := sysv.serviceRunLevel()
	require.NoError(t, err)
	assert.Equal(t, 3, len(level), "detected the right amount of services")
	assert.Equal(t, 4, len(level["sshd"]))
}

func TestParseSysvServicesRunning(t *testing.T) {
	mock, err := mock.NewFromTomlFile("./testdata/amzn1.toml")
	require.NoError(t, err)

	sysv := SysVServiceManager{provider: mock}
	// iterate over services and check if they are running
	running, err := sysv.running([]string{"sshd", "ntpd", "acpid"})
	require.NoError(t, err)
	assert.Equal(t, 3, len(running), "detected the right amount of services")
	assert.Equal(t, false, running["acpid"])
	assert.Equal(t, true, running["sshd"])
}
