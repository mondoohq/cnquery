// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/providers/slack/connection"
)

func (x *mqlSlackTeam) id() (string, error) {
	return "slack.team/" + x.Id.Data, nil
}

// init method for team
func (s *mqlSlackTeam) init(args map[string]interface{}) (map[string]interface{}, *mqlSlackTeam, error) {
	conn := s.MqlRuntime.Connection.(*connection.SlackConnection)
	client := conn.Client()
	teamInfo, err := client.GetTeamInfo()
	if err != nil {
		return nil, nil, err
	}

	args["id"] = llx.StringData(teamInfo.ID)
	args["name"] = llx.StringData(teamInfo.Name)
	args["domain"] = llx.StringData(teamInfo.Domain)
	args["emailDomain"] = llx.StringData(teamInfo.EmailDomain)

	return args, nil, nil
}
