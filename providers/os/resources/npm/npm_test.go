// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package npm

func findPkg(pkgs []*Package, name string) *Package {
	for _, p := range pkgs {
		if p.Name == name {
			return p
		}
	}
	panic("package not found")
}
