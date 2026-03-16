// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package recording

import (
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/utils/multierr"
	"go.mondoo.com/mql/v13/utils/syncx"
)

// MergeOpts controls how recordings are merged.
type MergeOpts struct {
	Path            string
	PrettyPrintJSON bool
}

// Merge combines multiple recordings into a single recording. Assets are
// matched by MRN first, then by overlapping platform IDs. When two assets
// match, their connections, resources, and ID lookups are merged (later
// recordings win on field-level conflicts). Non-matching assets are appended.
func Merge(recordings []*recording, opts MergeOpts) (*recording, error) {
	merged := &recording{
		Path:            opts.Path,
		prettyPrintJSON: opts.PrettyPrintJSON,
		assets:          syncx.Map[*Asset]{},
	}

	for _, rec := range recordings {
		for _, incoming := range rec.Assets {
			existing := findMatchingAsset(merged, incoming)
			if existing == nil {
				clone := cloneAsset(incoming)
				merged.Assets = append(merged.Assets, clone)
				continue
			}
			mergeAssets(existing, incoming)
		}
	}

	// Sync runtime maps back to serializable slices before refreshing caches,
	// since refreshCache rebuilds the maps from the slices.
	merged.finalize()
	merged.refreshCache()
	return merged, nil
}

// MergeFiles loads recordings from the given file paths and merges them into
// a single recording. If any file fails to load, the error is returned
// immediately.
func MergeFiles(paths []string, opts MergeOpts) (*recording, error) {
	recordings := make([]*recording, len(paths))
	for i, p := range paths {
		rec, err := LoadRecordingFile(p)
		if err != nil {
			return nil, multierr.Wrap(err, "failed to load recording from '"+p+"'")
		}
		recordings[i] = rec
	}
	return Merge(recordings, opts)
}

// findMatchingAsset returns the first asset in merged that matches incoming
// by MRN or by any overlapping platform ID.
func findMatchingAsset(merged *recording, incoming *Asset) *Asset {
	if incoming.Asset == nil {
		return nil
	}

	for _, existing := range merged.Assets {
		if existing.Asset == nil {
			continue
		}

		// Match by MRN.
		if incoming.Asset.Mrn != "" && existing.Asset.Mrn == incoming.Asset.Mrn {
			return existing
		}

		// Match by overlapping platform IDs.
		if platformIdsOverlap(existing.Asset.PlatformIds, incoming.Asset.PlatformIds) {
			return existing
		}
	}

	return nil
}

func platformIdsOverlap(a, b []string) bool {
	if len(a) == 0 || len(b) == 0 {
		return false
	}
	set := make(map[string]struct{}, len(a))
	for _, id := range a {
		set[id] = struct{}{}
	}
	for _, id := range b {
		if _, ok := set[id]; ok {
			return true
		}
	}
	return false
}

// mergeAssets folds incoming into existing, merging connections, resources,
// and ID lookups. For field-level conflicts the incoming value wins.
func mergeAssets(existing, incoming *Asset) {
	// Merge inventory metadata: prefer non-empty MRN and union platform IDs.
	if incoming.Asset.Mrn != "" {
		existing.Asset.Mrn = incoming.Asset.Mrn
	}
	existing.Asset.PlatformIds = unionStrings(existing.Asset.PlatformIds, incoming.Asset.PlatformIds)

	// Merge connections (keyed by their string ID).
	ensureAssetCaches(existing)
	ensureAssetCaches(incoming)
	for k, conn := range incoming.connections {
		existing.connections[k] = conn
	}

	// Merge resources (keyed by "Resource\x00ID").
	for key, res := range incoming.resources {
		if existingRes, ok := existing.resources[key]; ok {
			mergeResourceFields(existingRes, res)
		} else {
			clone := *res
			clone.Fields = make(map[string]*llx.RawData, len(res.Fields))
			for k, v := range res.Fields {
				clone.Fields[k] = v
			}
			existing.resources[key] = &clone
		}
	}

	// Merge ID lookups.
	if existing.IdsLookup == nil {
		existing.IdsLookup = map[string]string{}
	}
	for k, v := range incoming.IdsLookup {
		existing.IdsLookup[k] = v
	}
}

func mergeResourceFields(existing, incoming *Resource) {
	if existing.Fields == nil {
		existing.Fields = map[string]*llx.RawData{}
	}
	for k, v := range incoming.Fields {
		existing.Fields[k] = v
	}
}

// cloneAsset creates a shallow copy of an Asset with its own cache maps so
// that mutations to the merged recording don't affect the source.
func cloneAsset(a *Asset) *Asset {
	clone := &Asset{
		Asset:       a.Asset.CloneVT(),
		Connections: a.Connections,
		IdsLookup:   copyMap(a.IdsLookup),
		connections: make(map[string]*connection, len(a.connections)),
		resources:   make(map[string]*Resource, len(a.resources)),
	}
	for k, v := range a.connections {
		clone.connections[k] = v
	}
	for k, v := range a.resources {
		clone.resources[k] = v
	}
	return clone
}

func ensureAssetCaches(a *Asset) {
	if a.connections == nil || a.resources == nil {
		a.RefreshCache()
	}
}

func unionStrings(a, b []string) []string {
	seen := make(map[string]struct{}, len(a)+len(b))
	out := make([]string, 0, len(a)+len(b))
	for _, s := range a {
		if _, ok := seen[s]; !ok {
			seen[s] = struct{}{}
			out = append(out, s)
		}
	}
	for _, s := range b {
		if _, ok := seen[s]; !ok {
			seen[s] = struct{}{}
			out = append(out, s)
		}
	}
	return out
}

func copyMap(m map[string]string) map[string]string {
	if m == nil {
		return map[string]string{}
	}
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}
