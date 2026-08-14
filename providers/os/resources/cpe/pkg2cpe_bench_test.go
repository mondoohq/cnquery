// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package cpe

import (
	"fmt"
	"testing"
)

// BenchmarkNewPackage2Cpe models a package listing. Every package parser calls
// NewPackage2Cpe at least once per package, so this runs on every OS scan.
func BenchmarkNewPackage2Cpe(b *testing.B) {
	type pkg struct{ name, version, epoch, arch string }
	pkgs := make([]pkg, 0, 2000)
	for i := 0; i < 2000; i++ {
		pkgs = append(pkgs, pkg{
			name:    fmt.Sprintf("libexample-%d", i),
			version: fmt.Sprintf("1.%d.%d-3ubuntu2", i%20, i%7),
			epoch:   "",
			arch:    "amd64",
		})
	}
	// A quarter of the versions carry an epoch, which exercises the epoch pattern.
	for i := 0; i < len(pkgs); i += 4 {
		pkgs[i].version = "2:" + pkgs[i].version
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, p := range pkgs {
			if _, err := NewPackage2Cpe(p.name, p.name, p.version, p.epoch, p.arch); err != nil {
				b.Fatal(err)
			}
		}
	}
}
