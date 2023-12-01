// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/wafv2"
	waftypes "github.com/aws/aws-sdk-go-v2/service/wafv2/types"

	//"github.com/aws/aws-sdk-go/aws"
	"go.mondoo.com/cnquery/v9/llx"
	"go.mondoo.com/cnquery/v9/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v9/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v9/providers/aws/connection"
	"go.mondoo.com/cnquery/v9/types"
)

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
		var statement plugin.Resource
		if rule.Statement != nil {
			var sqlimatchstatement plugin.Resource
			var xssmatchstatement plugin.Resource
			var bytematchstatement plugin.Resource
			if rule.Statement.ByteMatchStatement != nil {
				var fieldToMatch plugin.Resource
				if rule.Statement.ByteMatchStatement.FieldToMatch != nil {
					var singleHeader plugin.Resource
					if rule.Statement.ByteMatchStatement.FieldToMatch.SingleHeader != nil {
						singleHeader, err = CreateResource(a.MqlRuntime, "aws.waf.rule.statement.bytematchstatement.fieldtomatch.singleheader", map[string]*llx.RawData{
							"name": llx.StringDataPtr(rule.Statement.ByteMatchStatement.FieldToMatch.SingleHeader.Name),
						})
						if err != nil {
							return nil, err
						}
					}
					fieldToMatch, err = CreateResource(a.MqlRuntime, "aws.waf.rule.statement.bytematchstatement.fieldtomatch", map[string]*llx.RawData{
						"singleHeader": llx.ResourceData(singleHeader, "aws.waf.rule.statement.bytematchstatement.fieldtomatch.singleheader"),
					})
					if err != nil {
						return nil, err
					}
				}
				bytematchstatement, err = CreateResource(a.MqlRuntime, "aws.waf.rule.statement.bytematchstatement", map[string]*llx.RawData{
					"fieldToMatch": llx.ResourceData(fieldToMatch, "aws.waf.rule.statement.bytematchstatement.fieldtomatch"),
				})
				if err != nil {
					return nil, err
				}
			}
			if rule.Statement.XssMatchStatement != nil {
				var fieldToMatch plugin.Resource
				if rule.Statement.XssMatchStatement.FieldToMatch != nil {
					var singleHeader plugin.Resource
					if rule.Statement.XssMatchStatement.FieldToMatch.SingleHeader != nil {
						singleHeader, err = CreateResource(a.MqlRuntime, "aws.waf.rule.statement.xssmatchstatement.fieldtomatch.singleheader", map[string]*llx.RawData{
							"name": llx.StringDataPtr(rule.Statement.XssMatchStatement.FieldToMatch.SingleHeader.Name),
						})
						if err != nil {
							return nil, err
						}
					}
					fieldToMatch, err = CreateResource(a.MqlRuntime, "aws.waf.rule.statement.xssmatchstatement.fieldtomatch", map[string]*llx.RawData{
						"singleHeader": llx.ResourceData(singleHeader, "aws.waf.rule.statement.xssmatchstatement.fieldtomatch.singleheader"),
					})
					if err != nil {
						return nil, err
					}
				}
				xssmatchstatement, err = CreateResource(a.MqlRuntime, "aws.waf.rule.statement.xssmatchstatement", map[string]*llx.RawData{
					"fieldToMatch": llx.ResourceData(fieldToMatch, "aws.waf.rule.statement.xssmatchstatement.fieldtomatch"),
				})
				if err != nil {
					return nil, err
				}
			}
			if rule.Statement.SqliMatchStatement != nil {
				var fieldToMatch plugin.Resource
				if rule.Statement.SqliMatchStatement.FieldToMatch != nil {
					var singleHeader plugin.Resource
					if rule.Statement.SqliMatchStatement.FieldToMatch.SingleHeader != nil {
						singleHeader, err = CreateResource(a.MqlRuntime, "aws.waf.rule.statement.sqlimatchstatement.fieldtomatch.singleheader", map[string]*llx.RawData{
							"name": llx.StringDataPtr(rule.Statement.SqliMatchStatement.FieldToMatch.SingleHeader.Name),
						})
						if err != nil {
							return nil, err
						}
					}
					fieldToMatch, err = CreateResource(a.MqlRuntime, "aws.waf.rule.statement.sqlimatchstatement.fieldtomatch", map[string]*llx.RawData{
						"singleHeader": llx.ResourceData(singleHeader, "aws.waf.rule.statement.sqlimatchstatement.fieldtomatch.singleheader"),
					})
					if err != nil {
						return nil, err
					}
				}
				sqlimatchstatement, err = CreateResource(a.MqlRuntime, "aws.waf.rule.statement.sqlimatchstatement", map[string]*llx.RawData{
					"fieldToMatch": llx.ResourceData(fieldToMatch, "aws.waf.rule.statement.sqlimatchstatement.fieldtomatch"),
				})
				if err != nil {
					return nil, err
				}
			}
			statement, err = CreateResource(a.MqlRuntime, "aws.waf.rule.statement", map[string]*llx.RawData{
				"sqliMatchStatement": llx.ResourceData(sqlimatchstatement, "aws.waf.rule.statement.sqlimatchstatement"),
				"xssMatchStatement":  llx.ResourceData(xssmatchstatement, "aws.waf.rule.statement.xssmatchstatement"),
				"byteMatchStatement": llx.ResourceData(bytematchstatement, "aws.waf.rule.statement.bytematchstatement"),
			})
			if err != nil {
				return nil, err
			}
			fmt.Println("Created statement:", statement)
		}
		fmt.Println("Statement:", statement)
		ruleAction, err := convert.JsonToDict(rule.Action)
		mqlRule, err := CreateResource(a.MqlRuntime, "aws.waf.rule",
			map[string]*llx.RawData{
				"name":      llx.StringDataPtr(rule.Name),
				"priority":  llx.IntData(int64(rule.Priority)),
				"action":    llx.DictData(ruleAction),
				"statement": llx.ResourceData(statement, "aws.waf.rule.statement"),
			},
		)
		if err != nil {
			return nil, err
		}
		rules = append(rules, mqlRule)
	}
	return rules, nil
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
