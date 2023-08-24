// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"time"

	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/providers/google-workspace/connection"
	directory "google.golang.org/api/admin/directory/v1"
)

func (g *mqlGoogleworkspace) domains() ([]interface{}, error) {
	conn := g.MqlRuntime.Connection.(*connection.GoogleWorkspaceConnection)
	directoryService, err := directoryService(conn, directory.AdminDirectoryDomainReadonlyScope)
	if err != nil {
		return nil, err
	}

	domains, err := directoryService.Domains.List(conn.CustomerID()).Do()
	if err != nil {
		return nil, err
	}

	res := []interface{}{}
	for i := range domains.Domains {
		r, err := newMqlGoogleWorkspaceDomain(g.MqlRuntime, domains.Domains[i])
		if err != nil {
			return nil, err
		}
		res = append(res, r)
	}
	return res, nil
}

func newMqlGoogleWorkspaceDomain(runtime *plugin.Runtime, entry *directory.Domains) (interface{}, error) {
	unixTimeUTC := time.UnixMilli(entry.CreationTime)
	return CreateResource(runtime, "googleworkspace.domain", map[string]*llx.RawData{
		"domainName":   llx.StringData(entry.DomainName),
		"isPrimary":    llx.BoolData(entry.IsPrimary),
		"verified":     llx.BoolData(entry.Verified),
		"creationTime": llx.TimeData(unixTimeUTC),
	})
}

func (g *mqlGoogleworkspaceDomain) id() (string, error) {
	return "googleworkspace.domain/" + g.DomainName.Data, g.DomainName.Error
}
