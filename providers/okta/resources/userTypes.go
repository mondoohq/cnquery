// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"encoding/json"
	"time"

	"github.com/okta/okta-sdk-golang/v5/okta"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/okta/connection"
)

// oktaUserTypeRaw is the user type wire shape. The v5 SDK's UserType struct
// declares only `id`; every other attribute the API returns lands in its
// untyped AdditionalProperties map. Its MarshalJSON writes those back out, so
// re-marshaling an SDK UserType and decoding it here recovers the full object.
type oktaUserTypeRaw struct {
	Id            string     `json:"id,omitempty"`
	Name          string     `json:"name,omitempty"`
	DisplayName   string     `json:"displayName,omitempty"`
	Description   string     `json:"description,omitempty"`
	Default       bool       `json:"default,omitempty"`
	CreatedBy     string     `json:"createdBy,omitempty"`
	LastUpdatedBy string     `json:"lastUpdatedBy,omitempty"`
	Created       *time.Time `json:"created,omitempty"`
	LastUpdated   *time.Time `json:"lastUpdated,omitempty"`
}

// mqlOktaUserTypeInternal caches the administrator ids behind the createdBy and
// lastUpdatedBy accessors so they resolve without public raw-id fields.
type mqlOktaUserTypeInternal struct {
	cacheCreatedBy     string
	cacheLastUpdatedBy string
}

func (o *mqlOkta) userTypes() ([]any, error) {
	conn := o.MqlRuntime.Connection.(*connection.OktaConnection)
	client := conn.Client()

	ctx := context.Background()
	slice, _, err := client.UserTypeAPI.ListUserTypes(ctx).Execute()
	if err != nil {
		return nil, err
	}

	list := []any{}
	for i := range slice {
		entry, err := decodeOktaUserType(&slice[i])
		if err != nil {
			return nil, err
		}
		r, err := newMqlOktaUserType(o.MqlRuntime, entry)
		if err != nil {
			return nil, err
		}
		list = append(list, r)
	}
	return list, nil
}

// decodeOktaUserType normalizes an SDK UserType through JSON into the full
// wire shape. See oktaUserTypeRaw for why the SDK type is not enough.
func decodeOktaUserType(src *okta.UserType) (*oktaUserTypeRaw, error) {
	raw, err := json.Marshal(src)
	if err != nil {
		return nil, err
	}
	entry := &oktaUserTypeRaw{}
	if err := json.Unmarshal(raw, entry); err != nil {
		return nil, err
	}
	return entry, nil
}

func initOktaUserType(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	// If we already have the full set of fields, no fetch needed.
	if len(args) > 1 {
		return args, nil, nil
	}

	idArg, ok := args["id"]
	if !ok || idArg == nil || idArg.Value == nil {
		// Bare resource construction (no id) is a valid empty state.
		return args, nil, nil
	}
	id, ok := idArg.Value.(string)
	if !ok || id == "" {
		return args, nil, nil
	}

	conn := runtime.Connection.(*connection.OktaConnection)
	client := conn.Client()
	ctx := context.Background()

	item, _, err := client.UserTypeAPI.GetUserType(ctx, id).Execute()
	if err != nil {
		return nil, nil, err
	}
	if item == nil {
		return nil, nil, nil
	}

	entry, err := decodeOktaUserType(item)
	if err != nil {
		return nil, nil, err
	}

	// Returning the built resource is the only way to populate the Internal
	// struct the createdBy and lastUpdatedBy accessors read from.
	res, err := newMqlOktaUserType(runtime, entry)
	if err != nil {
		return nil, nil, err
	}
	return nil, res, nil
}

func newMqlOktaUserType(runtime *plugin.Runtime, entry *oktaUserTypeRaw) (*mqlOktaUserType, error) {
	r, err := CreateResource(runtime, "okta.userType", map[string]*llx.RawData{
		"id":          llx.StringData(entry.Id),
		"name":        llx.StringData(entry.Name),
		"displayName": llx.StringData(entry.DisplayName),
		"description": llx.StringData(entry.Description),
		"default":     llx.BoolData(entry.Default),
		"created":     llx.TimeDataPtr(entry.Created),
		"lastUpdated": llx.TimeDataPtr(entry.LastUpdated),
	})
	if err != nil {
		return nil, err
	}

	mqlUserType := r.(*mqlOktaUserType)
	mqlUserType.cacheCreatedBy = entry.CreatedBy
	mqlUserType.cacheLastUpdatedBy = entry.LastUpdatedBy
	return mqlUserType, nil
}

func (o *mqlOktaUserType) id() (string, error) {
	return "okta.userType/" + o.Id.Data, o.Id.Error
}

func (o *mqlOktaUserType) createdBy() (*mqlOktaUser, error) {
	if o.cacheCreatedBy == "" {
		o.CreatedBy.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	return oktaUserRef(o.MqlRuntime, o.cacheCreatedBy)
}

func (o *mqlOktaUserType) lastUpdatedBy() (*mqlOktaUser, error) {
	if o.cacheLastUpdatedBy == "" {
		o.LastUpdatedBy.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	return oktaUserRef(o.MqlRuntime, o.cacheLastUpdatedBy)
}

// oktaUserRef resolves a user id to an okta.user. The runtime caches user
// instances by id, so repeated references across resources share one fetch.
func oktaUserRef(runtime *plugin.Runtime, id string) (*mqlOktaUser, error) {
	r, err := NewResource(runtime, "okta.user", map[string]*llx.RawData{
		"id": llx.StringData(id),
	})
	if err != nil {
		return nil, err
	}
	return r.(*mqlOktaUser), nil
}
