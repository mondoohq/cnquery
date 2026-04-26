// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"strconv"
	"strings"
	"sync"

	"github.com/Snowflake-Labs/terraform-provider-snowflake/pkg/sdk"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/snowflake/connection"
)

type mqlSnowflakeSecurityIntegrationInternal struct {
	descLock    sync.Mutex
	descLoaded  bool
	descProps   map[string]string
	descLoadErr error
}

func (r *mqlSnowflakeAccount) securityIntegrations() ([]any, error) {
	conn := r.MqlRuntime.Connection.(*connection.SnowflakeConnection)
	client := conn.Client()
	ctx := context.Background()

	integrations, err := client.SecurityIntegrations.Show(ctx, &sdk.ShowSecurityIntegrationRequest{})
	if err != nil {
		return nil, err
	}

	list := []any{}
	for i := range integrations {
		mqlSecurityIntegration, err := newMqlSnowflakeSecurityIntegration(r.MqlRuntime, integrations[i])
		if err != nil {
			return nil, err
		}
		list = append(list, mqlSecurityIntegration)
	}

	return list, nil
}

func newMqlSnowflakeSecurityIntegration(runtime *plugin.Runtime, integration sdk.SecurityIntegration) (*mqlSnowflakeSecurityIntegration, error) {
	r, err := CreateResource(runtime, "snowflake.securityIntegration", map[string]*llx.RawData{
		"__id":      llx.StringData(sdk.NewAccountObjectIdentifier(integration.Name).FullyQualifiedName()),
		"name":      llx.StringData(integration.Name),
		"type":      llx.StringData(integration.IntegrationType),
		"comment":   llx.StringData(integration.Comment),
		"enabled":   llx.BoolData(integration.Enabled),
		"createdAt": llx.TimeData(integration.CreatedOn),
		"category":  llx.StringData(integration.Category),
	})
	if err != nil {
		return nil, err
	}
	mqlResource := r.(*mqlSnowflakeSecurityIntegration)
	return mqlResource, nil
}

// describeProperties fetches DESCRIBE SECURITY INTEGRATION once and caches the
// flattened name->value map. SAML2_X509_CERT, SAML2_ISSUER, OAUTH_CLIENT_ID,
// etc. all live in this single result set.
func (r *mqlSnowflakeSecurityIntegration) describeProperties() (map[string]string, error) {
	if r.descLoaded {
		return r.descProps, r.descLoadErr
	}
	r.descLock.Lock()
	defer r.descLock.Unlock()
	if r.descLoaded {
		return r.descProps, r.descLoadErr
	}

	conn := r.MqlRuntime.Connection.(*connection.SnowflakeConnection)
	client := conn.Client()
	ctx := context.Background()

	props, err := client.SecurityIntegrations.Describe(ctx, sdk.NewAccountObjectIdentifier(r.Name.Data))
	if err != nil {
		r.descLoaded = true
		r.descLoadErr = err
		return nil, err
	}

	out := make(map[string]string, len(props))
	for _, p := range props {
		out[p.Name] = p.Value
	}
	r.descProps = out
	r.descLoaded = true
	return out, nil
}

func (r *mqlSnowflakeSecurityIntegration) properties() (map[string]any, error) {
	props, err := r.describeProperties()
	if err != nil {
		return nil, err
	}
	out := make(map[string]any, len(props))
	for k, v := range props {
		out[k] = v
	}
	return out, nil
}

func (r *mqlSnowflakeSecurityIntegration) saml2X509Cert() (string, error) {
	props, err := r.describeProperties()
	if err != nil {
		return "", err
	}
	return props["SAML2_X509_CERT"], nil
}

func (r *mqlSnowflakeSecurityIntegration) saml2Issuer() (string, error) {
	props, err := r.describeProperties()
	if err != nil {
		return "", err
	}
	return props["SAML2_ISSUER"], nil
}

func (r *mqlSnowflakeSecurityIntegration) saml2Provider() (string, error) {
	props, err := r.describeProperties()
	if err != nil {
		return "", err
	}
	return props["SAML2_PROVIDER"], nil
}

func (r *mqlSnowflakeSecurityIntegration) saml2SsoUrl() (string, error) {
	props, err := r.describeProperties()
	if err != nil {
		return "", err
	}
	return props["SAML2_SSO_URL"], nil
}

func (r *mqlSnowflakeSecurityIntegration) saml2SignRequest() (bool, error) {
	props, err := r.describeProperties()
	if err != nil {
		return false, err
	}
	return parseSnowflakeBool(props["SAML2_SIGN_REQUEST"]), nil
}

func (r *mqlSnowflakeSecurityIntegration) saml2ForceAuthn() (bool, error) {
	props, err := r.describeProperties()
	if err != nil {
		return false, err
	}
	return parseSnowflakeBool(props["SAML2_FORCE_AUTHN"]), nil
}

// parseSnowflakeBool handles the common rendering of bool properties from
// DESCRIBE results. Snowflake returns "true"/"false" (lowercase) for most
// booleans but parseBool tolerates uppercase variants.
func parseSnowflakeBool(value string) bool {
	if value == "" {
		return false
	}
	b, err := strconv.ParseBool(strings.TrimSpace(value))
	if err != nil {
		return false
	}
	return b
}
