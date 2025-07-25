// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/logger"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v11/providers/ms365/connection"
	"go.mondoo.com/cnquery/v11/types"
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

# Fetch all the necessary policies
$CsTeamsClientConfiguration = (Get-CsTeamsClientConfiguration)
$CsTenantFederationConfiguration = (Get-CsTenantFederationConfiguration)
$CsTeamsMeetingPolicy = (Get-CsTeamsMeetingPolicy -Identity Global)
$CsTeamsMessagingPolicy = (Get-CsTeamsMessagingPolicy -Identity Global)
# Fetch the calling policy to get the cloud recording setting
$callingPolicy = (Get-CsTeamsCallingPolicy -Identity Global)
$CsTeamsMeetingPolicy | Add-Member -NotePropertyName "AllowCloudRecordingForCalls" -NotePropertyValue $callingPolicy.AllowCloudRecordingForCalls

$allowedList = New-Object System.Collections.Generic.List[string]
if ($null -ne $CsTenantFederationConfiguration.AllowedDomains) {
  foreach ($item in $CsTenantFederationConfiguration.AllowedDomains) {
    $itemAsString = $item.ToString()
    $domainValue = ($itemAsString -split '=')[1]
    $allowedList.Add($domainValue)
  }
}
$CsTenantFederationConfiguration.AllowedDomains = $allowedList

$CsTenantFederationConfiguration | Add-Member -MemberType NoteProperty -Name "AllowedDomainsClean" -Value $allowedList

$msteams = New-Object PSObject
Add-Member -InputObject $msteams -MemberType NoteProperty -Name CsTeamsClientConfiguration -Value $CsTeamsClientConfiguration
Add-Member -InputObject $msteams -MemberType NoteProperty -Name CsTenantFederationConfiguration -Value $CsTenantFederationConfiguration
Add-Member -InputObject $msteams -MemberType NoteProperty -Name CsTeamsMeetingPolicy -Value $CsTeamsMeetingPolicy
Add-Member -InputObject $msteams -MemberType NoteProperty -Name CsTeamsMessagingPolicy -Value $CsTeamsMessagingPolicy

Disconnect-MicrosoftTeams -Confirm:$false
ConvertTo-Json -Depth 4 $msteams
`

type MsTeamsReport struct {
	CsTeamsClientConfiguration      interface{}                      `json:"CsTeamsClientConfiguration"`
	CsTenantFederationConfiguration *CsTenantFederationConfiguration `json:"CsTenantFederationConfiguration"`
	CsTeamsMeetingPolicy            *CsTeamsMeetingPolicy            `json:"CsTeamsMeetingPolicy"`
	CsTeamsMessagingPolicy          *CsTeamsMessagingPolicy          `json:"CsTeamsMessagingPolicy"`
}

type CsTenantFederationConfiguration struct {
	Identity                                    string   `json:"Identity"`
	AllowFederatedUsers                         bool     `json:"AllowFederatedUsers"`
	AllowPublicUsers                            bool     `json:"AllowPublicUsers"`
	AllowTeamsConsumer                          bool     `json:"AllowTeamsConsumer"`
	AllowTeamsConsumerInbound                   bool     `json:"AllowTeamsConsumerInbound"`
	TreatDiscoveredPartnersAsUnverified         bool     `json:"TreatDiscoveredPartnersAsUnverified"`
	SharedSipAddressSpace                       bool     `json:"SharedSipAddressSpace"`
	RestrictTeamsConsumerToExternalUserProfiles bool     `json:"RestrictTeamsConsumerToExternalUserProfiles"`
	AllowedDomains                              []string `json:"AllowedDomains"`
	// TODO: we need to figure out how to get this right when using Convert-ToJson
	// it currently comes back as an empty json object {} but the pwsh cmdlet spits out a string-looking value
	BlockedDomains interface{} `json:"BlockedDomains"`
}

type CsTeamsMeetingPolicy struct {
	AllowAnonymousUsersToJoinMeeting           bool   `json:"AllowAnonymousUsersToJoinMeeting"`
	AllowAnonymousUsersToStartMeeting          bool   `json:"AllowAnonymousUsersToStartMeeting"`
	AutoAdmittedUsers                          string `json:"AutoAdmittedUsers"`
	AllowPSTNUsersToBypassLobby                bool   `json:"AllowPSTNUsersToBypassLobby"`
	AllowExternalNonTrustedMeetingChat         bool   `json:"AllowExternalNonTrustedMeetingChat"`
	MeetingChatEnabledType                     string `json:"MeetingChatEnabledType"`
	DesignatedPresenterRoleMode                string `json:"DesignatedPresenterRoleMode"`
	AllowExternalParticipantGiveRequestControl bool   `json:"AllowExternalParticipantGiveRequestControl"`
	AllowSecurityEndUserReporting              bool   `json:"AllowSecurityEndUserReporting"`
	AllowCloudRecordingForCalls                bool   `json:"AllowCloudRecordingForCalls"`
}

type CsTeamsMessagingPolicy struct {
	AllowSecurityEndUserReporting bool `json:"AllowSecurityEndUserReporting"`
}

type mqlMs365TeamsInternal struct {
	teamsReportLock sync.Mutex
	fetched         bool
	fetchErr        error
}

func (r *mqlMs365Teams) gatherTeamsReport() error {
	conn := r.MqlRuntime.Connection.(*connection.Ms365Connection)

	r.teamsReportLock.Lock()
	defer r.teamsReportLock.Unlock()

	errHandler := func(err error) error {
		r.fetchErr = err
		return err
	}

	ctx := context.Background()
	token := conn.Token()
	teamsToken, err := token.GetToken(ctx, policy.TokenRequestOptions{
		Scopes: []string{teamsScope},
	})
	if err != nil {
		return errHandler(err)
	}
	graphToken, err := token.GetToken(ctx, policy.TokenRequestOptions{
		Scopes: []string{connection.DefaultMSGraphScope},
	})
	if err != nil {
		return errHandler(err)
	}

	fmtScript := fmt.Sprintf(teamsReport, graphToken.Token, teamsToken.Token)
	res, err := conn.CheckAndRunPowershellScript(fmtScript)
	if err != nil {
		return errHandler(err)
	}
	report := &MsTeamsReport{}
	if res.ExitStatus == 0 {
		data, err := io.ReadAll(res.Stdout)
		if err != nil {
			return errHandler(err)
		}
		str := string(data)
		// The Connect-MicrosoftTeams also displays a header for which there
		// are no params to hide it. To allow the JSON unmarshal to work
		// we strip away everything until the first '{' character.
		idx := strings.IndexByte(str, '{')
		if idx == -1 {
			return errHandler(errors.New("invalid JSON format"))
		}
		after := str[idx:]
		newData := []byte(after)

		logger.DebugDumpJSON("ms-teams-report", string(newData))
		err = json.Unmarshal(newData, report)
		if err != nil {
			return errHandler(err)
		}
	} else {
		data, err := io.ReadAll(res.Stderr)
		if err != nil {
			return errHandler(err)
		}

		str := string(data)
		if strings.Contains(strings.ToLower(str), "access denied") {
			return errHandler(errors.New("access denied, please ensure the credentials have the right permissions in Azure AD"))
		}

		logger.DebugDumpJSON("ms-teams-report", string(data))
		return errHandler(fmt.Errorf("failed to generate ms teams report (exit code %d): %s", res.ExitStatus, string(data)))
	}

	if report.CsTeamsClientConfiguration != nil {
		csTeamsConfiguration, csTeamsConfigurationErr := convert.JsonToDict(report.CsTeamsClientConfiguration)
		r.CsTeamsClientConfiguration = plugin.TValue[interface{}]{Data: csTeamsConfiguration, State: plugin.StateIsSet, Error: csTeamsConfigurationErr}
	} else {
		r.CsTeamsClientConfiguration = plugin.TValue[interface{}]{State: plugin.StateIsSet, Error: errors.New("CsTeamsClientConfiguration is nil")}
	}

	if report.CsTenantFederationConfiguration != nil {
		tenantConfig := report.CsTenantFederationConfiguration
		tenantConfigBlockedDomains, _ := convert.JsonToDict(tenantConfig.BlockedDomains)
		mqlTenantConfig, mqlTenantConfigErr := CreateResource(r.MqlRuntime, "ms365.teams.tenantFederationConfig",
			map[string]*llx.RawData{
				"identity":                                    llx.StringData(tenantConfig.Identity),
				"blockedDomains":                              llx.DictData(tenantConfigBlockedDomains),
				"allowedDomains":                              llx.ArrayData(convert.SliceAnyToInterface(tenantConfig.AllowedDomains), types.String),
				"allowFederatedUsers":                         llx.BoolData(tenantConfig.AllowFederatedUsers),
				"allowPublicUsers":                            llx.BoolData(tenantConfig.AllowPublicUsers),
				"allowTeamsConsumer":                          llx.BoolData(tenantConfig.AllowTeamsConsumer),
				"allowTeamsConsumerInbound":                   llx.BoolData(tenantConfig.AllowTeamsConsumerInbound),
				"treatDiscoveredPartnersAsUnverified":         llx.BoolData(tenantConfig.TreatDiscoveredPartnersAsUnverified),
				"sharedSipAddressSpace":                       llx.BoolData(tenantConfig.SharedSipAddressSpace),
				"restrictTeamsConsumerToExternalUserProfiles": llx.BoolData(tenantConfig.RestrictTeamsConsumerToExternalUserProfiles),
			})
		if mqlTenantConfigErr != nil {
			r.CsTenantFederationConfiguration = plugin.TValue[*mqlMs365TeamsTenantFederationConfig]{State: plugin.StateIsSet, Error: mqlTenantConfigErr}
		} else {
			r.CsTenantFederationConfiguration = plugin.TValue[*mqlMs365TeamsTenantFederationConfig]{Data: mqlTenantConfig.(*mqlMs365TeamsTenantFederationConfig), State: plugin.StateIsSet}
		}
	} else {
		r.CsTenantFederationConfiguration = plugin.TValue[*mqlMs365TeamsTenantFederationConfig]{State: plugin.StateIsSet, Error: errors.New("CsTenantFederationConfiguration is nil")}
	}

	if report.CsTeamsMeetingPolicy != nil {
		teamsPolicy := report.CsTeamsMeetingPolicy
		mqlTeamsPolicy, mqlTeamsPolicyErr := CreateResource(r.MqlRuntime, "ms365.teams.teamsMeetingPolicyConfig",
			map[string]*llx.RawData{
				"allowAnonymousUsersToJoinMeeting":           llx.BoolData(teamsPolicy.AllowAnonymousUsersToJoinMeeting),
				"allowAnonymousUsersToStartMeeting":          llx.BoolData(teamsPolicy.AllowAnonymousUsersToStartMeeting),
				"allowExternalNonTrustedMeetingChat":         llx.BoolData(teamsPolicy.AllowExternalNonTrustedMeetingChat),
				"autoAdmittedUsers":                          llx.StringData(teamsPolicy.AutoAdmittedUsers),
				"allowPSTNUsersToBypassLobby":                llx.BoolData(teamsPolicy.AllowPSTNUsersToBypassLobby),
				"meetingChatEnabledType":                     llx.StringData(teamsPolicy.MeetingChatEnabledType),
				"designatedPresenterRoleMode":                llx.StringData(teamsPolicy.DesignatedPresenterRoleMode),
				"allowExternalParticipantGiveRequestControl": llx.BoolData(teamsPolicy.AllowExternalParticipantGiveRequestControl),
				"allowSecurityEndUserReporting":              llx.BoolData(teamsPolicy.AllowSecurityEndUserReporting),
				"allowCloudRecordingForCalls":                llx.BoolData(teamsPolicy.AllowCloudRecordingForCalls),
			})
		if mqlTeamsPolicyErr != nil {
			r.CsTeamsMeetingPolicy = plugin.TValue[*mqlMs365TeamsTeamsMeetingPolicyConfig]{State: plugin.StateIsSet, Error: mqlTeamsPolicyErr}
		} else {
			r.CsTeamsMeetingPolicy = plugin.TValue[*mqlMs365TeamsTeamsMeetingPolicyConfig]{Data: mqlTeamsPolicy.(*mqlMs365TeamsTeamsMeetingPolicyConfig), State: plugin.StateIsSet}
		}
	} else {
		r.CsTeamsMeetingPolicy = plugin.TValue[*mqlMs365TeamsTeamsMeetingPolicyConfig]{State: plugin.StateIsSet, Error: errors.New("CsTeamsMeetingPolicy is nil")}
	}

	if report.CsTeamsMessagingPolicy != nil {
		teamsMessagePolicy := report.CsTeamsMessagingPolicy
		mqlTeamsMessagePolicy, mqlTeamsMessagePolicyErr := CreateResource(r.MqlRuntime, "ms365.teams.teamsMessagingPolicyConfig",
			map[string]*llx.RawData{
				"allowSecurityEndUserReporting": llx.BoolData(teamsMessagePolicy.AllowSecurityEndUserReporting),
			})
		if mqlTeamsMessagePolicyErr != nil {
			r.CsTeamsMessagingPolicy = plugin.TValue[*mqlMs365TeamsTeamsMessagingPolicyConfig]{State: plugin.StateIsSet, Error: mqlTeamsMessagePolicyErr}
		} else {
			r.CsTeamsMessagingPolicy = plugin.TValue[*mqlMs365TeamsTeamsMessagingPolicyConfig]{Data: mqlTeamsMessagePolicy.(*mqlMs365TeamsTeamsMessagingPolicyConfig), State: plugin.StateIsSet}
		}
	} else {
		r.CsTeamsMessagingPolicy = plugin.TValue[*mqlMs365TeamsTeamsMessagingPolicyConfig]{State: plugin.StateIsSet, Error: errors.New("CsTeamsMessagingPolicy is nil")}
	}

	return nil
}

func (r *mqlMs365Teams) csTeamsClientConfiguration() (interface{}, error) {
	return nil, r.gatherTeamsReport()
}

func (r *mqlMs365Teams) csTenantFederationConfiguration() (*mqlMs365TeamsTenantFederationConfig, error) {
	return nil, r.gatherTeamsReport()
}

func (r *mqlMs365Teams) csTeamsMeetingPolicy() (*mqlMs365TeamsTeamsMeetingPolicyConfig, error) {
	return nil, r.gatherTeamsReport()
}

func (r *mqlMs365Teams) csTeamsMessagingPolicy() (*mqlMs365TeamsTeamsMessagingPolicyConfig, error) {
	return nil, r.gatherTeamsReport()
}
