package microsoft

// These roles only apply for MSGraph. They are not used for azure resources
var DefaultRoles = []string{
	"Application.Read.All",
	"AuditLog.Read.All",
	"Calendars.Read",
	"Device.Read.All",
	"Group.Read.All",
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

func (p *Provider) MissingRoles(checkRoles ...string) []string {
	missing := []string{}
	roles := p.Roles()
	for _, role := range checkRoles {
		_, ok := roles[role]
		if !ok {
			missing = append(missing, role)
		}
	}
	return missing
}
