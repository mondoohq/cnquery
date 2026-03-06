// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"strconv"
	"time"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/gitlab/connection"
)

func (n *mqlGitlabNamespace) id() (string, error) {
	return "gitlab.namespace/" + strconv.FormatInt(n.Id.Data, 10), nil
}

func (g *mqlGitlabGroup) namespace() (*mqlGitlabNamespace, error) {
	conn := g.MqlRuntime.Connection.(*connection.GitLabConnection)

	groupID := int(g.Id.Data)
	ns, _, err := conn.Client().Namespaces.GetNamespace(groupID)
	if err != nil {
		return nil, err
	}

	var trialEndsOn *time.Time
	if ns.TrialEndsOn != nil {
		t := time.Time(*ns.TrialEndsOn)
		trialEndsOn = &t
	}

	var maxSeatsUsed int64
	if ns.MaxSeatsUsed != nil {
		maxSeatsUsed = *ns.MaxSeatsUsed
	}

	var seatsInUse int64
	if ns.SeatsInUse != nil {
		seatsInUse = *ns.SeatsInUse
	}

	nsArgs := map[string]*llx.RawData{
		"id":                          llx.IntData(ns.ID),
		"name":                        llx.StringData(ns.Name),
		"path":                        llx.StringData(ns.Path),
		"kind":                        llx.StringData(ns.Kind),
		"fullPath":                    llx.StringData(ns.FullPath),
		"parentId":                    llx.IntData(ns.ParentID),
		"webURL":                      llx.StringData(ns.WebURL),
		"membersCountWithDescendants": llx.IntData(ns.MembersCountWithDescendants),
		"billableMembersCount":        llx.IntData(ns.BillableMembersCount),
		"plan":                        llx.StringData(ns.Plan),
		"trial":                       llx.BoolData(ns.Trial),
		"trialEndsOn":                 llx.TimeDataPtr(trialEndsOn),
		"maxSeatsUsed":                llx.IntData(maxSeatsUsed),
		"seatsInUse":                  llx.IntData(seatsInUse),
	}

	mqlNs, err := CreateResource(g.MqlRuntime, "gitlab.namespace", nsArgs)
	if err != nil {
		return nil, err
	}

	return mqlNs.(*mqlGitlabNamespace), nil
}

func initGitlabNamespace(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}

	conn := runtime.Connection.(*connection.GitLabConnection)

	// If we have a group connection, get the namespace for that group
	if conn.IsGroup() {
		grp, err := conn.Group()
		if err != nil {
			return nil, nil, err
		}

		ns, _, err := conn.Client().Namespaces.GetNamespace(int(grp.ID))
		if err != nil {
			return nil, nil, err
		}

		var trialEndsOn *time.Time
		if ns.TrialEndsOn != nil {
			t := time.Time(*ns.TrialEndsOn)
			trialEndsOn = &t
		}

		var maxSeatsUsed int64
		if ns.MaxSeatsUsed != nil {
			maxSeatsUsed = *ns.MaxSeatsUsed
		}

		var seatsInUse int64
		if ns.SeatsInUse != nil {
			seatsInUse = *ns.SeatsInUse
		}

		args["id"] = llx.IntData(ns.ID)
		args["name"] = llx.StringData(ns.Name)
		args["path"] = llx.StringData(ns.Path)
		args["kind"] = llx.StringData(ns.Kind)
		args["fullPath"] = llx.StringData(ns.FullPath)
		args["parentId"] = llx.IntData(ns.ParentID)
		args["webURL"] = llx.StringData(ns.WebURL)
		args["membersCountWithDescendants"] = llx.IntData(ns.MembersCountWithDescendants)
		args["billableMembersCount"] = llx.IntData(ns.BillableMembersCount)
		args["plan"] = llx.StringData(ns.Plan)
		args["trial"] = llx.BoolData(ns.Trial)
		args["trialEndsOn"] = llx.TimeDataPtr(trialEndsOn)
		args["maxSeatsUsed"] = llx.IntData(maxSeatsUsed)
		args["seatsInUse"] = llx.IntData(seatsInUse)
	}

	return args, nil, nil
}
