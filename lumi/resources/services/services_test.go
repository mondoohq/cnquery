package services_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.io/mondoo/lumi/resources/services"
	mock "go.mondoo.io/mondoo/motor/mock/toml"
	"go.mondoo.io/mondoo/motor/types"
)

func TestParseServiceSystemDUnitFilesx(t *testing.T) {
	mock, err := mock.New(&types.Endpoint{Backend: "mock", Path: "services_systemd.toml"})
	if err != nil {
		t.Fatal(err)
	}
	c, err := mock.RunCommand("systemctl --all list-units")
	if err != nil {
		t.Fatal(err)
	}
	assert.Nil(t, err)

	m, err := services.ParseServiceSystemDUnitFiles(c.Stdout)
	assert.Nil(t, err)
	assert.Equal(t, 186, len(m), "detected the right amount of services")

	// check first element
	assert.Equal(t, "proc-sys-fs-binfmt_misc.automount", m[0].Name, "service name detected")
	assert.Equal(t, true, m[0].Running, "service is running")
	assert.Equal(t, true, m[0].Installed, "service is installed")
	assert.Equal(t, "systemd", m[0].Type, "service type is added")

	// check last element
	assert.Equal(t, "systemd-tmpfiles-clean.timer", m[185].Name, "service name detected")
	assert.Equal(t, true, m[185].Running, "service is running")
	assert.Equal(t, true, m[185].Installed, "service is installed")
	assert.Equal(t, "systemd", m[185].Type, "service type is added")
}

func TestParseServiceLaunchD(t *testing.T) {
	mock, err := mock.New(&types.Endpoint{Backend: "mock", Path: "services_launchd.toml"})
	if err != nil {
		t.Fatal(err)
	}
	c, err := mock.RunCommand("launchctl list")
	if err != nil {
		t.Fatal(err)
	}
	assert.Nil(t, err)
	m, err := services.ParseServiceLaunchD(c.Stdout)
	assert.Nil(t, err)
	assert.Equal(t, 369, len(m), "detected the right amount of services")

	assert.Equal(t, "com.apple.SafariHistoryServiceAgent", m[0].Name, "service name detected")
	assert.Equal(t, false, m[0].Running, "service is running")
	assert.Equal(t, true, m[0].Installed, "service is installed")
	assert.Equal(t, "launchd", m[0].Type, "service type is added")
}
