// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package mql

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetLatestVersion(t *testing.T) {
	client := &http.Client{}
	version, err := GetLatestVersion(client)

	assert.NoError(t, err)
	assert.NotNil(t, version)
	assert.Equal(t, mqlLatestReleaseUrl, "https://releases.mondoo.com/mql/latest.json?ignoreCache=1")
}
