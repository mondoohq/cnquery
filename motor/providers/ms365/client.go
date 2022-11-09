package ms365

import (
	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/cockroachdb/errors"
	a "github.com/microsoft/kiota-authentication-azure-go"
	"go.mondoo.com/cnquery/motor/providers/ms365/msgraphclient"
	"go.mondoo.com/cnquery/motor/vault"
)

var DefaultRoles = []string{
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

func (p *Provider) Auth() (*a.AzureIdentityAuthenticationProvider, error) {
	var credential azcore.TokenCredential
	var err error

	// we only support private key authentication for ms 365
	switch p.cred.Type {
	case vault.CredentialType_pkcs12:
		certs, privateKey, err := azidentity.ParseCertificates(p.cred.Secret, []byte(p.cred.Password))
		if err != nil {
			return nil, errors.Wrap(err, "could not parse pfx file")
		}

		credential, err = azidentity.NewClientCertificateCredential(p.tenantID, p.clientID, certs, privateKey, &azidentity.ClientCertificateCredentialOptions{})
		if err != nil {
			return nil, errors.Wrap(err, "error creating credentials")
		}
	case vault.CredentialType_password:
		credential, err = azidentity.NewClientSecretCredential(p.tenantID, p.clientID, string(p.cred.Secret), &azidentity.ClientSecretCredentialOptions{})
		if err != nil {
			return nil, errors.Wrap(err, "error creating credentials")
		}
	default:
		return nil, errors.New("invalid secret configuration for ms365 transport: " + p.cred.Type.String())
	}

	return a.NewAzureIdentityAuthenticationProviderWithScopes(credential, msgraphclient.DefaultMSGraphScopes)
}
