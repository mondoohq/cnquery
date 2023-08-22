// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package image

import (
	"io"
	"strings"

	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/daemon"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
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
	return img, mutate.Extract(img), nil
}
