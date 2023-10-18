// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package mqlc

import (
	"sort"
	"time"

	"go.mondoo.com/cnquery/v9/providers-sdk/v1/resources"
	"go.mondoo.com/cnquery/v9/types"
)

type CompilerStats interface {
	WalkSorted(f func(resource string, field string, info FieldStat))
	SetAutoExpand(on bool)
	CallResource(name string)
	CallField(resource string, field *resources.Field)
}

type FieldStat struct {
	Time       int64
	Type       types.Type
	AutoExpand bool
}

type compilerStatsNull struct{}

func (c compilerStatsNull) WalkSorted(f func(resource string, field string, info FieldStat)) {}
func (c compilerStatsNull) SetAutoExpand(on bool)                                            {}
func (c compilerStatsNull) CallResource(name string)                                         {}
func (c compilerStatsNull) CallField(resource string, field *resources.Field)                {}

type compilerStats struct {
	ResourceFields map[string]map[string]FieldStat

	// indicates that the following resource fields are done during
	// field expansion code (e.g. expand user => name + uid + home)
	isAutoExpand bool
}

func (c *compilerStats) SetAutoExpand(on bool) {
	if c == nil {
		return
	}
	c.isAutoExpand = on
}

func (c *compilerStats) CallResource(name string) {
	if _, ok := c.ResourceFields[name]; !ok {
		c.ResourceFields[name] = map[string]FieldStat{
			"": {
				Time: time.Now().UnixNano(),
			},
		}
	}
}

func (c *compilerStats) CallField(resource string, field *resources.Field) {
	c.CallResource(resource)
	if _, ok := c.ResourceFields[resource][field.Name]; ok {
		return
	}
	c.ResourceFields[resource][field.Name] = FieldStat{
		Time:       time.Now().UnixNano(),
		Type:       types.Type(field.Type),
		AutoExpand: c.isAutoExpand,
	}
}

func (c *compilerStats) WalkSorted(f func(resource string, field string, info FieldStat)) {
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
