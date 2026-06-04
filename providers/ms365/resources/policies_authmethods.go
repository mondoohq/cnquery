// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"github.com/microsoftgraph/msgraph-sdk-go/models"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/types"
)

type mqlMicrosoftAuthenticationMethodsPolicyInternal struct {
	cachePolicy models.AuthenticationMethodsPolicyable
}

// authMethodTargets renders the include/exclude target lists of an
// authentication method (or the registration campaign) as dictionaries. Every
// target type shares the id and targetType accessors.
func authMethodTargets[T interface {
	GetId() *string
	GetTargetType() *models.AuthenticationMethodTargetType
}](targets []T) []any {
	res := []any{}
	for _, t := range targets {
		d := map[string]any{}
		if t.GetId() != nil {
			d["id"] = *t.GetId()
		}
		if tt := t.GetTargetType(); tt != nil {
			d["targetType"] = tt.String()
		}
		res = append(res, d)
	}
	return res
}

// findAuthMethodConfig returns the first method configuration of type C from the
// cached policy, or false if the policy did not include it.
func findAuthMethodConfig[C models.AuthenticationMethodConfigurationable](a *mqlMicrosoftAuthenticationMethodsPolicy) (C, bool) {
	var zero C
	if a.cachePolicy == nil {
		return zero, false
	}
	for _, cfg := range a.cachePolicy.GetAuthenticationMethodConfigurations() {
		if c, ok := cfg.(C); ok {
			return c, true
		}
	}
	return zero, false
}

func (a *mqlMicrosoftAuthenticationMethodsPolicy) fido2() (*mqlMicrosoftAuthenticationMethodsPolicyFido2, error) {
	cfg, ok := findAuthMethodConfig[models.Fido2AuthenticationMethodConfigurationable](a)
	if !ok {
		a.Fido2.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}

	var keyRestrictionsEnforced *bool
	var enforcementType *string
	aaGuids := []any{}
	if kr := cfg.GetKeyRestrictions(); kr != nil {
		keyRestrictionsEnforced = kr.GetIsEnforced()
		enforcementType = enumPtrString(kr.GetEnforcementType())
		for _, g := range kr.GetAaGuids() {
			aaGuids = append(aaGuids, g)
		}
	}

	res, err := CreateResource(a.MqlRuntime, "microsoft.authenticationMethodsPolicy.fido2",
		map[string]*llx.RawData{
			"__id":                             llx.StringData(a.Id.Data + "/fido2"),
			"state":                            llx.StringDataPtr(enumPtrString(cfg.GetState())),
			"includeTargets":                   llx.ArrayData(authMethodTargets(cfg.GetIncludeTargets()), types.Dict),
			"excludeTargets":                   llx.ArrayData(authMethodTargets(cfg.GetExcludeTargets()), types.Dict),
			"isAttestationEnforced":            llx.BoolDataPtr(cfg.GetIsAttestationEnforced()),
			"isSelfServiceRegistrationAllowed": llx.BoolDataPtr(cfg.GetIsSelfServiceRegistrationAllowed()),
			"keyRestrictionsEnforced":          llx.BoolDataPtr(keyRestrictionsEnforced),
			"keyRestrictionsEnforcementType":   llx.StringDataPtr(enforcementType),
			"keyRestrictionsAaGuids":           llx.ArrayData(aaGuids, types.String),
		})
	if err != nil {
		return nil, err
	}
	return res.(*mqlMicrosoftAuthenticationMethodsPolicyFido2), nil
}

func (a *mqlMicrosoftAuthenticationMethodsPolicy) microsoftAuthenticator() (*mqlMicrosoftAuthenticationMethodsPolicyMicrosoftAuthenticator, error) {
	cfg, ok := findAuthMethodConfig[models.MicrosoftAuthenticatorAuthenticationMethodConfigurationable](a)
	if !ok {
		a.MicrosoftAuthenticator.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}

	var displayApp, displayLocation *string
	if fs := cfg.GetFeatureSettings(); fs != nil {
		if c := fs.GetDisplayAppInformationRequiredState(); c != nil {
			displayApp = enumPtrString(c.GetState())
		}
		if c := fs.GetDisplayLocationInformationRequiredState(); c != nil {
			displayLocation = enumPtrString(c.GetState())
		}
	}

	res, err := CreateResource(a.MqlRuntime, "microsoft.authenticationMethodsPolicy.microsoftAuthenticator",
		map[string]*llx.RawData{
			"__id":                               llx.StringData(a.Id.Data + "/microsoftAuthenticator"),
			"state":                              llx.StringDataPtr(enumPtrString(cfg.GetState())),
			"includeTargets":                     llx.ArrayData(authMethodTargets(cfg.GetIncludeTargets()), types.Dict),
			"excludeTargets":                     llx.ArrayData(authMethodTargets(cfg.GetExcludeTargets()), types.Dict),
			"isSoftwareOathEnabled":              llx.BoolDataPtr(cfg.GetIsSoftwareOathEnabled()),
			"displayAppInformationRequiredState": llx.StringDataPtr(displayApp),
			"displayLocationInformationRequiredState": llx.StringDataPtr(displayLocation),
		})
	if err != nil {
		return nil, err
	}
	return res.(*mqlMicrosoftAuthenticationMethodsPolicyMicrosoftAuthenticator), nil
}

func (a *mqlMicrosoftAuthenticationMethodsPolicy) temporaryAccessPass() (*mqlMicrosoftAuthenticationMethodsPolicyTemporaryAccessPass, error) {
	cfg, ok := findAuthMethodConfig[models.TemporaryAccessPassAuthenticationMethodConfigurationable](a)
	if !ok {
		a.TemporaryAccessPass.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}

	res, err := CreateResource(a.MqlRuntime, "microsoft.authenticationMethodsPolicy.temporaryAccessPass",
		map[string]*llx.RawData{
			"__id":                     llx.StringData(a.Id.Data + "/temporaryAccessPass"),
			"state":                    llx.StringDataPtr(enumPtrString(cfg.GetState())),
			"includeTargets":           llx.ArrayData(authMethodTargets(cfg.GetIncludeTargets()), types.Dict),
			"excludeTargets":           llx.ArrayData(authMethodTargets(cfg.GetExcludeTargets()), types.Dict),
			"defaultLifetimeInMinutes": llx.IntDataPtr(cfg.GetDefaultLifetimeInMinutes()),
			"defaultLength":            llx.IntDataPtr(cfg.GetDefaultLength()),
			"minimumLifetimeInMinutes": llx.IntDataPtr(cfg.GetMinimumLifetimeInMinutes()),
			"maximumLifetimeInMinutes": llx.IntDataPtr(cfg.GetMaximumLifetimeInMinutes()),
			"isUsableOnce":             llx.BoolDataPtr(cfg.GetIsUsableOnce()),
		})
	if err != nil {
		return nil, err
	}
	return res.(*mqlMicrosoftAuthenticationMethodsPolicyTemporaryAccessPass), nil
}

func (a *mqlMicrosoftAuthenticationMethodsPolicy) email() (*mqlMicrosoftAuthenticationMethodsPolicyEmail, error) {
	cfg, ok := findAuthMethodConfig[models.EmailAuthenticationMethodConfigurationable](a)
	if !ok {
		a.Email.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}

	res, err := CreateResource(a.MqlRuntime, "microsoft.authenticationMethodsPolicy.email",
		map[string]*llx.RawData{
			"__id":                         llx.StringData(a.Id.Data + "/email"),
			"state":                        llx.StringDataPtr(enumPtrString(cfg.GetState())),
			"includeTargets":               llx.ArrayData(authMethodTargets(cfg.GetIncludeTargets()), types.Dict),
			"excludeTargets":               llx.ArrayData(authMethodTargets(cfg.GetExcludeTargets()), types.Dict),
			"allowExternalIdToUseEmailOtp": llx.StringDataPtr(enumPtrString(cfg.GetAllowExternalIdToUseEmailOtp())),
		})
	if err != nil {
		return nil, err
	}
	return res.(*mqlMicrosoftAuthenticationMethodsPolicyEmail), nil
}

func (a *mqlMicrosoftAuthenticationMethodsPolicy) voice() (*mqlMicrosoftAuthenticationMethodsPolicyVoice, error) {
	cfg, ok := findAuthMethodConfig[models.VoiceAuthenticationMethodConfigurationable](a)
	if !ok {
		a.Voice.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}

	res, err := CreateResource(a.MqlRuntime, "microsoft.authenticationMethodsPolicy.voice",
		map[string]*llx.RawData{
			"__id":                 llx.StringData(a.Id.Data + "/voice"),
			"state":                llx.StringDataPtr(enumPtrString(cfg.GetState())),
			"includeTargets":       llx.ArrayData(authMethodTargets(cfg.GetIncludeTargets()), types.Dict),
			"excludeTargets":       llx.ArrayData(authMethodTargets(cfg.GetExcludeTargets()), types.Dict),
			"isOfficePhoneAllowed": llx.BoolDataPtr(cfg.GetIsOfficePhoneAllowed()),
		})
	if err != nil {
		return nil, err
	}
	return res.(*mqlMicrosoftAuthenticationMethodsPolicyVoice), nil
}

func (a *mqlMicrosoftAuthenticationMethodsPolicy) registrationCampaign() (*mqlMicrosoftAuthenticationMethodsPolicyRegistrationCampaign, error) {
	if a.cachePolicy == nil {
		a.RegistrationCampaign.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	enforcement := a.cachePolicy.GetRegistrationEnforcement()
	if enforcement == nil {
		a.RegistrationCampaign.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	campaign := enforcement.GetAuthenticationMethodsRegistrationCampaign()
	if campaign == nil {
		a.RegistrationCampaign.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}

	res, err := CreateResource(a.MqlRuntime, "microsoft.authenticationMethodsPolicy.registrationCampaign",
		map[string]*llx.RawData{
			"__id":                 llx.StringData(a.Id.Data + "/registrationCampaign"),
			"state":                llx.StringDataPtr(enumPtrString(campaign.GetState())),
			"snoozeDurationInDays": llx.IntDataPtr(campaign.GetSnoozeDurationInDays()),
			"includeTargets":       llx.ArrayData(authMethodTargets(campaign.GetIncludeTargets()), types.Dict),
			"excludeTargets":       llx.ArrayData(authMethodTargets(campaign.GetExcludeTargets()), types.Dict),
		})
	if err != nil {
		return nil, err
	}
	return res.(*mqlMicrosoftAuthenticationMethodsPolicyRegistrationCampaign), nil
}
