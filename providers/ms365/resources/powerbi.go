// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/logger"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/ms365/connection"
	"go.mondoo.com/mql/v13/types"
)

const (
	powerbiScope = "https://analysis.windows.net/powerbi/api/.default"
)

// powerbiReport connects to the Power BI service with the tenant credential and
// dumps the admin endpoints we model as a single JSON document. Each section is
// collected independently: the Power BI admin REST endpoints and the Fabric
// admin API have separate authorization requirements, so a permission gap in
// one section (for example tenant settings) must not blank the others. The
// PowerShell module emits a connection banner before any output, so the Go side
// strips everything up to the first '{'.
var powerbiReport = `
$ErrorActionPreference = "Stop"
$pbiToken = '%s'

if (-not (Get-Module -ListAvailable -Name MicrosoftPowerBIMgmt)) {
  Install-Module -Name MicrosoftPowerBIMgmt -Scope CurrentUser -Force
}
Import-Module MicrosoftPowerBIMgmt

Connect-PowerBIServiceAccount -Token $pbiToken | Out-Null

# Tenant settings live on the Fabric admin API, not the Power BI REST API, but
# accept the same Power BI bearer token.
$fabricHeaders = @{ Authorization = "Bearer $pbiToken" }

function Get-PbiSection([scriptblock]$call) {
  try {
    return [PSCustomObject]@{ data = (& $call); error = $null }
  } catch {
    return [PSCustomObject]@{ data = $null; error = $_.Exception.Message }
  }
}

# Get-PbiWorkspaces pages the admin groups endpoint (max 5000 per page) so
# tenants with more than 5000 workspaces are not silently truncated.
function Get-PbiWorkspaces() {
  $all = @()
  $skip = 0
  while ($true) {
    $url = 'admin/groups?$top=5000&$expand=users&$skip=' + $skip
    $page = @((Invoke-PowerBIRestMethod -Url $url -Method Get | ConvertFrom-Json).value)
    if ($page.Count -eq 0) { break }
    $all += $page
    if ($page.Count -lt 5000) { break }
    $skip += 5000
  }
  return $all
}

$report = [PSCustomObject]@{
  TenantSettings            = (Get-PbiSection { (Invoke-RestMethod -Uri 'https://api.fabric.microsoft.com/v1/admin/tenantsettings' -Headers $fabricHeaders -Method Get).tenantSettings })
  Workspaces                = (Get-PbiSection { Get-PbiWorkspaces })
  Capacities                = (Get-PbiSection { (Invoke-PowerBIRestMethod -Url 'admin/capacities' -Method Get | ConvertFrom-Json).value })
  PublishedToWeb            = (Get-PbiSection { (Invoke-PowerBIRestMethod -Url 'admin/widelySharedArtifacts/publishedToWeb' -Method Get | ConvertFrom-Json).ArtifactAccessEntities })
  SharedToWholeOrganization = (Get-PbiSection { (Invoke-PowerBIRestMethod -Url 'admin/widelySharedArtifacts/linksSharedToWholeOrganization' -Method Get | ConvertFrom-Json).ArtifactAccessEntities })
}

Disconnect-PowerBIServiceAccount | Out-Null

ConvertTo-Json -Depth 8 $report
`

// powerBiSection is one report section: its payload as raw JSON plus any error
// PowerShell captured while collecting it. A non-nil error is surfaced by the
// section's getter so callers see the real cause (for example missing admin API
// access) instead of an empty result.
type powerBiSection struct {
	Data  json.RawMessage `json:"data"`
	Error *string         `json:"error"`
}

type powerBiReportRaw struct {
	TenantSettings            powerBiSection `json:"TenantSettings"`
	Workspaces                powerBiSection `json:"Workspaces"`
	Capacities                powerBiSection `json:"Capacities"`
	PublishedToWeb            powerBiSection `json:"PublishedToWeb"`
	SharedToWholeOrganization powerBiSection `json:"SharedToWholeOrganization"`
}

type powerBiTenantSetting struct {
	SettingName              string `json:"settingName"`
	Title                    string `json:"title"`
	Enabled                  bool   `json:"enabled"`
	CanSpecifySecurityGroups bool   `json:"canSpecifySecurityGroups"`
	TenantSettingGroup       string `json:"tenantSettingGroup"`
	EnabledSecurityGroups    []any  `json:"enabledSecurityGroups"`
}

type powerBiWorkspace struct {
	Id                    string          `json:"id"`
	Name                  string          `json:"name"`
	Type                  string          `json:"type"`
	State                 string          `json:"state"`
	IsOnDedicatedCapacity bool            `json:"isOnDedicatedCapacity"`
	IsReadOnly            bool            `json:"isReadOnly"`
	CapacityId            string          `json:"capacityId"`
	Description           string          `json:"description"`
	Users                 []powerBiWsUser `json:"users"`
}

type powerBiWsUser struct {
	DisplayName          string `json:"displayName"`
	EmailAddress         string `json:"emailAddress"`
	Identifier           string `json:"identifier"`
	PrincipalType        string `json:"principalType"`
	GroupUserAccessRight string `json:"groupUserAccessRight"`
	GraphId              string `json:"graphId"`
}

type powerBiCapacity struct {
	Id          string   `json:"id"`
	DisplayName string   `json:"displayName"`
	Sku         string   `json:"sku"`
	State       string   `json:"state"`
	Region      string   `json:"region"`
	Admins      []string `json:"admins"`
}

type powerBiArtifactAccess struct {
	ArtifactId   string `json:"artifactId"`
	DisplayName  string `json:"displayName"`
	ArtifactType string `json:"artifactType"`
	AccessRight  string `json:"accessRight"`
	ShareType    string `json:"shareType"`
	Sharer       struct {
		EmailAddress string `json:"emailAddress"`
		DisplayName  string `json:"displayName"`
	} `json:"sharer"`
}

// unmarshalPowerBiSection returns the section error when present, otherwise
// decodes the payload into a slice. PowerShell's ConvertTo-Json collapses a
// single-element array into a bare object, so both forms are accepted.
func unmarshalPowerBiSection[T any](s powerBiSection) ([]T, error) {
	if s.Error != nil && *s.Error != "" {
		return nil, errors.New(*s.Error)
	}
	trimmed := bytes.TrimSpace(s.Data)
	if len(trimmed) == 0 || string(trimmed) == "null" {
		return nil, nil
	}
	if trimmed[0] == '[' {
		var out []T
		if err := json.Unmarshal(trimmed, &out); err != nil {
			return nil, err
		}
		return out, nil
	}
	var single T
	if err := json.Unmarshal(trimmed, &single); err != nil {
		return nil, err
	}
	return []T{single}, nil
}

type mqlMicrosoftPowerbiInternal struct {
	reportLock sync.Mutex
	fetched    bool
	fetchErr   error
	raw        *powerBiReportRaw
}

// gatherReport runs the PowerShell collection once and caches the raw sections.
// The returned error covers connection and transport failures that affect the
// whole report; per-section authorization errors are carried inside the raw
// sections and surfaced by the individual getters.
func (r *mqlMicrosoftPowerbi) gatherReport() (*powerBiReportRaw, error) {
	r.reportLock.Lock()
	defer r.reportLock.Unlock()

	if r.fetched {
		return r.raw, r.fetchErr
	}
	r.fetched = true

	raw, err := r.fetchReport()
	r.raw = raw
	r.fetchErr = err
	return raw, err
}

func (r *mqlMicrosoftPowerbi) fetchReport() (*powerBiReportRaw, error) {
	conn := r.MqlRuntime.Connection.(*connection.Ms365Connection)
	ctx := context.Background()

	pbiToken, err := conn.Token().GetToken(ctx, policy.TokenRequestOptions{
		Scopes: []string{powerbiScope},
	})
	if err != nil {
		return nil, err
	}

	// The token is interpolated into a single-quoted PowerShell string; double
	// any single quote so a token value can never break out of the quoting.
	escapedToken := strings.ReplaceAll(pbiToken.Token, "'", "''")
	script := fmt.Sprintf(powerbiReport, escapedToken)
	res, err := conn.CheckAndRunPowershellScript(script)
	if err != nil {
		return nil, err
	}

	if res.ExitStatus != 0 {
		data, _ := io.ReadAll(res.Stderr)
		str := strings.ToLower(string(data))
		if strings.Contains(str, "access denied") || strings.Contains(str, "unauthorized") || strings.Contains(str, "powerbinotlicensed") {
			return nil, errors.New("access denied; ensure the service principal is granted read-only admin API access in the Power BI tenant settings")
		}
		return nil, fmt.Errorf("failed to connect to Power BI (exit code %d): %s", res.ExitStatus, string(data))
	}

	data, err := io.ReadAll(res.Stdout)
	if err != nil {
		return nil, err
	}
	str := string(data)
	idx := strings.IndexByte(str, '{')
	if idx == -1 {
		return nil, errors.New("invalid JSON format in Power BI report")
	}
	jsonBytes := []byte(str[idx:])
	logger.DebugDumpJSON("ms-powerbi-report", string(jsonBytes))

	raw := &powerBiReportRaw{}
	if err := json.Unmarshal(jsonBytes, raw); err != nil {
		return nil, err
	}
	return raw, nil
}

func (r *mqlMicrosoftPowerbi) tenantSettings() ([]any, error) {
	raw, err := r.gatherReport()
	if err != nil {
		return nil, err
	}
	settings, err := unmarshalPowerBiSection[powerBiTenantSetting](raw.TenantSettings)
	if err != nil {
		return nil, err
	}

	res := make([]any, 0, len(settings))
	for i := range settings {
		s := settings[i]
		groups, err := convert.JsonToDictSlice(s.EnabledSecurityGroups)
		if err != nil {
			return nil, err
		}
		o, err := CreateResource(r.MqlRuntime, "microsoft.powerbi.tenantSetting", map[string]*llx.RawData{
			"__id":                     llx.StringData(s.SettingName),
			"name":                     llx.StringData(s.SettingName),
			"title":                    llx.StringData(s.Title),
			"enabled":                  llx.BoolData(s.Enabled),
			"canSpecifySecurityGroups": llx.BoolData(s.CanSpecifySecurityGroups),
			"tenantSettingGroup":       llx.StringData(s.TenantSettingGroup),
			"enabledSecurityGroups":    llx.ArrayData(groups, types.Dict),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, o)
	}
	return res, nil
}

func (r *mqlMicrosoftPowerbi) workspaces() ([]any, error) {
	raw, err := r.gatherReport()
	if err != nil {
		return nil, err
	}
	workspaces, err := unmarshalPowerBiSection[powerBiWorkspace](raw.Workspaces)
	if err != nil {
		return nil, err
	}

	res := make([]any, 0, len(workspaces))
	for i := range workspaces {
		ws := workspaces[i]
		o, err := CreateResource(r.MqlRuntime, "microsoft.powerbi.workspace", map[string]*llx.RawData{
			"__id":                  llx.StringData(ws.Id),
			"id":                    llx.StringData(ws.Id),
			"name":                  llx.StringData(ws.Name),
			"type":                  llx.StringData(ws.Type),
			"state":                 llx.StringData(ws.State),
			"isOnDedicatedCapacity": llx.BoolData(ws.IsOnDedicatedCapacity),
			"isReadOnly":            llx.BoolData(ws.IsReadOnly),
			"description":           llx.StringData(ws.Description),
		})
		if err != nil {
			return nil, err
		}
		mqlWs := o.(*mqlMicrosoftPowerbiWorkspace)
		mqlWs.cacheCapacityId = ws.CapacityId
		mqlWs.cacheUsers = ws.Users
		res = append(res, o)
	}
	return res, nil
}

func (r *mqlMicrosoftPowerbi) capacities() ([]any, error) {
	raw, err := r.gatherReport()
	if err != nil {
		return nil, err
	}
	capacities, err := unmarshalPowerBiSection[powerBiCapacity](raw.Capacities)
	if err != nil {
		return nil, err
	}

	res := make([]any, 0, len(capacities))
	for i := range capacities {
		c := capacities[i]
		admins := make([]any, 0, len(c.Admins))
		for _, a := range c.Admins {
			admins = append(admins, a)
		}
		o, err := CreateResource(r.MqlRuntime, "microsoft.powerbi.capacity", map[string]*llx.RawData{
			"__id":        llx.StringData(c.Id),
			"id":          llx.StringData(c.Id),
			"displayName": llx.StringData(c.DisplayName),
			"sku":         llx.StringData(c.Sku),
			"state":       llx.StringData(c.State),
			"region":      llx.StringData(c.Region),
			"admins":      llx.ArrayData(admins, types.String),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, o)
	}
	return res, nil
}

func (r *mqlMicrosoftPowerbi) publishedToWeb() ([]any, error) {
	raw, err := r.gatherReport()
	if err != nil {
		return nil, err
	}
	entries, err := unmarshalPowerBiSection[powerBiArtifactAccess](raw.PublishedToWeb)
	if err != nil {
		return nil, err
	}
	return r.createArtifactAccess(entries, "publishedToWeb")
}

func (r *mqlMicrosoftPowerbi) sharedToWholeOrganization() ([]any, error) {
	raw, err := r.gatherReport()
	if err != nil {
		return nil, err
	}
	entries, err := unmarshalPowerBiSection[powerBiArtifactAccess](raw.SharedToWholeOrganization)
	if err != nil {
		return nil, err
	}
	return r.createArtifactAccess(entries, "sharedToWholeOrganization")
}

func (r *mqlMicrosoftPowerbi) createArtifactAccess(entries []powerBiArtifactAccess, shareKind string) ([]any, error) {
	res := make([]any, 0, len(entries))
	for i := range entries {
		a := entries[i]
		o, err := CreateResource(r.MqlRuntime, "microsoft.powerbi.artifactAccess", map[string]*llx.RawData{
			"__id":               llx.StringData(shareKind + "/" + a.ArtifactId + "/" + a.ShareType),
			"artifactId":         llx.StringData(a.ArtifactId),
			"displayName":        llx.StringData(a.DisplayName),
			"artifactType":       llx.StringData(a.ArtifactType),
			"accessRight":        llx.StringData(a.AccessRight),
			"shareType":          llx.StringData(a.ShareType),
			"sharerEmailAddress": llx.StringData(a.Sharer.EmailAddress),
			"sharerDisplayName":  llx.StringData(a.Sharer.DisplayName),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, o)
	}
	return res, nil
}

type mqlMicrosoftPowerbiWorkspaceInternal struct {
	cacheCapacityId string
	cacheUsers      []powerBiWsUser
}

func (w *mqlMicrosoftPowerbiWorkspace) capacity() (*mqlMicrosoftPowerbiCapacity, error) {
	if w.cacheCapacityId == "" {
		w.Capacity.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}

	o, err := CreateResource(w.MqlRuntime, "microsoft.powerbi", map[string]*llx.RawData{})
	if err != nil {
		return nil, err
	}
	caps := o.(*mqlMicrosoftPowerbi).GetCapacities()
	if caps.Error != nil {
		return nil, caps.Error
	}
	for _, c := range caps.Data {
		mqlCap := c.(*mqlMicrosoftPowerbiCapacity)
		if mqlCap.Id.Data == w.cacheCapacityId {
			return mqlCap, nil
		}
	}

	w.Capacity.State = plugin.StateIsSet | plugin.StateIsNull
	return nil, nil
}

func (w *mqlMicrosoftPowerbiWorkspace) users() ([]any, error) {
	res := make([]any, 0, len(w.cacheUsers))
	for i := range w.cacheUsers {
		u := w.cacheUsers[i]
		ident := u.Identifier
		if ident == "" {
			ident = u.GraphId
		}
		if ident == "" {
			ident = u.EmailAddress
		}
		o, err := CreateResource(w.MqlRuntime, "microsoft.powerbi.workspace.user", map[string]*llx.RawData{
			"__id":          llx.StringData(w.Id.Data + "/" + ident),
			"displayName":   llx.StringData(u.DisplayName),
			"emailAddress":  llx.StringData(u.EmailAddress),
			"identifier":    llx.StringData(u.Identifier),
			"principalType": llx.StringData(u.PrincipalType),
			"accessRight":   llx.StringData(u.GroupUserAccessRight),
			"graphId":       llx.StringData(u.GraphId),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, o)
	}
	return res, nil
}
