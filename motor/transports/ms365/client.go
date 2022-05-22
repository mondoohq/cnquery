package ms365

import (
	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/cockroachdb/errors"
	a "github.com/microsoft/kiota-authentication-azure-go"
	"go.mondoo.io/mondoo/motor/vault"
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

func (t *Transport) Auth() (*a.AzureIdentityAuthenticationProvider, error) {
	var credential azcore.TokenCredential
	var err error

	// we only support private key authentication for ms 365
	switch t.cred.Type {
	case vault.CredentialType_pkcs12:
		certs, privateKey, err := azidentity.ParseCertificates(t.cred.Secret, []byte(t.cred.Password))
		if err != nil {
			return nil, errors.Wrap(err, "could not parse pfx file")
		}

		credential, err = azidentity.NewClientCertificateCredential(t.tenantID, t.clientID, certs, privateKey, &azidentity.ClientCertificateCredentialOptions{})
		if err != nil {
			return nil, errors.Wrap(err, "error creating credentials")
		}
	case vault.CredentialType_password:
		credential, err = azidentity.NewClientSecretCredential(t.tenantID, t.clientID, string(t.cred.Secret), &azidentity.ClientSecretCredentialOptions{})
		if err != nil {
			return nil, errors.Wrap(err, "error creating credentials")
		}
	default:
		return nil, errors.New("invalid secret configuration for ms365 transport: " + t.cred.Type.String())
	}

	return a.NewAzureIdentityAuthenticationProviderWithScopes(credential, DefaultMSGraphScopes)
}
