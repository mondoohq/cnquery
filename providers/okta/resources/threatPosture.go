// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"encoding/json"
	"time"

	"github.com/okta/okta-sdk-golang/v5/okta"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/okta/connection"
)

// --- attack protection (org-level singleton) ---

func (o *mqlOkta) attackProtection() (*mqlOktaAttackProtection, error) {
	conn := o.MqlRuntime.Connection.(*connection.OktaConnection)
	client := conn.Client()
	ctx := context.Background()

	var preventLockout bool
	lockout, _, err := client.AttackProtectionAPI.GetUserLockoutSettings(ctx).Execute()
	if err != nil {
		return nil, err
	}
	if len(lockout) > 0 && lockout[0].PreventBruteForceLockoutFromUnknownDevices != nil {
		preventLockout = *lockout[0].PreventBruteForceLockoutFromUnknownDevices
	}

	var verifyKnowledge bool
	authSettings, _, err := client.AttackProtectionAPI.GetAuthenticatorSettings(ctx).Execute()
	if err != nil {
		return nil, err
	}
	if len(authSettings) > 0 && authSettings[0].VerifyKnowledgeSecondWhen2faRequired != nil {
		verifyKnowledge = *authSettings[0].VerifyKnowledgeSecondWhen2faRequired
	}

	r, err := CreateResource(o.MqlRuntime, "okta.attackProtection", map[string]*llx.RawData{
		"__id": llx.StringData("okta.attackProtection"),
		"preventBruteForceLockoutFromUnknownDevices": llx.BoolData(preventLockout),
		"verifyKnowledgeSecondWhen2faRequired":       llx.BoolData(verifyKnowledge),
	})
	if err != nil {
		return nil, err
	}
	return r.(*mqlOktaAttackProtection), nil
}

// --- behavior detection rules ---

func (o *mqlOkta) behaviorRules() ([]any, error) {
	conn := o.MqlRuntime.Connection.(*connection.OktaConnection)
	client := conn.Client()

	ctx := context.Background()
	slice, resp, err := client.BehaviorAPI.ListBehaviorDetectionRules(ctx).Execute()
	if err != nil {
		return nil, err
	}

	list := []any{}
	appendEntry := func(datalist []okta.ListBehaviorDetectionRules200ResponseInner) error {
		for i := range datalist {
			r, err := newMqlOktaBehaviorRule(o.MqlRuntime, &datalist[i])
			if err != nil {
				return err
			}
			list = append(list, r)
		}
		return nil
	}

	if err := appendEntry(slice); err != nil {
		return nil, err
	}
	for resp != nil && resp.HasNextPage() {
		var page []okta.ListBehaviorDetectionRules200ResponseInner
		resp, err = resp.Next(&page)
		if err != nil {
			return nil, err
		}
		if err := appendEntry(page); err != nil {
			return nil, err
		}
	}
	return list, nil
}

// oktaBehaviorRuleRaw flattens the polymorphic behavior rule (anomalous
// device/IP/location or velocity). All variants share the base fields and a
// type-specific settings object, so re-marshaling to JSON gives one code path.
type oktaBehaviorRuleRaw struct {
	Id          string     `json:"id"`
	Name        string     `json:"name"`
	Type        string     `json:"type"`
	Status      string     `json:"status"`
	Settings    any        `json:"settings"`
	Created     *time.Time `json:"created"`
	LastUpdated *time.Time `json:"lastUpdated"`
}

func newMqlOktaBehaviorRule(runtime *plugin.Runtime, entry *okta.ListBehaviorDetectionRules200ResponseInner) (any, error) {
	raw, err := json.Marshal(entry.GetActualInstance())
	if err != nil {
		return nil, err
	}
	var rule oktaBehaviorRuleRaw
	if err := json.Unmarshal(raw, &rule); err != nil {
		return nil, err
	}
	settings, err := convert.JsonToDict(rule.Settings)
	if err != nil {
		return nil, err
	}

	return CreateResource(runtime, "okta.behaviorRule", map[string]*llx.RawData{
		"id":          llx.StringData(rule.Id),
		"name":        llx.StringData(rule.Name),
		"type":        llx.StringData(rule.Type),
		"status":      llx.StringData(rule.Status),
		"settings":    llx.DictData(settings),
		"created":     llx.TimeDataPtr(rule.Created),
		"lastUpdated": llx.TimeDataPtr(rule.LastUpdated),
	})
}

func (o *mqlOktaBehaviorRule) id() (string, error) {
	return "okta.behaviorRule/" + o.Id.Data, o.Id.Error
}

// --- risk providers ---

func (o *mqlOkta) riskProviders() ([]any, error) {
	conn := o.MqlRuntime.Connection.(*connection.OktaConnection)
	client := conn.Client()

	ctx := context.Background()
	slice, resp, err := client.RiskProviderAPI.ListRiskProviders(ctx).Execute()
	if err != nil {
		return nil, err
	}

	list := []any{}
	appendEntry := func(datalist []okta.RiskProvider) error {
		for i := range datalist {
			r, err := newMqlOktaRiskProvider(o.MqlRuntime, &datalist[i])
			if err != nil {
				return err
			}
			list = append(list, r)
		}
		return nil
	}

	if err := appendEntry(slice); err != nil {
		return nil, err
	}
	for resp != nil && resp.HasNextPage() {
		var page []okta.RiskProvider
		resp, err = resp.Next(&page)
		if err != nil {
			return nil, err
		}
		if err := appendEntry(page); err != nil {
			return nil, err
		}
	}
	return list, nil
}

func newMqlOktaRiskProvider(runtime *plugin.Runtime, entry *okta.RiskProvider) (any, error) {
	return CreateResource(runtime, "okta.riskProvider", map[string]*llx.RawData{
		"id":          llx.StringData(entry.Id),
		"name":        llx.StringData(entry.Name),
		"action":      llx.StringData(entry.Action),
		"clientId":    llx.StringData(entry.ClientId),
		"created":     llx.TimeDataPtr(entry.Created),
		"lastUpdated": llx.TimeDataPtr(entry.LastUpdated),
	})
}

func (o *mqlOktaRiskProvider) id() (string, error) {
	return "okta.riskProvider/" + o.Id.Data, o.Id.Error
}

// --- security events providers (shared signals framework) ---

func (o *mqlOkta) securityEventsProviders() ([]any, error) {
	conn := o.MqlRuntime.Connection.(*connection.OktaConnection)
	client := conn.Client()

	ctx := context.Background()
	slice, resp, err := client.SSFReceiverAPI.ListSecurityEventsProviderInstances(ctx).Execute()
	if err != nil {
		return nil, err
	}

	list := []any{}
	appendEntry := func(datalist []okta.SecurityEventsProviderResponse) error {
		for i := range datalist {
			r, err := newMqlOktaSecurityEventsProvider(o.MqlRuntime, &datalist[i])
			if err != nil {
				return err
			}
			list = append(list, r)
		}
		return nil
	}

	if err := appendEntry(slice); err != nil {
		return nil, err
	}
	for resp != nil && resp.HasNextPage() {
		var page []okta.SecurityEventsProviderResponse
		resp, err = resp.Next(&page)
		if err != nil {
			return nil, err
		}
		if err := appendEntry(page); err != nil {
			return nil, err
		}
	}
	return list, nil
}

func newMqlOktaSecurityEventsProvider(runtime *plugin.Runtime, entry *okta.SecurityEventsProviderResponse) (any, error) {
	settings, err := convert.JsonToDict(entry.Settings)
	if err != nil {
		return nil, err
	}

	return CreateResource(runtime, "okta.securityEventsProvider", map[string]*llx.RawData{
		"id":       llx.StringData(oktaStr(entry.Id)),
		"name":     llx.StringData(oktaStr(entry.Name)),
		"type":     llx.StringData(oktaStr(entry.Type)),
		"status":   llx.StringData(oktaStr(entry.Status)),
		"settings": llx.DictData(settings),
	})
}

func (o *mqlOktaSecurityEventsProvider) id() (string, error) {
	return "okta.securityEventsProvider/" + o.Id.Data, o.Id.Error
}
