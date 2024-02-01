// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package providers

import (
	"github.com/cockroachdb/errors"
)

var DefaultOsIDs = []string{
	"go.mondoo.com/cnquery/providers/os",
	// FIXME: DEPRECATED, remove in v12.0 vv
	// We specify providers without versions now. Also remove the providers
	// GetFirstID function, since it only exists for this use-case
	"go.mondoo.com/cnquery/v9/providers/os",
	"go.mondoo.com/cnquery/v10/providers/os",
	// ^^
}

var defaultRuntime *Runtime

func DefaultRuntime() *Runtime {
	if defaultRuntime == nil {
		defaultRuntime = NewCoordinator().NewRuntime()
	}
	return defaultRuntime
}

func SetDefaultRuntime(rt *Runtime) error {
	if rt == nil {
		return errors.New("attempted to set default runtime to null")
	}
	defaultRuntime = rt
	return nil
}
