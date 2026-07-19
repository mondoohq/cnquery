// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"

	"github.com/Snowflake-Labs/terraform-provider-snowflake/pkg/sdk"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/snowflake/connection"
	"go.mondoo.com/mql/v13/types"
)

func (r *mqlSnowflakeAccount) shares() ([]any, error) {
	conn := r.MqlRuntime.Connection.(*connection.SnowflakeConnection)
	client := conn.Client()
	ctx := context.Background()

	shares, err := client.Shares.Show(ctx, &sdk.ShowShareOptions{})
	if err != nil {
		return nil, err
	}

	list := []any{}
	for i := range shares {
		mqlShare, err := newMqlSnowflakeShare(r.MqlRuntime, shares[i])
		if err != nil {
			return nil, err
		}
		list = append(list, mqlShare)
	}

	return list, nil
}

func newMqlSnowflakeShare(runtime *plugin.Runtime, share sdk.Share) (*mqlSnowflakeShare, error) {
	to := make([]any, 0, len(share.To))
	for _, acct := range share.To {
		to = append(to, acct.FullyQualifiedName())
	}

	databaseName := ""
	if share.DatabaseName.Name() != "" {
		databaseName = share.DatabaseName.FullyQualifiedName()
	}

	r, err := CreateResource(runtime, "snowflake.share", map[string]*llx.RawData{
		"__id":         llx.StringData(string(share.Kind) + "/" + share.Name.FullyQualifiedName()),
		"name":         llx.StringData(share.Name.FullyQualifiedName()),
		"kind":         llx.StringData(string(share.Kind)),
		"databaseName": llx.StringData(databaseName),
		"to":           llx.ArrayData(to, types.String),
		"owner":        llx.StringData(share.Owner),
		"comment":      llx.StringData(share.Comment),
		"createdAt":    llx.TimeData(share.CreatedOn),
	})
	if err != nil {
		return nil, err
	}
	return r.(*mqlSnowflakeShare), nil
}

func (r *mqlSnowflakeShare) database() (*mqlSnowflakeDatabase, error) {
	return resolveDatabaseRef(r.MqlRuntime, r.DatabaseName.Data, &r.Database)
}

func (r *mqlSnowflakeShare) ownerRole() (*mqlSnowflakeRole, error) {
	return resolveOwnerRole(r.MqlRuntime, r.Owner.Data, &r.OwnerRole)
}
