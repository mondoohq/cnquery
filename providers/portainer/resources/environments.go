// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"strconv"

	"github.com/portainer/client-api-go/v2/pkg/models"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/portainer/connection"
)

type mqlPortainerEnvironmentInternal struct {
	cacheGroupId int64
	cacheTagIds  []int64
}

func accessPoliciesToDict(policies map[string]models.PortainerAccessPolicy) map[string]any {
	res := make(map[string]any, len(policies))
	for k, v := range policies {
		res[k] = v.RoleID
	}
	return res
}

// securitySettingsToDict flattens the per-environment container-security
// overrides into a dict keyed by setting name. Returns nil when the endpoint
// carries no overrides so the field resolves to an empty dict.
func securitySettingsToDict(s *models.PortainerEndpointSecuritySettings) map[string]any {
	if s == nil {
		return nil
	}
	return map[string]any{
		"allowBindMountsForRegularUsers":            s.AllowBindMountsForRegularUsers,
		"allowContainerCapabilitiesForRegularUsers": s.AllowContainerCapabilitiesForRegularUsers,
		"allowDeviceMappingForRegularUsers":         s.AllowDeviceMappingForRegularUsers,
		"allowHostNamespaceForRegularUsers":         s.AllowHostNamespaceForRegularUsers,
		"allowPrivilegedModeForRegularUsers":        s.AllowPrivilegedModeForRegularUsers,
		"allowStackManagementForRegularUsers":       s.AllowStackManagementForRegularUsers,
		"allowSysctlSettingForRegularUsers":         s.AllowSysctlSettingForRegularUsers,
		"allowVolumeBrowserForRegularUsers":         s.AllowVolumeBrowserForRegularUsers,
		"enableHostManagementFeatures":              s.EnableHostManagementFeatures,
	}
}

func newMqlPortainerEnvironment(runtime *plugin.Runtime, e *models.PortainereeEndpoint) (*mqlPortainerEnvironment, error) {
	res, err := CreateResource(runtime, "portainer.environment", map[string]*llx.RawData{
		"__id":                 llx.StringData("portainer.environment/" + strconv.FormatInt(e.ID, 10)),
		"id":                   llx.IntData(e.ID),
		"name":                 llx.StringData(e.Name),
		"type":                 llx.StringData(connection.EnvironmentType(e.Type)),
		"status":               llx.StringData(connection.EnvironmentStatus(e.Status)),
		"url":                  llx.StringData(e.URL),
		"publicUrl":            llx.StringData(e.PublicURL),
		"tlsEnabled":           llx.BoolData(e.TLS),
		"containerEngine":      llx.StringData(e.ContainerEngine),
		"teamAccessPolicies":   llx.DictData(accessPoliciesToDict(e.TeamAccessPolicies)),
		"userAccessPolicies":   llx.DictData(accessPoliciesToDict(e.UserAccessPolicies)),
		"edgeId":               llx.StringData(e.EdgeID),
		"heartbeat":            llx.BoolData(e.Heartbeat),
		"userTrusted":          llx.BoolData(e.UserTrusted),
		"gpuManagementEnabled": llx.BoolData(e.EnableGPUManagement),
		"securitySettings":     llx.DictData(securitySettingsToDict(e.SecuritySettings)),
	})
	if err != nil {
		return nil, err
	}
	mqlEnv := res.(*mqlPortainerEnvironment)
	mqlEnv.cacheGroupId = e.GroupID
	mqlEnv.cacheTagIds = e.TagIds
	return mqlEnv, nil
}

// tags resolves the cached tag ids to the tags assigned to the environment.
func (r *mqlPortainerEnvironment) tags() ([]any, error) {
	conn := r.MqlRuntime.Connection.(*connection.PortainerConnection)
	return resolvePortainerTags(r.MqlRuntime, conn, r.cacheTagIds)
}

// group resolves the environment group (endpoint group) this environment
// belongs to from the cached group id.
func (r *mqlPortainerEnvironment) group() (*mqlPortainerEnvironmentGroup, error) {
	// Portainer endpoint groups start at id 1 ("Unassigned"); 0 means unset.
	if r.cacheGroupId == 0 {
		r.Group.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	conn := r.MqlRuntime.Connection.(*connection.PortainerConnection)
	groups, err := conn.EndpointGroups()
	if err != nil {
		return nil, err
	}
	for _, g := range groups {
		if g.ID == r.cacheGroupId {
			return newMqlPortainerEnvironmentGroup(r.MqlRuntime, g)
		}
	}
	r.Group.State = plugin.StateIsNull | plugin.StateIsSet
	return nil, nil
}

func (r *mqlPortainer) environments() ([]any, error) {
	conn := r.MqlRuntime.Connection.(*connection.PortainerConnection)

	endpoints, err := conn.Endpoints()
	if err != nil {
		return nil, err
	}

	res := make([]any, 0, len(endpoints))
	for _, e := range endpoints {
		mqlEnv, err := newMqlPortainerEnvironment(r.MqlRuntime, e)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlEnv)
	}
	return res, nil
}
