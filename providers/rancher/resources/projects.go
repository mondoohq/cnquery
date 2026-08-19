// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/types"
)

// mqlRancherProjectInternal carries the cluster a project belongs to. The
// identifier is also the first half of the project's own id, but reading it off
// the record is what keeps the two independent of Rancher's id format.
type mqlRancherProjectInternal struct {
	cacheClusterID string
}

func (r *mqlRancher) projects() ([]any, error) {
	records, err := listRecords[projectRecord](r.MqlRuntime, pathProjects)
	if err != nil {
		return nil, err
	}

	res := make([]any, 0, len(records))
	for i := range records {
		mqlProject, err := buildProject(r.MqlRuntime, &records[i])
		if err != nil {
			return nil, err
		}
		res = append(res, mqlProject)
	}
	return res, nil
}

func buildProject(runtime *plugin.Runtime, record *projectRecord) (*mqlRancherProject, error) {
	resource, err := CreateResource(runtime, "rancher.project", map[string]*llx.RawData{
		"__id":                          llx.StringData(record.ID),
		"id":                            llx.StringData(record.ID),
		"name":                          llx.StringData(record.Name),
		"description":                   llx.StringData(record.Description),
		"state":                         llx.StringData(record.State),
		"created":                       llx.TimeDataPtr(parseTime(record.Created)),
		"backingNamespace":              llx.StringData(record.BackingNamespace),
		"labels":                        llx.MapData(toStringMap(record.Labels), types.String),
		"annotations":                   llx.MapData(toStringMap(record.Annotations), types.String),
		"resourceQuota":                 dictOrNil(record.ResourceQuota),
		"containerDefaultResourceLimit": dictOrNil(record.ContainerDefaultResourceLimit),
	})
	if err != nil {
		return nil, err
	}

	mqlProject := resource.(*mqlRancherProject)
	mqlProject.cacheClusterID = record.ClusterID
	return mqlProject, nil
}

// projectByID resolves a project from the shared project listing.
func projectByID(runtime *plugin.Runtime, id string) (*mqlRancherProject, error) {
	if id == "" {
		return nil, nil
	}
	records, err := listRecords[projectRecord](runtime, pathProjects)
	if err != nil {
		return nil, err
	}
	for i := range records {
		if records[i].ID == id {
			return buildProject(runtime, &records[i])
		}
	}
	return nil, nil
}

func (r *mqlRancherProject) cluster() (*mqlRancherCluster, error) {
	mqlCluster, err := clusterByID(r.MqlRuntime, r.cacheClusterID)
	if err != nil {
		return nil, err
	}
	if mqlCluster == nil {
		r.Cluster.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	return mqlCluster, nil
}

func (r *mqlRancherProject) roleTemplateBindings() ([]any, error) {
	records, err := listRecords[bindingRecord](r.MqlRuntime, pathProjectRoleTemplateBinding)
	if err != nil {
		return nil, err
	}

	projectID := r.Id.Data
	res := []any{}
	for i := range records {
		if records[i].ProjectID != projectID {
			continue
		}
		mqlBinding, err := buildProjectRoleTemplateBinding(r.MqlRuntime, &records[i])
		if err != nil {
			return nil, err
		}
		res = append(res, mqlBinding)
	}
	return res, nil
}

// dictOrNil reports an absent structure as null rather than as an empty map. A
// project with no quota and a project with an empty quota are different claims,
// and only the first is what Rancher means by leaving the field out.
func dictOrNil(value map[string]any) *llx.RawData {
	if value == nil {
		return llx.NilData
	}
	return llx.DictData(value)
}
