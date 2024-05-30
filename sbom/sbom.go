// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package sbom

//go:generate protoc --proto_path=. --go_out=. --go_opt=paths=source_relative sbom.proto

import (
	"fmt"
	"github.com/mitchellh/hashstructure/v2"
	"io"
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
