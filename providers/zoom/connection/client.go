// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/cockroachdb/errors"
)

// defaultBaseURL is the Zoom REST API v2 base URL.
const defaultBaseURL = "https://api.zoom.us/v2"

// Client is a minimal, hand-written typed HTTP client covering exactly the
// read endpoints the zoom.lr schema needs: account settings, users, roles,
// and groups. It is a pragmatic first implementation; see README.md for the
// production intent of a client generated from Zoom's published OpenAPI
// spec (ADR-033). Field names on the response types below are modeled after
// Zoom's documented account/user/role/group settings and may need
// adjustment once verified against (or regenerated from) that spec.
type Client struct {
	httpClient *http.Client
	baseURL    string
}

// newClient wraps an already-authenticated *http.Client (typically produced
// by clientcredentials.Config.Client) for calls against the Zoom API.
func newClient(httpClient *http.Client) *Client {
	return &Client{
		httpClient: httpClient,
		baseURL:    defaultBaseURL,
	}
}

// get issues an authenticated GET request against the Zoom API and decodes
// a JSON response into out. A nil out discards the response body after
// checking the status code.
func (c *Client) get(ctx context.Context, path string, query url.Values, out any) error {
	u := c.baseURL + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return errors.Wrap(err, "failed to build zoom API request")
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return errors.Wrapf(err, "zoom API request to %s failed", path)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return errors.Wrapf(err, "failed to read zoom API response from %s", path)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return errors.Newf("zoom API request to %s failed with status %d: %s", path, resp.StatusCode, string(body))
	}

	if out == nil || len(body) == 0 {
		return nil
	}
	if err := json.Unmarshal(body, out); err != nil {
		return errors.Wrapf(err, "failed to decode zoom API response from %s", path)
	}
	return nil
}

// ---- Users ----

// User is the subset of Zoom's user object this provider reads.
type User struct {
	ID          string `json:"id"`
	Email       string `json:"email"`
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
	DisplayName string `json:"display_name"`
	Type        int64  `json:"type"`
	Status      string `json:"status"`
	// Verified is 0 or 1 (whether the user's email address is verified).
	Verified      int        `json:"verified"`
	LoginType     int64      `json:"login_type"`
	RoleID        string     `json:"role_id"`
	GroupIDs      []string   `json:"group_ids"`
	LastLoginTime *time.Time `json:"last_login_time"`
	CreatedAt     *time.Time `json:"created_at"`
}

// UsersListResponse is the paginated response of the List Users endpoint.
type UsersListResponse struct {
	PageSize      int    `json:"page_size"`
	TotalRecords  int    `json:"total_records"`
	NextPageToken string `json:"next_page_token"`
	Users         []User `json:"users"`
}

// ListUsers returns one page of provisioned users. Pass an empty
// nextPageToken to fetch the first page.
func (c *Client) ListUsers(ctx context.Context, pageSize int, nextPageToken string) (*UsersListResponse, error) {
	q := url.Values{}
	q.Set("status", "active")
	q.Set("page_size", strconv.Itoa(pageSize))
	if nextPageToken != "" {
		q.Set("next_page_token", nextPageToken)
	}

	var out UsersListResponse
	if err := c.get(ctx, "/users", q, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetUser fetches a single user by ID or email.
func (c *Client) GetUser(ctx context.Context, userID string) (*User, error) {
	var out User
	if err := c.get(ctx, "/users/"+url.PathEscape(userID), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetMe fetches the user identified by the credentials used to connect,
// the cheapest authenticated read available to verify a connection.
func (c *Client) GetMe(ctx context.Context) (*User, error) {
	return c.GetUser(ctx, "me")
}

// ---- Account ----

// AccountInfo is the account identity returned by Get Account.
type AccountInfo struct {
	ID          string `json:"id"`
	AccountName string `json:"account_name"`
	OwnerEmail  string `json:"owner_email"`
}

// GetAccount fetches the account's identity (name, owner email).
func (c *Client) GetAccount(ctx context.Context, accountID string) (*AccountInfo, error) {
	var out AccountInfo
	if err := c.get(ctx, "/accounts/"+url.PathEscape(accountID), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// AccountSettings is the subset of Get Account Settings this provider reads:
// meeting-security defaults, cloud-recording encryption, and the sign-in
// session timeout.
type AccountSettings struct {
	MeetingSecurity struct {
		WaitingRoom                bool   `json:"waiting_room"`
		MeetingPasswordRequirement bool   `json:"meeting_password"`
		PmiPasswordRequirement     bool   `json:"pmi_password"`
		EncryptionType             string `json:"encryption_type"`
		E2eeAvailable              bool   `json:"end_to_end_encrypted_meetings"`
		MeetingAuthentication      bool   `json:"meeting_authentication"`
		OnlyAuthenticatedCanJoin   bool   `json:"only_authenticated_can_join"`
	} `json:"meeting_security"`
	Recording struct {
		CloudRecording           bool `json:"cloud_recording"`
		CloudRecordingEncryption bool `json:"cloud_recording_encryption"`
	} `json:"recording"`
	Security struct {
		SignInSessionTimeout int64 `json:"session_timeout"`
	} `json:"security"`
}

// GetAccountSettings fetches the account-wide meeting, recording, and
// sign-in security settings in a single call.
func (c *Client) GetAccountSettings(ctx context.Context, accountID string) (*AccountSettings, error) {
	var out AccountSettings
	if err := c.get(ctx, "/accounts/"+url.PathEscape(accountID)+"/settings", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// SsoSettings is the account's single sign-on configuration.
type SsoSettings struct {
	Enabled             bool     `json:"sso_enabled"`
	Domains             []string `json:"domains"`
	GroupMappingEnabled bool     `json:"group_mapping_enabled"`
	IdpIssuer           string   `json:"idp_issuer"`
	IdpSsoUrl           string   `json:"idp_sso_url"`
}

// GetSsoSettings fetches the account's SSO configuration.
func (c *Client) GetSsoSettings(ctx context.Context, accountID string) (*SsoSettings, error) {
	var out SsoSettings
	if err := c.get(ctx, "/accounts/"+url.PathEscape(accountID)+"/settings", url.Values{"option": {"sso"}}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ---- Roles ----

// Role is a Zoom role definition.
type Role struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	TotalMembers int64    `json:"total_members"`
	Privileges   []string `json:"privileges"`
}

// RolesListResponse is the response of the List Roles endpoint. Zoom does
// not paginate roles: an account's role set is small and returned in full.
type RolesListResponse struct {
	TotalRecords int    `json:"total_records"`
	Roles        []Role `json:"roles"`
}

// ListRoles returns every role defined on the account.
func (c *Client) ListRoles(ctx context.Context) (*RolesListResponse, error) {
	var out RolesListResponse
	if err := c.get(ctx, "/roles", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetRole fetches a single role by ID.
func (c *Client) GetRole(ctx context.Context, roleID string) (*Role, error) {
	var out Role
	if err := c.get(ctx, "/roles/"+url.PathEscape(roleID), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// RoleMember is a member entry returned by List Role Members.
type RoleMember struct {
	ID string `json:"id"`
}

// RoleMembersResponse is the paginated response of List Role Members.
type RoleMembersResponse struct {
	PageSize      int          `json:"page_size"`
	TotalRecords  int          `json:"total_records"`
	NextPageToken string       `json:"next_page_token"`
	Members       []RoleMember `json:"members"`
}

// ListRoleMembers returns one page of users assigned the given role.
func (c *Client) ListRoleMembers(ctx context.Context, roleID string, pageSize int, nextPageToken string) (*RoleMembersResponse, error) {
	q := url.Values{}
	q.Set("page_size", strconv.Itoa(pageSize))
	if nextPageToken != "" {
		q.Set("next_page_token", nextPageToken)
	}

	var out RoleMembersResponse
	if err := c.get(ctx, "/roles/"+url.PathEscape(roleID)+"/members", q, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ---- Groups ----

// Group is a Zoom user group.
type Group struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	TotalMembers int64  `json:"total_members"`
}

// GroupsListResponse is the response of the List Groups endpoint.
type GroupsListResponse struct {
	TotalRecords int     `json:"total_records"`
	Groups       []Group `json:"groups"`
}

// ListGroups returns every group defined on the account.
func (c *Client) ListGroups(ctx context.Context) (*GroupsListResponse, error) {
	var out GroupsListResponse
	if err := c.get(ctx, "/groups", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetGroup fetches a single group by ID.
func (c *Client) GetGroup(ctx context.Context, groupID string) (*Group, error) {
	var out Group
	if err := c.get(ctx, "/groups/"+url.PathEscape(groupID), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GroupSettings is the meeting-security overrides configured on a group,
// which take precedence over the account defaults for the group's members.
type GroupSettings struct {
	MeetingSecurity struct {
		WaitingRoom              bool `json:"waiting_room"`
		MeetingPasswordRequired  bool `json:"meeting_password"`
		E2eeAvailable            bool `json:"end_to_end_encrypted_meetings"`
		OnlyAuthenticatedCanJoin bool `json:"only_authenticated_can_join"`
	} `json:"meeting_security"`
}

// GetGroupSettings fetches the meeting-security overrides for a group.
func (c *Client) GetGroupSettings(ctx context.Context, groupID string) (*GroupSettings, error) {
	var out GroupSettings
	if err := c.get(ctx, "/groups/"+url.PathEscape(groupID)+"/settings", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GroupMember is a member entry returned by List Group Members.
type GroupMember struct {
	ID string `json:"id"`
}

// GroupMembersResponse is the paginated response of List Group Members.
type GroupMembersResponse struct {
	PageSize      int           `json:"page_size"`
	TotalRecords  int           `json:"total_records"`
	NextPageToken string        `json:"next_page_token"`
	Members       []GroupMember `json:"members"`
}

// ListGroupMembers returns one page of users belonging to the given group.
func (c *Client) ListGroupMembers(ctx context.Context, groupID string, pageSize int, nextPageToken string) (*GroupMembersResponse, error) {
	q := url.Values{}
	q.Set("page_size", strconv.Itoa(pageSize))
	if nextPageToken != "" {
		q.Set("next_page_token", nextPageToken)
	}

	var out GroupMembersResponse
	if err := c.get(ctx, "/groups/"+url.PathEscape(groupID)+"/members", q, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
