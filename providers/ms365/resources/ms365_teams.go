// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"go.mondoo.com/cnquery/v10/llx"
	"go.mondoo.com/cnquery/v10/logger"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v10/providers/ms365/connection"
)

const (
	teamsScope = "48ac35b8-9aa8-4d74-927d-1f4a14a0b239/.default"
)

var teamsReport = `
$ErrorActionPreference = "Stop"
$graphToken= '%s'
$teamsToken = '%s'

Install-Module -Name MicrosoftTeams -Scope CurrentUser -Force
Import-Module MicrosoftTeams
Connect-MicrosoftTeams -AccessTokens @("$graphToken", "$teamsToken")

$CsTeamsClientConfiguration = (Get-CsTeamsClientConfiguration)
$CsTenantFederationConfiguration = (Get-CsTenantFederationConfiguration)
$CsTeamsMeetingPolicy = (Get-CsTeamsMeetingPolicy -Identity Global)

$msteams = New-Object PSObject
Add-Member -InputObject $msteams -MemberType NoteProperty -Name CsTeamsClientConfiguration -Value $CsTeamsClientConfiguration
Add-Member -InputObject $msteams -MemberType NoteProperty -Name CsTenantFederationConfiguration -Value $CsTenantFederationConfiguration
Add-Member -InputObject $msteams -MemberType NoteProperty -Name CsTeamsMeetingPolicy -Value $CsTeamsMeetingPolicy

Disconnect-MicrosoftTeams -Confirm:$false
ConvertTo-Json -Depth 4 $msteams
`

type MsTeamsReport struct {
	CsTeamsClientConfiguration      interface{}                      `json:"CsTeamsClientConfiguration"`
	CsTenantFederationConfiguration *CsTenantFederationConfiguration `json:"CsTenantFederationConfiguration"`
	CsTeamsMeetingPolicy            *CsTeamsMeetingPolicy            `json:"CsTeamsMeetingPolicy"`
}

type CsTenantFederationConfiguration struct {
	Identity                                    string `json:"Identity"`
	AllowFederatedUsers                         bool   `json:"AllowFederatedUsers"`
	AllowPublicUsers                            bool   `json:"AllowPublicUsers"`
	AllowTeamsConsumer                          bool   `json:"AllowTeamsConsumer"`
	AllowTeamsConsumerInbound                   bool   `json:"AllowTeamsConsumerInbound"`
	TreatDiscoveredPartnersAsUnverified         bool   `json:"TreatDiscoveredPartnersAsUnverified"`
	SharedSipAddressSpace                       bool   `json:"SharedSipAddressSpace"`
	RestrictTeamsConsumerToExternalUserProfiles bool   `json:"RestrictTeamsConsumerToExternalUserProfiles"`
	// TODO: we need to figure out how to get this right when using Convert-ToJson
	// it currently comes back as an empty json object {} but the pwsh cmdlet spits out a string-looking value
	AllowedDomains interface{} `json:"AllowedDomains"`
	BlockedDomains interface{} `json:"BlockedDomains"`
}

type CsTeamsMeetingPolicy struct {
	AllowAnonymousUsersToJoinMeeting           bool   `json:"AllowAnonymousUsersToJoinMeeting"`
	AllowAnonymousUsersToStartMeeting          bool   `json:"AllowAnonymousUsersToStartMeeting"`
	AutoAdmittedUsers                          string `json:"AutoAdmittedUsers"`
	AllowPSTNUsersToBypassLobby                bool   `json:"AllowPSTNUsersToBypassLobby"`
	MeetingChatEnabledType                     string `json:"MeetingChatEnabledType"`
	DesignatedPresenterRoleMode                string `json:"DesignatedPresenterRoleMode"`
	AllowExternalParticipantGiveRequestControl bool   `json:"AllowExternalParticipantGiveRequestControl"`
	AllowSecurityEndUserReporting              bool   `json:"AllowSecurityEndUserReporting"`
}

type mqlMs365TeamsInternal struct {
	teamsReport     *MsTeamsReport
	teamsReportLock sync.Mutex
}

func (r *mqlMs365Teams) GetTeamsReport(ctx context.Context) (*MsTeamsReport, error) {
	fmt.Println("GetTeamsReport")
	conn := r.MqlRuntime.Connection.(*connection.Ms365Connection)
	r.teamsReportLock.Lock()
	defer r.teamsReportLock.Unlock()
	if r.teamsReport != nil {
		return r.teamsReport, nil
	}
	token := conn.Token()
	teamsToken, err := token.GetToken(ctx, policy.TokenRequestOptions{
		Scopes: []string{teamsScope},
	})
	if err != nil {
		return nil, err
	}
	graphToken, err := token.GetToken(ctx, policy.TokenRequestOptions{
		Scopes: []string{connection.DefaultMSGraphScope},
	})
	if err != nil {
		return nil, err
	}

	fmtScript := fmt.Sprintf(teamsReport, graphToken.Token, teamsToken.Token)
	res, err := conn.CheckAndRunPowershellScript(fmtScript)
	if err != nil {
		return nil, err
	}
	report := &MsTeamsReport{}
	if res.ExitStatus == 0 {
		data, err := io.ReadAll(res.Stdout)
		if err != nil {
			return nil, err
		}
		str := string(data)
		// The Connect-MicrosoftTeams also displays a header for which there
		// are no params to hide it. To allow the JSON unmarshal to work
		// we strip away everything until the first '{' character.
		idx := strings.IndexByte(str, '{')
		after := str[idx:]
		newData := []byte(after)

		logger.DebugDumpJSON("ms-teams-report", string(newData))
		err = json.Unmarshal(newData, report)
		if err != nil {
			return nil, err
		}
	} else {
		data, err := io.ReadAll(res.Stderr)
		if err != nil {
			return nil, err
		}

		logger.DebugDumpJSON("ms-teams-report", string(data))
		return nil, fmt.Errorf("failed to generate ms teams report (exit code %d): %s", res.ExitStatus, string(data))
	}

	r.teamsReport = report
	return report, nil
}

func (r *mqlMs365Teams) csTeamsClientConfiguration() (interface{}, error) {
	ctx := context.Background()
	report, err := r.GetTeamsReport(ctx)
	if err != nil {
		return nil, err
	}
	return convert.JsonToDict(report.CsTeamsClientConfiguration)
}

func (r *mqlMs365Teams) csTenantFederationConfiguration() (*mqlMs365TeamsTenantFederationConfig, error) {
	ctx := context.Background()
	report, err := r.GetTeamsReport(ctx)
	if err != nil {
		return nil, err
	}

	tenantConfig := report.CsTenantFederationConfiguration
	tenantConfigBlockedDomains, _ := convert.JsonToDict(tenantConfig.BlockedDomains)
	mqlTenantConfig, err := CreateResource(r.MqlRuntime, "ms365.teams.tenantFederationConfig",
		map[string]*llx.RawData{
			"identity":                                    llx.StringData(tenantConfig.Identity),
			"blockedDomains":                              llx.DictData(tenantConfigBlockedDomains),
			"allowFederatedUsers":                         llx.BoolData(tenantConfig.AllowFederatedUsers),
			"allowPublicUsers":                            llx.BoolData(tenantConfig.AllowPublicUsers),
			"allowTeamsConsumer":                          llx.BoolData(tenantConfig.AllowTeamsConsumer),
			"allowTeamsConsumerInbound":                   llx.BoolData(tenantConfig.AllowTeamsConsumerInbound),
			"treatDiscoveredPartnersAsUnverified":         llx.BoolData(tenantConfig.TreatDiscoveredPartnersAsUnverified),
			"sharedSipAddressSpace":                       llx.BoolData(tenantConfig.SharedSipAddressSpace),
			"restrictTeamsConsumerToExternalUserProfiles": llx.BoolData(tenantConfig.RestrictTeamsConsumerToExternalUserProfiles),
		})
	if err != nil {
		return nil, err
	}
	return mqlTenantConfig.(*mqlMs365TeamsTenantFederationConfig), nil
}

func (r *mqlMs365Teams) csTeamsMeetingPolicy() (*mqlMs365TeamsTeamsMeetingPolicyConfig, error) {
	ctx := context.Background()
	report, err := r.GetTeamsReport(ctx)
	if err != nil {
		return nil, err
	}

	teamsPolicy := report.CsTeamsMeetingPolicy
	mqlTeamsPolicy, err := CreateResource(r.MqlRuntime, "ms365.teams.teamsMeetingPolicyConfig",
		map[string]*llx.RawData{
			"allowAnonymousUsersToJoinMeeting":           llx.BoolData(teamsPolicy.AllowAnonymousUsersToJoinMeeting),
			"allowAnonymousUsersToStartMeeting":          llx.BoolData(teamsPolicy.AllowAnonymousUsersToStartMeeting),
			"autoAdmittedUsers":                          llx.StringData(teamsPolicy.AutoAdmittedUsers),
			"allowPSTNUsersToBypassLobby":                llx.BoolData(teamsPolicy.AllowPSTNUsersToBypassLobby),
			"meetingChatEnabledType":                     llx.StringData(teamsPolicy.MeetingChatEnabledType),
			"designatedPresenterRoleMode":                llx.StringData(teamsPolicy.DesignatedPresenterRoleMode),
			"allowExternalParticipantGiveRequestControl": llx.BoolData(teamsPolicy.AllowExternalParticipantGiveRequestControl),
			"allowSecurityEndUserReporting":              llx.BoolData(teamsPolicy.AllowSecurityEndUserReporting),
		})
	if err != nil {
		return nil, err
	}

	return mqlTeamsPolicy.(*mqlMs365TeamsTeamsMeetingPolicyConfig), nil
}
