package services

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"
	"go.mondoo.com/cnquery/motor/providers/mock"
)

func TestParseOpenbsdServicesRunning(t *testing.T) {
	mock, err := mock.NewFromTomlFile("./testdata/openbsd6.toml")
	require.NoError(t, err)

	openbsd := OpenBsdRcctlServiceManager{provider: mock}
	// iterate over services and check if they are running
	services, err := openbsd.List()
	require.NoError(t, err)
	assert.Equal(t, 70, len(services), "detected the right amount of services")
}
