// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package image

import (
	"io"
	"os"
	"strings"

	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/daemon"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	"go.mondoo.com/cnquery/v10/providers/os/connection/container/cache"
)

type ShaReference struct {
	SHA string
}

func (r ShaReference) Name() string {
	return r.SHA
}

func (r ShaReference) String() string {
	return r.SHA
}

func (r ShaReference) Context() name.Repository {
	return name.Repository{}
}

func (r ShaReference) Identifier() string {
	return r.SHA
}

func (r ShaReference) Scope(scope string) string {
	return ""
}

func LoadImageFromDockerEngine(sha string, disableBuffer bool) (v1.Image, io.ReadCloser, error) {
	opts := []daemon.Option{}
	if disableBuffer {
		opts = append(opts, daemon.WithUnbufferedOpener())
	}
	img, err := daemon.Image(&ShaReference{SHA: strings.Replace(sha, "sha256:", "", -1)}, opts...)
	if err != nil {
		return nil, nil, err
	}

	// write image to disk (conmpressed, unflattened)
	// Otherwise we can not later recognize it as a valid image
	f, err := writeCompressedTarImage(img, sha)
	if err != nil {
		return nil, nil, err
	}

	return img, f, nil
}

// writeCompressedTarImage writes image including the metradata unflattened to disk
func writeCompressedTarImage(img v1.Image, digest string) (*os.File, error) {
	f, err := cache.RandomFile()
	if err != nil {
		return nil, err
	}
	filename := f.Name()

	ref, err := name.ParseReference(digest, name.WeakValidation)
	if err != nil {
		os.Remove(filename)
		return nil, err
	}

	err = tarball.Write(ref, img, f)
	if err != nil {
		os.Remove(filename)
		return nil, err
	}

	// Rewind, to later read the complete file for uncompress
	f.Seek(0, io.SeekStart)

	return f, nil
}
