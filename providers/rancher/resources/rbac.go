// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/types"
)

// mqlRancherGlobalRoleInternal carries the rules the derived predicates are
// computed from, and the role templates the role inherits in downstream
// clusters. The rules are kept alongside the dict copy so the predicates read
// typed values rather than parsing their own field back out.
type mqlRancherGlobalRoleInternal struct {
	cacheRules                 []policyRule
	cacheInheritedClusterRoles []string
}

// mqlRancherRoleTemplateInternal carries the same for a role template, plus the
// templates it inherits rules from.
type mqlRancherRoleTemplateInternal struct {
	cacheRules             []policyRule
	cacheInheritedTemplate []string
}

// mqlRancherGlobalRoleBindingInternal carries the subject and role a binding
// names, both of which are resolved from a shared listing when asked for.
type mqlRancherGlobalRoleBindingInternal struct {
	cacheUserID       string
	cacheGlobalRoleID string
}

// mqlRancherClusterRoleTemplateBindingInternal carries the cluster, role
// template and user a cluster binding names.
type mqlRancherClusterRoleTemplateBindingInternal struct {
	cacheUserID         string
	cacheRoleTemplateID string
	cacheClusterID      string
}

// mqlRancherProjectRoleTemplateBindingInternal carries the project, role
// template and user a project binding names.
type mqlRancherProjectRoleTemplateBindingInternal struct {
	cacheUserID         string
	cacheRoleTemplateID string
	cacheProjectID      string
}

// -- global roles -----------------------------------------------------------

func (r *mqlRancher) globalRoles() ([]any, error) {
	records, err := listRecords[globalRoleRecord](r.MqlRuntime, pathGlobalRoles)
	if err != nil {
		return nil, err
	}

	res := make([]any, 0, len(records))
	for i := range records {
		mqlRole, err := buildGlobalRole(r.MqlRuntime, &records[i])
		if err != nil {
			return nil, err
		}
		res = append(res, mqlRole)
	}
	return res, nil
}

func buildGlobalRole(runtime *plugin.Runtime, record *globalRoleRecord) (*mqlRancherGlobalRole, error) {
	resource, err := CreateResource(runtime, "rancher.globalRole", map[string]*llx.RawData{
		"__id":                     llx.StringData(record.ID),
		"id":                       llx.StringData(record.ID),
		"name":                     llx.StringData(record.Name),
		"description":              llx.StringData(record.Description),
		"builtin":                  llx.BoolData(record.Builtin),
		"newUserDefault":           llx.BoolData(record.NewUserDefault),
		"created":                  llx.TimeDataPtr(parseTime(record.Created)),
		"rules":                    llx.ArrayData(rulesToDicts(record.Rules), types.Dict),
		"namespacedRules":          dictOrNil(namespacedRulesToDict(record.NamespacedRules)),
		"inheritedNamespacedRules": dictOrNil(namespacedRulesToDict(record.InheritedNamespacedRules)),
	})
	if err != nil {
		return nil, err
	}

	mqlRole := resource.(*mqlRancherGlobalRole)
	mqlRole.cacheRules = record.Rules
	mqlRole.cacheInheritedClusterRoles = record.InheritedClusterRoles
	return mqlRole, nil
}

func globalRoleByID(runtime *plugin.Runtime, id string) (*mqlRancherGlobalRole, error) {
	if id == "" {
		return nil, nil
	}
	records, err := listRecords[globalRoleRecord](runtime, pathGlobalRoles)
	if err != nil {
		return nil, err
	}
	for i := range records {
		if records[i].ID == id {
			return buildGlobalRole(runtime, &records[i])
		}
	}
	return nil, nil
}

func (r *mqlRancherGlobalRole) grantsFullAdmin() (bool, error) {
	return grantsFullAdmin(r.cacheRules), nil
}

func (r *mqlRancherGlobalRole) grantsPrivilegeEscalation() (bool, error) {
	return grantsPrivilegeEscalation(r.cacheRules), nil
}

func (r *mqlRancherGlobalRole) inheritedClusterRoleTemplates() ([]any, error) {
	return resolveRoleTemplates(r.MqlRuntime, r.cacheInheritedClusterRoles)
}

func (r *mqlRancherGlobalRole) bindings() ([]any, error) {
	records, err := listRecords[globalRoleBindingRecord](r.MqlRuntime, pathGlobalRoleBindings)
	if err != nil {
		return nil, err
	}

	roleID := r.Id.Data
	res := []any{}
	for i := range records {
		if records[i].GlobalRoleID != roleID {
			continue
		}
		mqlBinding, err := buildGlobalRoleBinding(r.MqlRuntime, &records[i])
		if err != nil {
			return nil, err
		}
		res = append(res, mqlBinding)
	}
	return res, nil
}

// -- global role bindings ---------------------------------------------------

func (r *mqlRancher) globalRoleBindings() ([]any, error) {
	records, err := listRecords[globalRoleBindingRecord](r.MqlRuntime, pathGlobalRoleBindings)
	if err != nil {
		return nil, err
	}

	res := make([]any, 0, len(records))
	for i := range records {
		mqlBinding, err := buildGlobalRoleBinding(r.MqlRuntime, &records[i])
		if err != nil {
			return nil, err
		}
		res = append(res, mqlBinding)
	}
	return res, nil
}

func buildGlobalRoleBinding(runtime *plugin.Runtime, record *globalRoleBindingRecord) (*mqlRancherGlobalRoleBinding, error) {
	resource, err := CreateResource(runtime, "rancher.globalRoleBinding", map[string]*llx.RawData{
		"__id":            llx.StringData(record.ID),
		"id":              llx.StringData(record.ID),
		"created":         llx.TimeDataPtr(parseTime(record.Created)),
		"subjectKind":     llx.StringData(subjectKind(record.UserID, record.UserPrincipalID, "", record.GroupPrincipalID, "")),
		"subjectName":     llx.StringData(subjectName(record.UserID, record.UserPrincipalID, "", record.GroupPrincipalID, "")),
		"userPrincipalId": llx.StringData(record.UserPrincipalID),
	})
	if err != nil {
		return nil, err
	}

	mqlBinding := resource.(*mqlRancherGlobalRoleBinding)
	mqlBinding.cacheUserID = record.UserID
	mqlBinding.cacheGlobalRoleID = record.GlobalRoleID
	return mqlBinding, nil
}

func (r *mqlRancherGlobalRoleBinding) globalRole() (*mqlRancherGlobalRole, error) {
	mqlRole, err := globalRoleByID(r.MqlRuntime, r.cacheGlobalRoleID)
	if err != nil {
		return nil, err
	}
	if mqlRole == nil {
		r.GlobalRole.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	return mqlRole, nil
}

func (r *mqlRancherGlobalRoleBinding) user() (*mqlRancherUser, error) {
	mqlUser, err := userByID(r.MqlRuntime, r.cacheUserID)
	if err != nil {
		return nil, err
	}
	if mqlUser == nil {
		r.User.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	return mqlUser, nil
}

// -- role templates ---------------------------------------------------------

func (r *mqlRancher) roleTemplates() ([]any, error) {
	records, err := listRecords[roleTemplateRecord](r.MqlRuntime, pathRoleTemplates)
	if err != nil {
		return nil, err
	}

	res := make([]any, 0, len(records))
	for i := range records {
		mqlTemplate, err := buildRoleTemplate(r.MqlRuntime, &records[i])
		if err != nil {
			return nil, err
		}
		res = append(res, mqlTemplate)
	}
	return res, nil
}

func buildRoleTemplate(runtime *plugin.Runtime, record *roleTemplateRecord) (*mqlRancherRoleTemplate, error) {
	resource, err := CreateResource(runtime, "rancher.roleTemplate", map[string]*llx.RawData{
		"__id":                  llx.StringData(record.ID),
		"id":                    llx.StringData(record.ID),
		"name":                  llx.StringData(record.Name),
		"description":           llx.StringData(record.Description),
		"context":               llx.StringData(record.Context),
		"builtin":               llx.BoolData(record.Builtin),
		"external":              llx.BoolData(record.External),
		"hidden":                llx.BoolData(record.Hidden),
		"locked":                llx.BoolData(record.Locked),
		"clusterCreatorDefault": llx.BoolData(record.ClusterCreatorDefault),
		"projectCreatorDefault": llx.BoolData(record.ProjectCreatorDefault),
		"administrative":        llx.BoolData(record.Administrative),
		"created":               llx.TimeDataPtr(parseTime(record.Created)),
		"rules":                 llx.ArrayData(rulesToDicts(record.Rules), types.Dict),
		"externalRules":         llx.ArrayData(rulesToDicts(record.ExternalRules), types.Dict),
	})
	if err != nil {
		return nil, err
	}

	mqlTemplate := resource.(*mqlRancherRoleTemplate)
	mqlTemplate.cacheRules = effectiveRules(record)
	mqlTemplate.cacheInheritedTemplate = record.RoleTemplateIDs
	return mqlTemplate, nil
}

func roleTemplateByID(runtime *plugin.Runtime, id string) (*mqlRancherRoleTemplate, error) {
	if id == "" {
		return nil, nil
	}
	records, err := listRecords[roleTemplateRecord](runtime, pathRoleTemplates)
	if err != nil {
		return nil, err
	}
	for i := range records {
		if records[i].ID == id {
			return buildRoleTemplate(runtime, &records[i])
		}
	}
	return nil, nil
}

// resolveRoleTemplates turns a list of role template names into resources. A
// name that does not resolve is left out rather than failing the whole list,
// because Rancher keeps a binding to a template that was deleted.
func resolveRoleTemplates(runtime *plugin.Runtime, ids []string) ([]any, error) {
	res := []any{}
	for _, id := range ids {
		mqlTemplate, err := roleTemplateByID(runtime, id)
		if err != nil {
			return nil, err
		}
		if mqlTemplate == nil {
			continue
		}
		res = append(res, mqlTemplate)
	}
	return res, nil
}

func (r *mqlRancherRoleTemplate) grantsFullAdmin() (bool, error) {
	return grantsFullAdmin(r.cacheRules), nil
}

func (r *mqlRancherRoleTemplate) grantsPrivilegeEscalation() (bool, error) {
	return grantsPrivilegeEscalation(r.cacheRules), nil
}

func (r *mqlRancherRoleTemplate) inheritedRoleTemplates() ([]any, error) {
	return resolveRoleTemplates(r.MqlRuntime, r.cacheInheritedTemplate)
}

// -- cluster role template bindings -----------------------------------------

func (r *mqlRancher) clusterRoleTemplateBindings() ([]any, error) {
	records, err := listRecords[bindingRecord](r.MqlRuntime, pathClusterRoleTemplateBinding)
	if err != nil {
		return nil, err
	}

	res := make([]any, 0, len(records))
	for i := range records {
		mqlBinding, err := buildClusterRoleTemplateBinding(r.MqlRuntime, &records[i])
		if err != nil {
			return nil, err
		}
		res = append(res, mqlBinding)
	}
	return res, nil
}

func buildClusterRoleTemplateBinding(runtime *plugin.Runtime, record *bindingRecord) (*mqlRancherClusterRoleTemplateBinding, error) {
	resource, err := CreateResource(runtime, "rancher.clusterRoleTemplateBinding", map[string]*llx.RawData{
		"__id":            llx.StringData(record.ID),
		"id":              llx.StringData(record.ID),
		"created":         llx.TimeDataPtr(parseTime(record.Created)),
		"subjectKind":     llx.StringData(subjectKind(record.UserID, record.UserPrincipalID, record.GroupID, record.GroupPrincipalID, "")),
		"subjectName":     llx.StringData(subjectName(record.UserID, record.UserPrincipalID, record.GroupID, record.GroupPrincipalID, "")),
		"userPrincipalId": llx.StringData(record.UserPrincipalID),
	})
	if err != nil {
		return nil, err
	}

	mqlBinding := resource.(*mqlRancherClusterRoleTemplateBinding)
	mqlBinding.cacheUserID = record.UserID
	mqlBinding.cacheRoleTemplateID = record.RoleTemplateID
	mqlBinding.cacheClusterID = record.ClusterID
	return mqlBinding, nil
}

func (r *mqlRancherClusterRoleTemplateBinding) cluster() (*mqlRancherCluster, error) {
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

func (r *mqlRancherClusterRoleTemplateBinding) roleTemplate() (*mqlRancherRoleTemplate, error) {
	mqlTemplate, err := roleTemplateByID(r.MqlRuntime, r.cacheRoleTemplateID)
	if err != nil {
		return nil, err
	}
	if mqlTemplate == nil {
		r.RoleTemplate.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	return mqlTemplate, nil
}

func (r *mqlRancherClusterRoleTemplateBinding) user() (*mqlRancherUser, error) {
	mqlUser, err := userByID(r.MqlRuntime, r.cacheUserID)
	if err != nil {
		return nil, err
	}
	if mqlUser == nil {
		r.User.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	return mqlUser, nil
}

// -- project role template bindings -----------------------------------------

func (r *mqlRancher) projectRoleTemplateBindings() ([]any, error) {
	records, err := listRecords[bindingRecord](r.MqlRuntime, pathProjectRoleTemplateBinding)
	if err != nil {
		return nil, err
	}

	res := make([]any, 0, len(records))
	for i := range records {
		mqlBinding, err := buildProjectRoleTemplateBinding(r.MqlRuntime, &records[i])
		if err != nil {
			return nil, err
		}
		res = append(res, mqlBinding)
	}
	return res, nil
}

func buildProjectRoleTemplateBinding(runtime *plugin.Runtime, record *bindingRecord) (*mqlRancherProjectRoleTemplateBinding, error) {
	resource, err := CreateResource(runtime, "rancher.projectRoleTemplateBinding", map[string]*llx.RawData{
		"__id":            llx.StringData(record.ID),
		"id":              llx.StringData(record.ID),
		"created":         llx.TimeDataPtr(parseTime(record.Created)),
		"subjectKind":     llx.StringData(subjectKind(record.UserID, record.UserPrincipalID, record.GroupID, record.GroupPrincipalID, record.ServiceAccount)),
		"subjectName":     llx.StringData(subjectName(record.UserID, record.UserPrincipalID, record.GroupID, record.GroupPrincipalID, record.ServiceAccount)),
		"userPrincipalId": llx.StringData(record.UserPrincipalID),
	})
	if err != nil {
		return nil, err
	}

	mqlBinding := resource.(*mqlRancherProjectRoleTemplateBinding)
	mqlBinding.cacheUserID = record.UserID
	mqlBinding.cacheRoleTemplateID = record.RoleTemplateID
	mqlBinding.cacheProjectID = record.ProjectID
	return mqlBinding, nil
}

func (r *mqlRancherProjectRoleTemplateBinding) project() (*mqlRancherProject, error) {
	mqlProject, err := projectByID(r.MqlRuntime, r.cacheProjectID)
	if err != nil {
		return nil, err
	}
	if mqlProject == nil {
		r.Project.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	return mqlProject, nil
}

func (r *mqlRancherProjectRoleTemplateBinding) roleTemplate() (*mqlRancherRoleTemplate, error) {
	mqlTemplate, err := roleTemplateByID(r.MqlRuntime, r.cacheRoleTemplateID)
	if err != nil {
		return nil, err
	}
	if mqlTemplate == nil {
		r.RoleTemplate.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	return mqlTemplate, nil
}

func (r *mqlRancherProjectRoleTemplateBinding) user() (*mqlRancherUser, error) {
	mqlUser, err := userByID(r.MqlRuntime, r.cacheUserID)
	if err != nil {
		return nil, err
	}
	if mqlUser == nil {
		r.User.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	return mqlUser, nil
}
