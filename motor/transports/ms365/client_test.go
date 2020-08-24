package ms365

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"gotest.tools/assert"
)

func TestParseJson(t *testing.T) {
	data := `{
    "subscriptionId": "<azure_subscription_id>",
    "tenantId": "<tenant_id>",
    "clientId": "<application_id>",
    "clientSecret": "<application_secret>",
    "activeDirectoryEndpointUrl": "https://login.microsoftonline.com",
    "resourceManagerEndpointUrl": "https://management.azure.com/",
    "activeDirectoryGraphResourceId": "https://graph.windows.net/",
    "sqlManagementEndpointUrl": "https://management.core.windows.net:8443/",
    "galleryEndpointUrl": "https://gallery.azure.com/",
    "managementEndpointUrl": "https://management.core.windows.net/"
	}`

	auth, err := ParseMicrosoftAuth(strings.NewReader(data))
	require.NoError(t, err)
	assert.Equal(t, "<tenant_id>", auth.TenantId)
	assert.Equal(t, "<azure_subscription_id>", auth.SubscriptionId)
	assert.Equal(t, "<application_id>", auth.ClientId)
	assert.Equal(t, "<application_secret>", auth.ClientSecret)
}
