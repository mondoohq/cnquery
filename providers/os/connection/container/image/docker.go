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

func LoadImageFromDockerEngine(sha string, disableBuffer bool) (v1.Image, error) {
	opts := []daemon.Option{}
	if disableBuffer {
		opts = append(opts, daemon.WithUnbufferedOpener())
	}
	img, err := daemon.Image(&ShaReference{SHA: strings.Replace(sha, "sha256:", "", -1)}, opts...)
	if err != nil {
		return nil, err
	}

	return img, nil
}

func WriteCompressedTarImageToFile(img v1.Image, digest string, f *os.File) error {
	ref, err := name.ParseReference(digest, name.WeakValidation)
	if err != nil {
		return err
	}

	err = tarball.Write(ref, img, f)
	if err != nil {
		return err
	}

	// Rewind, to later read the complete file for uncompress
	_, err = f.Seek(0, io.SeekStart)
	return err
}
