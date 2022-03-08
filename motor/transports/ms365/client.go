package ms365

import (
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/cockroachdb/errors"
	a "github.com/microsoft/kiota/authentication/go/azure"
	msgraphbetasdk "github.com/microsoftgraph/msgraph-beta-sdk-go"
)

const DefaultMSGraphScope = "https://graph.microsoft.com/.default"

var (
	DefaultMSGraphScopes = []string{DefaultMSGraphScope}
	DefaultRoles         = []string{
		"Application.Read.All",
		"AuditLog.Read.All",
		"Calendars.Read",
		"Device.Read.All",
		"DeviceManagementApps.Read.All",
		"DeviceManagementConfiguration.Read.All",
		"DeviceManagementManagedDevices.Read.All",
		"DeviceManagementRBAC.Read.All",
		"DeviceManagementServiceConfig.Read.All",
		"Directory.Read.All",
		"Domain.Read.All",
		"IdentityProvider.Read.All",
		"IdentityRiskEvent.Read.All",
		"IdentityRiskyUser.Read.All",
		"InformationProtectionPolicy.Read.All",
		"MailboxSettings.Read",
		"Organization.Read.All",
		"OrgContact.Read.All",
		"Policy.Read.All",
		"Policy.Read.ConditionalAccess",
		"Policy.Read.PermissionGrant",
		"RoleManagement.Read.All",
		"SecurityActions.Read.All",
		"SecurityEvents.Read.All",
		"TeamsApp.Read.All",
		"TeamSettings.Read.All",
		"ThreatAssessment.Read.All",
		"ThreatIndicators.Read.All",
		"User.Read.All",
	}
)

func (t *Transport) auth() (*a.AzureIdentityAuthenticationProvider, error) {
	cred, err := azidentity.NewClientSecretCredential(t.tenantID, t.clientID, t.clientSecret, &azidentity.ClientSecretCredentialOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "error creating credentials")
	}

	return a.NewAzureIdentityAuthenticationProviderWithScopes(cred, DefaultMSGraphScopes)
}

func (t *Transport) GraphBetaClient() (*msgraphbetasdk.GraphServiceClient, error) {
	auth, err := t.auth()
	if err != nil {
		return nil, errors.Wrap(err, "authentication provider error")
	}

	adapter, err := msgraphbetasdk.NewGraphRequestAdapter(auth)
	if err != nil {
		return nil, err
	}
	graphBetaClient := msgraphbetasdk.NewGraphServiceClient(adapter)
	return graphBetaClient, nil
}
