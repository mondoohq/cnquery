// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package awsebs

import "github.com/cockroachdb/errors"

func (m *ebsMetadata) windowsMetadata() (any, error) {
	return nil, errors.New("unimplemented")
}
