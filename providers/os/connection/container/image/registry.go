// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package image

import (
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"go.mondoo.com/cnquery/v10/providers/os/connection/container/auth"
)

func GetImageDescriptor(ref name.Reference, opts ...remote.Option) (*remote.Descriptor, error) {
	if len(opts) == 0 {
		opts = auth.DefaultOpts(ref.Name(), false)
	}
	return remote.Get(ref, opts...)
}

func LoadImageFromRegistry(ref name.Reference, opts ...remote.Option) (v1.Image, error) {
	if len(opts) == 0 {
		opts = auth.DefaultOpts(ref.Name(), false)
	}
	img, err := remote.Image(ref, opts...)
	if err != nil {
		return nil, err
	}
	return img, nil
}
