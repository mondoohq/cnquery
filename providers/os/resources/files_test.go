// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/testutils"
)

func TestResource_FilesFind(t *testing.T) {
	res := x.TestQuery(t, "files.find(from: '/etc').list")
	assert.NotEmpty(t, res)
	testutils.TestNoResultErrors(t, res)
	assert.Equal(t, 5, len(res[0].Data.Value.([]interface{})))
}
