// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package services_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v11/providers/os/connection/mock"
	"go.mondoo.com/cnquery/v11/providers/os/resources/services"
)

func TestParseServiceLaunchD(t *testing.T) {
	mock, err := mock.New(0, "./testdata/osx.toml", &inventory.Asset{
		Platform: &inventory.Platform{
			Name:   "macos",
			Family: []string{"unix", "darwin"},
		},
	})
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
