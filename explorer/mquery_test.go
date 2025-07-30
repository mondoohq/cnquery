// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package explorer

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v11"
	"go.mondoo.com/cnquery/v11/mqlc"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/testutils"
)

func TestMquery_RefreshAsAssetFilterStableChecksum(t *testing.T) {
	m := &Mquery{
		Mql: "true",
		Uid: "my-id0",
	}

	x := testutils.LinuxMock()
	conf := mqlc.NewConfig(x.Schema(), cnquery.DefaultFeatures)

	_, err := m.RefreshAsFilter("//owner/me", conf)
	require.NoError(t, err)
	assert.Equal(t, "//owner/me/filter/"+m.CodeId, m.Mrn)

	cs := m.Checksum
	_, err = m.RefreshAsFilter("//owner/me", conf)
	require.NoError(t, err)
	assert.Equal(t, cs, m.Checksum)
}

func TestMqueryWithTags_RefreshChecksum(t *testing.T) {
	x := testutils.LinuxMock()

	conf := mqlc.NewConfig(x.Schema(), cnquery.DefaultFeatures)

	m := &Mquery{
		Mql: "true",
		Uid: "my-id0",
		Tags: map[string]string{
			"tag1": "value1",
		},
	}

	getQuery := func(ctx context.Context, mrn string) (*Mquery, error) {
		return &Mquery{}, nil

	}

	err := m.RefreshChecksum(context.Background(), conf, getQuery)
	require.NoError(t, err)
	initialChecksum := m.Checksum

	m.Tags["tag2"] = "value2"
	err = m.RefreshChecksum(context.Background(), conf, getQuery)
	require.NoError(t, err)
	assert.NotEqual(t, initialChecksum, m.Checksum, "Checksum should change when tags are modified")
}

func TestMquery_Refresh(t *testing.T) {
	a := &Mquery{
		Mql:   "mondoo.version != props.world",
		Uid:   "my-id0",
		Props: []*Property{{Mql: "'hi'", Uid: "world"}},
	}

	err := a.RefreshMRN("//owner")
	require.NoError(t, err)
	err = a.Props[0].RefreshMRN(a.Mrn)
	require.NoError(t, err)
	assert.Equal(t, "//owner/queries/my-id0", a.Mrn)
	assert.Empty(t, a.Uid)
	assert.Equal(t, "//owner/queries/my-id0/properties/world", a.Props[0].Mrn)
	assert.Empty(t, a.Props[0].Uid)

	x := testutils.LinuxMock()
	conf := mqlc.NewConfig(x.Schema(), cnquery.DefaultFeatures)
	err = a.RefreshChecksum(
		context.Background(),
		conf,
		func(ctx context.Context, mrn string) (*Mquery, error) {
			return nil, nil
		},
	)
	require.NoError(t, err)
	assert.Equal(t, "g4cj2tMzHs4=", a.Checksum)
	assert.Equal(t, "OSGvkbAIHFU=", a.Props[0].Checksum)
}

func TestMqueryMerge(t *testing.T) {
	a := &Mquery{
		Mql:   "base",
		Title: "base title",
		Docs: &MqueryDocs{
			Desc:  "base desc",
			Audit: "base audit",
			Remediation: &Remediation{
				Items: []*TypedDoc{{
					Id:   "default",
					Desc: "a description",
				}},
			},
		},
	}
	b := &Mquery{
		Mql: "override",
		Docs: &MqueryDocs{
			Desc: "override desc",
			Remediation: &Remediation{
				Items: []*TypedDoc{{
					Id:   "default",
					Desc: "b description",
				}},
			},
		},
	}

	c := b.Merge(a)

	assert.NotEqual(t, a.Mql, c.Mql)
	assert.Equal(t, b.Mql, c.Mql)

	assert.Equal(t, a.Title, c.Title)
	assert.NotEqual(t, b.Title, c.Title)

	assert.NotEqual(t, a.Docs.Desc, c.Docs.Desc)
	assert.Equal(t, b.Docs.Desc, c.Docs.Desc)

	assert.Equal(t, a.Docs.Audit, c.Docs.Audit)
	assert.NotEqual(t, b.Docs.Audit, c.Docs.Audit)

	a.CodeId = "not this"
	b.CodeId = "not this either"
	assert.Equal(t, "", c.CodeId)

	// we want to make sure there are no residual shallow-copies
	cD := c.Docs.Remediation.Items[0].Desc
	a.Docs.Remediation.Items[0].Desc = "not this"
	a.Docs.Remediation.Items[0].Desc = "not this either"
	assert.Equal(t, cD, c.Docs.Remediation.Items[0].Desc)
}

func TestMquery_Remediation(t *testing.T) {
	tests := []struct {
		title string
		data  string
		out   *Remediation
	}{
		{
			"parse default remediation, string-only",
			"\"string-only remediation\"",
			&Remediation{Items: []*TypedDoc{
				{Id: "default", Desc: "string-only remediation"},
			}},
		},
		{
			"parse multiple remediation via array",
			"[{\"id\": \"one\", \"desc\": \"two\"}, {\"id\": \"three\", \"desc\": \"four\"}]",
			&Remediation{Items: []*TypedDoc{
				{Id: "one", Desc: "two"},
				{Id: "three", Desc: "four"},
			}},
		},
		{
			"parse internal structure, which uses items",
			"{\"items\":[{\"id\": \"one\", \"desc\": \"two\"}, {\"id\": \"three\", \"desc\": \"four\"}]}",
			&Remediation{Items: []*TypedDoc{
				{Id: "one", Desc: "two"},
				{Id: "three", Desc: "four"},
			}},
		},
	}

	for _, cur := range tests {
		t.Run(cur.title, func(t *testing.T) {
			var res Remediation
			err := json.Unmarshal([]byte(cur.data), &res)
			require.NoError(t, err)
			assert.Equal(t, cur.out, &res)
		})
	}

	t.Run("marshal remediation to json", func(t *testing.T) {
		initial := &Remediation{
			Items: []*TypedDoc{
				{Id: "default", Desc: "one remediation"},
			},
		}

		out, err := json.Marshal(initial)
		require.NoError(t, err)
		assert.Equal(t, "[{\"id\":\"default\",\"desc\":\"one remediation\"}]", string(out))

		var back Remediation
		err = json.Unmarshal(out, &back)
		require.NoError(t, err)
		assert.Equal(t, initial, &back)
	})
}
