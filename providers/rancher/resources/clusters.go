// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/types"
)

// LocalClusterID is the identifier Rancher gives the cluster it runs on itself.
const LocalClusterID = "local"

// mqlRancherClusterInternal carries the references a cluster names but does not
// resolve until they are asked for. Both are plain names on the wire, and the
// resources they point at are fetched from a listing that is shared with every
// other reader rather than looked up one cluster at a time.
type mqlRancherClusterInternal struct {
	cachePodSecurityTemplateName string
	cacheProjectMemberRoleName   string
}

func (r *mqlRancher) clusters() ([]any, error) {
	records, err := listRecords[clusterRecord](r.MqlRuntime, pathClusters)
	if err != nil {
		return nil, err
	}

	res := make([]any, 0, len(records))
	for i := range records {
		mqlCluster, err := buildCluster(r.MqlRuntime, &records[i])
		if err != nil {
			return nil, err
		}
		res = append(res, mqlCluster)
	}
	return res, nil
}

func buildCluster(runtime *plugin.Runtime, record *clusterRecord) (*mqlRancherCluster, error) {
	var kubernetesVersion *string
	if record.Version != nil && record.Version.GitVersion != "" {
		kubernetesVersion = &record.Version.GitVersion
	}

	authEndpointEnabled := false
	authEndpointFQDN := ""
	if record.LocalClusterAuthEndpoint != nil {
		authEndpointEnabled = record.LocalClusterAuthEndpoint.Enabled
		authEndpointFQDN = record.LocalClusterAuthEndpoint.FQDN
	}

	resource, err := CreateResource(runtime, "rancher.cluster", map[string]*llx.RawData{
		"__id":                            llx.StringData(record.ID),
		"id":                              llx.StringData(record.ID),
		"name":                            llx.StringData(record.Name),
		"description":                     llx.StringData(record.Description),
		"state":                           llx.StringData(record.State),
		"transitioning":                   llx.StringData(record.Transitioning),
		"transitioningMessage":            llx.StringData(record.TransitioningMessage),
		"driver":                          llx.StringData(record.Driver),
		"provider":                        llx.StringData(record.Provider),
		"kubernetesVersion":               llx.StringDataPtr(kubernetesVersion),
		"isLocalCluster":                  llx.BoolData(record.ID == LocalClusterID),
		"internal":                        llx.BoolData(record.Internal),
		"created":                         llx.TimeDataPtr(parseTime(record.Created)),
		"nodeCount":                       llx.IntData(record.NodeCount),
		"enableNetworkPolicy":             llx.BoolDataPtr(record.EnableNetworkPolicy),
		"appliedEnableNetworkPolicy":      llx.BoolData(record.AppliedEnableNetworkPolicy),
		"localClusterAuthEndpointEnabled": llx.BoolData(authEndpointEnabled),
		"localClusterAuthEndpointFqdn":    llx.StringData(authEndpointFQDN),
		"fleetWorkspaceName":              llx.StringData(record.FleetWorkspaceName),
		"apiEndpoint":                     llx.StringData(record.APIEndpoint),
		"labels":                          llx.MapData(toStringMap(record.Labels), types.String),
		"annotations":                     llx.MapData(toStringMap(record.Annotations), types.String),
	})
	if err != nil {
		return nil, err
	}

	mqlCluster := resource.(*mqlRancherCluster)
	mqlCluster.cachePodSecurityTemplateName = record.PodSecurityTemplateName
	mqlCluster.cacheProjectMemberRoleName = record.DefaultClusterRoleForProjectMembers
	return mqlCluster, nil
}

// clusterByID resolves a cluster from the shared cluster listing. Going through
// the listing rather than fetching one cluster at a time keeps a reference on a
// per-record resource, such as a token's cluster, from costing one call per
// record.
func clusterByID(runtime *plugin.Runtime, id string) (*mqlRancherCluster, error) {
	if id == "" {
		return nil, nil
	}
	records, err := listRecords[clusterRecord](runtime, pathClusters)
	if err != nil {
		return nil, err
	}
	for i := range records {
		if records[i].ID == id {
			return buildCluster(runtime, &records[i])
		}
	}
	return nil, nil
}

func (r *mqlRancherCluster) podSecurityAdmissionTemplate() (*mqlRancherPodSecurityAdmissionConfigurationTemplate, error) {
	if r.cachePodSecurityTemplateName == "" {
		r.PodSecurityAdmissionTemplate.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	template, err := podSecurityTemplateByName(r.MqlRuntime, r.cachePodSecurityTemplateName)
	if err != nil {
		return nil, err
	}
	if template == nil {
		r.PodSecurityAdmissionTemplate.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	return template, nil
}

func (r *mqlRancherCluster) defaultClusterRoleForProjectMembers() (*mqlRancherRoleTemplate, error) {
	if r.cacheProjectMemberRoleName == "" {
		r.DefaultClusterRoleForProjectMembers.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	template, err := roleTemplateByID(r.MqlRuntime, r.cacheProjectMemberRoleName)
	if err != nil {
		return nil, err
	}
	if template == nil {
		r.DefaultClusterRoleForProjectMembers.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	return template, nil
}

func (r *mqlRancherCluster) projects() ([]any, error) {
	records, err := listRecords[projectRecord](r.MqlRuntime, pathProjects)
	if err != nil {
		return nil, err
	}

	clusterID := r.Id.Data
	res := []any{}
	for i := range records {
		if records[i].ClusterID != clusterID {
			continue
		}
		mqlProject, err := buildProject(r.MqlRuntime, &records[i])
		if err != nil {
			return nil, err
		}
		res = append(res, mqlProject)
	}
	return res, nil
}

func (r *mqlRancherCluster) roleTemplateBindings() ([]any, error) {
	records, err := listRecords[bindingRecord](r.MqlRuntime, pathClusterRoleTemplateBinding)
	if err != nil {
		return nil, err
	}

	clusterID := r.Id.Data
	res := []any{}
	for i := range records {
		if records[i].ClusterID != clusterID {
			continue
		}
		mqlBinding, err := buildClusterRoleTemplateBinding(r.MqlRuntime, &records[i])
		if err != nil {
			return nil, err
		}
		res = append(res, mqlBinding)
	}
	return res, nil
}
