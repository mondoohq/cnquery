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

func (r *mqlSnowflakeAccount) tags() ([]any, error) {
	conn := r.MqlRuntime.Connection.(*connection.SnowflakeConnection)
	tags, err := conn.Client().Tags.Show(context.Background(), &sdk.ShowTagRequest{})
	if err != nil {
		return nil, err
	}

	out := make([]any, 0, len(tags))
	for i := range tags {
		mqlTag, err := newMqlSnowflakeTag(r.MqlRuntime, tags[i])
		if err != nil {
			return nil, err
		}
		out = append(out, mqlTag)
	}
	return out, nil
}

func newMqlSnowflakeTag(runtime *plugin.Runtime, tag sdk.Tag) (*mqlSnowflakeTag, error) {
	allowedValues := make([]any, 0, len(tag.AllowedValues))
	for _, v := range tag.AllowedValues {
		allowedValues = append(allowedValues, v)
	}

	r, err := CreateResource(runtime, "snowflake.tag", map[string]*llx.RawData{
		"__id":          llx.StringData(tag.ID().FullyQualifiedName()),
		"name":          llx.StringData(tag.Name),
		"databaseName":  llx.StringData(tag.DatabaseName),
		"schemaName":    llx.StringData(tag.SchemaName),
		"owner":         llx.StringData(tag.Owner),
		"ownerRoleType": llx.StringData(tag.OwnerRoleType),
		"comment":       llx.StringData(tag.Comment),
		"allowedValues": llx.ArrayData(allowedValues, types.String),
		"createdAt":     llx.TimeData(tag.CreatedOn),
	})
	if err != nil {
		return nil, err
	}
	return r.(*mqlSnowflakeTag), nil
}
