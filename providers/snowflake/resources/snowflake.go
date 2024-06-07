// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"go.mondoo.com/cnquery/v11/providers/snowflake/connection"
)

func (r *mqlSnowflake) id() (string, error) {
	return "snowflake", nil
}

func (r *mqlSnowflake) currentRole() (string, error) {
	conn := r.MqlRuntime.Connection.(*connection.SnowflakeConnection)
	client := conn.Client()
	ctx := context.Background()

	current, err := client.ContextFunctions.CurrentSessionDetails(ctx)
	if err != nil {
		return "", err
	}
	return current.Role, nil
}
