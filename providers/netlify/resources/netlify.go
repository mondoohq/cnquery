// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/netlify/connection"
	"go.mondoo.com/mql/v13/types"
)

func (r *mqlNetlify) id() (string, error) {
	return "netlify", nil
}

// --- shared helpers -------------------------------------------------------

// netlifyTime decodes a Netlify timestamp, which arrives as an RFC 3339 string
// and is absent or empty on records that never reached the state it describes.
type netlifyTime struct {
	t *time.Time
}

func (n *netlifyTime) UnmarshalJSON(b []byte) error {
	s := strings.TrimSpace(string(b))
	if s == "null" || s == "" {
		return nil
	}
	var str string
	if err := json.Unmarshal(b, &str); err != nil {
		return err
	}
	if str == "" {
		return nil
	}
	tt, err := time.Parse(time.RFC3339, str)
	if err != nil {
		// A timestamp the API changed the shape of is reported as null rather
		// than failing the whole resource, but it is logged so the change is
		// visible instead of looking like a record that never reached the
		// state the field describes.
		log.Warn().Str("value", str).Msg("netlify> could not parse timestamp; reporting it as null")
		return nil
	}
	n.t = &tt
	return nil
}

// Time returns the decoded time value, or nil when the source was absent.
func (n netlifyTime) Time() *time.Time {
	return n.t
}

// strSliceToAny widens a string slice into an any slice for llx.ArrayData.
func strSliceToAny(in []string) []any {
	out := make([]any, len(in))
	for i := range in {
		out[i] = in[i]
	}
	return out
}

// netlifyConn returns the Netlify connection backing the runtime.
func netlifyConn(runtime *plugin.Runtime) *connection.NetlifyConnection {
	return runtime.Connection.(*connection.NetlifyConnection)
}

// getNetlify returns the root resource, which every cross-resource accessor
// goes through so that its cached account and site lists are reused rather
// than refetched.
func getNetlify(runtime *plugin.Runtime) (*mqlNetlify, error) {
	res, err := CreateResource(runtime, "netlify", map[string]*llx.RawData{})
	if err != nil {
		return nil, err
	}
	return res.(*mqlNetlify), nil
}

// --- root resource --------------------------------------------------------

type userRecord struct {
	ID             string      `json:"id"`
	UID            string      `json:"uid"`
	Email          string      `json:"email"`
	FullName       string      `json:"full_name"`
	AvatarURL      string      `json:"avatar_url"`
	LoginProviders []string    `json:"login_providers"`
	SiteCount      int64       `json:"site_count"`
	CreatedAt      netlifyTime `json:"created_at"`
	LastLogin      netlifyTime `json:"last_login"`
}

func (n *mqlNetlify) currentUser() (*mqlNetlifyUser, error) {
	c := netlifyConn(n.MqlRuntime)

	var rec userRecord
	if err := c.Get(context.Background(), "/user", nil, &rec); err != nil {
		return nil, err
	}

	id := rec.ID
	if id == "" {
		id = rec.UID
	}

	res, err := CreateResource(n.MqlRuntime, "netlify.user", map[string]*llx.RawData{
		"id":             llx.StringData(id),
		"email":          llx.StringData(rec.Email),
		"fullName":       llx.StringData(rec.FullName),
		"avatarUrl":      llx.StringData(rec.AvatarURL),
		"loginProviders": llx.ArrayData(strSliceToAny(rec.LoginProviders), types.String),
		"siteCount":      llx.IntData(rec.SiteCount),
		"createdAt":      llx.TimeDataPtr(rec.CreatedAt.Time()),
		"lastLogin":      llx.TimeDataPtr(rec.LastLogin.Time()),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlNetlifyUser), nil
}

func (n *mqlNetlifyUser) id() (string, error) {
	return n.Id.Data, n.Id.Error
}

// sites flattens every site of every account in scope.
func (n *mqlNetlify) sites() ([]any, error) {
	accounts, err := n.accounts()
	if err != nil {
		return nil, err
	}

	var res []any
	for _, it := range accounts {
		account := it.(*mqlNetlifyAccount)
		sites := account.GetSites()
		if sites.Error != nil {
			return nil, sites.Error
		}
		res = append(res, sites.Data...)
	}
	return res, nil
}

// dnsZones lists the Netlify-managed DNS zones of every account in scope.
func (n *mqlNetlify) dnsZones() ([]any, error) {
	c := netlifyConn(n.MqlRuntime)

	records, err := connection.GetPaged[dnsZoneRecord](context.Background(), c, "/dns_zones", nil)
	if err != nil {
		return nil, err
	}

	accountFilter := c.AccountFilter()

	var res []any
	for i := range records {
		rec := records[i]
		if accountFilter != "" && rec.AccountID != accountFilter && rec.AccountSlug != accountFilter {
			continue
		}
		zone, err := newNetlifyDnsZone(n.MqlRuntime, &rec)
		if err != nil {
			return nil, err
		}
		res = append(res, zone)
	}
	return res, nil
}

type deployKeyRecord struct {
	ID        string      `json:"id"`
	PublicKey string      `json:"public_key"`
	CreatedAt netlifyTime `json:"created_at"`
}

func (n *mqlNetlify) deployKeys() ([]any, error) {
	c := netlifyConn(n.MqlRuntime)

	records, err := connection.GetList[deployKeyRecord](context.Background(), c, "/deploy_keys", nil)
	if err != nil {
		return nil, err
	}

	var res []any
	for i := range records {
		key, err := newNetlifyDeployKey(n.MqlRuntime, &records[i])
		if err != nil {
			return nil, err
		}
		res = append(res, key)
	}
	return res, nil
}
