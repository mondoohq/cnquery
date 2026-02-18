// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package providers

import (
	"sync"

	"github.com/cockroachdb/errors"
)

var DefaultOsIDs = []string{
	"go.mondoo.com/mql/providers/os",
	// FIXME: DEPRECATED, remove in v14.0 vv
	// We specify providers without versions now. Also remove the providers
	// GetFirstID function, since it only exists for this use-case
	"go.mondoo.com/cnquery/v9/providers/os",
	"go.mondoo.com/mql/v13/providers/os",
	// ^^
}

var (
	defaultRuntime      *Runtime
	defaultRuntimeMutex sync.Mutex
)

func DefaultRuntime() *Runtime {
	defaultRuntimeMutex.Lock()
	defer defaultRuntimeMutex.Unlock()
	if defaultRuntime == nil {
		defaultRuntime = Coordinator.NewRuntime()
	}
	return defaultRuntime
}

func SetDefaultRuntime(rt *Runtime) error {
	if rt == nil {
		return errors.New("attempted to set default runtime to null")
	}
	defaultRuntimeMutex.Lock()
	defaultRuntime = rt
	defaultRuntimeMutex.Unlock()
	return nil
}
