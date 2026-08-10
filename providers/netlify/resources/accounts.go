// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/netlify/connection"
	"go.mondoo.com/mql/v13/types"
)

// mqlNetlifyAccountInternal caches the values the account's own child lookups
// need: the slug that the member and site endpoints are keyed on rather than
// the account id, and the owner ids that owners resolves against the roster.
type mqlNetlifyAccountInternal struct {
	cacheSlug     string
	cacheOwnerIds []string
}

type accountRecord struct {
	ID           string      `json:"id"`
	Slug         string      `json:"slug"`
	Name         string      `json:"name"`
	Type         string      `json:"type"`
	TypeName     string      `json:"type_name"`
	OwnerIDs     []string    `json:"owner_ids"`
	RolesAllowed []string    `json:"roles_allowed"`
	CreatedAt    netlifyTime `json:"created_at"`
	UpdatedAt    netlifyTime `json:"updated_at"`
}

// accounts lists every account the token can reach, narrowed to the account the
// --account flag or a discovered child asset scopes the connection to.
func (n *mqlNetlify) accounts() ([]any, error) {
	c := netlifyConn(n.MqlRuntime)

	records, err := connection.GetList[accountRecord](context.Background(), c, "/accounts", nil)
	if err != nil {
		return nil, err
	}

	filter := c.AccountFilter()

	var res []any
	for i := range records {
		rec := records[i]
		if filter != "" && rec.ID != filter && rec.Slug != filter {
			continue
		}
		account, err := newNetlifyAccount(n.MqlRuntime, &rec)
		if err != nil {
			return nil, err
		}
		res = append(res, account)
	}
	return res, nil
}

func newNetlifyAccount(runtime *plugin.Runtime, rec *accountRecord) (*mqlNetlifyAccount, error) {
	res, err := CreateResource(runtime, "netlify.account", map[string]*llx.RawData{
		"id":           llx.StringData(rec.ID),
		"slug":         llx.StringData(rec.Slug),
		"name":         llx.StringData(rec.Name),
		"type":         llx.StringData(rec.Type),
		"typeName":     llx.StringData(rec.TypeName),
		"rolesAllowed": llx.ArrayData(strSliceToAny(rec.RolesAllowed), types.String),
		"createdAt":    llx.TimeDataPtr(rec.CreatedAt.Time()),
		"updatedAt":    llx.TimeDataPtr(rec.UpdatedAt.Time()),
	})
	if err != nil {
		return nil, err
	}

	account := res.(*mqlNetlifyAccount)
	account.cacheSlug = rec.Slug
	account.cacheOwnerIds = rec.OwnerIDs
	return account, nil
}

// initNetlifyAccount resolves the account a query targets: an explicit id or
// slug argument, the account a discovered asset is scoped to, or the --account
// connection option.
func initNetlifyAccount(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}

	c := netlifyConn(runtime)

	want := ""
	for _, key := range []string{"id", "slug"} {
		if data, ok := args[key]; ok {
			if s, ok := data.Value.(string); ok && s != "" {
				want = s
				break
			}
		}
	}
	if want == "" && c.Asset() != nil {
		for _, pid := range c.Asset().PlatformIds {
			if t := strings.TrimPrefix(pid, connection.PlatformIdNetlifyAccount); t != pid {
				want = t
				break
			}
		}
	}
	if want == "" {
		want = c.AccountFilter()
	}
	if want == "" {
		return nil, nil, errors.New("netlify.account requires an id or slug")
	}

	// The accounts endpoint is the only one that answers for either an id or a
	// slug, so the lookup runs against the list the token can reach.
	records, err := connection.GetList[accountRecord](context.Background(), c, "/accounts", nil)
	if err != nil {
		return nil, nil, err
	}
	for i := range records {
		if records[i].ID == want || records[i].Slug == want {
			account, err := newNetlifyAccount(runtime, &records[i])
			if err != nil {
				return nil, nil, err
			}
			return args, account, nil
		}
	}

	return nil, nil, fmt.Errorf("%w: %s", errAccountNotFound, want)
}

// errAccountNotFound reports that the wanted account is not one this token is a
// member of. Callers resolving an account by reference degrade to null on it
// rather than failing the resource that pointed at it.
var errAccountNotFound = errors.New("netlify.account not found")

func (a *mqlNetlifyAccount) id() (string, error) {
	return a.Id.Data, a.Id.Error
}

// --- account members ------------------------------------------------------

type memberRecord struct {
	ID        string `json:"id"`
	FullName  string `json:"full_name"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	AvatarURL string `json:"avatar"`
}

func (a *mqlNetlifyAccount) members() ([]any, error) {
	c := netlifyConn(a.MqlRuntime)

	slug := a.accountSlug()
	if slug == "" {
		return nil, errors.New("netlify.account.members requires the account slug")
	}

	records, err := connection.GetList[memberRecord](context.Background(), c,
		"/"+url.PathEscape(slug)+"/members", nil)
	if err != nil {
		// A member without administrative rights cannot read the roster.
		if connection.IsForbidden(err) {
			a.Members = plugin.TValue[[]any]{State: plugin.StateIsSet | plugin.StateIsNull}
			return nil, nil
		}
		return nil, err
	}

	var res []any
	for i := range records {
		rec := records[i]
		member, err := CreateResource(a.MqlRuntime, "netlify.account.member", map[string]*llx.RawData{
			"id":        llx.StringData(rec.ID),
			"fullName":  llx.StringData(rec.FullName),
			"email":     llx.StringData(rec.Email),
			"role":      llx.StringData(rec.Role),
			"avatarUrl": llx.StringData(rec.AvatarURL),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, member)
	}
	return res, nil
}

func (m *mqlNetlifyAccountMember) id() (string, error) {
	return m.Id.Data, m.Id.Error
}

// owners resolves the account's owner ids against its roster.
func (a *mqlNetlifyAccount) owners() ([]any, error) {
	members := a.GetMembers()
	if members.Error != nil {
		return nil, members.Error
	}
	if members.State&plugin.StateIsNull != 0 {
		a.Owners = plugin.TValue[[]any]{State: plugin.StateIsSet | plugin.StateIsNull}
		return nil, nil
	}

	owners := map[string]struct{}{}
	for _, id := range a.cacheOwnerIds {
		owners[id] = struct{}{}
	}

	var res []any
	for _, it := range members.Data {
		member, ok := it.(*mqlNetlifyAccountMember)
		if !ok {
			continue
		}
		if _, ok := owners[member.Id.Data]; ok {
			res = append(res, member)
		}
	}
	return res, nil
}

// accountSlug returns the slug the member and site endpoints are keyed on,
// preferring the cached value from the creating call and falling back to the
// field for a resource restored from a recording.
func (a *mqlNetlifyAccount) accountSlug() string {
	if a.cacheSlug != "" {
		return a.cacheSlug
	}
	return a.Slug.Data
}
