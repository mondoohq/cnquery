// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"fmt"
	"slices"

	albwaf "github.com/stackitcloud/stackit-sdk-go/services/albwaf/v1betaapi"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
)

type mqlStackitAlbCustomRuleInternal struct {
	// cacheConditions holds the rule's match conditions, captured when the
	// rule is built from its group so conditions() can expose them as typed
	// sub-resources without another API call. cacheIdBase is the rule's own
	// cache key, used to key the condition sub-resources.
	cacheConditions []albwaf.Condition
	cacheIdBase     string
}

type mqlStackitAlbManagedRuleSetInternal struct {
	// cacheGroups holds the rule set's groups, captured during init so rules()
	// need not re-fetch the set. A nil pointer means "not yet fetched".
	cacheGroups *map[string]albwaf.MRSRuleGroup
}

type mqlStackitAlbCustomRuleGroupInternal struct {
	// cacheRules holds the group's rules, captured during init so rules() need
	// not re-fetch the group. A nil pointer means "not yet fetched".
	cacheRules *[]albwaf.GetCustomRule
}

// ------------------------- WAF configurations -------------------------

func (r *mqlStackit) albWafs() ([]any, error) {
	c := conn(r.MqlRuntime)
	client, err := c.AlbWaf()
	if err != nil {
		return nil, err
	}
	resp, err := client.DefaultAPI.ListWAF(bgctx(), c.ProjectID(), c.Region()).Execute()
	if err != nil {
		if isAccessDenied(err) || isNotFound(err) {
			return []any{}, nil
		}
		return nil, err
	}
	items, _ := resp.GetItemsOk()
	out := make([]any, 0, len(items))
	for i := range items {
		res, err := buildAlbWaf(r.MqlRuntime, &items[i])
		if err != nil {
			return nil, err
		}
		out = append(out, res)
	}
	return out, nil
}

func buildAlbWaf(runtime *plugin.Runtime, w *albwaf.GetWAFResponse) (plugin.Resource, error) {
	args := map[string]*llx.RawData{
		"name":                llx.StringData(w.GetName()),
		"managedRuleSetName":  llx.StringData(w.GetManagedRuleSetName()),
		"customRuleGroupName": llx.StringData(w.GetCustomRuleGroupName()),
		"labels":              labelData(w.GetLabels()),
	}
	return CreateResource(runtime, "stackit.alb.waf", args)
}

func (r *mqlStackitAlbWaf) id() (string, error) {
	return "stackit.alb.waf/" + conn(r.MqlRuntime).ProjectID() + "/" + r.Name.Data, nil
}

func initStackitAlbWaf(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	name, ok := idArg(args, "name")
	if !ok {
		return args, nil, nil
	}
	c := conn(runtime)
	client, err := c.AlbWaf()
	if err != nil {
		return nil, nil, err
	}
	resp, err := client.DefaultAPI.GetWAF(bgctx(), c.ProjectID(), c.Region(), name).Execute()
	if err != nil {
		return nil, nil, err
	}
	res, err := buildAlbWaf(runtime, resp)
	if err != nil {
		return nil, nil, err
	}
	return nil, res, nil
}

func (r *mqlStackitAlbWaf) managedRuleSet() (*mqlStackitAlbManagedRuleSet, error) {
	if r.ManagedRuleSetName.Data == "" {
		return markNull[mqlStackitAlbManagedRuleSet](&r.ManagedRuleSet)
	}
	res, err := NewResource(r.MqlRuntime, "stackit.alb.managedRuleSet", map[string]*llx.RawData{
		"name": llx.StringData(r.ManagedRuleSetName.Data),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlStackitAlbManagedRuleSet), nil
}

func (r *mqlStackitAlbWaf) customRuleGroup() (*mqlStackitAlbCustomRuleGroup, error) {
	if r.CustomRuleGroupName.Data == "" {
		return markNull[mqlStackitAlbCustomRuleGroup](&r.CustomRuleGroup)
	}
	res, err := NewResource(r.MqlRuntime, "stackit.alb.customRuleGroup", map[string]*llx.RawData{
		"name": llx.StringData(r.CustomRuleGroupName.Data),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlStackitAlbCustomRuleGroup), nil
}

// wafs resolves the WAF configurations referenced by the load balancer's
// listeners (via each listener's wafConfigName), deduplicated.
func (r *mqlStackitAlbLoadBalancer) wafs() ([]any, error) {
	seen := map[string]struct{}{}
	names := []string{}
	for _, raw := range r.Listeners.Data {
		listener, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		// "wafConfigName" is the SDK's json key for the WAF a listener
		// references (alb.Listener.WafConfigName); listeners without a WAF omit
		// it, so a missing key just means "no WAF on this listener".
		name, ok := listener["wafConfigName"].(string)
		if !ok || name == "" {
			continue
		}
		if _, dup := seen[name]; dup {
			continue
		}
		seen[name] = struct{}{}
		names = append(names, name)
	}
	out := make([]any, 0, len(names))
	for _, name := range names {
		res, err := NewResource(r.MqlRuntime, "stackit.alb.waf", map[string]*llx.RawData{
			"name": llx.StringData(name),
		})
		if err != nil {
			return nil, err
		}
		out = append(out, res)
	}
	return out, nil
}

// ------------------------- managed rule sets -------------------------

func (r *mqlStackitAlbManagedRuleSet) id() (string, error) {
	return "stackit.alb.managedRuleSet/" + conn(r.MqlRuntime).ProjectID() + "/" + r.Name.Data, nil
}

func initStackitAlbManagedRuleSet(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	name, ok := idArg(args, "name")
	if !ok {
		return args, nil, nil
	}
	c := conn(runtime)
	client, err := c.AlbWaf()
	if err != nil {
		return nil, nil, err
	}
	resp, err := client.DefaultAPI.GetManagedRuleSet(bgctx(), c.ProjectID(), c.Region(), name).Execute()
	if err != nil {
		return nil, nil, err
	}
	res, err := CreateResource(runtime, "stackit.alb.managedRuleSet", map[string]*llx.RawData{
		"name":    llx.StringData(resp.GetName()),
		"type":    llx.StringData(string(resp.GetType())),
		"version": llx.StringData(resp.GetVersion()),
	})
	if err != nil {
		return nil, nil, err
	}
	// Keep the groups from this call so rules() doesn't re-fetch the set.
	groups := resp.GetGroups()
	res.(*mqlStackitAlbManagedRuleSet).cacheGroups = &groups
	return nil, res, nil
}

// groups returns the rule set's groups, using the copy captured during init
// when available and fetching the set otherwise.
func (r *mqlStackitAlbManagedRuleSet) groups() (map[string]albwaf.MRSRuleGroup, error) {
	if r.cacheGroups != nil {
		return *r.cacheGroups, nil
	}
	c := conn(r.MqlRuntime)
	client, err := c.AlbWaf()
	if err != nil {
		return nil, err
	}
	resp, err := client.DefaultAPI.GetManagedRuleSet(bgctx(), c.ProjectID(), c.Region(), r.Name.Data).Execute()
	if err != nil {
		if isAccessDenied(err) || isNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	groups := resp.GetGroups()
	r.cacheGroups = &groups
	return groups, nil
}

// rules flattens the managed rule set's groups into individual rules,
// each carrying its group name and mode.
func (r *mqlStackitAlbManagedRuleSet) rules() ([]any, error) {
	c := conn(r.MqlRuntime)
	groups, err := r.groups()
	if err != nil {
		return nil, err
	}
	out := []any{}
	// Iterate groups and rules in sorted order for deterministic output.
	for _, groupName := range sortedKeys(groups) {
		group := groups[groupName]
		rules := group.GetRules()
		for _, ruleName := range sortedKeys(rules) {
			rule := rules[ruleName]
			res, err := CreateResource(r.MqlRuntime, "stackit.alb.managedRule", map[string]*llx.RawData{
				"__id":        llx.StringData("stackit.alb.managedRule/" + c.ProjectID() + "/" + r.Name.Data + "/" + groupName + "/" + ruleName),
				"name":        llx.StringData(ruleName),
				"groupName":   llx.StringData(groupName),
				"mode":        llx.StringData(string(rule.GetMode())),
				"severity":    llx.StringData(rule.GetSeverity()),
				"description": llx.StringData(rule.GetDescription()),
			})
			if err != nil {
				return nil, err
			}
			out = append(out, res)
		}
	}
	return out, nil
}

// ------------------------- custom rule groups -------------------------

func (r *mqlStackitAlbCustomRuleGroup) id() (string, error) {
	return "stackit.alb.customRuleGroup/" + conn(r.MqlRuntime).ProjectID() + "/" + r.Name.Data, nil
}

func initStackitAlbCustomRuleGroup(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	name, ok := idArg(args, "name")
	if !ok {
		return args, nil, nil
	}
	c := conn(runtime)
	client, err := c.AlbWaf()
	if err != nil {
		return nil, nil, err
	}
	resp, err := client.DefaultAPI.GetCustomRuleGroup(bgctx(), c.ProjectID(), c.Region(), name).Execute()
	if err != nil {
		return nil, nil, err
	}
	res, err := CreateResource(runtime, "stackit.alb.customRuleGroup", map[string]*llx.RawData{
		"name": llx.StringData(resp.GetName()),
	})
	if err != nil {
		return nil, nil, err
	}
	// Keep the rules from this call so rules() doesn't re-fetch the group.
	rules := resp.GetRules()
	res.(*mqlStackitAlbCustomRuleGroup).cacheRules = &rules
	return nil, res, nil
}

// customRules returns the group's rules, using the copy captured during init
// when available and fetching the group otherwise.
func (r *mqlStackitAlbCustomRuleGroup) customRules() ([]albwaf.GetCustomRule, error) {
	if r.cacheRules != nil {
		return *r.cacheRules, nil
	}
	c := conn(r.MqlRuntime)
	client, err := c.AlbWaf()
	if err != nil {
		return nil, err
	}
	resp, err := client.DefaultAPI.GetCustomRuleGroup(bgctx(), c.ProjectID(), c.Region(), r.Name.Data).Execute()
	if err != nil {
		if isAccessDenied(err) || isNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	rules := resp.GetRules()
	r.cacheRules = &rules
	return rules, nil
}

func (r *mqlStackitAlbCustomRuleGroup) rules() ([]any, error) {
	c := conn(r.MqlRuntime)
	rules, err := r.customRules()
	if err != nil {
		return nil, err
	}
	out := make([]any, 0, len(rules))
	for i := range rules {
		rule := rules[i]
		var (
			action   string
			logMatch bool
			logMsg   string
			severity string
		)
		if b, ok := rule.GetBehaviourOk(); ok && b != nil {
			action = string(b.GetAction())
			logMatch = b.GetLog()
			logMsg = b.GetLogMsg()
			severity = string(b.GetSeverity())
		}
		idBase := fmt.Sprintf("stackit.alb.customRule/%s/%s/%d", c.ProjectID(), r.Name.Data, rule.GetId())
		res, err := CreateResource(r.MqlRuntime, "stackit.alb.customRule", map[string]*llx.RawData{
			"__id":        llx.StringData(idBase),
			"id":          llx.IntData(int64(rule.GetId())),
			"description": llx.StringData(rule.GetDescription()),
			"action":      llx.StringData(action),
			"log":         llx.BoolData(logMatch),
			"logMsg":      llx.StringData(logMsg),
			"severity":    llx.StringData(severity),
		})
		if err != nil {
			return nil, err
		}
		cr := res.(*mqlStackitAlbCustomRule)
		cr.cacheConditions = rule.GetConditions()
		cr.cacheIdBase = idBase
		out = append(out, res)
	}
	return out, nil
}

// conditions exposes the rule's match conditions as typed sub-resources,
// captured when the rule was built from its group.
func (r *mqlStackitAlbCustomRule) conditions() ([]any, error) {
	out := make([]any, 0, len(r.cacheConditions))
	for i := range r.cacheConditions {
		cond := r.cacheConditions[i]
		var variableType, variableValue, operatorType, operatorValue string
		if v, ok := cond.GetVariableOk(); ok && v != nil {
			variableType = string(v.GetType())
			variableValue = v.GetValue()
		}
		if o, ok := cond.GetOperatorOk(); ok && o != nil {
			operatorType = string(o.GetType())
			operatorValue = o.GetValue()
		}
		transforms := make([]string, 0, len(cond.GetTransformations()))
		for _, t := range cond.GetTransformations() {
			transforms = append(transforms, string(t))
		}
		res, err := CreateResource(r.MqlRuntime, "stackit.alb.customRuleCondition", map[string]*llx.RawData{
			"__id":            llx.StringData(fmt.Sprintf("%s/condition/%d", r.cacheIdBase, i)),
			"variableType":    llx.StringData(variableType),
			"variableValue":   llx.StringData(variableValue),
			"operatorType":    llx.StringData(operatorType),
			"operatorValue":   llx.StringData(operatorValue),
			"transformations": strSliceData(transforms),
		})
		if err != nil {
			return nil, err
		}
		out = append(out, res)
	}
	return out, nil
}

// sortedKeys returns the keys of a map in sorted order for deterministic
// iteration.
func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}
