// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"runtime"

	"go.mondoo.com/cnquery/v10"
	"go.mondoo.com/cnquery/v10/cli/execruntime"
)

func (m *mqlMondoo) version() (string, error) {
	return cnquery.GetVersion(), nil
}

func (m *mqlMondoo) build() (string, error) {
	return cnquery.GetBuild(), nil
}

func (m *mqlMondoo) arch() (string, error) {
	return runtime.GOOS + "-" + runtime.GOARCH, nil
}

func (m *mqlMondoo) jobEnvironment() (map[string]interface{}, error) {
	// get the local agent runtime information
	ciEnv := execruntime.Detect()

	return map[string]interface{}{
		"id":   ciEnv.Namespace,
		"name": ciEnv.Name,
	}, nil
}

func (m *mqlMondoo) capabilities() ([]interface{}, error) {
	// This method should never be reached.
	// These values are set during the `connect` call.
	return []interface{}{}, nil
}
