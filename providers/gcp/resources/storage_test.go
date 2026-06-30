// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/api/storage/v1"
)

func TestEvaluateBucketPublic(t *testing.T) {
	tests := []struct {
		name             string
		pap              string
		iamMembers       []string
		acl              []*storage.BucketAccessControl
		defaultObjectAcl []*storage.ObjectAccessControl
		want             bool
	}{
		{
			name: "PAP enforced wins over public IAM member",
			pap:  "enforced",
			iamMembers: []string{
				"user:alice@example.com",
				"allUsers",
			},
			want: false,
		},
		{
			name: "PAP enforced wins over public bucket ACL",
			pap:  "enforced",
			acl: []*storage.BucketAccessControl{
				{Entity: "allAuthenticatedUsers", Role: "READER"},
			},
			want: false,
		},
		{
			name: "PAP enforced wins over public default object ACL",
			pap:  "enforced",
			defaultObjectAcl: []*storage.ObjectAccessControl{
				{Entity: "allUsers", Role: "READER"},
			},
			want: false,
		},
		{
			name:       "PAP inherited + IAM allUsers is public",
			pap:        "inherited",
			iamMembers: []string{"user:alice@example.com", "allUsers"},
			want:       true,
		},
		{
			name:       "PAP unspecified + IAM allAuthenticatedUsers is public",
			pap:        "unspecified",
			iamMembers: []string{"allAuthenticatedUsers"},
			want:       true,
		},
		{
			name: "PAP inherited + bucket ACL allUsers is public",
			pap:  "inherited",
			acl: []*storage.BucketAccessControl{
				{Entity: "user-alice@example.com", Role: "OWNER"},
				{Entity: "allUsers", Role: "READER"},
			},
			want: true,
		},
		{
			name: "PAP inherited + default object ACL allAuthenticatedUsers is public",
			pap:  "inherited",
			defaultObjectAcl: []*storage.ObjectAccessControl{
				{Entity: "allAuthenticatedUsers", Role: "READER"},
			},
			want: true,
		},
		{
			name:       "PAP inherited + only private grants is not public",
			pap:        "inherited",
			iamMembers: []string{"user:alice@example.com", "serviceAccount:svc@p.iam.gserviceaccount.com"},
			acl: []*storage.BucketAccessControl{
				{Entity: "user-alice@example.com", Role: "OWNER"},
				{Entity: "project-owners-123", Role: "OWNER"},
			},
			defaultObjectAcl: []*storage.ObjectAccessControl{
				{Entity: "project-viewers-123", Role: "READER"},
			},
			want: false,
		},
		{
			name: "all empty is not public",
			pap:  "",
			want: false,
		},
		{
			name: "nil ACL entries are skipped",
			pap:  "inherited",
			acl: []*storage.BucketAccessControl{
				nil,
				{Entity: "allUsers", Role: "READER"},
			},
			defaultObjectAcl: []*storage.ObjectAccessControl{nil},
			want:             true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := evaluateBucketPublic(tc.pap, tc.iamMembers, tc.acl, tc.defaultObjectAcl)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestBucketRetentionHelpers(t *testing.T) {
	t.Run("nil policies return 0", func(t *testing.T) {
		assert.Equal(t, int64(0), bucketRetentionPeriodSeconds(nil))
		assert.Equal(t, int64(0), bucketSoftDeleteRetentionSeconds(nil))
	})
	t.Run("values are returned", func(t *testing.T) {
		assert.Equal(t, int64(86400), bucketRetentionPeriodSeconds(&storage.BucketRetentionPolicy{RetentionPeriod: 86400}))
		assert.Equal(t, int64(604800), bucketSoftDeleteRetentionSeconds(&storage.BucketSoftDeletePolicy{RetentionDurationSeconds: 604800}))
	})
}

func TestEffectiveTagsToInterface(t *testing.T) {
	tests := []struct {
		name string
		in   map[string]string
		want map[string]interface{}
	}{
		{
			name: "nil input returns empty map",
			in:   nil,
			want: map[string]interface{}{},
		},
		{
			name: "empty input returns empty map",
			in:   map[string]string{},
			want: map[string]interface{}{},
		},
		{
			name: "project-scoped tag key",
			in:   map[string]string{"123456789012/env": "production"},
			want: map[string]interface{}{"123456789012/env": "production"},
		},
		{
			name: "org-scoped tag key is preserved as-is",
			in:   map[string]string{"815471563813/glz-resource-optout": "true"},
			want: map[string]interface{}{"815471563813/glz-resource-optout": "true"},
		},
		{
			name: "multiple tags are all converted",
			in: map[string]string{
				"815471563813/glz-resource-optout":   "true",
				"815471563813/dtit:sec:infosecclass": "internal",
				"123456789012/env":                   "production",
			},
			want: map[string]interface{}{
				"815471563813/glz-resource-optout":   "true",
				"815471563813/dtit:sec:infosecclass": "internal",
				"123456789012/env":                   "production",
			},
		},
		{
			name: "tag with empty value is preserved",
			in:   map[string]string{"815471563813/cost-center": ""},
			want: map[string]interface{}{"815471563813/cost-center": ""},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := effectiveTagsToInterface(tc.in)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestBucketTagKeyMatching(t *testing.T) {
	// Demonstrates the correct MQL pattern for checking tag existence:
	// since org-managed keys are namespaced as "orgNumber/keyName",
	// a suffix match (contains) is needed rather than exact equality.
	tests := []struct {
		name       string
		tags       map[string]interface{}
		searchKey  string
		wantExists bool
	}{
		{
			name:       "exact match on namespaced key",
			tags:       map[string]interface{}{"815471563813/glz-resource-optout": "true"},
			searchKey:  "815471563813/glz-resource-optout",
			wantExists: true,
		},
		{
			name:       "suffix match (MQL regex pattern) finds namespaced key",
			tags:       map[string]interface{}{"815471563813/glz-resource-optout": "true"},
			searchKey:  "glz-resource-optout",
			wantExists: true,
		},
		{
			name:       "key not present",
			tags:       map[string]interface{}{"815471563813/env": "production"},
			searchKey:  "glz-resource-optout",
			wantExists: false,
		},
		{
			name:       "empty tags map",
			tags:       map[string]interface{}{},
			searchKey:  "glz-resource-optout",
			wantExists: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Simulate the MQL `tags.keys.any(_ == /suffix/)` pattern.
			found := false
			for k := range tc.tags {
				if k == tc.searchKey || (len(k) > len(tc.searchKey) && k[len(k)-len(tc.searchKey)-1] == '/' && k[len(k)-len(tc.searchKey):] == tc.searchKey) {
					found = true
					break
				}
			}
			assert.Equal(t, tc.wantExists, found)
		})
	}
}
