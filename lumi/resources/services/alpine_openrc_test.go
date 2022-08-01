package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.io/mondoo/motor"
	"go.mondoo.io/mondoo/motor/providers/mock"
)

func TestManagerAlpineImage(t *testing.T) {
	mock, err := mock.NewFromTomlFile("./testdata/alpine-image.toml")
	require.NoError(t, err)
	m, err := motor.New(mock)
	require.NoError(t, err)

	mm, err := ResolveManager(m)
	require.NoError(t, err)
	serviceList, err := mm.List()
	require.NoError(t, err)

	assert.Equal(t, 2, len(serviceList))

	assert.Contains(t, serviceList, &Service{
		Name:      "agetty",
		Running:   false, // service will not run, since its a container image
		Enabled:   true,
		Installed: true,
		Type:      "openrc",
	})

	assert.Contains(t, serviceList, &Service{
		Name:      "urandom",
		Running:   false,
		Enabled:   false,
		Installed: true,
		Type:      "openrc",
	})
}

func TestManagerAlpineContainer(t *testing.T) {
	mock, err := mock.NewFromTomlFile("./testdata/alpine-container.toml")
	require.NoError(t, err)
	m, err := motor.New(mock)
	require.NoError(t, err)

	mm, err := ResolveManager(m)
	require.NoError(t, err)
	serviceList, err := mm.List()
	require.NoError(t, err)

	assert.Equal(t, 2, len(serviceList))

	assert.Contains(t, serviceList, &Service{
		Name:      "agetty",
		Running:   true, // here this service is acutally running
		Enabled:   true,
		Installed: true,
		Type:      "openrc",
	})

	assert.Contains(t, serviceList, &Service{
		Name:      "urandom",
		Running:   false,
		Enabled:   false,
		Installed: true,
		Type:      "openrc",
	})
}
