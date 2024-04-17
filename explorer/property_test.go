// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package explorer

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v11"
	"go.mondoo.com/cnquery/v11/mqlc"
	"go.mondoo.com/cnquery/v11/providers"
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

func TestProperty_MustNotBeEmpty(t *testing.T) {
	in := &Property{
		Uid: "uid1",
		Mql: "",
	}

	schema := providers.DefaultRuntime().Schema()
	conf := mqlc.NewConfig(schema, cnquery.DefaultFeatures)
	_, err := in.RefreshChecksumAndType(conf)

	assert.Error(t, err)
}

func TestProperty_MustCompile(t *testing.T) {
	schema := providers.DefaultRuntime().Schema()
	conf := mqlc.NewConfig(schema, cnquery.DefaultFeatures)

	t.Run("fails on invalid MQL", func(t *testing.T) {
		in := &Property{
			Uid: "uid1",
			Mql: "ruri.ryu",
		}

		_, err := in.RefreshChecksumAndType(conf)
		assert.Error(t, err)
	})

	t.Run("works with a valid query", func(t *testing.T) {
		in := &Property{
			Uid: "uid1",
			Mql: "mondoo.version",
		}

		_, err := in.RefreshChecksumAndType(conf)
		assert.NoError(t, err)
	})
}
