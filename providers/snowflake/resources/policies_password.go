// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"

	"github.com/Snowflake-Labs/terraform-provider-snowflake/pkg/sdk"
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers/snowflake/connection"
)

func (r *mqlSnowflakeAccount) passwordPolicies() ([]interface{}, error) {
	conn := r.MqlRuntime.Connection.(*connection.SnowflakeConnection)
	client := conn.Client()
	ctx := context.Background()

	passwordPolicies, err := client.PasswordPolicies.Show(ctx, &sdk.ShowPasswordPolicyOptions{
		In: &sdk.In{
			Account: sdk.Bool(true),
		},
	})
	if err != nil {
		return nil, err
	}

	list := []interface{}{}
	for i := range passwordPolicies {
		mqlPasswordPolicy, err := newMqlSnowflakePasswordPolicy(r.MqlRuntime, passwordPolicies[i])
		if err != nil {
			return nil, err
		}
		list = append(list, mqlPasswordPolicy)
	}

	return list, nil
}

func newMqlSnowflakePasswordPolicy(runtime *plugin.Runtime, passwordPolicy sdk.PasswordPolicy) (*mqlSnowflakePasswordPolicy, error) {
	r, err := CreateResource(runtime, "snowflake.passwordPolicy", map[string]*llx.RawData{
		"__id":         llx.StringData(passwordPolicy.ID().FullyQualifiedName()),
		"name":         llx.StringData(passwordPolicy.Name),
		"databaseName": llx.StringData(passwordPolicy.DatabaseName),
		"schemaName":   llx.StringData(passwordPolicy.SchemaName),
		"kind":         llx.StringData(passwordPolicy.Kind),
		"owner":        llx.StringData(passwordPolicy.Owner),
		"comment":      llx.StringData(passwordPolicy.Comment),
		"createdAt":    llx.TimeData(passwordPolicy.CreatedOn),
	})
	if err != nil {
		return nil, err
	}
	mqlResource := r.(*mqlSnowflakePasswordPolicy)
	return mqlResource, nil
}

func (r *mqlSnowflakePasswordPolicy) gatherPasswordPolicyDetails() error {
	conn := r.MqlRuntime.Connection.(*connection.SnowflakeConnection)
	client := conn.Client()
	ctx := context.Background()

	passwordPolicy, err := client.PasswordPolicies.Describe(ctx, sdk.NewSchemaObjectIdentifier(r.DatabaseName.Data, r.SchemaName.Data, r.Name.Data))
	if err != nil {
		return err
	}

	r.PasswordMinLength = plugin.TValue[int64]{Data: 0, Error: nil, State: plugin.StateIsSet | plugin.StateIsNull}
	if passwordPolicy.PasswordMinLength != nil && passwordPolicy.PasswordMinLength.Value != nil {
		r.PasswordMinLength = plugin.TValue[int64]{Data: int64(*passwordPolicy.PasswordMinLength.Value), Error: nil, State: plugin.StateIsSet}
	}

	r.PasswordMinLength = plugin.TValue[int64]{Data: 0, Error: nil, State: plugin.StateIsSet | plugin.StateIsNull}
	if passwordPolicy.PasswordMinLength != nil && passwordPolicy.PasswordMinLength.Value != nil {
		r.PasswordMinLength = plugin.TValue[int64]{Data: int64(*passwordPolicy.PasswordMinLength.Value), Error: nil, State: plugin.StateIsSet}
	}

	r.PasswordMinUpperCaseChars = plugin.TValue[int64]{Data: 0, Error: nil, State: plugin.StateIsSet | plugin.StateIsNull}
	if passwordPolicy.PasswordMinUpperCaseChars != nil && passwordPolicy.PasswordMinUpperCaseChars.Value != nil {
		r.PasswordMinUpperCaseChars = plugin.TValue[int64]{Data: int64(*passwordPolicy.PasswordMinUpperCaseChars.Value), Error: nil, State: plugin.StateIsSet}
	}

	r.PasswordMinLowerCaseChars = plugin.TValue[int64]{Data: 0, Error: nil, State: plugin.StateIsSet | plugin.StateIsNull}
	if passwordPolicy.PasswordMinLowerCaseChars != nil && passwordPolicy.PasswordMinLowerCaseChars.Value != nil {
		r.PasswordMinLowerCaseChars = plugin.TValue[int64]{Data: int64(*passwordPolicy.PasswordMinLowerCaseChars.Value), Error: nil, State: plugin.StateIsSet}
	}

	r.PasswordMinNumericChars = plugin.TValue[int64]{Data: 0, Error: nil, State: plugin.StateIsSet | plugin.StateIsNull}
	if passwordPolicy.PasswordMinNumericChars != nil && passwordPolicy.PasswordMinNumericChars.Value != nil {
		r.PasswordMinNumericChars = plugin.TValue[int64]{Data: int64(*passwordPolicy.PasswordMinNumericChars.Value), Error: nil, State: plugin.StateIsSet}
	}

	r.PasswordMinSpecialChars = plugin.TValue[int64]{Data: 0, Error: nil, State: plugin.StateIsSet | plugin.StateIsNull}
	if passwordPolicy.PasswordMinSpecialChars != nil && passwordPolicy.PasswordMinSpecialChars.Value != nil {
		r.PasswordMinSpecialChars = plugin.TValue[int64]{Data: int64(*passwordPolicy.PasswordMinSpecialChars.Value), Error: nil, State: plugin.StateIsSet}
	}

	r.PasswordMinAgeDays = plugin.TValue[int64]{Data: 0, Error: nil, State: plugin.StateIsSet | plugin.StateIsNull}
	if passwordPolicy.PasswordMinAgeDays != nil && passwordPolicy.PasswordMinAgeDays.Value != nil {
		r.PasswordMinAgeDays = plugin.TValue[int64]{Data: int64(*passwordPolicy.PasswordMinAgeDays.Value), Error: nil, State: plugin.StateIsSet}
	}

	r.PasswordMaxAgeDays = plugin.TValue[int64]{Data: 0, Error: nil, State: plugin.StateIsSet | plugin.StateIsNull}
	if passwordPolicy.PasswordMaxAgeDays != nil && passwordPolicy.PasswordMaxAgeDays.Value != nil {
		r.PasswordMaxAgeDays = plugin.TValue[int64]{Data: int64(*passwordPolicy.PasswordMaxAgeDays.Value), Error: nil, State: plugin.StateIsSet}
	}

	r.PasswordMaxRetries = plugin.TValue[int64]{Data: 0, Error: nil, State: plugin.StateIsSet | plugin.StateIsNull}
	if passwordPolicy.PasswordMaxRetries != nil && passwordPolicy.PasswordMaxRetries.Value != nil {
		r.PasswordMaxRetries = plugin.TValue[int64]{Data: int64(*passwordPolicy.PasswordMaxRetries.Value), Error: nil, State: plugin.StateIsSet}
	}

	r.PasswordLockoutTimeMins = plugin.TValue[int64]{Data: 0, Error: nil, State: plugin.StateIsSet | plugin.StateIsNull}
	if passwordPolicy.PasswordLockoutTimeMins != nil && passwordPolicy.PasswordLockoutTimeMins.Value != nil {
		r.PasswordLockoutTimeMins = plugin.TValue[int64]{Data: int64(*passwordPolicy.PasswordLockoutTimeMins.Value), Error: nil, State: plugin.StateIsSet}
	}

	r.PasswordHistory = plugin.TValue[int64]{Data: 0, Error: nil, State: plugin.StateIsSet | plugin.StateIsNull}
	if passwordPolicy.PasswordHistory != nil && passwordPolicy.PasswordHistory.Value != nil {
		r.PasswordHistory = plugin.TValue[int64]{Data: int64(*passwordPolicy.PasswordHistory.Value), Error: nil, State: plugin.StateIsSet}
	}

	return nil
}

func (r *mqlSnowflakePasswordPolicy) passwordMinLength() (int64, error) {
	return 0, r.gatherPasswordPolicyDetails()
}

func (r *mqlSnowflakePasswordPolicy) passwordMaxLength() (int64, error) {
	return 0, r.gatherPasswordPolicyDetails()
}

func (r *mqlSnowflakePasswordPolicy) passwordMinUpperCaseChars() (int64, error) {
	return 0, r.gatherPasswordPolicyDetails()
}

func (r *mqlSnowflakePasswordPolicy) passwordMinLowerCaseChars() (int64, error) {
	return 0, r.gatherPasswordPolicyDetails()
}

func (r *mqlSnowflakePasswordPolicy) passwordMinNumericChars() (int64, error) {
	return 0, r.gatherPasswordPolicyDetails()
}

func (r *mqlSnowflakePasswordPolicy) passwordMinSpecialChars() (int64, error) {
	return 0, r.gatherPasswordPolicyDetails()
}

func (r *mqlSnowflakePasswordPolicy) passwordMinAgeDays() (int64, error) {
	return 0, r.gatherPasswordPolicyDetails()
}

func (r *mqlSnowflakePasswordPolicy) passwordMaxAgeDays() (int64, error) {
	return 0, r.gatherPasswordPolicyDetails()
}

func (r *mqlSnowflakePasswordPolicy) passwordMaxRetries() (int64, error) {
	return 0, r.gatherPasswordPolicyDetails()
}

func (r *mqlSnowflakePasswordPolicy) passwordLockoutTimeMins() (int64, error) {
	return 0, r.gatherPasswordPolicyDetails()
}

func (r *mqlSnowflakePasswordPolicy) passwordHistory() (int64, error) {
	return 0, r.gatherPasswordPolicyDetails()
}
