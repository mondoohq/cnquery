// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"sync"

	"github.com/Snowflake-Labs/terraform-provider-snowflake/pkg/sdk"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/snowflake/connection"
)

type mqlSnowflakeRowAccessPolicyInternal struct {
	descLock       sync.Mutex
	descLoaded     bool
	descLoadErr    error
	descSignature  []any
	descReturnType string
	descBody       string
	refsLock       sync.Mutex
	refsLoaded     bool
	refsLoadErr    error
	refs           []any
}

func (r *mqlSnowflakeAccount) rowAccessPolicies() ([]any, error) {
	conn := r.MqlRuntime.Connection.(*connection.SnowflakeConnection)
	client := conn.Client()
	ctx := context.Background()

	policies, err := client.RowAccessPolicies.Show(ctx, sdk.NewShowRowAccessPolicyRequest().
		WithIn(sdk.ExtendedIn{In: sdk.In{Account: sdk.Bool(true)}}))
	if err != nil {
		return nil, err
	}

	list := make([]any, 0, len(policies))
	for i := range policies {
		mqlPolicy, err := newMqlSnowflakeRowAccessPolicy(r.MqlRuntime, policies[i])
		if err != nil {
			return nil, err
		}
		list = append(list, mqlPolicy)
	}

	return list, nil
}

func newMqlSnowflakeRowAccessPolicy(runtime *plugin.Runtime, policy sdk.RowAccessPolicy) (*mqlSnowflakeRowAccessPolicy, error) {
	r, err := CreateResource(runtime, "snowflake.rowAccessPolicy", map[string]*llx.RawData{
		"__id":          llx.StringData(policy.ID().FullyQualifiedName()),
		"name":          llx.StringData(policy.Name),
		"databaseName":  llx.StringData(policy.DatabaseName),
		"schemaName":    llx.StringData(policy.SchemaName),
		"kind":          llx.StringData(policy.Kind),
		"owner":         llx.StringData(policy.Owner),
		"ownerRoleType": llx.StringData(policy.OwnerRoleType),
		"comment":       llx.StringData(policy.Comment),
		"options":       llx.StringData(policy.Options),
		"createdAt":     parseSnowflakeTime(policy.CreatedOn),
	})
	if err != nil {
		return nil, err
	}
	return r.(*mqlSnowflakeRowAccessPolicy), nil
}

func (r *mqlSnowflakeRowAccessPolicy) gatherDescribe() error {
	if r.descLoaded {
		return r.descLoadErr
	}
	r.descLock.Lock()
	defer r.descLock.Unlock()
	if r.descLoaded {
		return r.descLoadErr
	}

	conn := r.MqlRuntime.Connection.(*connection.SnowflakeConnection)
	client := conn.Client()
	ctx := context.Background()

	details, err := client.RowAccessPolicies.Describe(ctx,
		sdk.NewSchemaObjectIdentifier(r.DatabaseName.Data, r.SchemaName.Data, r.Name.Data),
	)
	if err != nil {
		r.descLoaded = true
		r.descLoadErr = err
		return err
	}

	sig := make([]any, 0, len(details.Signature))
	for _, s := range details.Signature {
		sig = append(sig, s.Name+":"+string(s.Type))
	}
	r.descSignature = sig
	r.descReturnType = details.ReturnType
	r.descBody = details.Body
	r.descLoaded = true
	return nil
}

func (r *mqlSnowflakeRowAccessPolicy) signature() ([]any, error) {
	if err := r.gatherDescribe(); err != nil {
		return nil, err
	}
	return r.descSignature, nil
}

func (r *mqlSnowflakeRowAccessPolicy) returnType() (string, error) {
	if err := r.gatherDescribe(); err != nil {
		return "", err
	}
	return r.descReturnType, nil
}

func (r *mqlSnowflakeRowAccessPolicy) body() (string, error) {
	if err := r.gatherDescribe(); err != nil {
		return "", err
	}
	return r.descBody, nil
}

func (r *mqlSnowflakeRowAccessPolicy) database() (*mqlSnowflakeDatabase, error) {
	return resolveDatabaseRef(r.MqlRuntime, r.DatabaseName.Data, &r.Database)
}

func (r *mqlSnowflakeRowAccessPolicy) schema() (*mqlSnowflakeSchema, error) {
	return resolveSchemaRef(r.MqlRuntime, r.DatabaseName.Data, r.SchemaName.Data, &r.Schema)
}

func (r *mqlSnowflakeRowAccessPolicy) references() ([]any, error) {
	if r.refsLoaded {
		return r.refs, r.refsLoadErr
	}
	r.refsLock.Lock()
	defer r.refsLock.Unlock()
	if r.refsLoaded {
		return r.refs, r.refsLoadErr
	}

	out, err := queryPolicyReferences(r.MqlRuntime, r.DatabaseName.Data, r.SchemaName.Data, r.Name.Data)
	if err != nil {
		r.refsLoaded = true
		r.refsLoadErr = err
		return nil, err
	}

	r.refs = out
	r.refsLoaded = true
	return r.refs, nil
}
