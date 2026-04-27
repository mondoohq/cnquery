// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"

	"github.com/ctreminiom/go-atlassian/v2/pkg/infra/models"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/atlassian/connection/confluence"
	"go.mondoo.com/mql/v13/types"
)

const (
	CONFLUENCE_PAGE_LIMIT  = 250
	CONFLUENCE_SPACE_LIMIT = 250
)

func (a *mqlAtlassianConfluence) id() (string, error) {
	return "confluence", nil
}

func (a *mqlAtlassianConfluence) users() ([]any, error) {
	conn, ok := a.MqlRuntime.Connection.(*confluence.ConfluenceConnection)
	if !ok {
		return nil, errors.New("Current connection does not allow confluence access")
	}
	confluence := conn.Client()
	cql := "type = user"
	users, _, err := confluence.Search.Users(context.Background(), cql, 0, 1000, nil)
	if err != nil {
		return nil, err
	}
	res := []any{}
	for _, user := range users.Results {
		mqlAtlassianConfluenceUser, err := CreateResource(a.MqlRuntime, "atlassian.confluence.user",
			map[string]*llx.RawData{
				"id":   llx.StringData(user.User.AccountID),
				"type": llx.StringData(user.User.AccountType),
				"name": llx.StringData(user.User.DisplayName),
			})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlAtlassianConfluenceUser)
	}
	return res, nil
}

func (a *mqlAtlassianConfluenceUser) id() (string, error) {
	return a.Id.Data, nil
}

// initAtlassianConfluenceUser supports creating a bare user reference from an
// account ID. The Confluence v1 SDK exposes no /user GET endpoint, so we cannot
// hydrate fields like name/type here — callers that traversed via a typed ref
// (e.g. permissionUsers, restriction.users) get the accountId only and any
// other accessed field will surface as null. Listing via atlassian.confluence.users
// remains the canonical way to enumerate user details.
func initAtlassianConfluenceUser(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	// A bare resource (just an id) is a valid empty state.
	return args, nil, nil
}

func (a *mqlAtlassianConfluenceGroup) id() (string, error) {
	return "atlassian.confluence.group/" + a.Id.Data, nil
}

// initAtlassianConfluenceGroup supports creating a bare group reference from a
// group id or name. The Confluence v1 SDK exposes no standalone group GET, so
// we cannot hydrate additional fields here. Callers receive a typed handle they
// can compare/filter on.
func initAtlassianConfluenceGroup(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	// A bare resource (just an id/name) is a valid empty state.
	return args, nil, nil
}

// spaces fetches all Confluence spaces with permission expansion so callers can
// inspect anonymous-access settings and per-subject permissions inline.
func (a *mqlAtlassianConfluence) spaces() ([]any, error) {
	conn, ok := a.MqlRuntime.Connection.(*confluence.ConfluenceConnection)
	if !ok {
		return nil, errors.New("Current connection does not allow confluence access")
	}
	client := conn.Client()

	options := &models.GetSpacesOptionScheme{Expand: []string{"permissions"}}
	res := []any{}
	startAt := 0

	for {
		page, _, err := client.Space.Gets(context.Background(), options, startAt, CONFLUENCE_SPACE_LIMIT)
		if err != nil {
			return nil, err
		}
		if page == nil || len(page.Results) == 0 {
			break
		}

		for _, space := range page.Results {
			if space == nil {
				continue
			}

			anon := false
			unlicensed := false
			perms := []any{}
			for _, perm := range space.Permissions {
				if perm == nil {
					continue
				}
				if perm.AnonymousAccess {
					anon = true
				}
				if perm.UnlicensedAccess {
					unlicensed = true
				}

				operation := ""
				targetType := ""
				if perm.Operation != nil {
					operation = perm.Operation.Operation
					targetType = perm.Operation.TargetType
				}

				if perm.Subject != nil {
					if perm.Subject.User != nil {
						for _, u := range perm.Subject.User.Results {
							if u == nil {
								continue
							}
							perms = append(perms, map[string]any{
								"operation":   operation,
								"targetType":  targetType,
								"subjectType": "user",
								"subjectKey":  u.AccountID,
								"subjectName": u.DisplayName,
							})
						}
					}
					if perm.Subject.Group != nil {
						for _, g := range perm.Subject.Group.Results {
							if g == nil {
								continue
							}
							perms = append(perms, map[string]any{
								"operation":   operation,
								"targetType":  targetType,
								"subjectType": "group",
								"subjectKey":  g.ID,
								"subjectName": g.Name,
							})
						}
					}
				}

				// If neither anonymous nor a subject was set but flags fired, still surface a row
				// so the count is non-zero.
				if perm.AnonymousAccess && perm.Subject == nil {
					perms = append(perms, map[string]any{
						"operation":   operation,
						"targetType":  targetType,
						"subjectType": "anonymous",
						"subjectKey":  "",
						"subjectName": "",
					})
				}
			}

			mqlSpace, err := CreateResource(a.MqlRuntime, "atlassian.confluence.space",
				map[string]*llx.RawData{
					"id":               llx.IntData(int64(space.ID)),
					"key":              llx.StringData(space.Key),
					"name":             llx.StringData(space.Name),
					"type":             llx.StringData(space.Type),
					"status":           llx.StringData(space.Status),
					"anonymousAccess":  llx.BoolData(anon),
					"unlicensedAccess": llx.BoolData(unlicensed),
					"permissions":      llx.ArrayData(perms, types.Dict),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlSpace)
		}

		startAt += len(page.Results)
		if len(page.Results) < CONFLUENCE_SPACE_LIMIT {
			break
		}
	}
	return res, nil
}

func (a *mqlAtlassianConfluenceSpace) id() (string, error) {
	return "atlassian.confluence.space/" + a.Key.Data, nil
}

// permissionUsers returns the distinct typed atlassian.confluence.user resources
// referenced by this space's permissions list. The permissions field contains
// subjectType="user" rows whose subjectKey is the user's accountId.
func (a *mqlAtlassianConfluenceSpace) permissionUsers() ([]any, error) {
	seen := map[string]bool{}
	res := []any{}
	for _, p := range a.Permissions.Data {
		row, ok := p.(map[string]any)
		if !ok {
			continue
		}
		if t, _ := row["subjectType"].(string); t != "user" {
			continue
		}
		key, _ := row["subjectKey"].(string)
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		name, _ := row["subjectName"].(string)
		args := map[string]*llx.RawData{
			"id":   llx.StringData(key),
			"name": llx.StringData(name),
			"type": llx.StringData(""),
		}
		mqlUser, err := NewResource(a.MqlRuntime, "atlassian.confluence.user", args)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlUser)
	}
	return res, nil
}

// permissionGroups returns the distinct typed atlassian.confluence.group resources
// referenced by this space's permissions list. The permissions field contains
// subjectType="group" rows whose subjectKey is the group id (and subjectName is
// the group display name).
func (a *mqlAtlassianConfluenceSpace) permissionGroups() ([]any, error) {
	seen := map[string]bool{}
	res := []any{}
	for _, p := range a.Permissions.Data {
		row, ok := p.(map[string]any)
		if !ok {
			continue
		}
		if t, _ := row["subjectType"].(string); t != "group" {
			continue
		}
		key, _ := row["subjectKey"].(string)
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		name, _ := row["subjectName"].(string)
		args := map[string]*llx.RawData{
			"id":   llx.StringData(key),
			"name": llx.StringData(name),
		}
		mqlGroup, err := NewResource(a.MqlRuntime, "atlassian.confluence.group", args)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlGroup)
	}
	return res, nil
}

// pages lists all "page" content within this space. Trashed/draft pages are excluded
// (the SDK defaults to type=page, status=current via the search call).
func (a *mqlAtlassianConfluenceSpace) pages() ([]any, error) {
	conn, ok := a.MqlRuntime.Connection.(*confluence.ConfluenceConnection)
	if !ok {
		return nil, errors.New("Current connection does not allow confluence access")
	}
	client := conn.Client()

	spaceKey := a.Key.Data
	if spaceKey == "" {
		return []any{}, nil
	}

	res := []any{}
	startAt := 0

	for {
		// Use ContentByType so we get only pages and don't mix in blogposts, comments, attachments.
		// status="current" excludes trashed and draft pages — better for audit use cases.
		page, _, err := client.Space.ContentByType(context.Background(), spaceKey, "page", "current", []string{"restrictions.read.restrictions.user", "restrictions.read.restrictions.group"}, startAt, CONFLUENCE_PAGE_LIMIT)
		if err != nil {
			return nil, err
		}
		if page == nil || len(page.Results) == 0 {
			break
		}
		for _, content := range page.Results {
			if content == nil {
				continue
			}
			mqlPage, err := CreateResource(a.MqlRuntime, "atlassian.confluence.page",
				map[string]*llx.RawData{
					"id":       llx.StringData(content.ID),
					"title":    llx.StringData(content.Title),
					"status":   llx.StringData(content.Status),
					"type":     llx.StringData(content.Type),
					"spaceKey": llx.StringData(spaceKey),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlPage)
		}
		startAt += len(page.Results)
		if len(page.Results) < CONFLUENCE_PAGE_LIMIT {
			break
		}
	}
	return res, nil
}

func (a *mqlAtlassianConfluencePage) id() (string, error) {
	return "atlassian.confluence.page/" + a.Id.Data, nil
}

// hasRestrictions returns true if either the read or update operation has any
// explicit user or group entries.
func (a *mqlAtlassianConfluencePage) hasRestrictions() (bool, error) {
	restrictions, err := a.fetchRestrictions()
	if err != nil {
		return false, err
	}
	for _, r := range restrictions {
		entry := r.(*mqlAtlassianConfluencePageRestriction)
		if len(entry.UserIds.Data) > 0 || len(entry.GroupNames.Data) > 0 {
			return true, nil
		}
	}
	return false, nil
}

func (a *mqlAtlassianConfluencePage) restrictions() ([]any, error) {
	return a.fetchRestrictions()
}

// fetchRestrictions loads page-level restrictions and caches them on first call.
// Both hasRestrictions() and restrictions() share this so we only call the API once.
func (a *mqlAtlassianConfluencePage) fetchRestrictions() ([]any, error) {
	if a.Restrictions.State == plugin.StateIsSet {
		return a.Restrictions.Data, nil
	}
	conn, ok := a.MqlRuntime.Connection.(*confluence.ConfluenceConnection)
	if !ok {
		return nil, errors.New("Current connection does not allow confluence access")
	}
	client := conn.Client()

	contentID := a.Id.Data
	if contentID == "" {
		a.Restrictions = plugin.TValue[[]any]{Data: []any{}, State: plugin.StateIsSet}
		return []any{}, nil
	}

	page, _, err := client.Content.Restriction.Gets(context.Background(),
		contentID,
		[]string{"restrictions.user", "restrictions.group"},
		0,
		CONFLUENCE_PAGE_LIMIT,
	)
	if err != nil {
		return nil, err
	}

	res := []any{}
	if page != nil {
		for _, restriction := range page.Results {
			if restriction == nil {
				continue
			}
			users := []any{}
			groups := []any{}
			if restriction.Restrictions != nil {
				if restriction.Restrictions.User != nil {
					for _, u := range restriction.Restrictions.User.Results {
						if u == nil {
							continue
						}
						if u.AccountID != "" {
							users = append(users, u.AccountID)
						}
					}
				}
				if restriction.Restrictions.Group != nil {
					for _, g := range restriction.Restrictions.Group.Results {
						if g == nil {
							continue
						}
						if g.Name != "" {
							groups = append(groups, g.Name)
						}
					}
				}
			}
			compositeID := contentID + "/" + restriction.Operation
			mqlRestriction, err := CreateResource(a.MqlRuntime, "atlassian.confluence.page.restriction",
				map[string]*llx.RawData{
					"id":         llx.StringData(compositeID),
					"operation":  llx.StringData(restriction.Operation),
					"userIds":    llx.ArrayData(users, types.String),
					"groupNames": llx.ArrayData(groups, types.String),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlRestriction)
		}
	}

	a.Restrictions = plugin.TValue[[]any]{Data: res, State: plugin.StateIsSet}
	return res, nil
}

func (a *mqlAtlassianConfluencePageRestriction) id() (string, error) {
	return "atlassian.confluence.page.restriction/" + a.Id.Data, nil
}

// users resolves the userIds list to typed atlassian.confluence.user references.
// Each id is treated as an accountId; the underlying user resource has no fetch
// endpoint so only the id is hydrated.
func (a *mqlAtlassianConfluencePageRestriction) users() ([]any, error) {
	res := []any{}
	for _, raw := range a.UserIds.Data {
		accountID, ok := raw.(string)
		if !ok || accountID == "" {
			continue
		}
		mqlUser, err := NewResource(a.MqlRuntime, "atlassian.confluence.user",
			map[string]*llx.RawData{
				"id":   llx.StringData(accountID),
				"name": llx.StringData(""),
				"type": llx.StringData(""),
			})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlUser)
	}
	return res, nil
}

// groups resolves the groupNames list to typed atlassian.confluence.group references.
// Confluence page restrictions identify groups by name (not id), so we use the
// name as the resource id for caching.
func (a *mqlAtlassianConfluencePageRestriction) groups() ([]any, error) {
	res := []any{}
	for _, raw := range a.GroupNames.Data {
		name, ok := raw.(string)
		if !ok || name == "" {
			continue
		}
		mqlGroup, err := NewResource(a.MqlRuntime, "atlassian.confluence.group",
			map[string]*llx.RawData{
				"id":   llx.StringData(name),
				"name": llx.StringData(name),
			})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlGroup)
	}
	return res, nil
}
