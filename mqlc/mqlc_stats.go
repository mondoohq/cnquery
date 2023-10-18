// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package mqlc

import (
	"sort"
	"time"

	"go.mondoo.com/cnquery/v9/providers-sdk/v1/resources"
	"go.mondoo.com/cnquery/v9/types"
)

type CompilerStats struct {
	ResourceFields map[string]map[string]FieldStat
}

type FieldStat struct {
	Time int64
	Type types.Type
}

func (c *CompilerStats) calledResource(name string) {
	if _, ok := c.ResourceFields[name]; !ok {
		c.ResourceFields[name] = map[string]FieldStat{
			"": {
				Time: time.Now().UnixNano(),
			},
		}
	}
}

func (c *CompilerStats) calledField(resource string, field *resources.Field) {
	c.calledResource(resource)
	c.ResourceFields[resource][field.Name] = FieldStat{
		Time: time.Now().UnixNano(),
		Type: types.Type(field.Type),
	}
}

func (c *CompilerStats) WalkSorted(f func(resource string, field string, info FieldStat)) {
	sortedResources := make([]string, len(c.ResourceFields))
	i := 0
	for key := range c.ResourceFields {
		sortedResources[i] = key
		i++
	}
	sort.Slice(sortedResources, func(i, j int) bool {
		return c.ResourceFields[sortedResources[i]][""].Time < c.ResourceFields[sortedResources[j]][""].Time
	})

	for _, resource := range sortedResources {
		m := c.ResourceFields[resource]

		fields := make([]string, len(m))
		i := 0
		for name := range m {
			fields[i] = name
			i++
		}

		sort.Slice(fields, func(i, j int) bool {
			return m[fields[i]].Time < m[fields[j]].Time
		})

		for _, field := range fields {
			f(resource, field, m[field])
		}
	}
}
