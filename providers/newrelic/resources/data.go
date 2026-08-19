// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"strconv"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/newrelic/connection"
)

// -----------------------------------------------------------------------------
// drop rule
// -----------------------------------------------------------------------------

// mqlNewrelicDropRuleInternal keeps the account the rule applies to and the
// person that created it.
type mqlNewrelicDropRuleInternal struct {
	cachedAccountID int
	cachedCreatorID string
}

// dropRuleCreatorID picks the creator's user ID out of the two places New Relic
// reports it. The creator object is the richer one but is absent for a rule
// registered by a system rather than a person, in which case the numeric
// createdBy is still there.
func dropRuleCreatorID(rule apiDropRule) string {
	if rule.Creator != nil && rule.Creator.ID > 0 {
		return strconv.Itoa(rule.Creator.ID)
	}
	if rule.CreatedBy > 0 {
		return strconv.Itoa(rule.CreatedBy)
	}
	return ""
}

func newDropRuleResource(runtime *plugin.Runtime, rule apiDropRule) (*mqlNewrelicDropRule, error) {
	res, err := CreateResource(runtime, "newrelic.dropRule", map[string]*llx.RawData{
		"__id":        llx.StringData("dropRule/" + strconv.Itoa(rule.AccountID) + "/" + rule.ID),
		"id":          llx.StringData(rule.ID),
		"action":      llx.StringData(rule.Action),
		"nrql":        llx.StringData(rule.Nrql),
		"description": llx.StringData(rule.Description),
		"source":      llx.StringData(rule.Source),
		"createdAt":   llx.TimeDataPtr(rule.CreatedAt.Time()),
	})
	if err != nil {
		return nil, err
	}

	mqlRule := res.(*mqlNewrelicDropRule)
	mqlRule.cachedAccountID = rule.AccountID
	mqlRule.cachedCreatorID = dropRuleCreatorID(rule)
	return mqlRule, nil
}

func (r *mqlNewrelicDropRule) account() (*mqlNewrelicAccount, error) {
	accountID := r.cachedAccountID
	if accountID <= 0 {
		// The list is fetched for one account, so a rule that does not repeat
		// the account ID still belongs to the connected one.
		conn, err := newrelicConn(r.MqlRuntime)
		if err != nil {
			return nil, err
		}
		accountID = conn.AccountID()
	}

	account, found, err := resolveAccount(r.MqlRuntime, accountID)
	if err != nil {
		return nil, err
	}
	if !found {
		r.Account.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	return account, nil
}

func (r *mqlNewrelicDropRule) creator() (*mqlNewrelicUser, error) {
	user, found, err := resolveUser(r.MqlRuntime, r.cachedCreatorID)
	if err != nil {
		return nil, err
	}
	if !found {
		r.Creator.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	return user, nil
}

// -----------------------------------------------------------------------------
// data retention rule
// -----------------------------------------------------------------------------

// mqlNewrelicDataRetentionRuleInternal keeps the accounts that created and
// deleted the rule.
type mqlNewrelicDataRetentionRuleInternal struct {
	cachedCreatedByID string
	cachedDeletedByID string
}

// retentionRuleID builds the cache key of a retention rule. A namespace carries
// both the rule in force and every rule that was deleted before it, so the
// namespace alone does not identify one, and the account is included because
// the same namespace exists on every account.
func retentionRuleID(conn *connection.NewrelicConnection, rule apiRetentionRule) string {
	return "dataRetentionRule/" + strconv.Itoa(conn.AccountID()) + "/" + rule.Namespace + "/" + rule.ID
}

func newRetentionRuleResource(runtime *plugin.Runtime, rule apiRetentionRule) (*mqlNewrelicDataRetentionRule, error) {
	conn, err := newrelicConn(runtime)
	if err != nil {
		return nil, err
	}

	res, err := CreateResource(runtime, "newrelic.dataRetentionRule", map[string]*llx.RawData{
		"__id":            llx.StringData(retentionRuleID(conn, rule)),
		"id":              llx.StringData(rule.ID),
		"namespace":       llx.StringData(rule.Namespace),
		"retentionInDays": llx.IntData(int64(rule.RetentionInDays)),
		"active":          llx.BoolData(isRetentionRuleActive(rule)),
		"createdAt":       llx.TimeDataPtr(rule.CreatedAt.Time()),
		"deletedAt":       llx.TimeDataPtr(rule.DeletedAt.Time()),
	})
	if err != nil {
		return nil, err
	}

	mqlRule := res.(*mqlNewrelicDataRetentionRule)
	mqlRule.cachedCreatedByID = rule.CreatedByID
	mqlRule.cachedDeletedByID = rule.DeletedByID
	return mqlRule, nil
}

func (r *mqlNewrelicDataRetentionRule) createdBy() (*mqlNewrelicUser, error) {
	user, found, err := resolveUser(r.MqlRuntime, r.cachedCreatedByID)
	if err != nil {
		return nil, err
	}
	if !found {
		r.CreatedBy.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	return user, nil
}

func (r *mqlNewrelicDataRetentionRule) deletedBy() (*mqlNewrelicUser, error) {
	user, found, err := resolveUser(r.MqlRuntime, r.cachedDeletedByID)
	if err != nil {
		return nil, err
	}
	if !found {
		r.DeletedBy.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	return user, nil
}
