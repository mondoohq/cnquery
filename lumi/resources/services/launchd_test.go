package services_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.io/mondoo/lumi/resources/services"
	"go.mondoo.io/mondoo/motor/motoros/mock"
	"go.mondoo.io/mondoo/motor/motoros/types"
)

func TestParseServiceLaunchD(t *testing.T) {
	mock, err := mock.NewFromToml(&types.Endpoint{Backend: "mock", Path: "./testdata/osx.toml"})
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
	assert.Equal(t, 15, len(m), "detected the right amount of services")

	assert.Equal(t, "com.apple.SafariHistoryServiceAgent", m[0].Name, "service name detected")
	assert.Equal(t, false, m[0].Running, "service is running")
	assert.Equal(t, true, m[0].Installed, "service is installed")
	assert.Equal(t, "launchd", m[0].Type, "service type is added")
}
