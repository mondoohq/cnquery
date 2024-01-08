// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package explorer

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProperty_RefreshMrn(t *testing.T) {
	in := &Property{
		Uid: "uid1",
		For: []*ObjectRef{{
			Uid: "uid2",
		}},
	}

	err := in.RefreshMRN("//my.owner")
	require.NoError(t, err)

	assert.Equal(t, "", in.Uid)
	assert.Equal(t, "//my.owner/queries/uid1", in.Mrn)
	assert.Equal(t, "", in.For[0].Uid)
	assert.Equal(t, "//my.owner/queries/uid2", in.For[0].Mrn)
}
