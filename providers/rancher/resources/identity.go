// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/types"
)

// mqlRancherTokenInternal carries the account and cluster a token names.
type mqlRancherTokenInternal struct {
	cacheUserID    string
	cacheClusterID string
}

// -- authentication providers -----------------------------------------------

func (r *mqlRancher) authProviders() ([]any, error) {
	records, err := listRecords[authConfigRecord](r.MqlRuntime, pathAuthConfigs)
	if err != nil {
		return nil, err
	}

	res := make([]any, 0, len(records))
	for i := range records {
		record := &records[i]
		id := record.ID
		if id == "" {
			id = record.Name
		}

		mqlProvider, err := CreateResource(r.MqlRuntime, "rancher.authProvider", map[string]*llx.RawData{
			"__id":                llx.StringData(id),
			"id":                  llx.StringData(id),
			"type":                llx.StringData(record.Type),
			"enabled":             llx.BoolData(record.Enabled),
			"accessMode":          llx.StringData(record.AccessMode),
			"allowedPrincipalIds": llx.ArrayData(toAnySlice(record.AllowedPrincipalIDs), types.String),
			"isLocalProvider":     llx.BoolData(record.Type == localAuthConfigType),
			"logoutAllSupported":  llx.BoolData(record.LogoutAllSupported),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlProvider)
	}
	return res, nil
}

// -- tokens -----------------------------------------------------------------

func (r *mqlRancher) tokens() ([]any, error) {
	records, err := listRecords[tokenRecord](r.MqlRuntime, pathTokens)
	if err != nil {
		return nil, err
	}

	res := make([]any, 0, len(records))
	for i := range records {
		record := &records[i]

		// The token value itself is deliberately absent from tokenRecord and
		// from everything below. Rancher declares it write-only and does not
		// return it, but not decoding it is what makes that a property of this
		// provider rather than of the server's current behavior.
		resource, err := CreateResource(r.MqlRuntime, "rancher.token", map[string]*llx.RawData{
			"__id":               llx.StringData(record.ID),
			"id":                 llx.StringData(record.ID),
			"userId":             llx.StringData(record.UserID),
			"description":        llx.StringData(record.Description),
			"authProvider":       llx.StringData(record.AuthProvider),
			"isDerived":          llx.BoolData(record.IsDerived),
			"enabled":            llx.BoolDataPtr(record.Enabled),
			"expired":            llx.BoolData(record.Expired),
			"current":            llx.BoolData(record.Current),
			"ttlMillis":          llx.IntData(record.TTLMillis),
			"expiresAt":          llx.TimeDataPtr(parseTime(record.ExpiresAt)),
			"lastUsedAt":         llx.TimeDataPtr(parseTime(record.LastUsedAt)),
			"activityLastSeenAt": llx.TimeDataPtr(parseTime(record.ActivityLastSeenAt)),
			"created":            llx.TimeDataPtr(parseTime(record.Created)),
		})
		if err != nil {
			return nil, err
		}

		mqlToken := resource.(*mqlRancherToken)
		mqlToken.cacheUserID = record.UserID
		mqlToken.cacheClusterID = record.ClusterID
		res = append(res, mqlToken)
	}
	return res, nil
}

func (r *mqlRancherToken) neverExpires() (bool, error) {
	return neverExpires(r.TtlMillis.Data), nil
}

func (r *mqlRancherToken) user() (*mqlRancherUser, error) {
	mqlUser, err := userByID(r.MqlRuntime, r.cacheUserID)
	if err != nil {
		return nil, err
	}
	if mqlUser == nil {
		r.User.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	return mqlUser, nil
}

func (r *mqlRancherToken) cluster() (*mqlRancherCluster, error) {
	mqlCluster, err := clusterByID(r.MqlRuntime, r.cacheClusterID)
	if err != nil {
		return nil, err
	}
	if mqlCluster == nil {
		r.Cluster.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	return mqlCluster, nil
}

// -- users ------------------------------------------------------------------

func (r *mqlRancher) users() ([]any, error) {
	records, err := listRecords[userRecord](r.MqlRuntime, pathUsers)
	if err != nil {
		return nil, err
	}

	res := make([]any, 0, len(records))
	for i := range records {
		mqlUser, err := buildUser(r.MqlRuntime, &records[i])
		if err != nil {
			return nil, err
		}
		res = append(res, mqlUser)
	}
	return res, nil
}

func buildUser(runtime *plugin.Runtime, record *userRecord) (*mqlRancherUser, error) {
	// No password material is read. The stored user object carries a password
	// field, which the API declares write-only; it is not part of userRecord.
	resource, err := CreateResource(runtime, "rancher.user", map[string]*llx.RawData{
		"__id":               llx.StringData(record.ID),
		"id":                 llx.StringData(record.ID),
		"username":           llx.StringData(record.Username),
		"name":               llx.StringData(record.Name),
		"description":        llx.StringData(record.Description),
		"enabled":            llx.BoolDataPtr(record.Enabled),
		"mustChangePassword": llx.BoolData(record.MustChangePassword),
		"created":            llx.TimeDataPtr(parseTime(record.Created)),
		"principalIds":       llx.ArrayData(toAnySlice(record.PrincipalIDs), types.String),
		"isSystemUser":       llx.BoolData(isSystemUser(record.PrincipalIDs)),
	})
	if err != nil {
		return nil, err
	}
	return resource.(*mqlRancherUser), nil
}

func userByID(runtime *plugin.Runtime, id string) (*mqlRancherUser, error) {
	if id == "" {
		return nil, nil
	}
	records, err := listRecords[userRecord](runtime, pathUsers)
	if err != nil {
		return nil, err
	}
	for i := range records {
		if records[i].ID == id {
			return buildUser(runtime, &records[i])
		}
	}
	return nil, nil
}

func (r *mqlRancherUser) globalRoleBindings() ([]any, error) {
	records, err := listRecords[globalRoleBindingRecord](r.MqlRuntime, pathGlobalRoleBindings)
	if err != nil {
		return nil, err
	}

	userID := r.Id.Data
	res := []any{}
	for i := range records {
		if records[i].UserID != userID {
			continue
		}
		mqlBinding, err := buildGlobalRoleBinding(r.MqlRuntime, &records[i])
		if err != nil {
			return nil, err
		}
		res = append(res, mqlBinding)
	}
	return res, nil
}
