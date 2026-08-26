// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func awsLikeSchema() *Schema {
	return &Schema{
		Resources: map[string]*ResourceInfo{
			"aws.vpc": {
				Id:                 "aws.vpc",
				Name:               "aws.vpc",
				Provider:           "go.mondoo.com/mql/providers/aws",
				MinProviderVersion: "11.15.2",
				Fields: map[string]*Field{
					"arn": {
						Name:     "arn",
						Type:     "\a",
						Provider: "go.mondoo.com/mql/providers/aws",
					},
					"blockPublicAccessOptions": {
						Name:               "blockPublicAccessOptions",
						Type:               "\n",
						Provider:           "go.mondoo.com/mql/providers/aws",
						MinProviderVersion: "13.52.0",
					},
				},
			},
		},
		Dependencies:     map[string]*ProviderInfo{},
		ProviderVersions: map[string]string{"go.mondoo.com/mql/providers/aws": "13.53.3"},
	}
}

// The aggregate schema is what the compiler resolves against, so version
// metadata that does not survive aggregation may as well not exist.
func TestAddPreservesVersionMetadata(t *testing.T) {
	agg := &Schema{
		Resources:    map[string]*ResourceInfo{},
		Dependencies: map[string]*ProviderInfo{},
	}
	agg.Add(awsLikeSchema())

	vpc := agg.Lookup("aws.vpc")
	require.NotNil(t, vpc)
	assert.Equal(t, "11.15.2", vpc.MinProviderVersion, "resource version must survive aggregation")
	assert.Equal(t, "13.52.0", vpc.Fields["blockPublicAccessOptions"].MinProviderVersion, "field version must survive aggregation")
	assert.Equal(t, "", vpc.Fields["arn"].MinProviderVersion, "a field at the resource version stays empty")

	v, ok := agg.ProviderVersion("go.mondoo.com/mql/providers/aws")
	assert.True(t, ok)
	assert.Equal(t, "13.53.3", v)

	_, ok = agg.ProviderVersion("go.mondoo.com/mql/providers/gcp")
	assert.False(t, ok, "an unknown provider is a miss, not an empty string")
}

// Add() appends to Others when two providers define the same field. It used to
// store the source pointer in the aggregate, so that append mutated the
// per-provider schema the coordinator caches, and every rebuild appended again.
func TestAddDoesNotMutateSourceSchema(t *testing.T) {
	awsSchema := awsLikeSchema()
	otherSchema := &Schema{
		Resources: map[string]*ResourceInfo{
			"aws.vpc": {
				Id:   "aws.vpc",
				Name: "aws.vpc",
				// An extension is what reaches the field-merge loop; two
				// competing bases short-circuit at the resource level.
				IsExtension: true,
				Provider:    "go.mondoo.com/mql/providers/other",
				Fields: map[string]*Field{
					"arn": {
						Name:     "arn",
						Type:     "\a",
						Provider: "go.mondoo.com/mql/providers/other",
					},
				},
			},
		},
		Dependencies:     map[string]*ProviderInfo{},
		ProviderVersions: map[string]string{"go.mondoo.com/mql/providers/other": "1.0.0"},
	}

	// Rebuild the aggregate repeatedly, the way unsafeRefresh does.
	for i := 0; i < 3; i++ {
		agg := &Schema{
			Resources:    map[string]*ResourceInfo{},
			Dependencies: map[string]*ProviderInfo{},
		}
		agg.Add(awsSchema)
		agg.Add(otherSchema)
	}

	assert.Empty(t, awsSchema.Resources["aws.vpc"].Fields["arn"].Others,
		"aggregating must not write back into the per-provider schema")

	// And the aggregate still has to record the cross-provider duplicate once.
	agg := &Schema{
		Resources:    map[string]*ResourceInfo{},
		Dependencies: map[string]*ProviderInfo{},
	}
	agg.Add(awsSchema)
	agg.Add(otherSchema)
	assert.Len(t, agg.Lookup("aws.vpc").Fields["arn"].Others, 1)
}

func TestAddMergesProviderVersions(t *testing.T) {
	agg := &Schema{
		Resources:    map[string]*ResourceInfo{},
		Dependencies: map[string]*ProviderInfo{},
	}
	agg.Add(awsLikeSchema())
	agg.Add(&Schema{
		Resources:        map[string]*ResourceInfo{},
		Dependencies:     map[string]*ProviderInfo{},
		ProviderVersions: map[string]string{"go.mondoo.com/mql/providers/gcp": "12.1.0"},
	})
	// A schema with no provenance at all must not blank out what we have.
	agg.Add(&Schema{
		Resources:    map[string]*ResourceInfo{},
		Dependencies: map[string]*ProviderInfo{},
	})

	assert.Equal(t, map[string]string{
		"go.mondoo.com/mql/providers/aws": "13.53.3",
		"go.mondoo.com/mql/providers/gcp": "12.1.0",
	}, agg.AllProviderVersions())
}
