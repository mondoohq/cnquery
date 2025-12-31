// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

type DiscoveryFilters struct {
	Ec2     Ec2DiscoveryFilters
	Ecr     EcrDiscoveryFilters
	Ecs     EcsDiscoveryFilters
	General GeneralDiscoveryFilters
}

func DiscoveryFiltersFromOpts(opts map[string]string) DiscoveryFilters {
	d := DiscoveryFilters{
		General: GeneralDiscoveryFilters{
			Regions:        parseCsvSliceOpt(opts, "regions"),
			ExcludeRegions: parseCsvSliceOpt(opts, "exclude:regions"),
			Tags:           parseMapOpt(opts, "tag:"),
			ExcludeTags:    parseMapOpt(opts, "exclude:tag:"),
		},
		Ec2: Ec2DiscoveryFilters{
			InstanceIds:        parseCsvSliceOpt(opts, "ec2:instance-ids"),
			ExcludeInstanceIds: parseCsvSliceOpt(opts, "ec2:exclude:instance-ids"),
		},
		Ecr: EcrDiscoveryFilters{
			Tags:        parseCsvSliceOpt(opts, "ecr:tags"),
			ExcludeTags: parseCsvSliceOpt(opts, "ecr:exclude:tags"),
		},
		Ecs: EcsDiscoveryFilters{
			OnlyRunningContainers: parseBoolOpt(opts, "ecs:only-running-containers", false),
			DiscoverInstances:     parseBoolOpt(opts, "ecs:discover-instances", false),
			DiscoverImages:        parseBoolOpt(opts, "ecs:discover-images", false),
		},
	}

	// TODO: backward compatibility, remove in future versions
	ec2Tags := parseMapOpt(opts, "ec2:tag:")
	ec2ExcludeTags := parseMapOpt(opts, "ec2:exclude:tag:")
	for k, v := range ec2Tags {
		if _, exists := d.General.Tags[k]; !exists {
			d.General.Tags[k] = v
		}
	}
	for k, v := range ec2ExcludeTags {
		if _, exists := d.General.ExcludeTags[k]; !exists {
			d.General.ExcludeTags[k] = v
		}
	}
	return d
}

type GeneralDiscoveryFilters struct {
	Regions        []string
	ExcludeRegions []string
	// note: values can be in a CSV format, e.g. "env": "prod,staging"
	Tags map[string]string
	// note: values can be in a CSV format, e.g. "env": "prod,staging"
	ExcludeTags map[string]string
}

func (f GeneralDiscoveryFilters) HasTags() bool {
	return len(f.Tags) > 0 || len(f.ExcludeTags) > 0
}

// helper function to improve the readability of filter application
// some resources do not support server-side filtering, so we need to apply filters client-side
func (f GeneralDiscoveryFilters) IsFilteredOutByTags(resourceTags map[string]string) bool {
	return !f.MatchesIncludeTags(resourceTags) || f.MatchesExcludeTags(resourceTags)
}

func (f GeneralDiscoveryFilters) MatchesIncludeTags(resourceTags map[string]string) bool {
	if len(f.Tags) == 0 {
		return true
	}

	for k, csv := range f.Tags {
		for v := range strings.SplitSeq(csv, ",") {
			if tagValue, ok := resourceTags[k]; ok && tagValue == v {
				return true
			}
		}
	}
	return false
}

// note: if this function returns `true`, it means that the resource should be skipped
func (f GeneralDiscoveryFilters) MatchesExcludeTags(resourceTags map[string]string) bool {
	for k, csv := range f.ExcludeTags {
		for v := range strings.SplitSeq(csv, ",") {
			if tagValue, ok := resourceTags[k]; ok && tagValue == v {
				return true
			}
		}
	}
	return false
}

// when possible, we should use AWS API filters to reduce data transfer
func (f GeneralDiscoveryFilters) ToServerSideEc2Filters() []ec2types.Filter {
	filters := []ec2types.Filter{}
	for k, v := range f.Tags {
		filters = append(filters, ec2types.Filter{
			Name:   aws.String(fmt.Sprintf("tag:%s", k)),
			Values: strings.Split(v, ","),
		})
	}
	return filters
}

type Ec2DiscoveryFilters struct {
	InstanceIds        []string
	ExcludeInstanceIds []string
}

// note: if this function returns `true`, it means that the resource should be skipped
func (f Ec2DiscoveryFilters) MatchesExcludeInstanceIds(instanceId *string) bool {
	return instanceId != nil && slices.Contains(f.ExcludeInstanceIds, *instanceId)
}

type EcrDiscoveryFilters struct {
	Tags        []string
	ExcludeTags []string
}

func (f EcrDiscoveryFilters) IsFilteredOutByTags(imageTags []string) bool {
	return !f.MatchesIncludeTags(imageTags) || f.MatchesExcludeTags(imageTags)
}

func (f EcrDiscoveryFilters) MatchesIncludeTags(imageTags []string) bool {
	if len(f.Tags) == 0 {
		return true
	}

	for _, filterTag := range f.Tags {
		if slices.Contains(imageTags, filterTag) {
			return true
		}
	}

	return false
}

// note: if this function returns `true`, it means that the resource should be skipped
func (f EcrDiscoveryFilters) MatchesExcludeTags(imageTags []string) bool {
	for _, filterTag := range f.ExcludeTags {
		if slices.Contains(imageTags, filterTag) {
			return true
		}
	}

	return false
}

type EcsDiscoveryFilters struct {
	OnlyRunningContainers bool
	DiscoverImages        bool
	DiscoverInstances     bool
}

func (f EcsDiscoveryFilters) MatchesOnlyRunningContainers(containerState string) bool {
	if !f.OnlyRunningContainers {
		return true
	}
	return containerState == "RUNNING"
}

// Given a key-value pair that matches a key, return the boolean value of the key.
// If the key is not found or the value cannot be parsed as a boolean, return the default value.
// Example: key = "ecs:only-running-containers", opts = {"ecs:only-running-containers": "true"}
// Returns: true
func parseBoolOpt(opts map[string]string, key string, defaultVal bool) bool {
	for k, v := range opts {
		if k == key {
			parsed, err := strconv.ParseBool(v)
			if err == nil {
				return parsed
			}
		}
	}
	return defaultVal
}

// Given a map of options and a key prefix, return a map of key-value pairs
// where the keys start with the given prefix, with the prefix removed.
// Example:
// keyPrefix = "tag:"
// opts = {"tag:env": "prod", "tag:role": "web"}
// returns {"env": "prod", "role": "web"}
func parseMapOpt(opts map[string]string, keyPrefix string) map[string]string {
	res := map[string]string{}
	for k, v := range opts {
		if k == "" || v == "" {
			continue
		}
		if !strings.HasPrefix(k, keyPrefix) {
			continue
		}
		res[strings.TrimPrefix(k, keyPrefix)] = v
	}
	return res
}

// Given a map of options and a key, return a slice of strings
// where the key matches the given key. The value is split by commas.
// Example:
// key = "regions"
// opts = {"regions": "us-east-1,us-west-2"}
// returns []string{"us-east-1", "us-west-2"}
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
