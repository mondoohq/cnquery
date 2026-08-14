// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"sync"
	"testing"

	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/utils/syncx"
)

// The cross-resource accessors resolve a parent without an API call whenever
// the runtime already holds it. These tests pin that resolution: which lookups
// answer from what the query already fetched, and which fall through to a read.
// A regression here is silent, because a lookup that finds nothing reports the
// reference as null rather than failing.

func testRuntime() *plugin.Runtime {
	return &plugin.Runtime{Resources: &syncx.Map[plugin.Resource]{}}
}

// seedRoot puts a root resource with already-resolved lists in the runtime, so
// getNeon hands it back instead of building one that would call the API.
func seedRoot(runtime *plugin.Runtime, projects, organizations []any) *mqlNeon {
	root := &mqlNeon{MqlRuntime: runtime, __id: "neon"}
	root.Projects = plugin.TValue[[]any]{Data: projects, State: plugin.StateIsSet}
	root.Organizations = plugin.TValue[[]any]{Data: organizations, State: plugin.StateIsSet}
	runtime.Resources.Set("neon\x00neon", root)
	return root
}

func testProject(runtime *plugin.Runtime, id string) *mqlNeonProject {
	return &mqlNeonProject{
		MqlRuntime: runtime,
		__id:       id,
		Id:         plugin.TValue[string]{Data: id, State: plugin.StateIsSet},
	}
}

func testOrganization(runtime *plugin.Runtime, id string) *mqlNeonOrganization {
	return &mqlNeonOrganization{
		MqlRuntime: runtime,
		__id:       id,
		Id:         plugin.TValue[string]{Data: id, State: plugin.StateIsSet},
	}
}

func TestProjectByIDResolvesFromTheProjectList(t *testing.T) {
	runtime := testRuntime()
	want := testProject(runtime, "proj-2")
	seedRoot(runtime, []any{testProject(runtime, "proj-1"), want}, nil)

	got, err := projectByID(runtime, "proj-2")
	if err != nil {
		t.Fatalf("projectByID: %v", err)
	}
	if got != want {
		t.Fatalf("expected the project from the list, got %v", got)
	}
}

// A personal project belongs to no organization, so it is not in the list the
// root collects per organization. It is still reachable, and a branch or
// endpoint of it must report its project rather than null.
func TestProjectByIDResolvesAProjectMissingFromTheList(t *testing.T) {
	runtime := testRuntime()
	seedRoot(runtime, []any{testProject(runtime, "proj-1")}, nil)

	personal := testProject(runtime, "proj-personal")
	runtime.Resources.Set("neon.project\x00proj-personal", personal)

	got, err := projectByID(runtime, "proj-personal")
	if err != nil {
		t.Fatalf("projectByID: %v", err)
	}
	if got != personal {
		t.Fatalf("expected the project the runtime already holds, got %v", got)
	}
}

func TestProjectByIDWithoutAnIDResolvesToNothing(t *testing.T) {
	runtime := testRuntime()
	seedRoot(runtime, nil, nil)

	got, err := projectByID(runtime, "")
	if err != nil {
		t.Fatalf("projectByID: %v", err)
	}
	if got != nil {
		t.Fatalf("expected no project, got %v", got)
	}
}

func TestOrganizationByIDResolvesFromTheOrganizationList(t *testing.T) {
	runtime := testRuntime()
	want := testOrganization(runtime, "org-2")
	seedRoot(runtime, nil, []any{testOrganization(runtime, "org-1"), want})

	got, err := organizationByID(runtime, "org-2")
	if err != nil {
		t.Fatalf("organizationByID: %v", err)
	}
	if got != want {
		t.Fatalf("expected the organization from the list, got %v", got)
	}
}

func TestOrganizationByIDResolvesOneMissingFromTheList(t *testing.T) {
	runtime := testRuntime()
	seedRoot(runtime, nil, []any{testOrganization(runtime, "org-1")})

	other := testOrganization(runtime, "org-other")
	runtime.Resources.Set("neon.organization\x00org-other", other)

	got, err := organizationByID(runtime, "org-other")
	if err != nil {
		t.Fatalf("organizationByID: %v", err)
	}
	if got != other {
		t.Fatalf("expected the organization the runtime already holds, got %v", got)
	}
}

// An organization neither listed nor already held resolves to nothing, which is
// the signal for the caller to read it directly.
func TestOrganizationByIDResolvesToNothingWhenUnknown(t *testing.T) {
	runtime := testRuntime()
	seedRoot(runtime, nil, []any{testOrganization(runtime, "org-1")})

	got, err := organizationByID(runtime, "org-unknown")
	if err != nil {
		t.Fatalf("organizationByID: %v", err)
	}
	if got != nil {
		t.Fatalf("expected no organization, got %v", got)
	}
}

// An organization list the key cannot read is not an answer about any one
// organization, so the lookup reports nothing rather than the list's error.
func TestOrganizationByIDIgnoresAnUnreadableList(t *testing.T) {
	runtime := testRuntime()
	root := seedRoot(runtime, nil, nil)
	root.Organizations = plugin.TValue[[]any]{State: plugin.StateIsSet | plugin.StateIsNull}

	got, err := organizationByID(runtime, "org-1")
	if err != nil {
		t.Fatalf("organizationByID: %v", err)
	}
	if got != nil {
		t.Fatalf("expected no organization, got %v", got)
	}
}

func TestNewNeonProjectSeedsTheOwnerFromThePayload(t *testing.T) {
	runtime := testRuntime()
	orgID := "org-1"
	rec := projectRecord{
		ID:    "proj-1",
		Name:  "production",
		OrgID: &orgID,
		Owner: &projectOwnerRecord{Email: "ada@example.com"},
	}

	project, err := newNeonProject(runtime, &rec)
	if err != nil {
		t.Fatalf("newNeonProject: %v", err)
	}
	if project.cacheOrgID != "org-1" {
		t.Errorf("organization linkage not cached: %q", project.cacheOrgID)
	}

	owner, err := project.ownerEmail()
	if err != nil {
		t.Fatalf("ownerEmail: %v", err)
	}
	if owner != "ada@example.com" {
		t.Errorf("expected the owner from the payload, got %q", owner)
	}
}

// The list endpoint omits the owner, so a project already built from the
// project endpoint must keep the owner it has when the same project comes back
// through a list.
func TestNewNeonProjectKeepsAKnownOwner(t *testing.T) {
	runtime := testRuntime()

	first, err := newNeonProject(runtime, &projectRecord{
		ID:    "proj-1",
		Owner: &projectOwnerRecord{Email: "ada@example.com"},
	})
	if err != nil {
		t.Fatalf("newNeonProject: %v", err)
	}

	second, err := newNeonProject(runtime, &projectRecord{ID: "proj-1"})
	if err != nil {
		t.Fatalf("newNeonProject: %v", err)
	}
	if second != first {
		t.Fatal("expected the project the runtime already holds")
	}

	owner, err := second.ownerEmail()
	if err != nil {
		t.Fatalf("ownerEmail: %v", err)
	}
	if owner != "ada@example.com" {
		t.Errorf("owner was lost on the second read: %q", owner)
	}
}

// A denied owner read is memoized as an answer, and a later payload must not
// quietly replace it, or the field would flip between reads of the same project.
func TestSeedOwnerEmailDoesNotOverwriteAnAnsweredRead(t *testing.T) {
	project := &mqlNeonProject{}
	project.ownerFetched.Store(true)

	project.seedOwnerEmail("ada@example.com")

	if project.ownerEmail_ != "" {
		t.Errorf("expected the answered read to stand, got %q", project.ownerEmail_)
	}
}

// CreateResource hands back an instance it already holds, so two queries can
// seed the same project at once. Run with -race, this pins that the seed is
// synchronized with the read it memoizes for.
func TestSeedOwnerEmailIsSafeUnderConcurrency(t *testing.T) {
	project := &mqlNeonProject{}

	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			project.seedOwnerEmail("ada@example.com")
		}()
	}
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if project.ownerFetched.Load() && project.ownerEmail_ != "ada@example.com" {
				t.Error("a seeded project reported an owner it was never given")
			}
		}()
	}
	wg.Wait()

	if project.ownerEmail_ != "ada@example.com" {
		t.Errorf("expected the seeded owner, got %q", project.ownerEmail_)
	}
}
