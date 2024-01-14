// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v10/providers/os/connection/mock"
)

func TestParseOpenbsdServicesRunning(t *testing.T) {
	mock, err := mock.New("./testdata/openbsd6.toml", &inventory.Asset{
		Platform: &inventory.Platform{
			Name: "openbsd",
		},
	})
	require.NoError(t, err)

	openbsd := OpenBsdRcctlServiceManager{conn: mock}
	// iterate over services and check if they are running
	services, err := openbsd.List()
	require.NoError(t, err)
	assert.Equal(t, 70, len(services), "detected the right amount of services")
}
