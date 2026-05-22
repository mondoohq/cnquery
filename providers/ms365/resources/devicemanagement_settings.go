// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"github.com/microsoftgraph/msgraph-sdk-go/models"
)

func compliancePolicyPlatform(policy models.DeviceCompliancePolicyable) string {
	if policy == nil {
		return "unknown"
	}
	t := policy.GetOdataType()
	if t == nil {
		return "unknown"
	}
	switch trimOdataType(*t) {
	case "iosCompliancePolicy":
		return "iOS"
	case "androidCompliancePolicy":
		return "android"
	case "androidWorkProfileCompliancePolicy":
		return "androidWorkProfile"
	case "windows10CompliancePolicy":
		return "windows10"
	case "windows10MobileCompliancePolicy":
		return "windows10Mobile"
	case "windows81CompliancePolicy":
		return "windows81"
	case "windowsPhone81CompliancePolicy":
		return "windowsPhone81"
	case "macOSCompliancePolicy":
		return "macOS"
	}
	return "unknown"
}

func complianceSettings(policy models.DeviceCompliancePolicyable) map[string]any {
	s := map[string]any{}
	if policy == nil {
		return s
	}
	if t := policy.GetOdataType(); t != nil {
		s["@odata.type"] = *t
	}

	switch p := policy.(type) {
	case *models.IosCompliancePolicy:
		setBool(s, "passcodeRequired", p.GetPasscodeRequired())
		setBool(s, "passcodeBlockSimple", p.GetPasscodeBlockSimple())
		setInt32(s, "passcodeMinimumLength", p.GetPasscodeMinimumLength())
		setInt32(s, "passcodeMinimumCharacterSetCount", p.GetPasscodeMinimumCharacterSetCount())
		setInt32(s, "passcodeExpirationDays", p.GetPasscodeExpirationDays())
		setInt32(s, "passcodePreviousPasscodeBlockCount", p.GetPasscodePreviousPasscodeBlockCount())
		setInt32(s, "passcodeMinutesOfInactivityBeforeLock", p.GetPasscodeMinutesOfInactivityBeforeLock())
		if v := p.GetPasscodeRequiredType(); v != nil {
			s["passcodeRequiredType"] = v.String()
		}
		setString(s, "osMinimumVersion", p.GetOsMinimumVersion())
		setString(s, "osMaximumVersion", p.GetOsMaximumVersion())
		setBool(s, "securityBlockJailbrokenDevices", p.GetSecurityBlockJailbrokenDevices())
		setBool(s, "deviceThreatProtectionEnabled", p.GetDeviceThreatProtectionEnabled())
		if v := p.GetDeviceThreatProtectionRequiredSecurityLevel(); v != nil {
			s["deviceThreatProtectionRequiredSecurityLevel"] = v.String()
		}
		setBool(s, "managedEmailProfileRequired", p.GetManagedEmailProfileRequired())

	case *models.AndroidCompliancePolicy:
		setBool(s, "passwordRequired", p.GetPasswordRequired())
		setInt32(s, "passwordMinimumLength", p.GetPasswordMinimumLength())
		setInt32(s, "passwordExpirationDays", p.GetPasswordExpirationDays())
		setInt32(s, "passwordPreviousPasswordBlockCount", p.GetPasswordPreviousPasswordBlockCount())
		setInt32(s, "passwordMinutesOfInactivityBeforeLock", p.GetPasswordMinutesOfInactivityBeforeLock())
		if v := p.GetPasswordRequiredType(); v != nil {
			s["passwordRequiredType"] = v.String()
		}
		setString(s, "osMinimumVersion", p.GetOsMinimumVersion())
		setString(s, "osMaximumVersion", p.GetOsMaximumVersion())
		setString(s, "minAndroidSecurityPatchLevel", p.GetMinAndroidSecurityPatchLevel())
		setBool(s, "storageRequireEncryption", p.GetStorageRequireEncryption())
		setBool(s, "securityBlockJailbrokenDevices", p.GetSecurityBlockJailbrokenDevices())
		setBool(s, "securityDisableUsbDebugging", p.GetSecurityDisableUsbDebugging())
		setBool(s, "securityPreventInstallAppsFromUnknownSources", p.GetSecurityPreventInstallAppsFromUnknownSources())
		setBool(s, "securityRequireCompanyPortalAppIntegrity", p.GetSecurityRequireCompanyPortalAppIntegrity())
		setBool(s, "securityRequireGooglePlayServices", p.GetSecurityRequireGooglePlayServices())
		setBool(s, "securityRequireSafetyNetAttestationBasicIntegrity", p.GetSecurityRequireSafetyNetAttestationBasicIntegrity())
		setBool(s, "securityRequireSafetyNetAttestationCertifiedDevice", p.GetSecurityRequireSafetyNetAttestationCertifiedDevice())
		setBool(s, "securityRequireUpToDateSecurityProviders", p.GetSecurityRequireUpToDateSecurityProviders())
		setBool(s, "securityRequireVerifyApps", p.GetSecurityRequireVerifyApps())
		setBool(s, "deviceThreatProtectionEnabled", p.GetDeviceThreatProtectionEnabled())
		if v := p.GetDeviceThreatProtectionRequiredSecurityLevel(); v != nil {
			s["deviceThreatProtectionRequiredSecurityLevel"] = v.String()
		}

	case *models.AndroidWorkProfileCompliancePolicy:
		setBool(s, "passwordRequired", p.GetPasswordRequired())
		setInt32(s, "passwordMinimumLength", p.GetPasswordMinimumLength())
		setInt32(s, "passwordExpirationDays", p.GetPasswordExpirationDays())
		setInt32(s, "passwordPreviousPasswordBlockCount", p.GetPasswordPreviousPasswordBlockCount())
		setInt32(s, "passwordMinutesOfInactivityBeforeLock", p.GetPasswordMinutesOfInactivityBeforeLock())
		if v := p.GetPasswordRequiredType(); v != nil {
			s["passwordRequiredType"] = v.String()
		}
		setString(s, "osMinimumVersion", p.GetOsMinimumVersion())
		setString(s, "osMaximumVersion", p.GetOsMaximumVersion())
		setString(s, "minAndroidSecurityPatchLevel", p.GetMinAndroidSecurityPatchLevel())
		setBool(s, "storageRequireEncryption", p.GetStorageRequireEncryption())
		setBool(s, "securityBlockJailbrokenDevices", p.GetSecurityBlockJailbrokenDevices())
		setBool(s, "securityDisableUsbDebugging", p.GetSecurityDisableUsbDebugging())
		setBool(s, "securityPreventInstallAppsFromUnknownSources", p.GetSecurityPreventInstallAppsFromUnknownSources())
		setBool(s, "securityRequireCompanyPortalAppIntegrity", p.GetSecurityRequireCompanyPortalAppIntegrity())
		setBool(s, "securityRequireGooglePlayServices", p.GetSecurityRequireGooglePlayServices())
		setBool(s, "securityRequireSafetyNetAttestationBasicIntegrity", p.GetSecurityRequireSafetyNetAttestationBasicIntegrity())
		setBool(s, "securityRequireSafetyNetAttestationCertifiedDevice", p.GetSecurityRequireSafetyNetAttestationCertifiedDevice())
		setBool(s, "securityRequireUpToDateSecurityProviders", p.GetSecurityRequireUpToDateSecurityProviders())
		setBool(s, "securityRequireVerifyApps", p.GetSecurityRequireVerifyApps())
		setBool(s, "deviceThreatProtectionEnabled", p.GetDeviceThreatProtectionEnabled())
		if v := p.GetDeviceThreatProtectionRequiredSecurityLevel(); v != nil {
			s["deviceThreatProtectionRequiredSecurityLevel"] = v.String()
		}

	case *models.Windows10CompliancePolicy:
		setBool(s, "passwordRequired", p.GetPasswordRequired())
		setBool(s, "passwordBlockSimple", p.GetPasswordBlockSimple())
		setInt32(s, "passwordMinimumLength", p.GetPasswordMinimumLength())
		setInt32(s, "passwordMinimumCharacterSetCount", p.GetPasswordMinimumCharacterSetCount())
		setInt32(s, "passwordExpirationDays", p.GetPasswordExpirationDays())
		setInt32(s, "passwordPreviousPasswordBlockCount", p.GetPasswordPreviousPasswordBlockCount())
		setInt32(s, "passwordMinutesOfInactivityBeforeLock", p.GetPasswordMinutesOfInactivityBeforeLock())
		setBool(s, "passwordRequiredToUnlockFromIdle", p.GetPasswordRequiredToUnlockFromIdle())
		if v := p.GetPasswordRequiredType(); v != nil {
			s["passwordRequiredType"] = v.String()
		}
		setString(s, "osMinimumVersion", p.GetOsMinimumVersion())
		setString(s, "osMaximumVersion", p.GetOsMaximumVersion())
		setString(s, "mobileOsMinimumVersion", p.GetMobileOsMinimumVersion())
		setString(s, "mobileOsMaximumVersion", p.GetMobileOsMaximumVersion())
		setBool(s, "storageRequireEncryption", p.GetStorageRequireEncryption())
		setBool(s, "bitLockerEnabled", p.GetBitLockerEnabled())
		setBool(s, "secureBootEnabled", p.GetSecureBootEnabled())
		setBool(s, "codeIntegrityEnabled", p.GetCodeIntegrityEnabled())
		setBool(s, "earlyLaunchAntiMalwareDriverEnabled", p.GetEarlyLaunchAntiMalwareDriverEnabled())
		setBool(s, "requireHealthyDeviceReport", p.GetRequireHealthyDeviceReport())

	case *models.MacOSCompliancePolicy:
		setBool(s, "passwordRequired", p.GetPasswordRequired())
		setBool(s, "passwordBlockSimple", p.GetPasswordBlockSimple())
		setInt32(s, "passwordMinimumLength", p.GetPasswordMinimumLength())
		setInt32(s, "passwordMinimumCharacterSetCount", p.GetPasswordMinimumCharacterSetCount())
		setInt32(s, "passwordExpirationDays", p.GetPasswordExpirationDays())
		setInt32(s, "passwordPreviousPasswordBlockCount", p.GetPasswordPreviousPasswordBlockCount())
		setInt32(s, "passwordMinutesOfInactivityBeforeLock", p.GetPasswordMinutesOfInactivityBeforeLock())
		if v := p.GetPasswordRequiredType(); v != nil {
			s["passwordRequiredType"] = v.String()
		}
		setString(s, "osMinimumVersion", p.GetOsMinimumVersion())
		setString(s, "osMaximumVersion", p.GetOsMaximumVersion())
		setBool(s, "storageRequireEncryption", p.GetStorageRequireEncryption())
		setBool(s, "systemIntegrityProtectionEnabled", p.GetSystemIntegrityProtectionEnabled())
		setBool(s, "firewallEnabled", p.GetFirewallEnabled())
		setBool(s, "firewallBlockAllIncoming", p.GetFirewallBlockAllIncoming())
		setBool(s, "firewallEnableStealthMode", p.GetFirewallEnableStealthMode())
		setBool(s, "deviceThreatProtectionEnabled", p.GetDeviceThreatProtectionEnabled())
		if v := p.GetDeviceThreatProtectionRequiredSecurityLevel(); v != nil {
			s["deviceThreatProtectionRequiredSecurityLevel"] = v.String()
		}
	}
	return s
}

func scheduledActionsForRule(policy models.DeviceCompliancePolicyable) []any {
	if policy == nil {
		return nil
	}
	rules := policy.GetScheduledActionsForRule()
	if len(rules) == 0 {
		return nil
	}
	out := make([]any, 0, len(rules))
	for _, rule := range rules {
		entry := map[string]any{}
		if v := rule.GetRuleName(); v != nil {
			entry["ruleName"] = *v
		}
		actions := []any{}
		for _, a := range rule.GetScheduledActionConfigurations() {
			ai := map[string]any{}
			if v := a.GetActionType(); v != nil {
				ai["actionType"] = v.String()
			}
			if v := a.GetGracePeriodHours(); v != nil {
				ai["gracePeriodHours"] = int64(*v)
			}
			if v := a.GetNotificationTemplateId(); v != nil {
				ai["notificationTemplateId"] = *v
			}
			if cc := a.GetNotificationMessageCCList(); len(cc) > 0 {
				list := make([]any, len(cc))
				for i, c := range cc {
					list[i] = c
				}
				ai["notificationMessageCCList"] = list
			}
			actions = append(actions, ai)
		}
		entry["scheduledActionConfigurations"] = actions
		out = append(out, entry)
	}
	return out
}

func deviceConfigurationPlatform(config models.DeviceConfigurationable) string {
	if config == nil {
		return "unknown"
	}
	t := config.GetOdataType()
	if t == nil {
		return "unknown"
	}
	switch trimOdataType(*t) {
	case "iosGeneralDeviceConfiguration":
		return "iOS"
	case "androidGeneralDeviceConfiguration":
		return "android"
	case "androidWorkProfileGeneralDeviceConfiguration":
		return "androidWorkProfile"
	case "macOSGeneralDeviceConfiguration":
		return "macOS"
	case "windows10GeneralConfiguration":
		return "windows10"
	}
	return "unknown"
}

func deviceConfigurationSettings(config models.DeviceConfigurationable) map[string]any {
	s := map[string]any{}
	if config == nil {
		return s
	}
	if t := config.GetOdataType(); t != nil {
		s["@odata.type"] = *t
	}

	switch c := config.(type) {
	case *models.IosGeneralDeviceConfiguration:
		setBool(s, "passcodeRequired", c.GetPasscodeRequired())
		setBool(s, "passcodeBlockSimple", c.GetPasscodeBlockSimple())
		setInt32(s, "passcodeMinimumLength", c.GetPasscodeMinimumLength())
		setInt32(s, "passcodeExpirationDays", c.GetPasscodeExpirationDays())
		setInt32(s, "passcodeMinutesOfInactivityBeforeLock", c.GetPasscodeMinutesOfInactivityBeforeLock())
		setInt32(s, "passcodeMinutesOfInactivityBeforeScreenTimeout", c.GetPasscodeMinutesOfInactivityBeforeScreenTimeout())
		setInt32(s, "passcodeSignInFailureCountBeforeWipe", c.GetPasscodeSignInFailureCountBeforeWipe())
		if v := c.GetPasscodeRequiredType(); v != nil {
			s["passcodeRequiredType"] = v.String()
		}

	case *models.AndroidGeneralDeviceConfiguration:
		setBool(s, "passwordRequired", c.GetPasswordRequired())
		setInt32(s, "passwordMinimumLength", c.GetPasswordMinimumLength())
		setInt32(s, "passwordExpirationDays", c.GetPasswordExpirationDays())
		setInt32(s, "passwordMinutesOfInactivityBeforeScreenTimeout", c.GetPasswordMinutesOfInactivityBeforeScreenTimeout())
		setInt32(s, "passwordSignInFailureCountBeforeFactoryReset", c.GetPasswordSignInFailureCountBeforeFactoryReset())
		if v := c.GetPasswordRequiredType(); v != nil {
			s["passwordRequiredType"] = v.String()
		}
		setBool(s, "storageRequireDeviceEncryption", c.GetStorageRequireDeviceEncryption())

	case *models.AndroidWorkProfileGeneralDeviceConfiguration:
		setInt32(s, "passwordMinimumLength", c.GetPasswordMinimumLength())
		setInt32(s, "passwordExpirationDays", c.GetPasswordExpirationDays())
		setInt32(s, "passwordMinutesOfInactivityBeforeScreenTimeout", c.GetPasswordMinutesOfInactivityBeforeScreenTimeout())
		setInt32(s, "passwordSignInFailureCountBeforeFactoryReset", c.GetPasswordSignInFailureCountBeforeFactoryReset())
		if v := c.GetPasswordRequiredType(); v != nil {
			s["passwordRequiredType"] = v.String()
		}
		setInt32(s, "workProfilePasswordMinutesOfInactivityBeforeScreenTimeout", c.GetWorkProfilePasswordMinutesOfInactivityBeforeScreenTimeout())
		if v := c.GetWorkProfilePasswordRequiredType(); v != nil {
			s["workProfilePasswordRequiredType"] = v.String()
		}

	case *models.MacOSGeneralDeviceConfiguration:
		setBool(s, "passwordRequired", c.GetPasswordRequired())
		setBool(s, "passwordBlockSimple", c.GetPasswordBlockSimple())
		setInt32(s, "passwordMinimumLength", c.GetPasswordMinimumLength())
		setInt32(s, "passwordExpirationDays", c.GetPasswordExpirationDays())
		setInt32(s, "passwordMinutesOfInactivityBeforeLock", c.GetPasswordMinutesOfInactivityBeforeLock())
		setInt32(s, "passwordMinutesOfInactivityBeforeScreenTimeout", c.GetPasswordMinutesOfInactivityBeforeScreenTimeout())
		if v := c.GetPasswordRequiredType(); v != nil {
			s["passwordRequiredType"] = v.String()
		}

	case *models.Windows10GeneralConfiguration:
		setBool(s, "passwordRequired", c.GetPasswordRequired())
		setBool(s, "passwordBlockSimple", c.GetPasswordBlockSimple())
		setInt32(s, "passwordMinimumLength", c.GetPasswordMinimumLength())
		setInt32(s, "passwordExpirationDays", c.GetPasswordExpirationDays())
		setInt32(s, "passwordMinutesOfInactivityBeforeScreenTimeout", c.GetPasswordMinutesOfInactivityBeforeScreenTimeout())
		setInt32(s, "passwordSignInFailureCountBeforeFactoryReset", c.GetPasswordSignInFailureCountBeforeFactoryReset())
		if v := c.GetPasswordRequiredType(); v != nil {
			s["passwordRequiredType"] = v.String()
		}
	}
	return s
}

func enrollmentConfigurationKind(config models.DeviceEnrollmentConfigurationable) string {
	if config == nil {
		return "unknown"
	}
	t := config.GetOdataType()
	if t == nil {
		return "unknown"
	}
	switch trimOdataType(*t) {
	case "deviceEnrollmentLimitConfiguration":
		return "limit"
	case "deviceEnrollmentPlatformRestrictionsConfiguration":
		return "platformRestrictions"
	case "deviceEnrollmentWindowsHelloForBusinessConfiguration":
		return "windowsHelloForBusiness"
	}
	return "unknown"
}

func enrollmentConfigurationSettings(config models.DeviceEnrollmentConfigurationable) map[string]any {
	s := map[string]any{}
	if config == nil {
		return s
	}
	if t := config.GetOdataType(); t != nil {
		s["@odata.type"] = *t
	}

	switch c := config.(type) {
	case *models.DeviceEnrollmentLimitConfiguration:
		setInt32(s, "limit", c.GetLimit())

	case *models.DeviceEnrollmentPlatformRestrictionsConfiguration:
		if r := c.GetIosRestriction(); r != nil {
			s["iosRestriction"] = platformRestrictionMap(r)
		}
		if r := c.GetAndroidRestriction(); r != nil {
			s["androidRestriction"] = platformRestrictionMap(r)
		}
		if r := c.GetMacOSRestriction(); r != nil {
			s["macOSRestriction"] = platformRestrictionMap(r)
		}
		if r := c.GetWindowsRestriction(); r != nil {
			s["windowsRestriction"] = platformRestrictionMap(r)
		}
		if r := c.GetWindowsMobileRestriction(); r != nil {
			s["windowsMobileRestriction"] = platformRestrictionMap(r)
		}

	case *models.DeviceEnrollmentWindowsHelloForBusinessConfiguration:
		if v := c.GetState(); v != nil {
			s["state"] = v.String()
		}
		setInt32(s, "pinMinimumLength", c.GetPinMinimumLength())
		setInt32(s, "pinMaximumLength", c.GetPinMaximumLength())
		setInt32(s, "pinExpirationInDays", c.GetPinExpirationInDays())
		setInt32(s, "pinPreviousBlockCount", c.GetPinPreviousBlockCount())
		if v := c.GetPinUppercaseCharactersUsage(); v != nil {
			s["pinUppercaseCharactersUsage"] = v.String()
		}
		if v := c.GetPinLowercaseCharactersUsage(); v != nil {
			s["pinLowercaseCharactersUsage"] = v.String()
		}
		if v := c.GetPinSpecialCharactersUsage(); v != nil {
			s["pinSpecialCharactersUsage"] = v.String()
		}
		setBool(s, "securityDeviceRequired", c.GetSecurityDeviceRequired())
		setBool(s, "unlockWithBiometricsEnabled", c.GetUnlockWithBiometricsEnabled())
		if v := c.GetEnhancedBiometricsState(); v != nil {
			s["enhancedBiometricsState"] = v.String()
		}
		setBool(s, "remotePassportEnabled", c.GetRemotePassportEnabled())
	}
	return s
}

func platformRestrictionMap(r models.DeviceEnrollmentPlatformRestrictionable) map[string]any {
	m := map[string]any{}
	setBool(m, "platformBlocked", r.GetPlatformBlocked())
	setBool(m, "personalDeviceEnrollmentBlocked", r.GetPersonalDeviceEnrollmentBlocked())
	setString(m, "osMinimumVersion", r.GetOsMinimumVersion())
	setString(m, "osMaximumVersion", r.GetOsMaximumVersion())
	return m
}

func setBool(m map[string]any, key string, v *bool) {
	if v != nil {
		m[key] = *v
	}
}

func setString(m map[string]any, key string, v *string) {
	if v != nil {
		m[key] = *v
	}
}

func setInt32(m map[string]any, key string, v *int32) {
	if v != nil {
		m[key] = int64(*v)
	}
}
