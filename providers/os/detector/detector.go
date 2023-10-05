// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package detector

import (
	"runtime"

	"go.mondoo.com/cnquery/v9/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v9/providers/os/connection"
	"go.mondoo.com/cnquery/v9/providers/os/connection/shared"
)

func DetectOS(conn shared.Connection) (*inventory.Platform, bool) {
	if conn.Type() == connection.Local && runtime.GOOS == "windows" {
		return WindowsFamily.Resolve(conn)
	}
	return OperatingSystems.Resolve(conn)
}

// map that is organized by platform name, to quickly determine its families
var osTree = platformParents(OperatingSystems)

func platformParents(r *PlatformResolver) map[string][]string {
	return traverseFamily(r, []string{})
}

func traverseFamily(r *PlatformResolver, parents []string) map[string][]string {
	if r.IsFamily {
		// make sure we completely copy the values, otherwise they are going to overwrite themselves
		p := make([]string, len(parents))
		copy(p, parents)
		// add the current family
		p = append(p, r.Name)
		res := map[string][]string{}

		// iterate over children
		for i := range r.Children {
			child := r.Children[i]
			// recursively walk through the tree
			collect := traverseFamily(child, p)
			for k := range collect {
				res[k] = collect[k]
			}
		}
		return res
	}

	// return child (no family)
	return map[string][]string{
		r.Name: parents,
	}
}

func Family(platform string) []string {
	parents, ok := osTree[platform]
	if !ok {
		return []string{}
	}
	return parents
}

// gathers the family for the provided platform
// NOTE: at this point only operating systems have families
func IsFamily(platform string, family string) bool {
	// 1. determine the families of the platform
	parents, ok := osTree[platform]
	if !ok {
		return false
	}

	// 2. check that the platform is part of the family
	for i := range parents {
		if parents[i] == family {
			return true
		}
	}
	return false
}
