// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package plugin

import (
	"errors"

	"google.golang.org/grpc/status"
)

var (
	ErrProviderTypeDoesNotMatch = errors.New("provider type does not match")
	ErrUnsupportedProvider      = errors.New("unsupported provider")
	ErrRunCommandNotImplemented = errors.New("provider does not implement RunCommand")
	ErrFileInfoNotImplemented   = errors.New("provider does not implement FileInfo")
)

// IsUnsupportedProviderError checks if the given errors indicates an unsupported provider
// for either a direct (non-grpc) transmission or a GRPC-based call
func IsUnsupportedProviderError(e error) bool {
	if e == ErrUnsupportedProvider {
		return true
	}
	st, ok := status.FromError(e)
	if !ok {
		return false
	}
	return st.Message() == ErrUnsupportedProvider.Error()
}
