// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"time"

	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers/shodan/connection"
)

func (r *mqlShodan) id() (string, error) {
	return "shodan", nil
}

func initShodanProfile(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	conn := runtime.Connection.(*connection.ShodanConnection)
	client := conn.Client()
	if client == nil {
		return nil, nil, errors.New("cannot retrieve new data while using a mock connection")
	}

	profile, err := client.AccountProfile(context.Background())
	if err != nil {
		return nil, nil, err
	}

	args["member"] = llx.BoolData(profile.Member)
	args["credits"] = llx.IntData(profile.Credits)
	args["displayName"] = llx.StringData(profile.DisplayName)
	args["createdAt"] = llx.NilData

	t, err := time.Parse(time.RFC3339, profile.Created)
	if err == nil {
		args["createdAt"] = llx.TimeData(t)
	}

	return args, nil, nil
}

func (r *mqlShodanProfile) id() (string, error) {
	return "shodan/profile", nil
}

func initShodanApiPlan(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	conn := runtime.Connection.(*connection.ShodanConnection)
	client := conn.Client()
	if client == nil {
		return nil, nil, errors.New("cannot retrieve new data while using a mock connection")
	}

	apiInfo, err := client.ApiInfo(context.Background())
	if err != nil {
		return nil, nil, err
	}

	args["scanCredits"] = llx.IntData(apiInfo.ScanCredits)
	args["plan"] = llx.StringData(apiInfo.Plan)
	args["unlocked"] = llx.BoolData(apiInfo.Unlocked)
	args["unlockedLeft"] = llx.IntData(apiInfo.UnlockedLeft)
	args["telnet"] = llx.BoolData(apiInfo.Telnet)
	args["monitoredIps"] = llx.IntData(apiInfo.MonitoredIps)

	return args, nil, nil
}

func (r *mqlShodanApiPlan) id() (string, error) {
	return "shodan/api-plan", nil
}
