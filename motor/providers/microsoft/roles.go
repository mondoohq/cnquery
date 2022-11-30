package microsoft

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
	for i := range checkRoles {
		_, ok := p.rolesMap[checkRoles[i]]
		if !ok {
			missing = append(missing, checkRoles[i])
		}
	}
	return missing
}
