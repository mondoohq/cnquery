// Copyright Mondoo, Inc. 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"encoding/json"
	"fmt"
	"maps"
	"sort"
	"strings"

	"go.mondoo.com/mql/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/providers/k8s/connection/shared"
)

// LabelNamespaceLabelConflict marks a container image whose in-scope namespaces
// disagree on an inherited label key. Its value lists the conflicting keys,
// sorted and comma-separated, so a routing rule can match on it.
const LabelNamespaceLabelConflict = "k8s.mondoo.com/namespace-label-conflict"

// imageLabelVerdict is what the in-scope namespaces running an image agreed on:
// the inherited labels every voter shares, and the keys they disagree on.
type imageLabelVerdict struct {
	Labels    map[string]string `json:"labels,omitempty"`
	Conflicts []string          `json:"conflicts,omitempty"`
}

// imageLabelVerdicts folds the votes. imagesByNamespace lists the images each
// in-scope namespace runs; nsLabels holds each namespace's own labels; patterns
// are the inherited keys. Only namespaces carrying a key vote on it: all voters
// agree → the key becomes a label; they disagree → the key is a conflict; no
// voter → nothing. Images with neither are absent from the result.
func imageLabelVerdicts(imagesByNamespace map[string][]string, nsLabels map[string]map[string]string, patterns []string) map[string]imageLabelVerdict {
	// votes[image][key] is the set of values the namespaces running image carry for key.
	votes := map[string]map[string]map[string]struct{}{}
	for ns, images := range imagesByNamespace {
		inherited := inheritedNamespaceLabels(nsLabels[ns], patterns)
		if len(inherited) == 0 {
			continue
		}
		for _, image := range images {
			keys := votes[image]
			if keys == nil {
				keys = map[string]map[string]struct{}{}
				votes[image] = keys
			}
			for k, v := range inherited {
				if keys[k] == nil {
					keys[k] = map[string]struct{}{}
				}
				keys[k][v] = struct{}{}
			}
		}
	}

	verdicts := make(map[string]imageLabelVerdict, len(votes))
	for image, keys := range votes {
		var verdict imageLabelVerdict
		for k, values := range keys {
			if len(values) > 1 {
				verdict.Conflicts = append(verdict.Conflicts, k)
				continue
			}
			if verdict.Labels == nil {
				verdict.Labels = map[string]string{}
			}
			for v := range values {
				verdict.Labels[k] = v
			}
		}
		sort.Strings(verdict.Conflicts)
		verdicts[image] = verdict
	}
	return verdicts
}

// verdictsForImages returns the subset of verdicts for the given images.
func verdictsForImages(verdicts map[string]imageLabelVerdict, images []string) map[string]imageLabelVerdict {
	subset := map[string]imageLabelVerdict{}
	for _, image := range images {
		if v, ok := verdicts[image]; ok {
			subset[image] = v
		}
	}
	return subset
}

// imageLabelVerdictsMaxBytes caps the encoded verdicts handed to one namespace
// stage. Beyond it the stage receives imageLabelVerdictsSentinel and labels no
// image rather than labeling some wrongly. A var so tests can lower it.
var imageLabelVerdictsMaxBytes = 256 << 10

// imageLabelVerdictsSentinel tells a namespace stage that its verdicts did not
// fit the transport.
const imageLabelVerdictsSentinel = "*"

// encodeImageLabelVerdicts marshals verdicts for the stage-2 option: "{}" for
// an empty map, the sentinel when the JSON is longer than imageLabelVerdictsMaxBytes.
func encodeImageLabelVerdicts(verdicts map[string]imageLabelVerdict) string {
	if len(verdicts) == 0 {
		return "{}"
	}
	raw, err := json.Marshal(verdicts)
	if err != nil || len(raw) > imageLabelVerdictsMaxBytes {
		return imageLabelVerdictsSentinel
	}
	return string(raw)
}

// decodeImageLabelVerdicts parses the stage-2 option; skip is true for the sentinel.
func decodeImageLabelVerdicts(raw string) (verdicts map[string]imageLabelVerdict, skip bool, err error) {
	if raw == imageLabelVerdictsSentinel {
		return nil, true, nil
	}
	if err := json.Unmarshal([]byte(raw), &verdicts); err != nil {
		return nil, false, fmt.Errorf("invalid %s option: %w", shared.OPTION_INHERIT_NAMESPACE_LABELS_VERDICTS, err)
	}
	return verdicts, false, nil
}

func isImageAsset(a *inventory.Asset) bool {
	return len(a.Connections) > 0 && a.Connections[0].Type == registryImageConnectionType
}

// applyImageLabelVerdicts stamps each registry-image asset in assets (matched by
// Connections[0].Host) with its verdict's labels and, when it has conflicts,
// the marker label. Assets without a verdict are left untouched.
func applyImageLabelVerdicts(assets []*inventory.Asset, verdicts map[string]imageLabelVerdict) {
	if len(verdicts) == 0 {
		return
	}
	for _, a := range assets {
		if !isImageAsset(a) {
			continue
		}
		verdict, ok := verdicts[a.Connections[0].Host]
		if !ok || (len(verdict.Labels) == 0 && len(verdict.Conflicts) == 0) {
			continue
		}
		if a.Labels == nil {
			a.Labels = map[string]string{}
		}
		maps.Copy(a.Labels, verdict.Labels)
		if len(verdict.Conflicts) > 0 {
			a.Labels[LabelNamespaceLabelConflict] = strings.Join(verdict.Conflicts, ",")
		}
	}
}
