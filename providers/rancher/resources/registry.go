// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"sort"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/types"
)

// Registry credentials live under the project API rather than the management
// API, and each project answers for its own. A credential bound to the project
// is copied into every namespace of that project; a namespaced one exists in
// exactly one. Both are read, because the difference between them is the reach
// this resource is here to report.
const (
	scopeProject   = "project"
	scopeNamespace = "namespace"
)

// mqlRancherRegistryCredentialInternal carries the project a credential belongs
// to.
type mqlRancherRegistryCredentialInternal struct {
	cacheProjectID string
}

func (r *mqlRancher) registryCredentials() ([]any, error) {
	projects, err := listRecords[projectRecord](r.MqlRuntime, pathProjects)
	if err != nil {
		return nil, err
	}

	res := []any{}
	for i := range projects {
		projectID := projects[i].ID
		if projectID == "" {
			continue
		}

		for _, source := range []struct {
			path  string
			scope string
		}{
			{"/v3/project/" + projectID + "/dockerCredentials", scopeProject},
			{"/v3/project/" + projectID + "/namespacedDockerCredentials", scopeNamespace},
		} {
			// A project the token cannot read its credentials for, or a Rancher
			// that does not serve the namespaced collection, must not take the
			// whole listing down with it. Only a 404 is skipped silently; a
			// refusal or a transport failure still surfaces.
			records, err := listOptionalRecords[dockerCredentialRecord](r.MqlRuntime, source.path)
			if err != nil {
				return nil, err
			}

			for j := range records {
				mqlCredential, err := buildRegistryCredential(r.MqlRuntime, projectID, &records[j])
				if err != nil {
					return nil, err
				}
				res = append(res, mqlCredential)
			}
		}
	}
	return res, nil
}

func buildRegistryCredential(runtime *plugin.Runtime, projectID string, record *dockerCredentialRecord) (*mqlRancherRegistryCredential, error) {
	// Only the registry host and the user name are taken from the credential.
	// The password and the pre-encoded auth header the API also carries are not
	// decoded anywhere in this provider, so there is nothing for a query, a
	// report, or a recording to leak.
	registries := make([]string, 0, len(record.Registries))
	usernames := make(map[string]any, len(record.Registries))
	for host, account := range record.Registries {
		registries = append(registries, host)
		usernames[host] = account.Username
	}
	sort.Strings(registries)

	scope := scopeProject
	if record.NamespaceID != "" {
		scope = scopeNamespace
	}

	resource, err := CreateResource(runtime, "rancher.registryCredential", map[string]*llx.RawData{
		// The credential's own id repeats across projects for a namespaced
		// credential, since it is only namespace-qualified. Adding the project
		// keeps two credentials of the same name in different projects from
		// collapsing onto one another in the resource cache.
		"__id":              llx.StringData(projectID + "/" + record.ID),
		"id":                llx.StringData(record.ID),
		"name":              llx.StringData(record.Name),
		"description":       llx.StringData(record.Description),
		"created":           llx.TimeDataPtr(parseTime(record.Created)),
		"scope":             llx.StringData(scope),
		"namespaceName":     llx.StringData(record.NamespaceID),
		"registries":        llx.ArrayData(toAnySlice(registries), types.String),
		"registryUsernames": llx.MapData(usernames, types.String),
	})
	if err != nil {
		return nil, err
	}

	mqlCredential := resource.(*mqlRancherRegistryCredential)
	mqlCredential.cacheProjectID = projectID
	return mqlCredential, nil
}

func (r *mqlRancherRegistryCredential) project() (*mqlRancherProject, error) {
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
