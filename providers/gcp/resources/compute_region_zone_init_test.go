// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.mondoo.com/mql/llx"
	"google.golang.org/api/cloudresourcemanager/v3"
	"google.golang.org/api/compute/v1"
)

func regionZoneScopes() []string {
	return []string{cloudresourcemanager.CloudPlatformReadOnlyScope, compute.ComputeReadonlyScope}
}

// TestRegionInitDoesNotLeakProjectIdIntoArgs pins the two ways a region or zone
// lookup can end.
//
// projectId scopes the lookup but is not a field on either resource. The init
// used to write it into args and then, on any path that returned without a
// resource, hand those args to SetAllData -- which rejects an unknown field. A
// bare `gcp.project.computeService.region` therefore failed with
//
//	[gcp] cannot set 'projectId' in resource
//	'gcp.project.computeService.region', field not found
//
// which names an internal mechanism rather than the missing argument. Dropping
// projectId from args without also reporting the missing name only traded that
// for an empty husk whose every field is unset, which surfaces as "encountered
// a primitive with no type information" with nothing naming the cause. Both
// were verified against a live project.
func TestRegionInitDoesNotLeakProjectIdIntoArgs(t *testing.T) {
	t.Run("region without a name says so", func(t *testing.T) {
		env := setupTestEnv(t, regionZoneScopes())
		_, err := NewResource(env.Runtime, "gcp.project.computeService.region", map[string]*llx.RawData{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), `requires a "name" argument`)
		assert.NotContains(t, err.Error(), "field not found",
			"the caller should learn which argument is missing, not that an internal setter rejected a key")
	})

	t.Run("zone without a name says so", func(t *testing.T) {
		env := setupTestEnv(t, regionZoneScopes())
		_, err := NewResource(env.Runtime, "gcp.project.computeService.zone", map[string]*llx.RawData{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), `requires a "name" argument`)
		assert.NotContains(t, err.Error(), "field not found")
	})

	t.Run("region with a name and a projectId resolves", func(t *testing.T) {
		env := setupTestEnv(t, regionZoneScopes())
		env.Mux.HandleFunc("/compute/v1/projects/"+testProjectId+"/regions/us-central1",
			func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprint(w, `{"id": "1000", "name": "us-central1", "status": "UP", "description": "us-central1"}`)
			})

		res, err := NewResource(env.Runtime, "gcp.project.computeService.region", map[string]*llx.RawData{
			"name":      llx.StringData("us-central1"),
			"projectId": llx.StringData(testProjectId),
		})
		require.NoError(t, err)
		region := res.(*mqlGcpProjectComputeServiceRegion)
		assert.Equal(t, "us-central1", region.Name.Data)
		assert.Equal(t, "UP", region.Status.Data)
	})

	t.Run("zone with a name and a projectId resolves", func(t *testing.T) {
		env := setupTestEnv(t, regionZoneScopes())
		env.Mux.HandleFunc("/compute/v1/projects/"+testProjectId+"/zones/us-central1-a",
			func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprint(w, `{"id": "2000", "name": "us-central1-a", "status": "UP", "description": "us-central1-a"}`)
			})

		res, err := NewResource(env.Runtime, "gcp.project.computeService.zone", map[string]*llx.RawData{
			"name":      llx.StringData("us-central1-a"),
			"projectId": llx.StringData(testProjectId),
		})
		require.NoError(t, err)
		zone := res.(*mqlGcpProjectComputeServiceZone)
		assert.Equal(t, "us-central1-a", zone.Name.Data)
		assert.Equal(t, "UP", zone.Status.Data)
	})
}

// TestInitsSurviveANullNameArgument pins the arg-reading guards.
//
// An init that reads args["name"].Value.(string) without the comma-ok form
// panics when the value is null rather than a string, and a panic in a provider
// takes the whole scan down rather than one field. These inits now all use the
// guarded form.
//
// The guarded form then has to report the unusable argument rather than hand
// back partial args: the runtime builds the resource from those, leaving every
// field unset, which surfaces as "encountered a primitive with no type
// information" with nothing naming the cause. So an unusable name is an error,
// not an empty resource.
//
// Both scope sets are registered because they are disjoint beyond the shared
// cloudresourcemanager scope: the region and zone lookups need
// compute.ComputeReadonlyScope, while the project-backed inits need
// iam.CloudPlatformScope and compute.CloudPlatformScope. A resource whose scope
// set is not registered cannot build its client.
func TestInitsSurviveANullNameArgument(t *testing.T) {
	for _, name := range []string{
		"gcp.project.spannerService.instanceConfig",
		"gcp.project.memcacheService.instance",
		"gcp.project.pubsubService.schema",
		"gcp.service",
	} {
		t.Run(name, func(t *testing.T) {
			env := setupTestEnv(t, regionZoneScopes(), projectScopes())
			var err error
			assert.NotPanics(t, func() {
				_, err = NewResource(env.Runtime, name, map[string]*llx.RawData{
					"name": {Type: "\x07"},
				})
			})
			assert.Error(t, err, "a null name must be reported, not resolved to a husk")
		})
	}
}

// TestInitsRejectAnUnusableName pins the same contract for the two other ways a
// name can be unusable: absent entirely, and present but empty. Both used to
// return (args, nil, nil), which the runtime turns into a resource with every
// field unset.
func TestInitsRejectAnUnusableName(t *testing.T) {
	for _, resource := range []string{
		"gcp.project.spannerService.instanceConfig",
		"gcp.project.memcacheService.instance",
		"gcp.service",
	} {
		for _, tc := range []struct {
			title string
			args  map[string]*llx.RawData
		}{
			{"no name argument", map[string]*llx.RawData{"projectId": llx.StringData(testProjectId)}},
			{"empty name argument", map[string]*llx.RawData{"name": llx.StringData("")}},
		} {
			t.Run(resource+"/"+tc.title, func(t *testing.T) {
				env := setupTestEnv(t, regionZoneScopes(), projectScopes())
				_, err := NewResource(env.Runtime, resource, tc.args)
				require.Error(t, err)
				assert.Contains(t, err.Error(), "name",
					"the error should name the argument that is missing")
			})
		}
	}
}
