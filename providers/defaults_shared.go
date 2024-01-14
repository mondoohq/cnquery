// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package providers

import (
	"github.com/cockroachdb/errors"
)

const (
	DefaultOsID           = "go.mondoo.com/cnquery/v10/providers/os"
	DeprecatedDefaultOsID = "go.mondoo.com/cnquery/providers/os" // temp to migrate v9 beta users
)

var defaultRuntime *Runtime

func DefaultRuntime() *Runtime {
	if defaultRuntime == nil {
		defaultRuntime = Coordinator.NewRuntime()
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
