// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package sbom

//go:generate protoc --proto_path=. --go_out=. --go_opt=paths=source_relative sbom.proto

import (
	"cmp"
	"fmt"
	"io"

	"github.com/mitchellh/hashstructure/v2"
)

type Decoder interface {
	Parse(r io.Reader) (*Sbom, error)
}

func (b *Package) Hash() (string, error) {
	hash, err := hashstructure.Hash(b, hashstructure.FormatV2, nil)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%016x", hash), nil
}

// SortFn is a helper function for slices.SortFunc to sort a slice of Package
// by name and version. Use it like this: slices.SortFunc(packages, sbom.SortFn)
func SortFn(a, b *Package) int {
	if n := cmp.Compare(a.Name, b.Name); n != 0 {
		return n
	}
	// if names are equal, order by version
	return cmp.Compare(a.Version, b.Version)
}
