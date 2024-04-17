// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/wafv2"
	waftypes "github.com/aws/aws-sdk-go-v2/service/wafv2/types"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	//"github.com/aws/aws-sdk-go/aws"
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v11/providers/aws/connection"
	"go.mondoo.com/cnquery/v11/types"
)

func (a *mqlAwsWaf) id() (string, error) {
	return "aws.waf", nil
}

func (a *mqlAwsWafAcl) id() (string, error) {
	return a.Arn.Data, nil
}

func (a *mqlAwsWafRule) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAwsWafRuleStatement) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAwsWafRulegroup) id() (string, error) {
	return a.Arn.Data, nil
}

func (a *mqlAwsWafIpset) id() (string, error) {
	return a.Arn.Data, nil
}

func initAwsWaf(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 1 {
		return args, nil, nil
	}

	scope := ""
	if x, ok := args["scope"]; ok {
		scope = x.Value.(string)
	} else {
		scope = "CLOUDFRONT"
	}
	args["scope"] = llx.StringData(scope)

	log.Debug().Msgf("AWS WAF using scope: %s", scope)

	return args, nil, nil

}

func (a *mqlAwsWaf) acls() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	region := ""
	svc := conn.Wafv2(region)
	ctx := context.Background()
	acls := []interface{}{}
	nextMarker := aws.String("No-Marker-to-begin-with")
	scopeString := a.Scope.Data
	scope := waftypes.Scope(scopeString)
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
					"scope":                    llx.StringData(scopeString),
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

func (a *mqlAwsWafRuleStatementSqlimatchstatement) id() (string, error) {
	return a.StatementID.Data, nil
}

func (a *mqlAwsWafRuleStatementBytematchstatement) id() (string, error) {
	return a.StatementID.Data, nil
}

func (a *mqlAwsWafRuleStatementXssmatchstatement) id() (string, error) {
	return a.StatementID.Data, nil
}

func (a *mqlAwsWafRuleStatementRegexmatchstatement) id() (string, error) {
	return a.StatementID.Data, nil
}

func (a *mqlAwsWafRuleStatementGeomatchstatement) id() (string, error) {
	return a.StatementID.Data, nil
}

func (a *mqlAwsWafRuleStatementIpsetreferencestatement) id() (string, error) {
	return a.StatementID.Data, nil
}

func (a *mqlAwsWafRuleStatementLabelmatchstatement) id() (string, error) {
	return a.StatementID.Data, nil
}

func (a *mqlAwsWafRuleStatementManagedrulegroupstatement) id() (string, error) {
	return a.StatementID.Data, nil
}

func (a *mqlAwsWafRuleStatementNotstatement) id() (string, error) {
	return a.StatementID.Data, nil
}

func (a *mqlAwsWafRuleStatementOrstatement) id() (string, error) {
	return a.StatementID.Data, nil
}

func (a *mqlAwsWafRuleStatementAndstatement) id() (string, error) {
	return a.StatementID.Data, nil
}

func (a *mqlAwsWafRuleStatementRatebasedstatement) id() (string, error) {
	return "not implemented", nil
}

func (a *mqlAwsWafRuleStatementRegexpatternsetreferencestatement) id() (string, error) {
	return a.StatementID.Data, nil
}

func (a *mqlAwsWafRuleStatementRulegroupreferencestatement) id() (string, error) {
	return a.StatementID.Data, nil
}

func (a *mqlAwsWafRuleStatementSizeconstraintstatement) id() (string, error) {
	return a.StatementID.Data, nil
}

func (a *mqlAwsWafRuleFieldtomatch) id() (string, error) {
	return a.StatementID.Data, nil
}

func (a *mqlAwsWafRuleFieldtomatchBody) id() (string, error) {
	return a.StatementID.Data, nil
}

func (a *mqlAwsWafRuleFieldtomatchJsonbody) id() (string, error) {
	return a.StatementID.Data, nil
}

func (a *mqlAwsWafRuleFieldtomatchCookie) id() (string, error) {
	return a.StatementID.Data, nil
}

func (a *mqlAwsWafRuleFieldtomatchHeaders) id() (string, error) {
	return a.StatementID.Data, nil
}

func (a *mqlAwsWafRuleFieldtomatchHeaderorder) id() (string, error) {
	return a.StatementID.Data, nil
}

func (a *mqlAwsWafRuleFieldtomatchSingleheader) id() (string, error) {
	return a.StatementID.Data, nil
}

func (a *mqlAwsWafRuleFieldtomatchSinglequeryargument) id() (string, error) {
	return a.StatementID.Data, nil
}

func (a *mqlAwsWafRuleFieldtomatchHeadersMatchpattern) id() (string, error) {
	return a.StatementID.Data, nil
}

func (a *mqlAwsWafRuleFieldtomatchJsonbodyMatchpattern) id() (string, error) {
	return a.StatementID.Data, nil
}

func (a *mqlAwsWafRuleFieldtomatchJa3fingerprint) id() (string, error) {
	return a.StatementID.Data, nil
}

func (a *mqlAwsWafRulegroup) rules() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	scopeString := a.Scope.Data
	scope := waftypes.Scope(scopeString)
	ctx := context.Background()
	region := ""
	svc := conn.Wafv2(region)
	rules := []interface{}{}
	params := &wafv2.GetRuleGroupInput{
		Id:    &a.Id.Data,
		Name:  &a.Name.Data,
		Scope: scope,
	}
	ruleGroupDetails, err := svc.GetRuleGroup(ctx, params)
	if err != nil {
		return nil, err
	}
	for _, rule := range ruleGroupDetails.RuleGroup.Rules {
		ruleID := a.Arn.Data + "/" + *rule.Name
		mqlStatement, err := createStatementResource(a.MqlRuntime, rule.Statement, rule.Name, ruleID)
		ruleAction, err := createActionResource(a.MqlRuntime, rule.Action, rule.Name)
		mqlRule, err := CreateResource(a.MqlRuntime, "aws.waf.rule",
			map[string]*llx.RawData{
				"id":        llx.StringData(ruleID),
				"name":      llx.StringDataPtr(rule.Name),
				"priority":  llx.IntData(int64(rule.Priority)),
				"action":    llx.ResourceData(ruleAction, "aws.waf.rule.action"),
				"statement": llx.ResourceData(mqlStatement, "aws.waf.rule.statement"),
				"belongsTo": llx.StringData(a.Arn.Data),
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

	region := ""
	svc := conn.Wafv2(region)
	ctx := context.Background()
	acls := []interface{}{}
	nextMarker := aws.String("No-Marker-to-begin-with")
	scopeString := a.Scope.Data
	scope := waftypes.Scope(scopeString)
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
					"scope":       llx.StringData(scopeString),
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

	region := ""
	svc := conn.Wafv2(region)
	ctx := context.Background()
	acls := []interface{}{}
	nextMarker := aws.String("No-Marker-to-begin-with")
	scopeString := a.Scope.Data
	scope := waftypes.Scope(scopeString)
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
					"scope":       llx.StringData(scopeString),
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

func (a *mqlAwsWafAcl) rules() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	scopeString := a.Scope.Data
	scope := waftypes.Scope(scopeString)
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
		ruleID := a.Arn.Data + "/" + *rule.Name
		mqlStatement, err := createStatementResource(a.MqlRuntime, rule.Statement, rule.Name, ruleID)
		ruleAction, err := createActionResource(a.MqlRuntime, rule.Action, rule.Name)
		mqlRule, err := CreateResource(a.MqlRuntime, "aws.waf.rule",
			map[string]*llx.RawData{
				"id":        llx.StringData(ruleID),
				"name":      llx.StringDataPtr(rule.Name),
				"priority":  llx.IntData(int64(rule.Priority)),
				"action":    llx.ResourceData(ruleAction, "aws.waf.rule.action"),
				"statement": llx.ResourceData(mqlStatement, "aws.waf.rule.statement"),
				"belongsTo": llx.StringData(a.Arn.Data),
			},
		)
		if err != nil {
			return nil, err
		}
		rules = append(rules, mqlRule)
	}
	return rules, nil
}

func createActionResource(runtime *plugin.Runtime, ruleAction *waftypes.RuleAction, ruleName *string) (plugin.Resource, error) {
	var mqlAction plugin.Resource
	var err error

	var action string
	responseCode := ""
	if ruleAction != nil {
		if ruleAction.Allow != nil {
			action = "allow"
		}

		if ruleAction.Block != nil {
			action = "block"
			if ruleAction.Block.CustomResponse != nil {
				responseCodeNumber := *ruleAction.Block.CustomResponse.ResponseCode
				responseCode = string(responseCodeNumber)
			} else {
				responseCode = "403" // Default for Block
			}
		}

		if ruleAction.Count != nil {
			action = "count"
		}
		if ruleAction.Captcha != nil {
			action = "captcha"
		}
	}
	mqlAction, err = CreateResource(runtime, "aws.waf.rule.action", map[string]*llx.RawData{
		"ruleName":     llx.StringDataPtr(ruleName),
		"action":       llx.StringData(action),
		"responseCode": llx.StringData(responseCode),
	})
	return mqlAction, err
}

func createStatementResource(runtime *plugin.Runtime, statement *waftypes.Statement, ruleName *string, ruleID string) (plugin.Resource, error) {
	var err error
	var sqlimatchstatement plugin.Resource
	var xssmatchstatement plugin.Resource
	var bytematchstatement plugin.Resource
	var regexmatchstatement plugin.Resource
	var geomatchstatement plugin.Resource
	var ipsetreferencestatement plugin.Resource
	var labelmatchstatement plugin.Resource
	var managedrulegroupstatement plugin.Resource
	var andStatement plugin.Resource
	var notstatement plugin.Resource
	var orstatement plugin.Resource
	var ratebasedstatement plugin.Resource
	var regexpatternsetreferencestatement plugin.Resource
	var rulegroupreferencestatement plugin.Resource
	var sizeconstraintstatement plugin.Resource
	var statementJson map[string]interface{}
	mqlStatementID := ruleID + "/" + uuid.New().String()
	var kind string
	if statement != nil {
		statementJson, err = convert.JsonToDict(statement)
		if statement.RegexMatchStatement != nil {
			kind = "RegexMatchStatement"
			var fieldToMatch plugin.Resource
			fieldToMatch, err = createFieldToMatchResource(runtime, statement.RegexMatchStatement.FieldToMatch, ruleName, mqlStatementID)
			if statement.RegexMatchStatement.FieldToMatch != nil {
			}
			regexmatchstatement, err = CreateResource(runtime, "aws.waf.rule.statement.regexmatchstatement", map[string]*llx.RawData{
				"statementID":  llx.StringData(mqlStatementID),
				"ruleName":     llx.StringDataPtr(ruleName),
				"fieldToMatch": llx.ResourceData(fieldToMatch, "aws.waf.rule.statement.regexmatchstatement.fieldtomatch"),
				"regexString":  llx.StringDataPtr(statement.RegexMatchStatement.RegexString),
			})
			if err != nil {
				return nil, err
			}
		}
		if statement.ByteMatchStatement != nil {
			kind = "ByteMatchStatement"
			var fieldToMatch plugin.Resource
			fieldToMatch, err = createFieldToMatchResource(runtime, statement.ByteMatchStatement.FieldToMatch, ruleName, mqlStatementID)
			bytematchstatement, err = CreateResource(runtime, "aws.waf.rule.statement.bytematchstatement", map[string]*llx.RawData{
				"statementID":  llx.StringData(mqlStatementID),
				"ruleName":     llx.StringDataPtr(ruleName),
				"fieldToMatch": llx.ResourceData(fieldToMatch, "aws.waf.rule.fieldtomatch"),
				"searchString": llx.StringData(string(statement.ByteMatchStatement.SearchString)),
			})
			if err != nil {
				return nil, err
			}
		}
		if statement.XssMatchStatement != nil {
			kind = "XssMatchStatement"
			var fieldToMatch plugin.Resource
			fieldToMatch, err = createFieldToMatchResource(runtime, statement.XssMatchStatement.FieldToMatch, ruleName, mqlStatementID)
			xssmatchstatement, err = CreateResource(runtime, "aws.waf.rule.statement.xssmatchstatement", map[string]*llx.RawData{
				"statementID":  llx.StringData(mqlStatementID),
				"ruleName":     llx.StringDataPtr(ruleName),
				"fieldToMatch": llx.ResourceData(fieldToMatch, "aws.waf.rule.fieldtomatch"),
			})
			if err != nil {
				return nil, err
			}
		}
		if statement.SqliMatchStatement != nil {
			kind = "SqliMatchStatement"
			var fieldToMatch plugin.Resource
			fieldToMatch, err := createFieldToMatchResource(runtime, statement.SqliMatchStatement.FieldToMatch, ruleName, mqlStatementID)
			sqlimatchstatement, err = CreateResource(runtime, "aws.waf.rule.statement.sqlimatchstatement", map[string]*llx.RawData{
				"statementID":      llx.StringData(mqlStatementID),
				"ruleName":         llx.StringDataPtr(ruleName),
				"fieldToMatch":     llx.ResourceData(fieldToMatch, "aws.waf.rule.fieldtomatch"),
				"sensitivityLevel": llx.StringData(string(statement.SqliMatchStatement.SensitivityLevel)),
			})
			if err != nil {
				return nil, err
			}
		}
		if statement.GeoMatchStatement != nil {
			kind = "GeoMatchStatement"
			var countryCodes []string
			for _, countryCode := range statement.GeoMatchStatement.CountryCodes {
				countryCodes = append(countryCodes, string(countryCode))
			}
			countryCodesArray := convert.SliceAnyToInterface(countryCodes)
			geomatchstatement, err = CreateResource(runtime, "aws.waf.rule.statement.geomatchstatement", map[string]*llx.RawData{
				"statementID":  llx.StringData(mqlStatementID),
				"ruleName":     llx.StringDataPtr(ruleName),
				"countryCodes": llx.ArrayData(countryCodesArray, types.String),
			})
			if err != nil {
				return nil, err
			}
		}
		if statement.IPSetReferenceStatement != nil {
			kind = "IPSetReferenceStatement"
			var IPSetForwardedIPConfig plugin.Resource
			if statement.IPSetReferenceStatement.IPSetForwardedIPConfig != nil {
				IPSetForwardedIPConfig, err = CreateResource(runtime, "aws.waf.rule.statement.ipsetreferencestatement.ipsetforwardedipconfig", map[string]*llx.RawData{
					"statementID":      llx.StringData(mqlStatementID),
					"ruleName":         llx.StringDataPtr(ruleName),
					"headerName":       llx.StringDataPtr(statement.IPSetReferenceStatement.IPSetForwardedIPConfig.HeaderName),
					"position":         llx.StringData(string(statement.IPSetReferenceStatement.IPSetForwardedIPConfig.Position)),
					"fallbackBehavior": llx.StringData(string(statement.IPSetReferenceStatement.IPSetForwardedIPConfig.FallbackBehavior)),
				})
				if err != nil {
					return nil, err
				}
			}
			ipsetreferencestatement, err = CreateResource(runtime, "aws.waf.rule.statement.ipsetreferencestatement", map[string]*llx.RawData{
				"statementID":            llx.StringData(mqlStatementID),
				"ruleName":               llx.StringDataPtr(ruleName),
				"arn":                    llx.StringDataPtr(statement.IPSetReferenceStatement.ARN),
				"ipSetForwardedIPConfig": llx.ResourceData(IPSetForwardedIPConfig, "aws.waf.rule.statement.ipsetreferencestatement.ipsetforwardedipconfig"),
			})
			if err != nil {
				return nil, err
			}
		}
		if statement.LabelMatchStatement != nil {
			kind = "LabelMatchStatement"
			labelmatchstatement, err = CreateResource(runtime, "aws.waf.rule.statement.labelmatchstatement", map[string]*llx.RawData{
				"statementID": llx.StringData(mqlStatementID),
				"ruleName":    llx.StringDataPtr(ruleName),
				"key":         llx.StringDataPtr(statement.LabelMatchStatement.Key),
				"scope":       llx.StringData(string(statement.LabelMatchStatement.Scope)),
			})
			if err != nil {
				return nil, err
			}
		}
		if statement.ManagedRuleGroupStatement != nil {
			kind = "ManagedRuleGroupStatement"
			managedrulegroupstatement, err = CreateResource(runtime, "aws.waf.rule.statement.managedrulegroupstatement", map[string]*llx.RawData{
				"statementID": llx.StringData(mqlStatementID),
				"ruleName":    llx.StringDataPtr(ruleName),
				"name":        llx.StringDataPtr(statement.ManagedRuleGroupStatement.Name),
				"vendorName":  llx.StringDataPtr(statement.ManagedRuleGroupStatement.VendorName),
			})
			if err != nil {
				return nil, err
			}
		}
		if statement.AndStatement != nil {
			kind = "AndStatement"
			var statements []interface{}
			for _, statement := range statement.AndStatement.Statements {
				andStatementMqlStatement, err := createStatementResource(runtime, &statement, ruleName, ruleID)
				if err != nil {
					return nil, err
				}
				statements = append(statements, andStatementMqlStatement)
			}
			andStatement, err = CreateResource(runtime, "aws.waf.rule.statement.andstatement", map[string]*llx.RawData{
				"statementID": llx.StringData(mqlStatementID),
				"statements":  llx.ArrayData(statements, types.ResourceLike),
				"ruleName":    llx.StringDataPtr(ruleName),
			})
			if err != nil {
				return nil, err
			}
		}
		if statement.NotStatement != nil {
			kind = "NotStatement"
			var notStatementMqlStatement plugin.Resource
			notStatementMqlStatement, err = createStatementResource(runtime, statement.NotStatement.Statement, ruleName, ruleID)
			if err != nil {
				return nil, err
			}
			notstatement, err = CreateResource(runtime, "aws.waf.rule.statement.notstatement", map[string]*llx.RawData{
				"statementID": llx.StringData(mqlStatementID),
				"statement":   llx.ResourceData(notStatementMqlStatement, "aws.waf.rule.statement.notstatement"),
				"ruleName":    llx.StringDataPtr(ruleName),
			})
		}
		if statement.OrStatement != nil {
			kind = "OrStatement"
			var statements []interface{}
			for _, statement := range statement.OrStatement.Statements {
				orStatementMqlStatement, err := createStatementResource(runtime, &statement, ruleName, ruleID)
				if err != nil {
					return nil, err
				}
				statements = append(statements, orStatementMqlStatement)
			}
			orstatement, err = CreateResource(runtime, "aws.waf.rule.statement.orstatement", map[string]*llx.RawData{
				"statementID": llx.StringData(mqlStatementID),
				"statements":  llx.ArrayData(statements, types.ResourceLike),
				"ruleName":    llx.StringDataPtr(ruleName),
			})
		}
		if statement.RateBasedStatement != nil {
			kind = "RateBasedStatement"
			ratebasedstatement, err = CreateResource(runtime, "aws.waf.rule.statement.ratebasedstatement", map[string]*llx.RawData{})
			if err != nil {
				return nil, err
			}
		}
		if statement.RegexPatternSetReferenceStatement != nil {
			kind = "RegexPatternSetReferenceStatement"
			var fieldToMatch plugin.Resource
			fieldToMatch, err = createFieldToMatchResource(runtime, statement.RegexPatternSetReferenceStatement.FieldToMatch, ruleName, mqlStatementID)
			regexpatternsetreferencestatement, err = CreateResource(runtime, "aws.waf.rule.statement.regexpatternsetreferencestatement", map[string]*llx.RawData{
				"statementID":  llx.StringData(mqlStatementID),
				"ruleName":     llx.StringDataPtr(ruleName),
				"arn":          llx.StringDataPtr(statement.RegexPatternSetReferenceStatement.ARN),
				"fieldToMatch": llx.ResourceData(fieldToMatch, "aws.waf.rule.fieldtomatch"),
			})
			if err != nil {
				return nil, err
			}
		}
		if statement.RuleGroupReferenceStatement != nil {
			kind = "RuleGroupReferenceStatement"
			excludeRules := convert.SliceAnyToInterface(statement.RuleGroupReferenceStatement.ExcludedRules)
			rulegroupreferencestatement, err = CreateResource(runtime, "aws.waf.rule.statement.rulegroupreferencestatement", map[string]*llx.RawData{
				"statementID":  llx.StringData(mqlStatementID),
				"ruleName":     llx.StringDataPtr(ruleName),
				"arn":          llx.StringDataPtr(statement.RuleGroupReferenceStatement.ARN),
				"excludeRules": llx.ArrayData(excludeRules, types.String),
			})
			if err != nil {
				return nil, err
			}
		}
		if statement.SizeConstraintStatement != nil {
			kind = "SizeConstraintStatement"
			var fieldToMatch plugin.Resource
			fieldToMatch, err = createFieldToMatchResource(runtime, statement.SizeConstraintStatement.FieldToMatch, ruleName, mqlStatementID)
			sizeconstraintstatement, err = CreateResource(runtime, "aws.waf.rule.statement.sizeconstraintstatement", map[string]*llx.RawData{
				"statementID":        llx.StringData(mqlStatementID),
				"ruleName":           llx.StringDataPtr(ruleName),
				"size":               llx.IntData(statement.SizeConstraintStatement.Size),
				"comparisonOperator": llx.StringData(string(statement.SizeConstraintStatement.ComparisonOperator)),
				"fieldToMatch":       llx.ResourceData(fieldToMatch, "aws.waf.rule.fieldtomatch"),
			})
			if err != nil {
				return nil, err
			}
		}
	}
	var mqlStatement plugin.Resource
	mqlStatement, err = CreateResource(runtime, "aws.waf.rule.statement",
		map[string]*llx.RawData{
			"id":                                llx.StringData(mqlStatementID),
			"kind":                              llx.StringData(kind),
			"json":                              llx.DictData(statementJson),
			"regexMatchStatement":               llx.ResourceData(regexmatchstatement, "aws.waf.rule.statement.regexmatchstatement"),
			"byteMatchStatement":                llx.ResourceData(bytematchstatement, "aws.waf.rule.statement.bytematchstatement"),
			"xssMatchStatement":                 llx.ResourceData(xssmatchstatement, "aws.waf.rule.statement.xssmatchstatement"),
			"sqliMatchStatement":                llx.ResourceData(sqlimatchstatement, "aws.waf.rule.statement.sqlimatchstatement"),
			"geoMatchStatement":                 llx.ResourceData(geomatchstatement, "aws.waf.rule.statement.geomatchstatement"),
			"ipSetReferenceStatement":           llx.ResourceData(ipsetreferencestatement, "aws.waf.rule.statement.ipsetreferencestatement"),
			"labelMatchStatement":               llx.ResourceData(labelmatchstatement, "aws.waf.rule.statement.labelmatchstatement"),
			"managedRuleGroupStatement":         llx.ResourceData(managedrulegroupstatement, "aws.waf.rule.statement.managedrulegroupstatement"),
			"notStatement":                      llx.ResourceData(notstatement, "aws.waf.rule.statement.notstatement"),
			"orStatement":                       llx.ResourceData(orstatement, "aws.waf.rule.statement.orstatement"),
			"andStatement":                      llx.ResourceData(andStatement, "aws.waf.rule.statement.andStatement"),
			"rateBasedStatement":                llx.ResourceData(ratebasedstatement, "aws.waf.rule.statement.ratebasedstatement"),
			"regexPatternSetReferenceStatement": llx.ResourceData(regexpatternsetreferencestatement, "aws.waf.rule.statement.regexpatternsetreferencestatement"),
			"ruleGroupReferenceStatement":       llx.ResourceData(rulegroupreferencestatement, "aws.waf.rule.statement.rulegroupreferencestatement"),
			"sizeConstraintStatement":           llx.ResourceData(sizeconstraintstatement, "aws.waf.rule.statement.sizeconstraintstatement"),
		},
	)

	return mqlStatement, nil
}

func createFieldToMatchResource(runtime *plugin.Runtime, fieldToMatch *waftypes.FieldToMatch, ruleName *string, mqlStatementID string) (plugin.Resource, error) {
	var err error
	var singleHeader plugin.Resource
	var singleQueryArgument plugin.Resource
	var body plugin.Resource
	var cookie plugin.Resource
	var headerOrder plugin.Resource
	var headers plugin.Resource
	var ja3Fingerprint plugin.Resource
	var jsonBody plugin.Resource
	var target string
	if fieldToMatch.SingleHeader != nil {
		target = "SingleHeader"
		singleHeader, err = CreateResource(runtime, "aws.waf.rule.fieldtomatch.singleheader", map[string]*llx.RawData{
			"statementID": llx.StringData(mqlStatementID),
			"ruleName":    llx.StringDataPtr(ruleName),
			"name":        llx.StringDataPtr(fieldToMatch.SingleHeader.Name),
		})
		if err != nil {
			return nil, err
		}
	}
	if fieldToMatch.SingleQueryArgument != nil {
		target = "SingleQueryArgument"
		singleQueryArgument, err = CreateResource(runtime, "aws.waf.rule.fieldtomatch.singlequeryargument", map[string]*llx.RawData{
			"statementID": llx.StringData(mqlStatementID),
			"ruleName":    llx.StringDataPtr(ruleName),
			"name":        llx.StringDataPtr(fieldToMatch.SingleQueryArgument.Name),
		})
		if err != nil {
			return nil, err
		}
	}
	if fieldToMatch.Body != nil {
		target = "Body"
		body, err = CreateResource(runtime, "aws.waf.rule.fieldtomatch.body", map[string]*llx.RawData{
			"statementID":      llx.StringData(mqlStatementID),
			"ruleName":         llx.StringDataPtr(ruleName),
			"overSizeHandling": llx.StringData(string(fieldToMatch.Body.OversizeHandling)),
		})
	}
	if fieldToMatch.Cookies != nil {
		target = "Cookies"
		cookie, err = CreateResource(runtime, "aws.waf.rule.fieldtomatch.cookie", map[string]*llx.RawData{
			"statementID":      llx.StringData(mqlStatementID),
			"ruleName":         llx.StringDataPtr(ruleName),
			"overSizeHandling": llx.StringData(string(fieldToMatch.Cookies.OversizeHandling)),
		})
	}
	if fieldToMatch.HeaderOrder != nil {
		target = "HeaderOrder"
		headerOrder, err = CreateResource(runtime, "aws.waf.rule.fieldtomatch.headerOrder", map[string]*llx.RawData{
			"statementID":      llx.StringData(mqlStatementID),
			"ruleName":         llx.StringDataPtr(ruleName),
			"overSizeHandling": llx.StringData(string(fieldToMatch.Headers.OversizeHandling)),
		})
	}
	if fieldToMatch.SingleQueryArgument != nil {
		target = "SingleQueryArgument"
		singleQueryArgument, err = CreateResource(runtime, "aws.waf.rule.fieldtomatch.singlequeryargument", map[string]*llx.RawData{
			"statementID": llx.StringData(mqlStatementID),
			"ruleName":    llx.StringDataPtr(ruleName),
			"name":        llx.StringDataPtr(fieldToMatch.SingleQueryArgument.Name),
		})
	}

	if fieldToMatch.JA3Fingerprint != nil {
		target = "JA3Fingerprint"
		ja3Fingerprint, err = CreateResource(runtime, "aws.waf.rule.fieldtomatch.ja3fingerprint", map[string]*llx.RawData{
			"statementID":      llx.StringData(mqlStatementID),
			"ruleName":         llx.StringDataPtr(ruleName),
			"fallbackBehavior": llx.StringData(string(fieldToMatch.JA3Fingerprint.FallbackBehavior)),
		})
	}

	if fieldToMatch.Headers != nil {
		target = "Headers"
		var matchPattern plugin.Resource
		if fieldToMatch.Headers.MatchPattern != nil {
			includeHeaders := convert.SliceAnyToInterface(fieldToMatch.Headers.MatchPattern.IncludedHeaders)
			excludeHeaders := convert.SliceAnyToInterface(fieldToMatch.Headers.MatchPattern.ExcludedHeaders)
			matchPattern, err = CreateResource(runtime, "aws.waf.rule.fieldtomatch.jsonbody.matchpattern", map[string]*llx.RawData{
				"statementID":    llx.StringData(mqlStatementID),
				"ruleName":       llx.StringDataPtr(ruleName),
				"all":            llx.BoolData(fieldToMatch.Headers.MatchPattern.All != nil),
				"includeHeaders": llx.ArrayData(includeHeaders, types.String),
				"excludeHeaders": llx.ArrayData(excludeHeaders, types.String),
			})
		}
		headers, err = CreateResource(runtime, "aws.waf.rule.fieldtomatch.headers", map[string]*llx.RawData{
			"statementID":      llx.StringData(mqlStatementID),
			"ruleName":         llx.StringDataPtr(ruleName),
			"matchPattern":     llx.ResourceData(matchPattern, "aws.waf.rule.fieldtomatch.headers.matchpatern"),
			"overSizeHandling": llx.StringData(string(fieldToMatch.Headers.OversizeHandling)),
			"matchScope":       llx.StringData(string(fieldToMatch.Headers.MatchScope)),
		})

	}
	if fieldToMatch.JsonBody != nil {
		target = "JsonBody"
		var matchPattern plugin.Resource
		includePathsArray := convert.SliceAnyToInterface(fieldToMatch.JsonBody.MatchPattern.IncludedPaths)
		if fieldToMatch.JsonBody.MatchPattern != nil {
			matchPattern, err = CreateResource(runtime, "aws.waf.rule.fieldtomatch.jsonbody.matchpattern", map[string]*llx.RawData{
				"statementID":  llx.StringData(mqlStatementID),
				"ruleName":     llx.StringDataPtr(ruleName),
				"all":          llx.BoolData(fieldToMatch.JsonBody.MatchPattern.All != nil),
				"includePaths": llx.ArrayData(includePathsArray, types.String),
			})
			if err != nil {
				return nil, err
			}
		}
		jsonBody, err = CreateResource(runtime, "aws.waf.rule.fieldtomatch.jsonbody", map[string]*llx.RawData{
			"statementID":             llx.StringData(mqlStatementID),
			"ruleName":                llx.StringDataPtr(ruleName),
			"overSizeHandling":        llx.StringData(string(fieldToMatch.JsonBody.OversizeHandling)),
			"invalidFallbackBehavior": llx.StringData(string(fieldToMatch.JsonBody.InvalidFallbackBehavior)),
			"matchScope":              llx.StringData(string(fieldToMatch.JsonBody.MatchScope)),
			"matchPattern":            llx.ResourceData(matchPattern, "aws.waf.rule.fieldtomatch.jsonbody.matchpattern"),
		})
		if err != nil {
			return nil, err
		}
	}
	if fieldToMatch.QueryString != nil {
		target = "QueryString"
	}
	if fieldToMatch.Method != nil {
		target = "Method"
	}
	if fieldToMatch.UriPath != nil {
		target = "UriPath"
	}
	if fieldToMatch.AllQueryArguments != nil {
		target = "allQueryArguments"
	}
	mqlFieldToMatch, err := CreateResource(runtime, "aws.waf.rule.fieldtomatch", map[string]*llx.RawData{
		"statementID":         llx.StringData(mqlStatementID),
		"target":              llx.StringData(target),
		"ruleName":            llx.StringDataPtr(ruleName),
		"queryString":         llx.BoolData(fieldToMatch.QueryString != nil),
		"method":              llx.BoolData(fieldToMatch.Method != nil),
		"uriPath":             llx.BoolData(fieldToMatch.UriPath != nil),
		"allQueryArguments":   llx.BoolData(fieldToMatch.AllQueryArguments != nil),
		"singleHeader":        llx.ResourceData(singleHeader, "aws.waf.rule.fieldtomatch.singleheader"),
		"singleQueryArgument": llx.ResourceData(singleQueryArgument, "aws.waf.rule.fieldtomatch.singlequeryargument"),
		"body":                llx.ResourceData(body, "aws.waf.rule.fieldtomatch.body"),
		"cookie":              llx.ResourceData(cookie, "aws.waf.rule.fieldtomatch.cookie"),
		"headerOrder":         llx.ResourceData(headerOrder, "aws.waf.rule.fieldToMatch.headerorder"),
		"headers":             llx.ResourceData(headers, "aws.waf.rule.fieldToMatch.headers"),
		"ja3Fingerprint":      llx.ResourceData(ja3Fingerprint, "aws.waf.rule.fieldToMatch.ja3fingerprint"),
		"jsonBody":            llx.ResourceData(jsonBody, "aws.waf.rule.fieldToMatch.jsonbody"),
	})
	if err != nil {
		return nil, err
	}

	return mqlFieldToMatch, nil
}
