package slack

import "go.mondoo.com/cnquery/motor/platform"

func (p *Provider) PlatformInfo() (*platform.Platform, error) {
	return &platform.Platform{
		Name:    "slack",
		Title:   "Slack",
		Runtime: p.Runtime(),
		Kind:    p.Kind(),
	}, nil
}

func (p *Provider) Identifier() (string, error) {
	return "//platformid.api.mondoo.app/runtime/slack/team/" + p.teamInfo.ID, nil
}

func (p *Provider) TeamName() (string, error) {
	return p.teamInfo.Name, nil
}
