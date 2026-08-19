// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"strconv"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/types"
)

// -----------------------------------------------------------------------------
// alert policy
// -----------------------------------------------------------------------------

// mqlNewrelicAlertPolicyInternal keeps the account the policy belongs to.
type mqlNewrelicAlertPolicyInternal struct {
	cachedAccountID int
}

func newAlertPolicyResource(runtime *plugin.Runtime, policy apiAlertPolicy) (*mqlNewrelicAlertPolicy, error) {
	res, err := CreateResource(runtime, "newrelic.alertPolicy", map[string]*llx.RawData{
		"__id":               llx.StringData("alertPolicy/" + strconv.Itoa(policy.AccountID) + "/" + policy.ID),
		"id":                 llx.StringData(policy.ID),
		"name":               llx.StringData(policy.Name),
		"incidentPreference": llx.StringData(policy.IncidentPreference),
	})
	if err != nil {
		return nil, err
	}

	mqlPolicy := res.(*mqlNewrelicAlertPolicy)
	mqlPolicy.cachedAccountID = policy.AccountID
	return mqlPolicy, nil
}

func (r *mqlNewrelicAlertPolicy) account() (*mqlNewrelicAccount, error) {
	account, found, err := resolveAccount(r.MqlRuntime, r.cachedAccountID)
	if err != nil {
		return nil, err
	}
	if !found {
		r.Account.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	return account, nil
}

// conditions lists the conditions evaluated under the policy by filtering the
// account's condition list, which is fetched once. Asking New Relic for the
// conditions of one policy would work too, but it would cost one search per
// policy and the search filter is the kind of argument that returns a wrong
// answer rather than an error when it is off.
func (r *mqlNewrelicAlertPolicy) conditions() ([]any, error) {
	conditions, err := cachedAlertConditions(r.MqlRuntime)
	if err != nil {
		return nil, err
	}

	res := []any{}
	for _, condition := range conditions {
		if condition.PolicyID != r.Id.Data {
			continue
		}
		mqlCondition, err := newAlertConditionResource(r.MqlRuntime, condition)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlCondition)
	}
	return res, nil
}

// resolveAlertPolicy hands back the policy resource for an ID, or null when the
// policy is not in the account's policy list.
func resolveAlertPolicy(runtime *plugin.Runtime, policyID string) (*mqlNewrelicAlertPolicy, bool, error) {
	if policyID == "" {
		return nil, false, nil
	}

	policies, err := cachedAlertPolicies(runtime)
	if err != nil {
		return nil, false, err
	}
	for _, policy := range policies {
		if policy.ID == policyID {
			mqlPolicy, err := newAlertPolicyResource(runtime, policy)
			if err != nil {
				return nil, false, err
			}
			return mqlPolicy, true, nil
		}
	}
	return nil, false, nil
}

// -----------------------------------------------------------------------------
// alert condition
// -----------------------------------------------------------------------------

// mqlNewrelicAlertConditionInternal keeps the policy the condition runs under.
type mqlNewrelicAlertConditionInternal struct {
	cachedPolicyID string
}

// alertTermsDict renders a condition's thresholds as dicts. The shape is fixed
// across every condition type, and none of the five values is worth a resource
// of its own.
func alertTermsDict(terms []apiAlertTerm) []any {
	out := make([]any, 0, len(terms))
	for _, term := range terms {
		out = append(out, map[string]any{
			"operator":             term.Operator,
			"priority":             term.Priority,
			"threshold":            term.Threshold,
			"thresholdDuration":    int64(term.ThresholdDuration),
			"thresholdOccurrences": term.ThresholdOccurrences,
		})
	}
	return out
}

func newAlertConditionResource(runtime *plugin.Runtime, condition apiAlertCondition) (*mqlNewrelicAlertCondition, error) {
	res, err := CreateResource(runtime, "newrelic.alertCondition", map[string]*llx.RawData{
		"__id":                      llx.StringData("alertCondition/" + condition.ID),
		"id":                        llx.StringData(condition.ID),
		"name":                      llx.StringData(condition.Name),
		"description":               llx.StringData(condition.Description),
		"enabled":                   llx.BoolData(condition.Enabled),
		"type":                      llx.StringData(condition.Type),
		"nrql":                      llx.StringData(condition.Nrql.Query),
		"runbookUrl":                llx.StringData(condition.RunbookURL),
		"violationTimeLimitSeconds": llx.IntData(int64(condition.ViolationTimeLimitSeconds)),
		"terms":                     llx.ArrayData(alertTermsDict(condition.Terms), types.Dict),
	})
	if err != nil {
		return nil, err
	}

	mqlCondition := res.(*mqlNewrelicAlertCondition)
	mqlCondition.cachedPolicyID = condition.PolicyID
	return mqlCondition, nil
}

func (r *mqlNewrelicAlertCondition) policy() (*mqlNewrelicAlertPolicy, error) {
	policy, found, err := resolveAlertPolicy(r.MqlRuntime, r.cachedPolicyID)
	if err != nil {
		return nil, err
	}
	if !found {
		r.Policy.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	return policy, nil
}

// -----------------------------------------------------------------------------
// notification destination
// -----------------------------------------------------------------------------

// mqlNewrelicNotificationDestinationInternal keeps the account the destination
// belongs to.
type mqlNewrelicNotificationDestinationInternal struct {
	cachedAccountID int
}

func newNotificationDestinationResource(runtime *plugin.Runtime, destination apiNotificationDestination) (*mqlNewrelicNotificationDestination, error) {
	res, err := CreateResource(runtime, "newrelic.notificationDestination", map[string]*llx.RawData{
		"__id":              llx.StringData("notificationDestination/" + destination.ID),
		"id":                llx.StringData(destination.ID),
		"name":              llx.StringData(destination.Name),
		"type":              llx.StringData(destination.Type),
		"active":            llx.BoolData(destination.Active),
		"status":            llx.StringData(destination.Status),
		"userAuthenticated": llx.BoolData(destination.IsUserAuthenticated),
		"createdAt":         llx.TimeDataPtr(destination.CreatedAt.Time()),
		"updatedAt":         llx.TimeDataPtr(destination.UpdatedAt.Time()),
		"lastSent":          llx.TimeDataPtr(destination.LastSent.Time()),
	})
	if err != nil {
		return nil, err
	}

	mqlDestination := res.(*mqlNewrelicNotificationDestination)
	mqlDestination.cachedAccountID = destination.AccountID
	return mqlDestination, nil
}

func (r *mqlNewrelicNotificationDestination) account() (*mqlNewrelicAccount, error) {
	account, found, err := resolveAccount(r.MqlRuntime, r.cachedAccountID)
	if err != nil {
		return nil, err
	}
	if !found {
		r.Account.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	return account, nil
}

// resolveNotificationDestination hands back the destination resource for an ID,
// or null when the destination is not in the account's destination list.
func resolveNotificationDestination(runtime *plugin.Runtime, destinationID string) (*mqlNewrelicNotificationDestination, bool, error) {
	if destinationID == "" {
		return nil, false, nil
	}

	destinations, err := cachedNotificationDestinations(runtime)
	if err != nil {
		return nil, false, err
	}
	for _, destination := range destinations {
		if destination.ID == destinationID {
			mqlDestination, err := newNotificationDestinationResource(runtime, destination)
			if err != nil {
				return nil, false, err
			}
			return mqlDestination, true, nil
		}
	}
	return nil, false, nil
}

// -----------------------------------------------------------------------------
// notification channel
// -----------------------------------------------------------------------------

// mqlNewrelicNotificationChannelInternal keeps the destination and account the
// channel points at.
type mqlNewrelicNotificationChannelInternal struct {
	cachedDestinationID string
	cachedAccountID     int
}

func newNotificationChannelResource(runtime *plugin.Runtime, channel apiNotificationChannel) (*mqlNewrelicNotificationChannel, error) {
	res, err := CreateResource(runtime, "newrelic.notificationChannel", map[string]*llx.RawData{
		"__id":      llx.StringData("notificationChannel/" + channel.ID),
		"id":        llx.StringData(channel.ID),
		"name":      llx.StringData(channel.Name),
		"type":      llx.StringData(channel.Type),
		"product":   llx.StringData(channel.Product),
		"active":    llx.BoolData(channel.Active),
		"status":    llx.StringData(channel.Status),
		"createdAt": llx.TimeDataPtr(channel.CreatedAt.Time()),
		"updatedAt": llx.TimeDataPtr(channel.UpdatedAt.Time()),
	})
	if err != nil {
		return nil, err
	}

	mqlChannel := res.(*mqlNewrelicNotificationChannel)
	mqlChannel.cachedDestinationID = channel.DestinationID
	mqlChannel.cachedAccountID = channel.AccountID
	return mqlChannel, nil
}

func (r *mqlNewrelicNotificationChannel) destination() (*mqlNewrelicNotificationDestination, error) {
	destination, found, err := resolveNotificationDestination(r.MqlRuntime, r.cachedDestinationID)
	if err != nil {
		return nil, err
	}
	if !found {
		r.Destination.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	return destination, nil
}

func (r *mqlNewrelicNotificationChannel) account() (*mqlNewrelicAccount, error) {
	account, found, err := resolveAccount(r.MqlRuntime, r.cachedAccountID)
	if err != nil {
		return nil, err
	}
	if !found {
		r.Account.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	return account, nil
}
