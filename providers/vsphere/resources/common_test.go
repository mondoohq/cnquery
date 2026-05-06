// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vmware/govmomi/vapi/tags"
	"github.com/vmware/govmomi/vim25/types"
)

func attachedFor(moid string, tagPairs ...struct{ name, categoryID string }) tags.AttachedTags {
	tagsList := make([]tags.Tag, len(tagPairs))
	for i, p := range tagPairs {
		tagsList[i] = tags.Tag{Name: p.name, CategoryID: p.categoryID}
	}
	return tags.AttachedTags{
		ObjectID: types.ManagedObjectReference{Type: "VirtualMachine", Value: moid},
		Tags:     tagsList,
	}
}

func TestResolveAttachedTags_BasicAndCachedAcrossObjects(t *testing.T) {
	calls := map[string]int{}
	getCategory := func(_ context.Context, id string) (string, error) {
		calls[id]++
		return "env", nil
	}

	out := resolveAttachedTags(context.Background(), []tags.AttachedTags{
		attachedFor("vm-1", struct{ name, categoryID string }{"prod", "cat-1"}),
		attachedFor("vm-2", struct{ name, categoryID string }{"prod", "cat-1"}),
	}, getCategory)

	assert.Equal(t, []string{"env:prod"}, out["vm-1"])
	assert.Equal(t, []string{"env:prod"}, out["vm-2"])
	assert.Equal(t, 1, calls["cat-1"], "category should be fetched once across the batch")
}

func TestResolveAttachedTags_TransientFailureDoesNotPoisonCache(t *testing.T) {
	calls := map[string]int{}
	getCategory := func(_ context.Context, id string) (string, error) {
		calls[id]++
		// First call fails; subsequent calls succeed. This simulates a
		// transient network blip that resolves before the next tag is
		// processed.
		if calls[id] == 1 {
			return "", errors.New("transient")
		}
		return "env", nil
	}

	out := resolveAttachedTags(context.Background(), []tags.AttachedTags{
		attachedFor("vm-1", struct{ name, categoryID string }{"prod", "cat-1"}),
		attachedFor("vm-2", struct{ name, categoryID string }{"prod", "cat-1"}),
	}, getCategory)

	// vm-1 hit the failure: tag emitted without category prefix.
	assert.Equal(t, []string{"prod"}, out["vm-1"])
	// vm-2 retried (because failure wasn't cached) and got the prefix.
	assert.Equal(t, []string{"env:prod"}, out["vm-2"])
	assert.Equal(t, 2, calls["cat-1"], "transient failure should not be cached")
}

func TestResolveAttachedTags_PersistentFailureFallsBackToTagName(t *testing.T) {
	getCategory := func(_ context.Context, _ string) (string, error) {
		return "", errors.New("vAPI down")
	}

	out := resolveAttachedTags(context.Background(), []tags.AttachedTags{
		attachedFor("vm-1", struct{ name, categoryID string }{"prod", "cat-1"}),
	}, getCategory)

	assert.Equal(t, []string{"prod"}, out["vm-1"])
}

func TestResolveAttachedTags_EmptyTagsYieldsEmptySlice(t *testing.T) {
	getCategory := func(_ context.Context, _ string) (string, error) {
		t.Fatal("getCategory should not be called when there are no tags")
		return "", nil
	}

	out := resolveAttachedTags(context.Background(), []tags.AttachedTags{
		attachedFor("vm-1"),
	}, getCategory)

	assert.Empty(t, out["vm-1"])
	assert.Contains(t, out, "vm-1")
}
