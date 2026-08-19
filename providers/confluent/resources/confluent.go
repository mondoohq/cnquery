// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/confluent/connection"
)

// mqlConfluentInternal memoizes the organization record. The identifier, the
// display name and the just-in-time provisioning flag all come from the same
// call, so a query touching more than one of them costs a single request.
type mqlConfluentInternal struct {
	orgOnce sync.Once
	org     *OrganizationRecord
	orgErr  error
}

func (r *mqlConfluent) id() (string, error) {
	conn, err := confluentConn(r.MqlRuntime)
	if err != nil {
		return "", err
	}
	if orgID := conn.OrganizationID(); orgID != "" {
		return connection.NewConfluentOrgIdentifier(orgID), nil
	}
	return "confluent", nil
}

// --- shared plumbing ------------------------------------------------------

// confluentConn pulls the connection off the runtime.
func confluentConn(runtime *plugin.Runtime) (*connection.ConfluentConnection, error) {
	conn, ok := runtime.Connection.(*connection.ConfluentConnection)
	if !ok {
		return nil, errors.New("no Confluent Cloud connection on the runtime")
	}
	return conn, nil
}

// getConfluent returns the root resource. Cross-resource accessors go through
// it so they reuse its cached listings rather than refetching a collection each
// time a child needs to reach one.
func getConfluent(runtime *plugin.Runtime) (*mqlConfluent, error) {
	res, err := CreateResource(runtime, "confluent", map[string]*llx.RawData{})
	if err != nil {
		return nil, err
	}
	return res.(*mqlConfluent), nil
}

// --- shared record shapes -------------------------------------------------

// objectMeta is the metadata block every management API object carries.
type objectMeta struct {
	Self         string        `json:"self"`
	ResourceName string        `json:"resource_name"`
	CreatedAt    confluentTime `json:"created_at"`
	UpdatedAt    confluentTime `json:"updated_at"`
	DeletedAt    confluentTime `json:"deleted_at"`
}

// objectReference is how one management API object points at another. Only the
// identifier is used here; the related URL and the CRN restate it.
type objectReference struct {
	ID           string `json:"id"`
	Environment  string `json:"environment"`
	Related      string `json:"related"`
	ResourceName string `json:"resource_name"`
	APIVersion   string `json:"api_version"`
	Kind         string `json:"kind"`
}

// refID returns the identifier of an optional reference, or the empty string
// when the object carries no reference at all.
func refID(ref *objectReference) string {
	if ref == nil {
		return ""
	}
	return ref.ID
}

// confluentTime decodes a management API timestamp, which arrives as an RFC
// 3339 string and is absent on objects that never reached the state it
// describes.
type confluentTime struct {
	t *time.Time
}

func (c *confluentTime) UnmarshalJSON(b []byte) error {
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
	parsed, err := time.Parse(time.RFC3339, str)
	if err != nil {
		// A timestamp whose shape changed is reported as null rather than
		// failing the whole resource, but it is logged so the change is visible
		// instead of looking like an object that never reached the state the
		// field describes.
		log.Warn().Str("value", str).Msg("confluent> could not parse timestamp; reporting it as null")
		return nil
	}
	c.t = &parsed
	return nil
}

// Time returns the decoded value, or nil when the source was absent.
func (c confluentTime) Time() *time.Time { return c.t }

// --- shared helpers -------------------------------------------------------

// strSliceToAny widens a string slice into an any slice for llx.ArrayData.
func strSliceToAny(in []string) []any {
	out := make([]any, len(in))
	for i := range in {
		out[i] = in[i]
	}
	return out
}

// strMapToAny widens a string map into an any map for llx.MapData.
func strMapToAny(in map[string]string) map[string]any {
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

// optionalInt reports an absent number as null rather than as zero, so a value
// the API does not report is distinguishable from one that is genuinely zero.
func optionalInt(v *int32) *llx.RawData {
	if v == nil {
		return llx.NilData
	}
	return llx.IntData(int64(*v))
}

// ageInDays reports how many whole days have passed since t, or null when the
// timestamp is absent. A key of unknown age must not read as a key created
// today, which is the reading a zero would carry.
func ageInDays(t *time.Time, now time.Time) *llx.RawData {
	if t == nil {
		return llx.NilData
	}
	days := now.Sub(*t).Hours() / 24
	if days < 0 {
		// A creation time in the future is not an age. Report zero rather than
		// a negative number, which no rotation policy is written against.
		days = 0
	}
	return llx.IntData(int64(math.Floor(days)))
}

// --- organization ---------------------------------------------------------

// OrganizationRecord is one entry of the organizations listing.
type OrganizationRecord struct {
	ID          string     `json:"id"`
	DisplayName string     `json:"display_name"`
	JitEnabled  *bool      `json:"jit_enabled"`
	Metadata    objectMeta `json:"metadata"`
}

// FetchOrganization reads the organization the API key belongs to. A Cloud API
// key is issued inside exactly one organization, so the listing holds one
// entry; the first is taken and an empty listing is an error rather than an
// anonymous asset.
func FetchOrganization(ctx context.Context, conn *connection.ConfluentConnection) (*OrganizationRecord, error) {
	orgs, err := connection.GetPaged[OrganizationRecord](ctx, conn, conn.CloudTarget(), "/org/v2/organizations", nil)
	if err != nil {
		return nil, err
	}
	if len(orgs) == 0 {
		return nil, errors.New("the Confluent Cloud API key reaches no organization")
	}
	return &orgs[0], nil
}

func (r *mqlConfluent) fetchOrganization() (*OrganizationRecord, error) {
	r.orgOnce.Do(func() {
		conn, err := confluentConn(r.MqlRuntime)
		if err != nil {
			r.orgErr = err
			return
		}
		r.org, r.orgErr = FetchOrganization(context.Background(), conn)
	})
	return r.org, r.orgErr
}

func (r *mqlConfluent) organizationId() (string, error) {
	org, err := r.fetchOrganization()
	if err != nil {
		return "", err
	}
	return org.ID, nil
}

func (r *mqlConfluent) organizationName() (string, error) {
	org, err := r.fetchOrganization()
	if err != nil {
		return "", err
	}
	return org.DisplayName, nil
}

func (r *mqlConfluent) jitEnabled() (bool, error) {
	org, err := r.fetchOrganization()
	if err != nil {
		return false, err
	}
	if org.JitEnabled == nil {
		// The flag is only reported for organizations where the feature is
		// available. Reporting false would say the feature is off on an
		// organization that never answered the question.
		r.JitEnabled.State = plugin.StateIsSet | plugin.StateIsNull
		return false, nil
	}
	return *org.JitEnabled, nil
}
