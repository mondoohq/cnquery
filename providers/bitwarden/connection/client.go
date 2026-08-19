// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/cockroachdb/errors"
)

// Client is a minimal, hand-written net/http client for the read endpoints
// of the Bitwarden Public API this provider needs. It is NOT generated from
// Bitwarden's OpenAPI spec; see providers/bitwarden/README.md for why, and
// for the production intent (an OpenAPI-generated client per ADR-037).
type Client struct {
	baseUrl    string
	httpClient *http.Client
}

// NewClient wraps an already-authenticated *http.Client (an OAuth2
// client-credentials client, see connection.go) with the Bitwarden Public
// API base URL.
func NewClient(baseUrl string, httpClient *http.Client) *Client {
	return &Client{
		baseUrl:    strings.TrimSuffix(baseUrl, "/"),
		httpClient: httpClient,
	}
}

// SelectionReadOnly is Bitwarden's embedded "who can access this" record,
// returned inline on both member/group responses (their granted
// collections) and collection responses (their granted groups). Alongside
// the target ID it carries the permission flags of the access grant:
// readOnly, hidePasswords, and manage.
type SelectionReadOnly struct {
	Id            string `json:"id"`
	ReadOnly      bool   `json:"readOnly"`
	HidePasswords bool   `json:"hidePasswords"`
	Manage        bool   `json:"manage"`
}

// flexEnum decodes a JSON value the Bitwarden Public API may return either
// as a numeric enum ordinal or as its string name, depending on API
// version/endpoint. Translation helpers (policyTypeName, roleName,
// statusName) turn this into the stable lowercase string the .lr schema
// documents, regardless of which wire form was sent.
type flexEnum struct {
	asInt    *int
	asString *string
}

func (f *flexEnum) UnmarshalJSON(b []byte) error {
	if string(b) == "null" {
		return nil
	}
	var i int
	if err := json.Unmarshal(b, &i); err == nil {
		f.asInt = &i
		return nil
	}
	var s string
	if err := json.Unmarshal(b, &s); err == nil {
		f.asString = &s
		return nil
	}
	return errors.Newf("bitwarden: cannot decode enum value %s", string(b))
}

// listResponse is the envelope Bitwarden's Public API wraps list endpoints
// in: {"object": "list", "data": [...], "continuationToken": null}.
//
// Known limitation: the list readers below (ListPolicies, ListMembers,
// ListGroups, ListCollections) return only the first page (out.Data) and do
// not follow ContinuationToken. The Bitwarden Public API returns the full set
// in a single response for these organization-scoped endpoints (the token is
// null in practice), so results are complete today. If a future API version
// starts paginating any of these, add a loop that re-requests with the
// continuation token until it is nil, or results will be silently truncated.
type listResponse[T any] struct {
	Object            string  `json:"object"`
	Data              []T     `json:"data"`
	ContinuationToken *string `json:"continuationToken"`
}

// Policy is a single organization-wide security policy, GET /policies.
type Policy struct {
	Id      string         `json:"id"`
	Type    flexEnum       `json:"type"`
	Enabled bool           `json:"enabled"`
	Data    map[string]any `json:"data"`
}

// Member is a single organization membership record, GET /members.
type Member struct {
	Id                    string              `json:"id"`
	UserId                *string             `json:"userId"`
	Name                  *string             `json:"name"`
	Email                 string              `json:"email"`
	Type                  flexEnum            `json:"type"`
	Status                flexEnum            `json:"status"`
	TwoFactorEnabled      bool                `json:"twoFactorEnabled"`
	ResetPasswordEnrolled bool                `json:"resetPasswordEnrolled"`
	ExternalId            *string             `json:"externalId"`
	Collections           []SelectionReadOnly `json:"collections"`
}

// Group is a single organization group record, GET /groups.
type Group struct {
	Id          string              `json:"id"`
	Name        string              `json:"name"`
	ExternalId  *string             `json:"externalId"`
	Collections []SelectionReadOnly `json:"collections"`
}

// Collection is a single organization collection record, GET /collections.
type Collection struct {
	Id         string              `json:"id"`
	Name       string              `json:"name"`
	ExternalId *string             `json:"externalId"`
	Groups     []SelectionReadOnly `json:"groups"`
}

// get issues an authenticated GET against the Public API and decodes the
// JSON response body into out. A non-2xx response is returned as an error
// carrying the status code and response body for diagnosis.
func (c *Client) get(ctx context.Context, path string, out any) error {
	u := c.baseUrl + path
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return errors.Wrapf(err, "bitwarden: request to %s failed", path)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return errors.Wrapf(err, "bitwarden: failed to read response body for %s", path)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return errors.Newf("bitwarden: request to %s failed with status %d: %s", path, resp.StatusCode, string(body))
	}

	if out == nil {
		return nil
	}
	if err := json.Unmarshal(body, out); err != nil {
		return errors.Wrapf(err, "bitwarden: failed to decode response for %s", path)
	}
	return nil
}

// ListPolicies lists every security policy configured for the organization.
func (c *Client) ListPolicies(ctx context.Context) ([]Policy, error) {
	var out listResponse[Policy]
	if err := c.get(ctx, "/policies", &out); err != nil {
		return nil, err
	}
	return out.Data, nil
}

// ListMembers lists every member of the organization.
func (c *Client) ListMembers(ctx context.Context) ([]Member, error) {
	var out listResponse[Member]
	if err := c.get(ctx, "/members", &out); err != nil {
		return nil, err
	}
	return out.Data, nil
}

// GetMember reads a single member by its member (organization user) ID.
func (c *Client) GetMember(ctx context.Context, id string) (*Member, error) {
	var out Member
	if err := c.get(ctx, "/members/"+url.PathEscape(id), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetMemberGroupIds reads the IDs of the groups a member belongs to.
func (c *Client) GetMemberGroupIds(ctx context.Context, id string) ([]string, error) {
	var out []string
	if err := c.get(ctx, fmt.Sprintf("/members/%s/group-ids", url.PathEscape(id)), &out); err != nil {
		return nil, err
	}
	return out, nil
}

// ListGroups lists every group defined in the organization.
func (c *Client) ListGroups(ctx context.Context) ([]Group, error) {
	var out listResponse[Group]
	if err := c.get(ctx, "/groups", &out); err != nil {
		return nil, err
	}
	return out.Data, nil
}

// GetGroup reads a single group by its ID.
func (c *Client) GetGroup(ctx context.Context, id string) (*Group, error) {
	var out Group
	if err := c.get(ctx, "/groups/"+url.PathEscape(id), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetGroupMemberIds reads the IDs of the members belonging to a group.
func (c *Client) GetGroupMemberIds(ctx context.Context, id string) ([]string, error) {
	var out []string
	if err := c.get(ctx, fmt.Sprintf("/groups/%s/member-ids", url.PathEscape(id)), &out); err != nil {
		return nil, err
	}
	return out, nil
}

// ListCollections lists every collection defined in the organization.
func (c *Client) ListCollections(ctx context.Context) ([]Collection, error) {
	var out listResponse[Collection]
	if err := c.get(ctx, "/collections", &out); err != nil {
		return nil, err
	}
	return out.Data, nil
}

// GetCollection reads a single collection by its ID.
func (c *Client) GetCollection(ctx context.Context, id string) (*Collection, error) {
	var out Collection
	if err := c.get(ctx, "/collections/"+url.PathEscape(id), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// policyTypeNames translates the bitwarden/server PolicyType enum ordinal
// (see ADR-037 References) to the stable lowercase name the .lr schema
// documents. Kept in ordinal order for readability; if Bitwarden adds a new
// policy type, an unrecognized ordinal falls back to a numbered placeholder
// rather than silently mislabeling an existing one.
var policyTypeNames = []string{
	"twoFactorAuthentication",
	"masterPassword",
	"passwordGenerator",
	"singleOrg",
	"requireSso",
	"organizationDataOwnership",
	"disableSend",
	"sendOptions",
	"resetPassword",
	"maximumVaultTimeout",
	"disablePersonalVaultExport",
	"activateAutofill",
	"automaticAppLogIn",
	"freeFamiliesSponsorshipPolicy",
	"removeUnlockWithPin",
	"restrictedItemTypesPolicy",
	"uriMatchDefaults",
	"autotypeDefaultSetting",
	"automaticUserConfirmation",
	"blockClaimedDomainAccountCreation",
	"organizationUserNotification",
	"sendControls",
	"fillAssist",
}

// PolicyTypeName resolves a policy's flexEnum type to its stable string
// name, regardless of whether the API sent an ordinal or a name.
func PolicyTypeName(v flexEnum) string {
	return resolveEnum(v, policyTypeNames)
}

// memberRoleNames translates the bitwarden/server OrganizationUserType enum
// ordinal to the stable lowercase role name the .lr schema documents.
var memberRoleNames = []string{
	"owner",
	"admin",
	"user",
	"manager",
	"custom",
}

// MemberRoleName resolves a member's flexEnum type to its stable role name.
func MemberRoleName(v flexEnum) string {
	return resolveEnum(v, memberRoleNames)
}

// memberStatusNames translates the bitwarden/server OrganizationUserStatusType
// enum ordinal to the stable lowercase status name the .lr schema documents.
// Bitwarden's Revoked status is the numeric value -1; resolveEnum's
// index-by-ordinal lookup can't represent that, so it's handled separately.
var memberStatusNames = []string{
	"invited",
	"accepted",
	"confirmed",
	"staged",
}

// MemberStatusName resolves a member's flexEnum status to its stable status
// name.
func MemberStatusName(v flexEnum) string {
	if v.asInt != nil && *v.asInt == -1 {
		return "revoked"
	}
	return resolveEnum(v, memberStatusNames)
}

// resolveEnum normalizes a flexEnum against a name table (for numeric wire
// values) or a simple lowercase-first-letter transform (for string wire
// values), so translation is correct whichever form the API sent.
func resolveEnum(v flexEnum, names []string) string {
	switch {
	case v.asInt != nil:
		if *v.asInt >= 0 && *v.asInt < len(names) {
			return names[*v.asInt]
		}
		return fmt.Sprintf("unknown(%d)", *v.asInt)
	case v.asString != nil:
		s := *v.asString
		if s == "" {
			return s
		}
		return strings.ToLower(s[:1]) + s[1:]
	default:
		return ""
	}
}
