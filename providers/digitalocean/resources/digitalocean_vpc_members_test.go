// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
)

func setStr(s string) plugin.TValue[string] {
	return plugin.TValue[string]{Data: s, State: plugin.StateIsSet}
}

func setList(items ...any) plugin.TValue[[]any] {
	return plugin.TValue[[]any]{Data: items, State: plugin.StateIsSet}
}

func TestGroupByVpcID(t *testing.T) {
	a := &mqlDigitaloceanDroplet{Id: plugin.TValue[int64]{Data: 1, State: plugin.StateIsSet}, VpcUuid: setStr("vpc-a")}
	b := &mqlDigitaloceanDroplet{Id: plugin.TValue[int64]{Data: 2, State: plugin.StateIsSet}, VpcUuid: setStr("vpc-a")}
	unattached := &mqlDigitaloceanDroplet{Id: plugin.TValue[int64]{Data: 3, State: plugin.StateIsSet}, VpcUuid: setStr("")}
	whitespace := &mqlDigitaloceanDroplet{Id: plugin.TValue[int64]{Data: 4, State: plugin.StateIsSet}, VpcUuid: setStr("   ")}

	idx := groupByVpcID([]any{a, b, unattached, whitespace}, dropletVpcIDs)

	assert.Equal(t, []any{a, b}, idx["vpc-a"], "both droplets land under their VPC")
	assert.Empty(t, idx["vpc-b"], "a VPC with no members has no bucket")
	assert.NotContains(t, idx, "", "a droplet with no VPC is dropped, not filed under the empty id")
	assert.Len(t, idx, 1)
}

func TestGroupByVpcIDMultiAttach(t *testing.T) {
	// An NFS share is reachable from several VPCs at once, so it has to
	// appear under each of them rather than only the first.
	share := &mqlDigitaloceanNfs{
		Id:     setStr("share-1"),
		VpcIds: setList("vpc-a", "vpc-b"),
	}
	onlyB := &mqlDigitaloceanNfs{Id: setStr("share-2"), VpcIds: setList("vpc-b")}
	detached := &mqlDigitaloceanNfs{Id: setStr("share-3"), VpcIds: setList()}

	idx := groupByVpcID([]any{share, onlyB, detached}, nfsShareVpcIDs)

	assert.Equal(t, []any{share}, idx["vpc-a"])
	assert.Equal(t, []any{share, onlyB}, idx["vpc-b"])
	assert.Len(t, idx, 2)
}

func TestGroupByVpcIDSkipsForeignEntries(t *testing.T) {
	// A cached collection holding an unexpected type must be skipped rather
	// than panic the scan on a bare type assertion.
	idx := groupByVpcID([]any{"not a droplet", nil, &mqlDigitaloceanDroplet{VpcUuid: setStr("vpc-a")}}, dropletVpcIDs)
	assert.Len(t, idx["vpc-a"], 1)
	assert.Len(t, idx, 1)
}

func TestVpcIDExtractors(t *testing.T) {
	// Pins which field each collection is grouped by. Reading the wrong id
	// field would silently report an empty VPC instead of failing.
	tests := []struct {
		name    string
		item    any
		extract func(any) []string
		want    []string
	}{
		{
			name:    "droplet uses vpcUuid",
			item:    &mqlDigitaloceanDroplet{VpcUuid: setStr("vpc-a")},
			extract: dropletVpcIDs,
			want:    []string{"vpc-a"},
		},
		{
			name:    "database uses privateNetworkUuid",
			item:    &mqlDigitaloceanDatabase{PrivateNetworkUuid: setStr("vpc-b")},
			extract: databaseVpcIDs,
			want:    []string{"vpc-b"},
		},
		{
			name:    "load balancer uses vpcUuid",
			item:    &mqlDigitaloceanLoadBalancer{VpcUuid: setStr("vpc-c")},
			extract: loadBalancerVpcIDs,
			want:    []string{"vpc-c"},
		},
		{
			name:    "kubernetes cluster uses vpcUuid",
			item:    &mqlDigitaloceanKubernetesCluster{VpcUuid: setStr("vpc-d")},
			extract: kubernetesClusterVpcIDs,
			want:    []string{"vpc-d"},
		},
		{
			name:    "nfs share uses vpcIds",
			item:    &mqlDigitaloceanNfs{VpcIds: setList("vpc-e", "vpc-f")},
			extract: nfsShareVpcIDs,
			want:    []string{"vpc-e", "vpc-f"},
		},
		{
			name: "nat gateway uses its cached attachment uuids",
			item: &mqlDigitaloceanVpcNatGateway{
				mqlDigitaloceanVpcNatGatewayInternal: mqlDigitaloceanVpcNatGatewayInternal{
					vpcUUIDs: []string{"vpc-g", "vpc-h"},
				},
			},
			extract: natGatewayVpcIDs,
			want:    []string{"vpc-g", "vpc-h"},
		},
		{
			name:    "wrong resource type yields nothing",
			item:    &mqlDigitaloceanVpc{Id: setStr("vpc-a")},
			extract: dropletVpcIDs,
			want:    nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.extract(tt.item))
		})
	}
}

func TestNfsShareVpcIDsDropsNonStrings(t *testing.T) {
	share := &mqlDigitaloceanNfs{VpcIds: setList("vpc-a", 42, nil, "vpc-b")}
	assert.Equal(t, []string{"vpc-a", "vpc-b"}, nfsShareVpcIDs(share))
}

func TestVpcMemberList(t *testing.T) {
	droplet := &mqlDigitaloceanDroplet{VpcUuid: setStr("vpc-a")}
	idx := map[string][]any{"vpc-a": {droplet}}

	t.Run("vpc with matches", func(t *testing.T) {
		got, ok := vpcMemberList(idx, nil, "vpc-a")
		require.True(t, ok)
		assert.Equal(t, []any{droplet}, got)
	})

	t.Run("vpc with no matches is empty, not null", func(t *testing.T) {
		got, ok := vpcMemberList(idx, nil, "vpc-b")
		require.True(t, ok, "an empty result is a real answer")
		assert.Empty(t, got)
		assert.NotNil(t, got)
	})

	t.Run("unreadable collection is null, not empty", func(t *testing.T) {
		got, ok := vpcMemberList(nil, errors.New("403 forbidden"), "vpc-a")
		assert.False(t, ok, "an unreadable collection must not read as an empty VPC")
		assert.Nil(t, got)
	})

	t.Run("empty vpc id is null", func(t *testing.T) {
		got, ok := vpcMemberList(idx, nil, "")
		assert.False(t, ok)
		assert.Nil(t, got)

		got, ok = vpcMemberList(idx, nil, "   ")
		assert.False(t, ok)
		assert.Nil(t, got)
	})

	t.Run("result is a copy of the cached bucket", func(t *testing.T) {
		got, ok := vpcMemberList(idx, nil, "vpc-a")
		require.True(t, ok)
		got[0] = nil
		assert.Equal(t, []any{droplet}, idx["vpc-a"], "the memoized index must not be mutated by a caller")
	})
}

func TestStringsFromRawList(t *testing.T) {
	assert.Equal(t, []string{}, stringsFromRawList(nil))
	assert.Equal(t, []string{}, stringsFromRawList([]any{}))
	assert.Equal(t, []string{"a", "b"}, stringsFromRawList([]any{"a", 1, true, nil, "b"}))
}
