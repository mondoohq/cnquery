// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	coreresources "go.mondoo.com/mql/v13/providers/core/resources"
)

func (g *mqlGithubRepository) findings() (plugin.Resource, error) {
	if g.Name.Error != nil {
		return nil, g.Name.Error
	}
	repoName := g.Name.Data

	if g.Owner.Error != nil {
		return nil, g.Owner.Error
	}
	owner := g.Owner.Data

	if owner.Login.Error != nil {
		return nil, owner.Login.Error
	}
	ownerLogin := owner.Login.Data

	return dependabotAlertFindings(g, ownerLogin, repoName)
}

func dependabotAlertFindings(g *mqlGithubRepository, owner, repository string) (plugin.Resource, error) {
	alertsResult := g.GetDependabotAlerts()
	if alertsResult.Error != nil {
		return nil, alertsResult.Error
	}

	if len(alertsResult.Data) == 0 {
		return nil, nil
	}
	a := alertsResult.Data[0]
	if a == nil {
		return nil, nil
	}

	alert, ok := a.(*mqlGithubDependabotAlert)
	if !ok {
		return nil, nil
	}

	alertID, err := alert.id()
	if err != nil {
		return nil, err
	}

	id := coreresources.ResourceFinding + "/" + owner + "/" + repository + "/" + alertID
	args := map[string]*llx.RawData{
		"id":   llx.StringData(id),
		"__id": llx.StringData(id),
	}
	finding, err := CreateResource(g.MqlRuntime, coreresources.ResourceFinding, args)
	if err != nil {
		return nil, err
	}

	return finding, nil
	mqlId := finding.MqlID()
	v, err := g.MqlRuntime.GetSharedData("finding", mqlId, "")
	if err != nil {
		return nil, err
	}
	casted := v.Value.(plugin.Resource)
	return casted, nil
}
