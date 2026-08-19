// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/types"
)

// mqlRancherClusterTemplateInternal carries the revision a template offers by
// default, which is resolved from the shared revision listing when asked for.
type mqlRancherClusterTemplateInternal struct {
	cacheDefaultRevisionID string
}

// mqlRancherClusterTemplateRevisionInternal carries the template a revision
// belongs to.
type mqlRancherClusterTemplateRevisionInternal struct {
	cacheClusterTemplateID string
}

// -- cluster templates ------------------------------------------------------

func (r *mqlRancher) clusterTemplates() ([]any, error) {
	// Cluster templates were removed in Rancher 2.12 along with RKE1, so the
	// endpoint is simply not there on a newer server. Only a 404 is read that
	// way; a refused or unreachable server still fails.
	records, err := listOptionalRecords[clusterTemplateRecord](r.MqlRuntime, pathClusterTemplates)
	if err != nil {
		return nil, err
	}

	res := make([]any, 0, len(records))
	for i := range records {
		mqlTemplate, err := buildClusterTemplate(r.MqlRuntime, &records[i])
		if err != nil {
			return nil, err
		}
		res = append(res, mqlTemplate)
	}
	return res, nil
}

func buildClusterTemplate(runtime *plugin.Runtime, record *clusterTemplateRecord) (*mqlRancherClusterTemplate, error) {
	members := make([]any, 0, len(record.Members))
	for _, member := range record.Members {
		members = append(members, member)
	}

	resource, err := CreateResource(runtime, "rancher.clusterTemplate", map[string]*llx.RawData{
		"__id":        llx.StringData(record.ID),
		"id":          llx.StringData(record.ID),
		"name":        llx.StringData(record.Name),
		"description": llx.StringData(record.Description),
		"created":     llx.TimeDataPtr(parseTime(record.Created)),
		"members":     llx.ArrayData(members, types.Dict),
	})
	if err != nil {
		return nil, err
	}

	mqlTemplate := resource.(*mqlRancherClusterTemplate)
	mqlTemplate.cacheDefaultRevisionID = record.DefaultRevisionID
	return mqlTemplate, nil
}

func clusterTemplateByID(runtime *plugin.Runtime, id string) (*mqlRancherClusterTemplate, error) {
	if id == "" {
		return nil, nil
	}
	records, err := listOptionalRecords[clusterTemplateRecord](runtime, pathClusterTemplates)
	if err != nil {
		return nil, err
	}
	for i := range records {
		if records[i].ID == id {
			return buildClusterTemplate(runtime, &records[i])
		}
	}
	return nil, nil
}

func (r *mqlRancherClusterTemplate) revisions() ([]any, error) {
	records, err := listOptionalRecords[clusterTemplateRevisionRecord](r.MqlRuntime, pathClusterTemplateRevisions)
	if err != nil {
		return nil, err
	}

	templateID := r.Id.Data
	res := []any{}
	for i := range records {
		if records[i].ClusterTemplateID != templateID {
			continue
		}
		mqlRevision, err := buildClusterTemplateRevision(r.MqlRuntime, &records[i])
		if err != nil {
			return nil, err
		}
		res = append(res, mqlRevision)
	}
	return res, nil
}

func (r *mqlRancherClusterTemplate) defaultRevision() (*mqlRancherClusterTemplateRevision, error) {
	if r.cacheDefaultRevisionID == "" {
		r.DefaultRevision.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}

	records, err := listOptionalRecords[clusterTemplateRevisionRecord](r.MqlRuntime, pathClusterTemplateRevisions)
	if err != nil {
		return nil, err
	}
	for i := range records {
		if records[i].ID == r.cacheDefaultRevisionID {
			return buildClusterTemplateRevision(r.MqlRuntime, &records[i])
		}
	}

	r.DefaultRevision.State = plugin.StateIsSet | plugin.StateIsNull
	return nil, nil
}

func buildClusterTemplateRevision(runtime *plugin.Runtime, record *clusterTemplateRevisionRecord) (*mqlRancherClusterTemplateRevision, error) {
	questions := make([]any, 0, len(record.Questions))
	for _, question := range record.Questions {
		questions = append(questions, question)
	}

	resource, err := CreateResource(runtime, "rancher.clusterTemplate.revision", map[string]*llx.RawData{
		"__id":      llx.StringData(record.ID),
		"id":        llx.StringData(record.ID),
		"name":      llx.StringData(record.Name),
		"created":   llx.TimeDataPtr(parseTime(record.Created)),
		"state":     llx.StringData(record.State),
		"enabled":   llx.BoolDataPtr(record.Enabled),
		"questions": llx.ArrayData(questions, types.Dict),
	})
	if err != nil {
		return nil, err
	}

	mqlRevision := resource.(*mqlRancherClusterTemplateRevision)
	mqlRevision.cacheClusterTemplateID = record.ClusterTemplateID
	return mqlRevision, nil
}

func (r *mqlRancherClusterTemplateRevision) clusterTemplate() (*mqlRancherClusterTemplate, error) {
	mqlTemplate, err := clusterTemplateByID(r.MqlRuntime, r.cacheClusterTemplateID)
	if err != nil {
		return nil, err
	}
	if mqlTemplate == nil {
		r.ClusterTemplate.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	return mqlTemplate, nil
}

// -- pod security admission configuration templates -------------------------

func (r *mqlRancher) podSecurityAdmissionConfigurationTemplates() ([]any, error) {
	records, err := listRecords[podSecurityTemplateRecord](r.MqlRuntime, pathPodSecurityTemplates)
	if err != nil {
		return nil, err
	}

	res := make([]any, 0, len(records))
	for i := range records {
		mqlTemplate, err := buildPodSecurityTemplate(r.MqlRuntime, &records[i])
		if err != nil {
			return nil, err
		}
		res = append(res, mqlTemplate)
	}
	return res, nil
}

func buildPodSecurityTemplate(runtime *plugin.Runtime, record *podSecurityTemplateRecord) (*mqlRancherPodSecurityAdmissionConfigurationTemplate, error) {
	var defaults podSecurityDefaults
	var exemptions podSecurityExemptions
	if record.Configuration != nil {
		if record.Configuration.Defaults != nil {
			defaults = *record.Configuration.Defaults
		}
		if record.Configuration.Exemptions != nil {
			exemptions = *record.Configuration.Exemptions
		}
	}

	name := record.Name
	if name == "" {
		name = record.ID
	}

	resource, err := CreateResource(runtime, "rancher.podSecurityAdmissionConfigurationTemplate", map[string]*llx.RawData{
		"__id":                 llx.StringData(name),
		"name":                 llx.StringData(name),
		"description":          llx.StringData(record.Description),
		"created":              llx.TimeDataPtr(parseTime(record.Created)),
		"enforce":              llx.StringDataPtr(nilIfEmpty(defaults.Enforce)),
		"enforceVersion":       llx.StringDataPtr(nilIfEmpty(defaults.EnforceVersion)),
		"audit":                llx.StringDataPtr(nilIfEmpty(defaults.Audit)),
		"auditVersion":         llx.StringDataPtr(nilIfEmpty(defaults.AuditVersion)),
		"warn":                 llx.StringDataPtr(nilIfEmpty(defaults.Warn)),
		"warnVersion":          llx.StringDataPtr(nilIfEmpty(defaults.WarnVersion)),
		"exemptNamespaces":     llx.ArrayData(toAnySlice(exemptions.Namespaces), types.String),
		"exemptUsernames":      llx.ArrayData(toAnySlice(exemptions.Usernames), types.String),
		"exemptRuntimeClasses": llx.ArrayData(toAnySlice(exemptions.RuntimeClasses), types.String),
	})
	if err != nil {
		return nil, err
	}
	return resource.(*mqlRancherPodSecurityAdmissionConfigurationTemplate), nil
}

func podSecurityTemplateByName(runtime *plugin.Runtime, name string) (*mqlRancherPodSecurityAdmissionConfigurationTemplate, error) {
	if name == "" {
		return nil, nil
	}
	records, err := listRecords[podSecurityTemplateRecord](runtime, pathPodSecurityTemplates)
	if err != nil {
		return nil, err
	}
	for i := range records {
		key := records[i].Name
		if key == "" {
			key = records[i].ID
		}
		if key == name {
			return buildPodSecurityTemplate(runtime, &records[i])
		}
	}
	return nil, nil
}

func (r *mqlRancherPodSecurityAdmissionConfigurationTemplate) clusters() ([]any, error) {
	records, err := listRecords[clusterRecord](r.MqlRuntime, pathClusters)
	if err != nil {
		return nil, err
	}

	name := r.Name.Data
	res := []any{}
	for i := range records {
		if records[i].PodSecurityTemplateName != name {
			continue
		}
		mqlCluster, err := buildCluster(r.MqlRuntime, &records[i])
		if err != nil {
			return nil, err
		}
		res = append(res, mqlCluster)
	}
	return res, nil
}
