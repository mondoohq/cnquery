// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"time"

	"go.mondoo.com/mql/v13/providers/portainer/connection"
)

// unixTimePtr converts a Portainer Unix-seconds timestamp into a *time.Time,
// returning nil when the value is 0 so the field resolves to null ("unset")
// instead of the 1970 epoch.
func unixTimePtr(sec int64) *time.Time {
	if sec == 0 {
		return nil
	}
	t := time.Unix(sec, 0).UTC()
	return &t
}

func (r *mqlPortainer) id() (string, error) {
	conn := r.MqlRuntime.Connection.(*connection.PortainerConnection)
	if id := conn.InstanceID(); id != "" {
		return "portainer/" + id, nil
	}
	return "portainer", nil
}

func (r *mqlPortainer) version() (string, error) {
	conn := r.MqlRuntime.Connection.(*connection.PortainerConnection)
	return conn.Version(), nil
}

func (r *mqlPortainer) instanceId() (string, error) {
	conn := r.MqlRuntime.Connection.(*connection.PortainerConnection)
	return conn.InstanceID(), nil
}
