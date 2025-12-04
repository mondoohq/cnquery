// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"fmt"
	"slices"
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

// ensure all underlying reference types aren't `nil`
func EmptyDiscoveryFilters() DiscoveryFilters {
	return DiscoveryFilters{
		General: GeneralDiscoveryFilters{Regions: []string{}, ExcludeRegions: []string{}, Tags: map[string]string{}, ExcludeTags: map[string]string{}},
		Ec2:     Ec2DiscoveryFilters{InstanceIds: []string{}, ExcludeInstanceIds: []string{}},
		Ecr:     EcrDiscoveryFilters{Tags: []string{}, ExcludeTags: []string{}},
		Ecs:     EcsDiscoveryFilters{},
	}
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

type EcsDiscoveryFilters struct {
	OnlyRunningContainers bool
	DiscoverImages        bool
	DiscoverInstances     bool
}
