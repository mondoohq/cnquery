// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package purl

import "github.com/package-url/packageurl-go"

type Type string

// These are only an extension of the known purl types defined at:
//
// https://github.com/package-url/purl-spec#known-purl-types
var (
	// TypeWindows is a pkg:windows purl.
	TypeWindows Type = "windows"
	// TypeAppx is a pkg:appx purl.
	TypeAppx Type = "appx"
	// TypeMacos is a pkg:macos purl.
	TypeMacos Type = "macos"
	// Type_X_Platform is a pkg:platform purl.
	Type_X_Platform Type = "platform"
	// TypeSnap is a pkg:snap purl.
	TypeSnap Type = "snap"
	// TypeCos is a pkg:cos purl for Google Container-Optimized OS packages.
	// Tracks the shape proposed in package-url/purl-spec#270 and emitted by
	// osv-scalibr; not (yet) in the formal purl-spec registry.
	TypeCos Type = "cos"
	// Types we use coming from:
	// https://github.com/package-url/packageurl-go/blob/master/packageurl.go#L54
	TypeGeneric = Type(packageurl.TypeGeneric)
	TypeApk     = Type(packageurl.TypeApk)
	TypeDebian  = Type(packageurl.TypeDebian)
	TypeAlpm    = Type(packageurl.TypeAlpm)
	TypeEbuild  = Type(packageurl.TypeEbuild)
	TypeNix     = Type(packageurl.TypeNix)
	TypeRPM     = Type(packageurl.TypeRPM)

	KnownTypes = map[Type]struct{}{
		TypeAppx:        {},
		TypeWindows:     {},
		TypeMacos:       {},
		Type_X_Platform: {},
		TypeGeneric:     {},
		TypeApk:         {},
		TypeDebian:      {},
		TypeAlpm:        {},
		TypeEbuild:      {},
		TypeNix:         {},
		TypeRPM:         {},
		TypeCos:         {},
	}
)

func ValidTypeString(t string) bool {
	return ValidType(Type(t))
}

func ValidType(t Type) bool {
	_, ok := KnownTypes[t]
	return ok
}
