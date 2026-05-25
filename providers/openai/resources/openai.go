// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"time"

	"github.com/openai/openai-go/v3"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/openai/connection"
)

func openaiConn(runtime *plugin.Runtime) *connection.OpenaiConnection {
	return runtime.Connection.(*connection.OpenaiConnection)
}

func unixToTime(ts int64) time.Time {
	if ts == 0 {
		return time.Time{}
	}
	return time.Unix(ts, 0)
}

func isAccessDenied(err error) bool {
	if err == nil {
		return false
	}
	var apiErr *openai.Error
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode == 401 || apiErr.StatusCode == 403
	}
	return false
}

func (r *mqlOpenai) id() (string, error) {
	return "openai", nil
}

func (r *mqlOpenai) organization() (string, error) {
	return openaiConn(r.MqlRuntime).Organization(), nil
}

func (r *mqlOpenai) projectId() (string, error) {
	return openaiConn(r.MqlRuntime).Project(), nil
}
