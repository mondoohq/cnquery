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
