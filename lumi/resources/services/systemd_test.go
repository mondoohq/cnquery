package services_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.io/mondoo/lumi/resources/services"
	"go.mondoo.io/mondoo/motor/transports"
	"go.mondoo.io/mondoo/motor/transports/mock"
)

func TestParseServiceSystemDUnitFiles(t *testing.T) {
	mock, err := mock.NewFromToml(&transports.TransportConfig{Backend: transports.TransportBackend_CONNECTION_MOCK, Path: "./testdata/linux_systemd.toml"})
	if err != nil {
		t.Fatal(err)
	}
	c, err := mock.RunCommand("systemctl --all list-units --type service")
	if err != nil {
		t.Fatal(err)
	}
	assert.Nil(t, err)

	m, err := services.ParseServiceSystemDUnitFiles(c.Stdout)
	assert.Nil(t, err)
	assert.Equal(t, 102, len(m), "detected the right amount of services")

	// check first element
	assert.Equal(t, "auditd", m[0].Name, "service name detected")
	assert.Equal(t, true, m[0].Running, "service is running")
	assert.Equal(t, true, m[0].Installed, "service is installed")
	assert.Equal(t, "systemd", m[0].Type, "service type is added")

	// check last element
	assert.Equal(t, "ypxfrd", m[101].Name, "service name detected")
	assert.Equal(t, false, m[101].Running, "service is running")
	assert.Equal(t, false, m[101].Installed, "service is installed")
	assert.Equal(t, "systemd", m[101].Type, "service type is added")

	// check for masked element
	assert.Equal(t, "nfs-server", m[30].Name, "service name detected")
	assert.Equal(t, true, m[30].Masked, "service is masked")
}
