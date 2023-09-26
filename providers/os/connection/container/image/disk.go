// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package image

import (
	"io"
	"os"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
)

func LoadImageFromDisk(filepath string) (v1.Image, io.ReadCloser, error) {
	img, err := tarball.ImageFromPath(filepath, nil)
	if err != nil {
		return nil, nil, err
	}
	rc, err := os.Open(filepath)
	if err != nil {
		return nil, nil, err
	}

	return img, rc, nil
}
