// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"slices"
	"strings"
)

// DiscoveryFilters holds the per-service filters used to narrow discovery.
type DiscoveryFilters struct {
	Storage StorageDiscoveryFilters
}

// DiscoveryFiltersFromOpts builds the discovery filters from the raw --filters
// key/value options passed on the connection config.
func DiscoveryFiltersFromOpts(opts map[string]string) DiscoveryFilters {
	return DiscoveryFilters{
		Storage: StorageDiscoveryFilters{
			BucketNames:        parseCsvSliceOpt(opts, "storage:bucket-names"),
			ExcludeBucketNames: parseCsvSliceOpt(opts, "storage:exclude:bucket-names"),
		},
	}
}

type StorageDiscoveryFilters struct {
	BucketNames        []string
	ExcludeBucketNames []string
}

// note: if this function returns `true`, it means that the bucket should be skipped
func (f StorageDiscoveryFilters) IsFilteredOut(bucketName string) bool {
	if len(f.BucketNames) > 0 && !slices.Contains(f.BucketNames, bucketName) {
		return true
	}
	return slices.Contains(f.ExcludeBucketNames, bucketName)
}

// parseCsvSliceOpt returns the comma-separated values for the given key as a
// slice. Empty keys or values are skipped, and a non-nil empty slice is
// returned when there is nothing to parse.
func parseCsvSliceOpt(opts map[string]string, key string) []string {
	res := []string{}
	for k, v := range opts {
		if k == "" || v == "" {
			continue
		}
		if k == key {
			res = append(res, strings.Split(v, ",")...)
		}
	}
	return res
}
