// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package llx_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/types"
)

func TestVersion_Conversions(t *testing.T) {
	sv := llx.StringPrimitive("1.2.3")
	sv.Type = string(types.Version)
	rd := sv.RawData()
	require.NoError(t, rd.Error, "no error converting version to raw data")
	require.Equal(t, "1.2.3", rd.Value, "version to raw data is the same")
}
