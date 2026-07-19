// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers/alicloud/connection"
)

func (r *mqlAlicloud) id() (string, error) {
	return "alicloud", nil
}

func (r *mqlAlicloud) accountId() (string, error) {
	conn := r.MqlRuntime.Connection.(*connection.AlicloudConnection)
	return conn.Identify()
}

func (r *mqlAlicloud) regions() ([]any, error) {
	conn := r.MqlRuntime.Connection.(*connection.AlicloudConnection)
	regions, err := conn.GetRegions()
	if err != nil {
		return nil, err
	}
	return llx.TArr2Raw(regions), nil
}
