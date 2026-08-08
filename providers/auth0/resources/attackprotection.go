// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/types"
)

// attackProtection reads the tenant-wide brute-force, suspicious-IP, and
// breached-password defenses. Its cache key is the tenant domain, since there
// is exactly one attack-protection configuration per tenant.
func (a *mqlAuth0) attackProtection() (*mqlAuth0AttackProtection, error) {
	conn := a.conn()
	client := conn.Client()
	ctx := context.Background()

	bruteForce, err := client.AttackProtection.GetBruteForceProtection(ctx)
	if err != nil {
		return nil, err
	}
	suspicious, err := client.AttackProtection.GetSuspiciousIPThrottling(ctx)
	if err != nil {
		return nil, err
	}
	breached, err := client.AttackProtection.GetBreachedPasswordDetection(ctx)
	if err != nil {
		return nil, err
	}

	var preLoginMaxAttempts, preRegistrationMaxAttempts *int
	if suspicious.Stage != nil {
		if suspicious.Stage.PreLogin != nil {
			preLoginMaxAttempts = suspicious.Stage.PreLogin.MaxAttempts
		}
		if suspicious.Stage.PreUserRegistration != nil {
			preRegistrationMaxAttempts = suspicious.Stage.PreUserRegistration.MaxAttempts
		}
	}

	res, err := CreateResource(a.MqlRuntime, "auth0.attackProtection", map[string]*llx.RawData{
		"__id":                                                llx.StringData(conn.Domain()),
		"bruteForceEnabled":                                   llx.BoolDataPtr(bruteForce.Enabled),
		"bruteForceMode":                                      llx.StringDataPtr(bruteForce.Mode),
		"bruteForceMaxAttempts":                               llx.IntDataPtr(bruteForce.MaxAttempts),
		"bruteForceAllowlist":                                 llx.ArrayData(strList(bruteForce.AllowList), types.String),
		"suspiciousIpThrottlingEnabled":                       llx.BoolDataPtr(suspicious.Enabled),
		"suspiciousIpThrottlingAllowlist":                     llx.ArrayData(strList(suspicious.AllowList), types.String),
		"suspiciousIpThrottlingPreLoginMaxAttempts":           llx.IntDataPtr(preLoginMaxAttempts),
		"suspiciousIpThrottlingPreRegistrationMaxAttempts":    llx.IntDataPtr(preRegistrationMaxAttempts),
		"breachedPasswordDetectionEnabled":                    llx.BoolDataPtr(breached.Enabled),
		"breachedPasswordDetectionMethod":                     llx.StringDataPtr(breached.Method),
		"breachedPasswordDetectionShields":                    llx.ArrayData(strList(breached.Shields), types.String),
		"breachedPasswordDetectionAdminNotificationFrequency": llx.ArrayData(strList(breached.AdminNotificationFrequency), types.String),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAuth0AttackProtection), nil
}
