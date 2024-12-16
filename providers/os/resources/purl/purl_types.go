// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package purl

import "github.com/package-url/packageurl-go"

type Type string

// These are only an extension of the known purl types defined at:
//
// https://github.com/package-url/packageurl-go/blob/master/packageurl.go#L54
// https://github.com/package-url/purl-spec#known-purl-types
var (
	// TypeWindows is a pkg:windows purl.
	TypeWindows = "windows"
	// TypeWindowsAppx is a pkg:appx purl.
	TypeWindowsAppx = "appx"
	// TypeMacos is a pkg:macos purl.
	TypeMacos = "macos"

	KnownTypes = map[string]struct{}{
		TypeWindows: {},
		TypeMacos:   {},
	}
)

func init() {
	// merge packageurl.KnownTypes and the extension types
	for t := range packageurl.KnownTypes {
		KnownTypes[t] = struct{}{}
	}
}

func ValidType(t string) bool {
	_, ok := KnownTypes[t]
	return ok
}
