// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package slack

import "go.mondoo.com/cnquery/motor/platform"

func (p *Provider) PlatformInfo() (*platform.Platform, error) {
	return &platform.Platform{
		Name:    "slack-team",
		Title:   "Slack Team",
		Runtime: p.Runtime(),
		Kind:    p.Kind(),
		Family:  []string{"slack"},
	}, nil
}

func (p *Provider) Identifier() (string, error) {
	return "//platformid.api.mondoo.app/runtime/slack/team/" + p.teamInfo.ID, nil
}

func (p *Provider) TeamName() (string, error) {
	return p.teamInfo.Name, nil
}
