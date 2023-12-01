// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/wafv2"
	waftypes "github.com/aws/aws-sdk-go-v2/service/wafv2/types"

	//"github.com/aws/aws-sdk-go/aws"
	"go.mondoo.com/cnquery/v9/llx"
	"go.mondoo.com/cnquery/v9/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v9/providers/aws/connection"
	"go.mondoo.com/cnquery/v9/types"
)

type mqlAwsWafRuleInternal struct {
	lock sync.Mutex
	rule waftypes.Rule
}

type mqlAwsWafRuleStatementInternal struct {
	lock sync.Mutex
	rule waftypes.Rule
}

func (a *mqlAwsWaf) id() (string, error) {
	return "aws.waf", nil
}

func (a *mqlAwsWafAcl) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAwsWafRule) id() (string, error) {
	return a.Name.Data, nil
}

func (a *mqlAwsWafRulegroup) id() (string, error) {
	return a.Name.Data, nil
}

func (a *mqlAwsWafIpset) id() (string, error) {
	return a.Name.Data, nil
}

func (a *mqlAwsWaf) acls() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	//waf := a.Id.Data

	region := ""
	svc := conn.Wafv2(region)
	ctx := context.Background()
	acls := []interface{}{}
	//scope := "REGIONAL"
	nextMarker := aws.String("No-Marker-to-begin-with")
	var scope waftypes.Scope
	scope = "REGIONAL"
	params := &wafv2.ListWebACLsInput{Scope: scope}
	for nextMarker != nil {
		aclsRes, err := svc.ListWebACLs(ctx, params)
		if err != nil {
			return nil, err
		}
		nextMarker = aclsRes.NextMarker
		if aclsRes.NextMarker != nil {
			params.NextMarker = nextMarker
		}

		for _, acl := range aclsRes.WebACLs {
			params := &wafv2.GetWebACLInput{
				Id:    acl.Id,
				Name:  acl.Name,
				Scope: scope,
			}
			aclDetails, err := svc.GetWebACL(ctx, params)
			if err != nil {
				return nil, err
			}
			mqlAcl, err := CreateResource(a.MqlRuntime, "aws.waf.acl",
				map[string]*llx.RawData{
					"id":                       llx.StringDataPtr(acl.Id),
					"arn":                      llx.StringDataPtr(acl.ARN),
					"name":                     llx.StringDataPtr(acl.Name),
					"description":              llx.StringDataPtr(acl.Description),
					"managedByFirewallManager": llx.BoolData(aclDetails.WebACL.ManagedByFirewallManager),
				},
			)
			if err != nil {
				return nil, err
			}
			acls = append(acls, mqlAcl)
		}
	}
	return acls, nil
}

func (a *mqlAwsWafAcl) rules() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	//waf := a.Id.Data

	var scope waftypes.Scope
	scope = "REGIONAL"
	ctx := context.Background()
	region := ""
	svc := conn.Wafv2(region)
	rules := []interface{}{}
	params := &wafv2.GetWebACLInput{
		Id:    &a.Id.Data,
		Name:  &a.Name.Data,
		Scope: scope,
	}
	aclDetails, err := svc.GetWebACL(ctx, params)
	if err != nil {
		return nil, err
	}
	for _, rule := range aclDetails.WebACL.Rules {
		ruleStatement, err := convert.JsonToDict(rule.Statement)
		if err != nil {
			return nil, err
		}
		ruleAction, err := convert.JsonToDict(rule.Action)
		mqlRule, err := CreateResource(a.MqlRuntime, "aws.waf.rule",
			map[string]*llx.RawData{
				"name":      llx.StringDataPtr(rule.Name),
				"priority":  llx.IntData(int64(rule.Priority)),
				"statement": llx.DictData(ruleStatement),
				"action":    llx.DictData(ruleAction),
			},
		)
		if err != nil {
			return nil, err
		}
		mqlRule.(*mqlAwsWafRule).rule = rule
		rules = append(rules, mqlRule)
	}
	return rules, nil
}

func (a *mqlAwsWafRule) statement() (*mqlAwsWafRuleStatement, error) {
	//conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	//var scope waftypes.Scope
	//scope = "REGIONAL"
	//ctx := context.Background()
	//region := ""
	//svc := conn.Wafv2(region)
	var statement *mqlAwsWafRuleStatement
	//svc.listRule
	return statement, nil
}

func (a *mqlAwsWafRuleStatement) andStatement() (*mqlAwsWafRuleStatement, error) {
	//conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	//var scope waftypes.Scope
	//scope = "REGIONAL"
	//ctx := context.Background()
	//region := ""
	//svc := conn.Wafv2(region)
	var statement *mqlAwsWafRuleStatement
	//svc.listRule
	return statement, nil
}

func (a *mqlAwsWafRulegroup) rules() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	//waf := a.Id.Data

	var scope waftypes.Scope
	scope = "REGIONAL"
	ctx := context.Background()
	region := ""
	svc := conn.Wafv2(region)
	rules := []interface{}{}
	params := &wafv2.GetWebACLInput{
		Id:    &a.Id.Data,
		Name:  &a.Name.Data,
		Scope: scope,
	}
	aclDetails, err := svc.GetWebACL(ctx, params)
	if err != nil {
		return nil, err
	}
	for _, rule := range aclDetails.WebACL.Rules {
		mqlRule, err := CreateResource(a.MqlRuntime, "aws.waf.rule",
			map[string]*llx.RawData{
				"name":     llx.StringDataPtr(rule.Name),
				"priority": llx.IntData(int64(rule.Priority)),
			},
		)
		if err != nil {
			return nil, err
		}
		rules = append(rules, mqlRule)
	}
	return rules, nil
}

func (a *mqlAwsWaf) ruleGroups() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	//waf := a.Id.Data

	region := ""
	svc := conn.Wafv2(region)
	ctx := context.Background()
	acls := []interface{}{}
	//scope := "REGIONAL"
	nextMarker := aws.String("No-Marker-to-begin-with")
	var scope waftypes.Scope
	scope = "REGIONAL"
	params := &wafv2.ListRuleGroupsInput{Scope: scope}
	for nextMarker != nil {
		aclsRes, err := svc.ListRuleGroups(ctx, params)
		if err != nil {
			return nil, err
		}
		nextMarker = aclsRes.NextMarker
		if aclsRes.NextMarker != nil {
			params.NextMarker = nextMarker
		}

		for _, ruleGroup := range aclsRes.RuleGroups {
			mqlRuleGroup, err := CreateResource(a.MqlRuntime, "aws.waf.rulegroup",
				map[string]*llx.RawData{
					"id":          llx.StringDataPtr(ruleGroup.Id),
					"arn":         llx.StringDataPtr(ruleGroup.ARN),
					"name":        llx.StringDataPtr(ruleGroup.Name),
					"description": llx.StringDataPtr(ruleGroup.Description),
				},
			)
			if err != nil {
				return nil, err
			}
			acls = append(acls, mqlRuleGroup)
		}
	}
	return acls, nil
}

func (a *mqlAwsWaf) ipSets() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	//waf := a.Id.Data

	region := ""
	svc := conn.Wafv2(region)
	ctx := context.Background()
	acls := []interface{}{}
	//scope := "REGIONAL"
	nextMarker := aws.String("No-Marker-to-begin-with")
	var scope waftypes.Scope
	scope = "REGIONAL"
	params := &wafv2.ListIPSetsInput{Scope: scope}
	for nextMarker != nil {
		aclsRes, err := svc.ListIPSets(ctx, params)
		if err != nil {
			return nil, err
		}
		nextMarker = aclsRes.NextMarker
		if aclsRes.NextMarker != nil {
			params.NextMarker = nextMarker
		}

		for _, ipset := range aclsRes.IPSets {
			params := &wafv2.GetIPSetInput{
				Id:    ipset.Id,
				Name:  ipset.Name,
				Scope: scope,
			}
			ipsetDetails, err := svc.GetIPSet(ctx, params)
			if err != nil {
				return nil, err
			}
			ipsetAddresses := convert.SliceAnyToInterface(ipsetDetails.IPSet.Addresses)
			if err != nil {
				return nil, err
			}
			mqlIPSet, err := CreateResource(a.MqlRuntime, "aws.waf.ipset",
				map[string]*llx.RawData{
					"id":          llx.StringDataPtr(ipset.Id),
					"arn":         llx.StringDataPtr(ipset.ARN),
					"name":        llx.StringDataPtr(ipset.Name),
					"description": llx.StringDataPtr(ipset.Description),
					"addressType": llx.StringDataPtr((*string)(&ipsetDetails.IPSet.IPAddressVersion)),
					"addresses":   llx.ArrayData(ipsetAddresses, types.String),
				},
			)
			if err != nil {
				return nil, err
			}
			acls = append(acls, mqlIPSet)
		}
	}
	return acls, nil
}

// Root structure to match the top-level JSON object
type Statement struct {
	SqliMatchStatement *SqliMatchStatement `json:"SqliMatchStatement,omitempty"`
}

// SqliMatchStatement represents the SQL injection match statement
type SqliMatchStatement struct {
	FieldToMatch        FieldToMatch         `json:"FieldToMatch"`
	TextTransformations []TextTransformation `json:"TextTransformations"`
	SensitivityLevel    string               `json:"SensitivityLevel"`
}

// FieldToMatch represents the field to match in SQL injection match statement
type FieldToMatch struct {
	SingleHeader *SingleHeader `json:"SingleHeader,omitempty"`
	// Add other fields as necessary
}

// SingleHeader represents the single header field to match
type SingleHeader struct {
	Name string `json:"Name"`
}

// TextTransformation represents a text transformation in SQL injection match statement
type TextTransformation struct {
	Priority int    `json:"Priority"`
	Type     string `json:"Type"`
}
