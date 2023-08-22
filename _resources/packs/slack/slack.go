// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package slack

import (
	"errors"

	"go.mondoo.com/cnquery/resources/packs/slack/info"

	"go.mondoo.com/cnquery/motor/providers"
	slack_provider "go.mondoo.com/cnquery/motor/providers/slack"
)

var Registry = info.Registry

func init() {
	Init(Registry)
}

func (k *mqlSlack) id() (string, error) {
	return "slack", nil
}

func slackProvider(p providers.Instance) (*slack_provider.Provider, error) {
	at, ok := p.(*slack_provider.Provider)
	if !ok {
		return nil, errors.New("slack resource is not supported on this provider")
	}
	return at, nil
}
