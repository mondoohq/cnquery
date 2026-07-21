// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import "strings"

// The v5 Okta SDK is OpenAPI-generated and models almost every scalar as a
// pointer so it can distinguish "unset" from a zero value. The previous v2
// types were plain values, so these helpers dereference v5 pointers back to the
// zero-value semantics the resource mappers (and existing MQL queries) expect.

func oktaStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// resolveOktaResourceTarget maps an Okta resource reference (an ORN and/or a
// self-link URL) to a modeled resource type and id. It returns "user",
// "group", or "application" with the target id, or two empty strings when the
// reference does not name one of those resources.
//
// ORN form:  orn:okta:<service>:<orgId>:<resourceType>:<id>[:...]
// URL form:  https://<org>/api/v1/<collection>/<id>[/...]
func resolveOktaResourceTarget(orn, href string) (targetType string, id string) {
	if orn != "" {
		parts := strings.Split(orn, ":")
		for i := 0; i+1 < len(parts); i++ {
			switch parts[i] {
			case "users":
				return "user", parts[i+1]
			case "groups":
				return "group", parts[i+1]
			case "apps":
				// app ORNs carry an app-type segment before the id
				// (orn:okta:idp:<org>:apps:<appType>:<appId>).
				return "application", parts[len(parts)-1]
			}
		}
	}

	if href != "" {
		for _, m := range []struct{ seg, typ string }{
			{"/users/", "user"},
			{"/groups/", "group"},
			{"/apps/", "application"},
		} {
			if i := strings.Index(href, m.seg); i >= 0 {
				rest := href[i+len(m.seg):]
				if j := strings.IndexByte(rest, '/'); j >= 0 {
					rest = rest[:j]
				}
				if rest != "" {
					return m.typ, rest
				}
			}
		}
	}

	return "", ""
}

// oktaLinkHref extracts an href from an Okta HAL `_links` entry, which the v5
// SDK surfaces as an untyped map[string]interface{} of the shape
// {"href": "..."}. Returns "" when the entry is missing or malformed.
func oktaLinkHref(link any) string {
	m, ok := link.(map[string]interface{})
	if !ok {
		return ""
	}
	href, ok := m["href"].(string)
	if !ok {
		return ""
	}
	return href
}

// lastPathSegment returns the final non-empty segment of a URL path.
func lastPathSegment(s string) string {
	s = strings.TrimRight(s, "/")
	if i := strings.LastIndex(s, "/"); i >= 0 {
		return s[i+1:]
	}
	return s
}

// oktaRoleIdFromPermissionsHref pulls a custom-role id out of a permissions
// self-link of the form ".../iam/roles/<roleId>/permissions".
func oktaRoleIdFromPermissionsHref(href string) string {
	const marker = "/roles/"
	i := strings.Index(href, marker)
	if i < 0 {
		return ""
	}
	rest := href[i+len(marker):]
	if j := strings.IndexByte(rest, '/'); j >= 0 {
		rest = rest[:j]
	}
	return rest
}

func oktaBool(b *bool) bool {
	if b == nil {
		return false
	}
	return *b
}
