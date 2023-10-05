// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package detector

import (
	"errors"

	"go.mondoo.com/cnquery/v9/motor/platform"
	"go.mondoo.com/cnquery/v9/motor/providers"
)

func New(p providers.Instance) *Detector {
	return &Detector{
		provider: p,
	}
}

type Detector struct {
	provider providers.Instance
	cache    *platform.Platform
}

func (d *Detector) Platform() (*platform.Platform, error) {
	if d.provider == nil {
		return nil, errors.New("cannot detect platform without a transport")
	}

	panic("ALL GONE IN PLATFORM DETECTOR")
	return nil, nil
}
