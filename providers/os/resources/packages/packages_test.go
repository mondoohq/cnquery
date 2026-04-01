// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package packages

func findPkg(pkgs []Package, name string) Package {
	for _, p := range pkgs {
		if p.Name == name {
			return p
		}
	}
	panic("package not found: " + name)
}
