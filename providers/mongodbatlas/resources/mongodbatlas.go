// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"sync"
	"time"

	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/mongodbatlas/connection"
	"go.mongodb.org/atlas-sdk/v20250312006/admin"
)

type mqlMongodbatlasInternal struct {
	orgSettingsLock sync.Mutex
	orgSettingsDone bool
	orgSettings     *admin.OrganizationSettings
}

func (r *mqlMongodbatlas) id() (string, error) {
	conn := r.MqlRuntime.Connection.(*connection.MongoDBAtlasConnection)
	if conn.Plane() == connection.PlaneProject {
		return "mongodbatlas/project/" + conn.ProjectID(), nil
	}
	return "mongodbatlas/org/" + conn.OrgID(), nil
}

func atlasClient(runtime *plugin.Runtime) *admin.APIClient {
	return runtime.Connection.(*connection.MongoDBAtlasConnection).Client()
}

// orgID returns the connected organization id, deriving it once (race-safe)
// from the accessible organizations when it was not supplied.
func orgID(runtime *plugin.Runtime) (string, error) {
	conn := runtime.Connection.(*connection.MongoDBAtlasConnection)
	return conn.EnsureOrgID(context.Background())
}

// projectID returns the connected project id, or an error when the asset is an
// organization rather than a single project.
func projectID(runtime *plugin.Runtime) (string, error) {
	conn := runtime.Connection.(*connection.MongoDBAtlasConnection)
	if conn.Plane() != connection.PlaneProject {
		return "", errors.New("this resource requires connecting to a single Atlas project (use --project-id or query a discovered project asset)")
	}
	return conn.ProjectID(), nil
}

// fetchOrgSettings loads the organization settings once and caches them for the
// several settings fields on the root resource.
func (r *mqlMongodbatlas) fetchOrgSettings() (*admin.OrganizationSettings, error) {
	r.orgSettingsLock.Lock()
	defer r.orgSettingsLock.Unlock()
	if r.orgSettingsDone {
		return r.orgSettings, nil
	}
	oid, err := orgID(r.MqlRuntime)
	if err != nil {
		return nil, err
	}
	settings, _, err := atlasClient(r.MqlRuntime).OrganizationsApi.GetOrganizationSettings(context.Background(), oid).Execute()
	if err != nil {
		return nil, err
	}
	r.orgSettings = settings
	r.orgSettingsDone = true
	return settings, nil
}

func (r *mqlMongodbatlas) organizationId() (string, error) {
	return orgID(r.MqlRuntime)
}

func (r *mqlMongodbatlas) organizationName() (string, error) {
	oid, err := orgID(r.MqlRuntime)
	if err != nil {
		return "", err
	}
	org, _, err := atlasClient(r.MqlRuntime).OrganizationsApi.GetOrganization(context.Background(), oid).Execute()
	if err != nil {
		return "", err
	}
	return org.GetName(), nil
}

func (r *mqlMongodbatlas) multiFactorAuthRequired() (bool, error) {
	s, err := r.fetchOrgSettings()
	if err != nil {
		return false, err
	}
	return s.GetMultiFactorAuthRequired(), nil
}

func (r *mqlMongodbatlas) apiAccessListRequired() (bool, error) {
	s, err := r.fetchOrgSettings()
	if err != nil {
		return false, err
	}
	return s.GetApiAccessListRequired(), nil
}

func (r *mqlMongodbatlas) restrictEmployeeAccess() (bool, error) {
	s, err := r.fetchOrgSettings()
	if err != nil {
		return false, err
	}
	return s.GetRestrictEmployeeAccess(), nil
}

func (r *mqlMongodbatlas) genAiFeaturesEnabled() (bool, error) {
	s, err := r.fetchOrgSettings()
	if err != nil {
		return false, err
	}
	return s.GetGenAIFeaturesEnabled(), nil
}

func (r *mqlMongodbatlas) maxServiceAccountSecretValidityInHours() (int64, error) {
	s, err := r.fetchOrgSettings()
	if err != nil {
		return 0, err
	}
	return int64(s.GetMaxServiceAccountSecretValidityInHours()), nil
}

func (r *mqlMongodbatlas) securityContact() (string, error) {
	s, err := r.fetchOrgSettings()
	if err != nil {
		return "", err
	}
	return s.GetSecurityContact(), nil
}

// timePtr returns a pointer to t, or nil for the zero time, for llx.TimeDataPtr.
func timePtr(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}

// strSlice converts a string slice to the []any form llx.ArrayData expects.
func strSlice(vals []string) []any {
	out := make([]any, 0, len(vals))
	for _, v := range vals {
		out = append(out, v)
	}
	return out
}
