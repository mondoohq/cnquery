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

type mqlSnowflakeSessionPolicyInternal struct {
	descLock          sync.Mutex
	descLoaded        bool
	descLoadErr       error
	descIdleTimeout   int64
	descUiIdleTimeout int64
}

func (r *mqlSnowflakeAccount) sessionPolicies() ([]any, error) {
	conn := r.MqlRuntime.Connection.(*connection.SnowflakeConnection)
	client := conn.Client()
	ctx := context.Background()

	policies, err := client.SessionPolicies.Show(ctx, sdk.NewShowSessionPolicyRequest())
	if err != nil {
		return nil, err
	}

	list := []any{}
	for i := range policies {
		mqlPolicy, err := newMqlSnowflakeSessionPolicy(r.MqlRuntime, policies[i])
		if err != nil {
			return nil, err
		}
		list = append(list, mqlPolicy)
	}

	return list, nil
}

func newMqlSnowflakeSessionPolicy(runtime *plugin.Runtime, policy sdk.SessionPolicy) (*mqlSnowflakeSessionPolicy, error) {
	r, err := CreateResource(runtime, "snowflake.sessionPolicy", map[string]*llx.RawData{
		"__id":          llx.StringData(policy.ID().FullyQualifiedName()),
		"name":          llx.StringData(policy.Name),
		"databaseName":  llx.StringData(policy.DatabaseName),
		"schemaName":    llx.StringData(policy.SchemaName),
		"kind":          llx.StringData(policy.Kind),
		"owner":         llx.StringData(policy.Owner),
		"ownerRoleType": llx.StringData(policy.OwnerRoleType),
		"comment":       llx.StringData(policy.Comment),
		"options":       llx.StringData(policy.Options),
	})
	if err != nil {
		return nil, err
	}
	return r.(*mqlSnowflakeSessionPolicy), nil
}

func (r *mqlSnowflakeSessionPolicy) gatherSessionPolicyDetails() error {
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

	desc, err := client.SessionPolicies.Describe(ctx, sdk.NewSchemaObjectIdentifier(r.DatabaseName.Data, r.SchemaName.Data, r.Name.Data))
	if err != nil {
		r.descLoaded = true
		r.descLoadErr = err
		return err
	}

	r.descIdleTimeout = int64(desc.SessionIdleTimeoutMins)
	r.descUiIdleTimeout = int64(desc.SessionUIIdleTimeoutMins)
	r.descLoaded = true
	return nil
}

func (r *mqlSnowflakeSessionPolicy) sessionIdleTimeoutMins() (int64, error) {
	if err := r.gatherSessionPolicyDetails(); err != nil {
		return 0, err
	}
	return r.descIdleTimeout, nil
}

func (r *mqlSnowflakeSessionPolicy) sessionUiIdleTimeoutMins() (int64, error) {
	if err := r.gatherSessionPolicyDetails(); err != nil {
		return 0, err
	}
	return r.descUiIdleTimeout, nil
}
