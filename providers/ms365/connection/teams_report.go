// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
)

var teamsReport = `
$graphToken= '%s'
$teamsToken = '%s'

Install-Module -Name MicrosoftTeams -Scope CurrentUser -Force
Import-Module MicrosoftTeams
Connect-MicrosoftTeams -AccessTokens @("$graphToken", "$teamsToken")

$CsTeamsClientConfiguration = (Get-CsTeamsClientConfiguration)

$msteams = New-Object PSObject
Add-Member -InputObject $msteams -MemberType NoteProperty -Name CsTeamsClientConfiguration -Value $CsTeamsClientConfiguration

Disconnect-MicrosoftTeams -Confirm:$false

ConvertTo-Json -Depth 4 $msteams
`

func (c *Ms365Connection) GetTeamsReport(ctx context.Context) (*MsTeamsReport, error) {
	c.teamsReportLock.Lock()
	defer c.teamsReportLock.Unlock()
	if c.teamsReport != nil {
		return c.teamsReport, nil
	}
	token := c.Token()
	teamsToken, err := token.GetToken(ctx, policy.TokenRequestOptions{
		Scopes: []string{TeamsScope},
	})
	if err != nil {
		return nil, err
	}
	graphToken, err := token.GetToken(ctx, policy.TokenRequestOptions{
		Scopes: []string{DefaultMSGraphScope},
	})
	if err != nil {
		return nil, err
	}
	report, err := c.getTeamsReport(graphToken.Token, teamsToken.Token)
	if err != nil {
		return nil, err
	}
	c.teamsReport = report
	return report, nil
}

func (c *Ms365Connection) getTeamsReport(accessToken, teamsToken string) (*MsTeamsReport, error) {
	fmtScript := fmt.Sprintf(teamsReport, accessToken, teamsToken)
	res, err := c.runPowershellScript(fmtScript)
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
		err = json.Unmarshal(newData, report)
		if err != nil {
			return nil, err
		}
	} else {
		data, err := io.ReadAll(res.Stderr)
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("failed to generate ms365 report: %s", string(data))

	}
	return report, nil
}

type MsTeamsReport struct {
	CsTeamsClientConfiguration interface{} `json:"CsTeamsClientConfiguration"`
}
