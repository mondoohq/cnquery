// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/iru/connection"
	"go.mondoo.com/mql/v13/providers/iru/connection/client"
)

func (r *mqlIru) tenant() (*mqlIruTenant, error) {
	conn := r.MqlRuntime.Connection.(*connection.IruConnection)
	t, err := conn.Client.GetTenant()
	if err != nil {
		if client.IsAccessDenied(err) {
			log.Warn().Err(err).Msg("iru> access denied to tenant settings")
			r.Tenant.State = plugin.StateIsSet | plugin.StateIsNull
			return nil, nil
		}
		return nil, err
	}

	item, err := CreateResource(r.MqlRuntime, "iru.tenant", map[string]*llx.RawData{
		"subdomain":           llx.StringData(t.Subdomain),
		"organizationName":    llx.StringData(t.OrganizationName),
		"region":              llx.StringData(t.Region),
		"agentMinimumVersion": llx.StringData(t.AgentMinimumVersion),
		"agentLatestVersion":  llx.StringData(t.AgentLatestVersion),
	})
	if err != nil {
		return nil, err
	}
	return item.(*mqlIruTenant), nil
}

func (t *mqlIruTenant) id() (string, error) {
	return "iru.tenant/" + t.Subdomain.Data, nil
}
