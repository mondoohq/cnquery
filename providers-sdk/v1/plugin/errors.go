// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package plugin

import (
	"errors"
)

var (
	ErrProviderTypeDoesNotMatch = errors.New("provider type does not match")
	ErrUnsupportedProvider      = errors.New("unsupported provider")
	ErrRunCommandNotImplemented = errors.New("provider does not implement RunCommand")
	ErrFileInfoNotImplemented   = errors.New("provider does not implement FileInfo")
)
