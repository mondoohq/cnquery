// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	resourcemanager "github.com/stackitcloud/stackit-sdk-go/services/resourcemanager/v0api"
	"go.mondoo.com/mql/llx"
	"go.mondoo.com/mql/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/providers/stackit/connection"
)

func (r *mqlStackit) id() (string, error) {
	return "stackit/" + conn(r.MqlRuntime).ProjectID(), nil
}

func (r *mqlStackit) region() (string, error) {
	return conn(r.MqlRuntime).Region(), nil
}

func conn(runtime *plugin.Runtime) *connection.StackitConnection {
	return runtime.Connection.(*connection.StackitConnection)
}

// mqlStackitProjectInternal caches the ancestry chain the project response
// carries when parents are requested, so ancestors() builds its records
// without another call.
type mqlStackitProjectInternal struct {
	cacheParents []resourcemanager.ParentListInner
}

// project fetches the project metadata from the resource-manager API. Parents
// are requested on the same call: that populates the ancestry chain up to the
// organization, which is the only way a project-scoped credential learns
// which organization owns it without organization-level read access.
func (r *mqlStackit) project() (*mqlStackitProject, error) {
	c := conn(r.MqlRuntime)
	client, err := c.ResourceManager()
	if err != nil {
		return nil, err
	}
	resp, err := client.DefaultAPI.GetProject(bgctx(), c.ProjectID()).IncludeParents(true).Execute()
	if err != nil {
		if isAccessDenied(err) {
			return markNull[mqlStackitProject](&r.Project)
		}
		return nil, err
	}

	lifecycle, _ := resp.GetLifecycleStateOk()
	labels, _ := resp.GetLabelsOk()
	createdAt, ok := resp.GetCreationTimeOk()
	updatedAt, okUpdated := resp.GetUpdateTimeOk()

	parentID, parentUUID, parentType := "", "", ""
	if p, hasParent := resp.GetParentOk(); hasParent {
		parentID = p.GetContainerId()
		if parentID == "" {
			parentID = p.GetId()
		}
		parentUUID = p.GetId()
		if t, ok := p.GetTypeOk(); ok && t != nil {
			parentType = string(*t)
		}
	}

	args := map[string]*llx.RawData{
		"id":             llx.StringData(resp.GetProjectId()),
		"containerId":    llx.StringData(resp.GetContainerId()),
		"name":           llx.StringData(resp.GetName()),
		"parent":         llx.StringData(parentID),
		"parentId":       llx.StringData(parentUUID),
		"parentType":     llx.StringData(parentType),
		"lifecycleState": llx.StringData(ptrEnumStr(lifecycle)),
		"creationTime":   llx.TimeDataPtr(timeOrNil(createdAt, ok)),
		"updateTime":     llx.TimeDataPtr(timeOrNil(updatedAt, okUpdated)),
		"labels":         labelData(labels),
	}
	res, err := CreateResource(r.MqlRuntime, "stackit.project", args)
	if err != nil {
		return nil, err
	}
	project := res.(*mqlStackitProject)
	project.cacheParents = resp.GetParents()
	return project, nil
}

func (r *mqlStackitProject) id() (string, error) {
	return "stackit.project/" + r.Id.Data, nil
}

// ancestors lists the containers above the project, nearest first: the
// folder (or folders) it sits in and finally the organization. The chain
// comes from the same project response, requested with parents included.
func (r *mqlStackitProject) ancestors() ([]any, error) {
	out := make([]any, 0, len(r.cacheParents))
	for i := range r.cacheParents {
		p := &r.cacheParents[i]
		res, err := CreateResource(r.MqlRuntime, "stackit.project.ancestor", projectAncestorArgs(p))
		if err != nil {
			return nil, err
		}
		out = append(out, res)
	}
	return out, nil
}

// organizationId reports the container id of the organization at the top of
// the ancestry chain, or "" when the chain is empty or names no organization.
func (r *mqlStackitProject) organizationId() (string, error) {
	return projectOrganizationID(r.cacheParents), nil
}

// projectAncestorArgs maps one ancestry entry onto stackit.project.ancestor.
// The container id is the identifier every resource-manager endpoint takes,
// so it is the resource id; the UUID rides along as `uuid`.
func projectAncestorArgs(p *resourcemanager.ParentListInner) map[string]*llx.RawData {
	ancestorType := ""
	if t, ok := p.GetTypeOk(); ok && t != nil {
		ancestorType = string(*t)
	}
	parentID := ""
	if v, ok := p.GetContainerParentIdOk(); ok && v != nil {
		parentID = *v
	}
	return map[string]*llx.RawData{
		"id":       llx.StringData(p.GetContainerId()),
		"uuid":     llx.StringData(p.GetId()),
		"name":     llx.StringData(p.GetName()),
		"type":     llx.StringData(ancestorType),
		"parentId": llx.StringData(parentID),
	}
}

// projectOrganizationID finds the organization in an ancestry chain by its
// type, not its position, since the API does not promise an ordering.
func projectOrganizationID(parents []resourcemanager.ParentListInner) string {
	for i := range parents {
		if t, ok := parents[i].GetTypeOk(); ok && t != nil && *t == resourcemanager.PARENTLISTINNERTYPE_ORGANIZATION {
			return parents[i].GetContainerId()
		}
	}
	return ""
}

func (r *mqlStackitProjectAncestor) id() (string, error) {
	return "stackit.project.ancestor/" + r.Id.Data, nil
}

// Each namespace resource has a stable id.

func (r *mqlStackitSke) id() (string, error) {
	return "stackit.ske/" + conn(r.MqlRuntime).ProjectID(), nil
}

func (r *mqlStackitObjectStorage) id() (string, error) {
	return "stackit.objectStorage/" + conn(r.MqlRuntime).ProjectID(), nil
}

func (r *mqlStackitDns) id() (string, error) {
	return "stackit.dns/" + conn(r.MqlRuntime).ProjectID(), nil
}

func (r *mqlStackitPostgresFlex) id() (string, error) {
	return "stackit.postgresFlex/" + conn(r.MqlRuntime).ProjectID(), nil
}

func (r *mqlStackitMongoDbFlex) id() (string, error) {
	return "stackit.mongoDbFlex/" + conn(r.MqlRuntime).ProjectID(), nil
}

func (r *mqlStackitOpenSearch) id() (string, error) {
	return "stackit.openSearch/" + conn(r.MqlRuntime).ProjectID(), nil
}

func (r *mqlStackitMariaDb) id() (string, error) {
	return "stackit.mariaDb/" + conn(r.MqlRuntime).ProjectID(), nil
}

func (r *mqlStackitRedis) id() (string, error) {
	return "stackit.redis/" + conn(r.MqlRuntime).ProjectID(), nil
}

func (r *mqlStackitRabbitMq) id() (string, error) {
	return "stackit.rabbitMq/" + conn(r.MqlRuntime).ProjectID(), nil
}

func (r *mqlStackitSecretsManager) id() (string, error) {
	return "stackit.secretsManager/" + conn(r.MqlRuntime).ProjectID(), nil
}

func (r *mqlStackitObservability) id() (string, error) {
	return "stackit.observability/" + conn(r.MqlRuntime).ProjectID(), nil
}

// Namespace getters on `stackit` just return the singleton namespace resource.

func (r *mqlStackit) ske() (*mqlStackitSke, error) {
	res, err := makeNamespace(r.MqlRuntime, "stackit.ske")
	if err != nil {
		return nil, err
	}
	return res.(*mqlStackitSke), nil
}

func (r *mqlStackit) objectStorage() (*mqlStackitObjectStorage, error) {
	res, err := makeNamespace(r.MqlRuntime, "stackit.objectStorage")
	if err != nil {
		return nil, err
	}
	return res.(*mqlStackitObjectStorage), nil
}

func (r *mqlStackit) dns() (*mqlStackitDns, error) {
	res, err := makeNamespace(r.MqlRuntime, "stackit.dns")
	if err != nil {
		return nil, err
	}
	return res.(*mqlStackitDns), nil
}

func (r *mqlStackit) postgresFlex() (*mqlStackitPostgresFlex, error) {
	res, err := makeNamespace(r.MqlRuntime, "stackit.postgresFlex")
	if err != nil {
		return nil, err
	}
	return res.(*mqlStackitPostgresFlex), nil
}

func (r *mqlStackit) mongoDbFlex() (*mqlStackitMongoDbFlex, error) {
	res, err := makeNamespace(r.MqlRuntime, "stackit.mongoDbFlex")
	if err != nil {
		return nil, err
	}
	return res.(*mqlStackitMongoDbFlex), nil
}

func (r *mqlStackit) openSearch() (*mqlStackitOpenSearch, error) {
	res, err := makeNamespace(r.MqlRuntime, "stackit.openSearch")
	if err != nil {
		return nil, err
	}
	return res.(*mqlStackitOpenSearch), nil
}

func (r *mqlStackit) sqlServerFlex() (*mqlStackitSqlServerFlex, error) {
	res, err := makeNamespace(r.MqlRuntime, "stackit.sqlServerFlex")
	if err != nil {
		return nil, err
	}
	return res.(*mqlStackitSqlServerFlex), nil
}

func (r *mqlStackit) logMe() (*mqlStackitLogMe, error) {
	res, err := makeNamespace(r.MqlRuntime, "stackit.logMe")
	if err != nil {
		return nil, err
	}
	return res.(*mqlStackitLogMe), nil
}

func (r *mqlStackit) mariaDb() (*mqlStackitMariaDb, error) {
	res, err := makeNamespace(r.MqlRuntime, "stackit.mariaDb")
	if err != nil {
		return nil, err
	}
	return res.(*mqlStackitMariaDb), nil
}

func (r *mqlStackit) redis() (*mqlStackitRedis, error) {
	res, err := makeNamespace(r.MqlRuntime, "stackit.redis")
	if err != nil {
		return nil, err
	}
	return res.(*mqlStackitRedis), nil
}

func (r *mqlStackit) rabbitMq() (*mqlStackitRabbitMq, error) {
	res, err := makeNamespace(r.MqlRuntime, "stackit.rabbitMq")
	if err != nil {
		return nil, err
	}
	return res.(*mqlStackitRabbitMq), nil
}

func (r *mqlStackit) secretsManager() (*mqlStackitSecretsManager, error) {
	res, err := makeNamespace(r.MqlRuntime, "stackit.secretsManager")
	if err != nil {
		return nil, err
	}
	return res.(*mqlStackitSecretsManager), nil
}

func (r *mqlStackit) observability() (*mqlStackitObservability, error) {
	res, err := makeNamespace(r.MqlRuntime, "stackit.observability")
	if err != nil {
		return nil, err
	}
	return res.(*mqlStackitObservability), nil
}

func (r *mqlStackit) modelServing() (*mqlStackitModelServing, error) {
	res, err := makeNamespace(r.MqlRuntime, "stackit.modelServing")
	if err != nil {
		return nil, err
	}
	return res.(*mqlStackitModelServing), nil
}

func (r *mqlStackit) telemetry() (*mqlStackitTelemetry, error) {
	res, err := makeNamespace(r.MqlRuntime, "stackit.telemetry")
	if err != nil {
		return nil, err
	}
	return res.(*mqlStackitTelemetry), nil
}

func (r *mqlStackit) sfs() (*mqlStackitSfs, error) {
	res, err := makeNamespace(r.MqlRuntime, "stackit.sfs")
	if err != nil {
		return nil, err
	}
	return res.(*mqlStackitSfs), nil
}

func (r *mqlStackit) kms() (*mqlStackitKms, error) {
	res, err := makeNamespace(r.MqlRuntime, "stackit.kms")
	if err != nil {
		return nil, err
	}
	return res.(*mqlStackitKms), nil
}

func (r *mqlStackit) iam() (*mqlStackitIam, error) {
	res, err := makeNamespace(r.MqlRuntime, "stackit.iam")
	if err != nil {
		return nil, err
	}
	return res.(*mqlStackitIam), nil
}
