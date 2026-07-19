// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"strings"
	"sync"

	"github.com/Snowflake-Labs/terraform-provider-snowflake/pkg/sdk"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/snowflake/connection"
)

type mqlSnowflakeAuthenticationPolicyInternal struct {
	descLock          sync.Mutex
	descLoaded        bool
	descLoadErr       error
	descAuthMethods   []any
	descMfaMethods    []any
	descMfaEnrollment string
	descClientTypes   []any
	descSecIntegs     []any
}

func (r *mqlSnowflakeAccount) authenticationPolicies() ([]any, error) {
	conn := r.MqlRuntime.Connection.(*connection.SnowflakeConnection)
	client := conn.Client()
	ctx := context.Background()

	policies, err := client.AuthenticationPolicies.Show(ctx,
		sdk.NewShowAuthenticationPolicyRequest().WithIn(sdk.In{Account: sdk.Bool(true)}),
	)
	if err != nil {
		return nil, err
	}

	list := make([]any, 0, len(policies))
	for i := range policies {
		mqlPolicy, err := newMqlSnowflakeAuthenticationPolicy(r.MqlRuntime, policies[i])
		if err != nil {
			return nil, err
		}
		list = append(list, mqlPolicy)
	}

	return list, nil
}

func newMqlSnowflakeAuthenticationPolicy(runtime *plugin.Runtime, policy sdk.AuthenticationPolicy) (*mqlSnowflakeAuthenticationPolicy, error) {
	r, err := CreateResource(runtime, "snowflake.authenticationPolicy", map[string]*llx.RawData{
		"__id":          llx.StringData(policy.ID().FullyQualifiedName()),
		"name":          llx.StringData(policy.Name),
		"databaseName":  llx.StringData(policy.DatabaseName),
		"schemaName":    llx.StringData(policy.SchemaName),
		"owner":         llx.StringData(policy.Owner),
		"ownerRoleType": llx.StringData(policy.OwnerRoleType),
		"comment":       llx.StringData(policy.Comment),
		"options":       llx.StringData(policy.Options),
		"createdAt":     llx.StringData(policy.CreatedOn),
	})
	if err != nil {
		return nil, err
	}
	return r.(*mqlSnowflakeAuthenticationPolicy), nil
}

func (r *mqlSnowflakeAuthenticationPolicy) gatherDescribe() error {
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

	rows, err := client.AuthenticationPolicies.Describe(ctx,
		sdk.NewSchemaObjectIdentifier(r.DatabaseName.Data, r.SchemaName.Data, r.Name.Data),
	)
	if err != nil {
		r.descLoaded = true
		r.descLoadErr = err
		return err
	}

	for _, row := range rows {
		switch row.Property {
		case "AUTHENTICATION_METHODS":
			r.descAuthMethods = parseAuthPolicyList(row.Value)
		case "MFA_AUTHENTICATION_METHODS":
			r.descMfaMethods = parseAuthPolicyList(row.Value)
		case "MFA_ENROLLMENT":
			r.descMfaEnrollment = strings.TrimSpace(row.Value)
		case "CLIENT_TYPES":
			r.descClientTypes = parseAuthPolicyList(row.Value)
		case "SECURITY_INTEGRATIONS":
			r.descSecIntegs = parseAuthPolicyList(row.Value)
		}
	}

	r.descLoaded = true
	return nil
}

// parseAuthPolicyList parses DESCRIBE AUTHENTICATION POLICY list values into an
// []any of strings. Snowflake returns these lists wrapped in square brackets
// (`[ALL]`, `[PASSWORD, SAML]`); older docs show parentheses (`('ALL')`), so
// both wrappers are stripped. Empty list values (`[]`, `()`, or empty string)
// yield an empty slice.
func parseAuthPolicyList(s string) []any {
	s = strings.TrimSpace(s)
	s = strings.Trim(s, "()[]")
	if s == "" {
		return []any{}
	}
	parts := strings.Split(s, ",")
	out := make([]any, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		p = strings.Trim(p, "'\"")
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func (r *mqlSnowflakeAuthenticationPolicy) authenticationMethods() ([]any, error) {
	if err := r.gatherDescribe(); err != nil {
		return nil, err
	}
	return r.descAuthMethods, nil
}

func (r *mqlSnowflakeAuthenticationPolicy) mfaAuthenticationMethods() ([]any, error) {
	if err := r.gatherDescribe(); err != nil {
		return nil, err
	}
	return r.descMfaMethods, nil
}

func (r *mqlSnowflakeAuthenticationPolicy) mfaEnrollment() (string, error) {
	if err := r.gatherDescribe(); err != nil {
		return "", err
	}
	return r.descMfaEnrollment, nil
}

func (r *mqlSnowflakeAuthenticationPolicy) clientTypes() ([]any, error) {
	if err := r.gatherDescribe(); err != nil {
		return nil, err
	}
	return r.descClientTypes, nil
}

func (r *mqlSnowflakeAuthenticationPolicy) securityIntegrations() ([]any, error) {
	if err := r.gatherDescribe(); err != nil {
		return nil, err
	}
	return r.descSecIntegs, nil
}

func (r *mqlSnowflakeAuthenticationPolicy) securityIntegrationRefs() ([]any, error) {
	if err := r.gatherDescribe(); err != nil {
		return nil, err
	}
	out := []any{}
	for _, s := range r.descSecIntegs {
		name, ok := s.(string)
		if !ok || name == "" {
			continue
		}
		res, err := NewResource(r.MqlRuntime, "snowflake.securityIntegration", map[string]*llx.RawData{
			"name": llx.StringData(name),
		})
		if err != nil {
			return nil, err
		}
		out = append(out, res)
	}
	return out, nil
}
