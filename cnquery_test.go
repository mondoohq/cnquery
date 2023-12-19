// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package cnquery

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
	assert.Equal(t, cnqueryLatestReleaseUrl, "https://releases.mondoo.com/cnquery/latest.json?ignoreCache=1")
}
