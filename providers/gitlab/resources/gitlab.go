// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"strconv"
	"strings"
	"time"

	gitlab "gitlab.com/gitlab-org/api/client-go"
)

// isoTimePtr converts the SDK's date-only ISOTime into a *time.Time, keeping
// nil as nil. Callers must always set the resulting field (via
// llx.TimeDataPtr, which maps nil to a proper MQL null) rather than skipping
// the map key: an absent key leaves the field *unset* rather than null, and
// unset fields cross the plugin boundary as an empty DataRes that surfaces as
// "llx: encountered a primitive with no type information, coercing to null".
func isoTimePtr(t *gitlab.ISOTime) *time.Time {
	if t == nil {
		return nil
	}
	converted := time.Time(*t)
	return &converted
}

func mapAccessLevelToRole(accessLevel int) string {
	switch accessLevel {
	case 0:
		return "No access"
	case 5:
		return "Minimal Access"
	case 10:
		return "Guest"
	case 15:
		return "Planner"
	case 20:
		return "Reporter"
	case 30:
		return "Developer"
	case 40:
		return "Maintainer"
	case 50:
		return "Owner"
	case 60:
		return "Admin"
	default:
		return "Unknown"
	}
}

// projectScopedID builds a cache key for a resource that is only unique within
// a single project. GitLab identifies many project children by values that
// repeat across projects (branch names, file paths, release tags, variable
// keys), so the owning project id has to be part of the key. Without it, a
// query that walks several projects at once (`gitlab.group.projects { ... }`)
// resolves every project after the first to the first one's resource, because
// the runtime caches on resourceName + "\x00" + __id.
func projectScopedID(resource string, projectID int64, parts ...string) string {
	return scopedID(resource, projectID, parts...)
}

// groupScopedID is the group-level counterpart of projectScopedID, for children
// that repeat across groups (SAML links, CI/CD variables).
func groupScopedID(resource string, groupID int64, parts ...string) string {
	return scopedID(resource, groupID, parts...)
}

// userScopedID is the user-level counterpart, for children that repeat across
// users (external identities).
func userScopedID(resource string, userID int64, parts ...string) string {
	return scopedID(resource, userID, parts...)
}

func scopedID(resource string, ownerID int64, parts ...string) string {
	segments := make([]string, 0, len(parts)+2)
	segments = append(segments, resource, strconv.FormatInt(ownerID, 10))
	segments = append(segments, parts...)
	return strings.Join(segments, "/")
}
