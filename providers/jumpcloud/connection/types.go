// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import "time"

// SystemUser is a user account as returned by the v1 /systemusers endpoint.
// JumpCloud historically keyed v1 objects on `_id` and newer responses also
// carry `id`; EffectiveID prefers whichever is populated.
type SystemUser struct {
	ID               string   `json:"id"`
	XID              string   `json:"_id"`
	Username         string   `json:"username"`
	Email            string   `json:"email"`
	Firstname        string   `json:"firstname"`
	Lastname         string   `json:"lastname"`
	Displayname      string   `json:"displayname"`
	Activated        bool     `json:"activated"`
	Suspended        bool     `json:"suspended"`
	AccountLocked    bool     `json:"account_locked"`
	PasswordExpired  bool     `json:"password_expired"`
	PasswordlessSudo bool     `json:"passwordless_sudo"`
	Sudo             bool     `json:"sudo"`
	LdapBindingUser  bool     `json:"ldap_binding_user"`
	EnableManagedUID bool     `json:"enable_managed_uid"`
	TotpEnabled      bool     `json:"totp_enabled"`
	State            string   `json:"state"`
	Created          string   `json:"created"`
	MFA              *userMFA `json:"mfa"`
}

// userMFA is the nested multi-factor configuration on a user.
type userMFA struct {
	Configured bool `json:"configured"`
	Exclusion  bool `json:"exclusion"`
}

// EffectiveID returns the user's identifier, preferring `id` and falling back
// to the legacy `_id`.
func (u *SystemUser) EffectiveID() string {
	if u == nil {
		return ""
	}
	if u.ID != "" {
		return u.ID
	}
	return u.XID
}

// System is an enrolled device as returned by the v1 /systems endpoint.
type System struct {
	ID                             string       `json:"id"`
	XID                            string       `json:"_id"`
	Hostname                       string       `json:"hostname"`
	DisplayName                    string       `json:"displayName"`
	OS                             string       `json:"os"`
	Version                        string       `json:"version"`
	AgentVersion                   string       `json:"agentVersion"`
	Arch                           string       `json:"arch"`
	Active                         bool         `json:"active"`
	AllowSshRootLogin              bool         `json:"allowSshRootLogin"`
	AllowSshPasswordAuthentication bool         `json:"allowSshPasswordAuthentication"`
	AllowMultiFactorAuthentication bool         `json:"allowMultiFactorAuthentication"`
	AllowPublicKeyAuthentication   bool         `json:"allowPublicKeyAuthentication"`
	FDE                            *systemFDE   `json:"fde"`
	SystemInsights                 *systemState `json:"systemInsights"`
	HasServiceAccount              bool         `json:"hasServiceAccount"`
	RemoteIP                       string       `json:"remoteIP"`
	LastContact                    string       `json:"lastContact"`
	Created                        string       `json:"created"`
}

// systemFDE holds a system's full-disk-encryption status.
type systemFDE struct {
	Active bool `json:"active"`
}

// systemState holds a nested feature-state object (for example System Insights).
type systemState struct {
	State string `json:"state"`
}

// EffectiveID returns the system's identifier, preferring `id` over `_id`.
func (s *System) EffectiveID() string {
	if s == nil {
		return ""
	}
	if s.ID != "" {
		return s.ID
	}
	return s.XID
}

// FdeActive reports whether full-disk encryption is active on the system.
func (s *System) FdeActive() bool {
	return s != nil && s.FDE != nil && s.FDE.Active
}

// InsightsEnabled reports whether System Insights is enabled on the system.
func (s *System) InsightsEnabled() bool {
	return s != nil && s.SystemInsights != nil && s.SystemInsights.State == "enabled"
}

// Group is a user group or system group as returned by the v2 group endpoints.
type Group struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Email       string `json:"email"`
	Type        string `json:"type"`
}

// Application is an SSO application as returned by the v2 /applications endpoint.
type Application struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	SsoURL      string `json:"ssoUrl"`
	Active      bool   `json:"active"`
}

// Policy is a configuration or security policy as returned by the v2 /policies
// endpoint.
type Policy struct {
	ID       string          `json:"id"`
	Name     string          `json:"name"`
	Active   bool            `json:"active"`
	Template *policyTemplate `json:"template"`
}

// policyTemplate names the template a policy is based on.
type policyTemplate struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
}

// TemplateName returns the display name of the policy's template, falling back
// to its internal name.
func (p *Policy) TemplateName() string {
	if p == nil || p.Template == nil {
		return ""
	}
	if p.Template.DisplayName != "" {
		return p.Template.DisplayName
	}
	return p.Template.Name
}

// Command is a command definition as returned by the v1 /commands endpoint.
type Command struct {
	ID          string `json:"id"`
	XID         string `json:"_id"`
	Name        string `json:"name"`
	CommandType string `json:"commandType"`
	LaunchType  string `json:"launchType"`
	Sudo        bool   `json:"sudo"`
	Timeout     string `json:"timeout"`
}

// EffectiveID returns the command's identifier, preferring `id` over `_id`.
func (c *Command) EffectiveID() string {
	if c == nil {
		return ""
	}
	if c.ID != "" {
		return c.ID
	}
	return c.XID
}

// RadiusServer is a RADIUS server configuration as returned by the v2
// /radiusservers endpoint.
type RadiusServer struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	NetworkSourceIP string `json:"networkSourceIp"`
	MFA             string `json:"mfa"`
}

// Directory is an external directory integration as returned by the v2
// /directories endpoint.
type Directory struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

// Organization is an organization as returned by the v1 /organizations
// endpoint.
type Organization struct {
	ID          string `json:"id"`
	XID         string `json:"_id"`
	DisplayName string `json:"displayName"`
	Name        string `json:"name"`
}

// EffectiveID returns the organization's identifier, preferring `id` over `_id`.
func (o *Organization) EffectiveID() string {
	if o == nil {
		return ""
	}
	if o.ID != "" {
		return o.ID
	}
	return o.XID
}

// GraphConnection is a single entry from a JumpCloud v2 graph (membership or
// association) endpoint. The related object is named by GraphConnection.To.
type GraphConnection struct {
	To graphTarget `json:"to"`
}

// graphTarget names the object on the far side of a graph connection.
type graphTarget struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

// GraphTargetIDs extracts the target ids from a set of graph connections. When
// filterType is non-empty, only targets of that type are returned; a target
// with an empty id is always skipped. The result preserves input order and
// contains no duplicates, so callers can resolve each id exactly once.
func GraphTargetIDs(conns []*GraphConnection, filterType string) []string {
	seen := make(map[string]struct{}, len(conns))
	out := make([]string, 0, len(conns))
	for _, c := range conns {
		if c == nil {
			continue
		}
		if filterType != "" && c.To.Type != filterType {
			continue
		}
		id := c.To.ID
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

// UserMFAConfigured reports whether the account has any multi-factor method set
// up, treating an enrolled TOTP factor and a configured MFA object as
// equivalent signals.
func UserMFAConfigured(u *SystemUser) bool {
	if u == nil {
		return false
	}
	if u.TotpEnabled {
		return true
	}
	return u.MFA != nil && u.MFA.Configured
}

// ParseTime parses a JumpCloud RFC3339 timestamp, returning nil for an empty or
// unparseable value so an absent time surfaces as null rather than a zero date.
func ParseTime(s string) *time.Time {
	if s == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return nil
	}
	return &t
}
