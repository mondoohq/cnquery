// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"sync"
	"time"

	"github.com/Snowflake-Labs/terraform-provider-snowflake/pkg/sdk"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/snowflake/connection"
	"go.mondoo.com/mql/v13/types"
)

type mqlSnowflakeSecretInternal struct {
	descLock        sync.Mutex
	descLoaded      bool
	descLoadErr     error
	descUsername    string
	descIntegration string
	descAccessExp   *time.Time
	descRefreshExp  *time.Time
}

func (r *mqlSnowflakeAccount) secrets() ([]any, error) {
	conn := r.MqlRuntime.Connection.(*connection.SnowflakeConnection)
	client := conn.Client()
	ctx := context.Background()

	secrets, err := client.Secrets.Show(ctx,
		sdk.NewShowSecretRequest().WithIn(sdk.ExtendedIn{In: sdk.In{Account: sdk.Bool(true)}}),
	)
	if err != nil {
		return nil, err
	}

	list := make([]any, 0, len(secrets))
	for i := range secrets {
		mqlSecret, err := newMqlSnowflakeSecret(r.MqlRuntime, secrets[i])
		if err != nil {
			return nil, err
		}
		list = append(list, mqlSecret)
	}

	return list, nil
}

func newMqlSnowflakeSecret(runtime *plugin.Runtime, secret sdk.Secret) (*mqlSnowflakeSecret, error) {
	comment := ""
	if secret.Comment != nil {
		comment = *secret.Comment
	}

	scopes := make([]any, 0, len(secret.OauthScopes))
	for _, s := range secret.OauthScopes {
		scopes = append(scopes, s)
	}

	r, err := CreateResource(runtime, "snowflake.secret", map[string]*llx.RawData{
		"__id":          llx.StringData(secret.ID().FullyQualifiedName()),
		"name":          llx.StringData(secret.Name),
		"databaseName":  llx.StringData(secret.DatabaseName),
		"schemaName":    llx.StringData(secret.SchemaName),
		"owner":         llx.StringData(secret.Owner),
		"ownerRoleType": llx.StringData(secret.OwnerRoleType),
		"comment":       llx.StringData(comment),
		"secretType":    llx.StringData(secret.SecretType),
		"oauthScopes":   llx.ArrayData(scopes, types.String),
		"createdAt":     llx.TimeData(secret.CreatedOn),
	})
	if err != nil {
		return nil, err
	}
	return r.(*mqlSnowflakeSecret), nil
}

func (r *mqlSnowflakeSecret) gatherDescribe() error {
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

	details, err := client.Secrets.Describe(ctx,
		sdk.NewSchemaObjectIdentifier(r.DatabaseName.Data, r.SchemaName.Data, r.Name.Data),
	)
	if err != nil {
		r.descLoaded = true
		r.descLoadErr = err
		return err
	}

	if details.Username != nil {
		r.descUsername = *details.Username
	}
	if details.IntegrationName != nil {
		r.descIntegration = *details.IntegrationName
	}
	r.descAccessExp = details.OauthAccessTokenExpiryTime
	r.descRefreshExp = details.OauthRefreshTokenExpiryTime

	r.descLoaded = true
	return nil
}

func (r *mqlSnowflakeSecret) username() (string, error) {
	if err := r.gatherDescribe(); err != nil {
		return "", err
	}
	return r.descUsername, nil
}

func (r *mqlSnowflakeSecret) integrationName() (string, error) {
	if err := r.gatherDescribe(); err != nil {
		return "", err
	}
	return r.descIntegration, nil
}

func (r *mqlSnowflakeSecret) oauthAccessTokenExpiryTime() (*time.Time, error) {
	if err := r.gatherDescribe(); err != nil {
		return nil, err
	}
	if r.descAccessExp == nil {
		r.OauthAccessTokenExpiryTime.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	return r.descAccessExp, nil
}

func (r *mqlSnowflakeSecret) oauthRefreshTokenExpiryTime() (*time.Time, error) {
	if err := r.gatherDescribe(); err != nil {
		return nil, err
	}
	if r.descRefreshExp == nil {
		r.OauthRefreshTokenExpiryTime.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	return r.descRefreshExp, nil
}
