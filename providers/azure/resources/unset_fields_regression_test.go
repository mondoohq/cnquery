// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"os"
	"strings"
	"testing"

	web "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appservice/armappservice/v6"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// staticFields reads azure.lr and returns the fields a resource's creator is
// responsible for: everything declared in the schema that is not a computed
// accessor (`field() type`, which the runtime resolves by calling a Go
// method).
//
// A static field has no other source. If the creator leaves it out of the args
// map it is never set -- not null, unset -- and every read of it crosses the
// plugin boundary as an empty DataRes, which the client reports as "provider
// returned no data and no error for a field" followed by "llx: encountered a
// primitive with no type information, coercing to null".
//
// Scanned as text rather than parsed so the provider module does not take on
// the schema parser as a dependency for one test.
func staticFields(t *testing.T, resource string) []string {
	t.Helper()

	raw, err := os.ReadFile("azure.lr")
	require.NoError(t, err)

	var fields []string
	inBody := false
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)

		if !inBody {
			// the header is `[private] <name> [@options] {`; match the name
			// exactly so sub-resources sharing the prefix don't open the body
			head := strings.TrimPrefix(line, "private ")
			name, rest, ok := strings.Cut(head, " ")
			inBody = ok && name == resource && strings.HasSuffix(rest, "{")
			continue
		}
		if line == "}" {
			return fields
		}
		if line == "" || strings.HasPrefix(line, "//") || strings.HasPrefix(line, "embed ") {
			continue
		}

		name, _, _ := strings.Cut(line, " ")
		// `field() type` is computed, `init(...)` is not a field at all
		if strings.Contains(name, "(") {
			continue
		}
		fields = append(fields, name)
	}

	t.Fatalf("resource %q not found in azure.lr", resource)
	return nil
}

// TestSiteConfigArgsSetsEveryStaticField guards the app-site configuration
// creator against reintroducing conditional `args["x"] = ...` assignments.
//
// minTlsVersion, ftpsState, minTlsCipherSuite and scmMinTlsVersion used to be
// assigned only when the SDK pointer was non-nil, and the nine booleans only
// when Azure returned a properties block at all. Every app whose config omits
// any of them logged two errors per field per scan and reported the field as a
// typeless null.
func TestSiteConfigArgsSetsEveryStaticField(t *testing.T) {
	want := staticFields(t, "azure.subscription.webService.appsiteconfig")
	// pin the scanner itself, so a silently-empty or over-eager field list
	// cannot make the assertions below pass for the wrong reason
	require.Subset(t, want, []string{
		"id", "name", "kind", "type", "properties", "corsAllowedOrigins",
		"minTlsVersion", "ftpsState", "minTlsCipherSuite", "scmMinTlsVersion",
		"remoteDebuggingEnabled", "http20Enabled", "alwaysOn", "webSocketsEnabled",
		"httpLoggingEnabled", "detailedErrorLoggingEnabled", "autoHealEnabled",
	})
	require.NotContains(t, want, "ipSecurityRestrictions", "computed accessors are not the creator's job")

	t.Run("fully populated properties", func(t *testing.T) {
		args, err := siteConfigArgs(&web.SiteConfigResource{
			ID:   strp("/subscriptions/sub/sites/app/config/web"),
			Name: strp("web"),
			Properties: &web.SiteConfig{
				MinTLSVersion:               ptr(web.SupportedTLSVersionsOne2),
				FtpsState:                   ptr(web.FtpsStateDisabled),
				MinTLSCipherSuite:           ptr(web.TLSCipherSuitesTLSAES128GCMSHA256),
				ScmMinTLSVersion:            ptr(web.SupportedTLSVersionsOne2),
				RemoteDebuggingEnabled:      boolp(false),
				Http20Enabled:               boolp(true),
				AlwaysOn:                    boolp(true),
				WebSocketsEnabled:           boolp(false),
				HTTPLoggingEnabled:          boolp(true),
				DetailedErrorLoggingEnabled: boolp(true),
				AutoHealEnabled:             boolp(false),
			},
		})
		require.NoError(t, err)
		for _, field := range want {
			assert.Contains(t, args, field, "field %q must be set by the creator", field)
		}
		assert.Equal(t, "TLS_AES_128_GCM_SHA256", args["minTlsCipherSuite"].Value)
		assert.Equal(t, "Disabled", args["ftpsState"].Value)
	})

	// Azure answers without a properties block for apps whose configuration
	// has never been written. Every field is null then, and none is unset.
	t.Run("nil properties", func(t *testing.T) {
		args, err := siteConfigArgs(&web.SiteConfigResource{ID: strp("/subscriptions/sub/sites/app/config/web")})
		require.NoError(t, err)
		for _, field := range want {
			assert.Contains(t, args, field, "field %q must be set by the creator", field)
		}
		assert.Nil(t, args["minTlsCipherSuite"].Value)
		assert.Nil(t, args["alwaysOn"].Value)
		assert.Equal(t, "/subscriptions/sub/sites/app/config/web", args["id"].Value)
	})
}

// TestSubResourcesRejectDirectQueries covers the sub-resources whose name is
// also the dotted path that reaches them, e.g. the cluster resource
// azure.subscription.aksService.cluster plus its autoUpgradeProfile field.
//
// The compiler resolves the longest matching resource name first, so writing
// the full path builds the sub-resource bare, with no id and no fields. Each
// field read on that husk then reported an anonymous malformed-primitive error
// rather than the mistake in the query. Their init must say what went wrong.
func TestSubResourcesRejectDirectQueries(t *testing.T) {
	t.Run("outboundVnetRouting", func(t *testing.T) {
		_, res, err := initAzureSubscriptionWebServiceAppsiteOutboundVnetRouting(nil, nil)
		require.Error(t, err)
		assert.Nil(t, res)
		assert.Contains(t, err.Error(), "azure.subscription.webService.appsite.outboundVnetRouting")
		assert.Contains(t, err.Error(), "azure.subscription.webService.apps {")
	})

	t.Run("autoUpgradeProfile", func(t *testing.T) {
		_, res, err := initAzureSubscriptionAksServiceClusterAutoUpgradeProfile(nil, nil)
		require.Error(t, err)
		assert.Nil(t, res)
		assert.Contains(t, err.Error(), "azure.subscription.aksService.cluster.autoUpgradeProfile")
		assert.Contains(t, err.Error(), "azure.subscription.aksService.clusters {")
	})

	t.Run("advancedNetworking", func(t *testing.T) {
		_, res, err := initAzureSubscriptionAksServiceClusterAdvancedNetworking(nil, nil)
		require.Error(t, err)
		assert.Nil(t, res)
		assert.Contains(t, err.Error(), "azure.subscription.aksService.cluster.advancedNetworking")
		assert.Contains(t, err.Error(), "azure.subscription.aksService.clusters {")
	})
}
