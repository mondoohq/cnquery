// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/wafv2"
	waftypes "github.com/aws/aws-sdk-go-v2/service/wafv2/types"
	"github.com/google/uuid"

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

func (a *mqlAwsWafRuleStatement) id() (string, error) {
	return a.Id.Data, nil
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

func (a *mqlAwsWafRuleStatementSqlimatchstatement) id() (string, error) {
	return a.RuleName.Data, nil
}

func (a *mqlAwsWafRuleStatementBytematchstatement) id() (string, error) {
	return a.RuleName.Data, nil
}

func (a *mqlAwsWafRuleFieldtomatch) id() (string, error) {
	return a.RuleName.Data, nil
}

func (a *mqlAwsWafRuleFieldtomatchBody) id() (string, error) {
	return a.RuleName.Data, nil
}

func (a *mqlAwsWafRuleFieldtomatchJsonbody) id() (string, error) {
	return a.RuleName.Data, nil
}

func (a *mqlAwsWafRuleFieldtomatchCookie) id() (string, error) {
	return a.RuleName.Data, nil
}

func (a *mqlAwsWafRuleFieldtomatchHeaders) id() (string, error) {
	return a.RuleName.Data, nil
}

func (a *mqlAwsWafRuleFieldtomatchHeaderorder) id() (string, error) {
	return a.RuleName.Data, nil
}

func (a *mqlAwsWafRuleFieldtomatchSingleheader) id() (string, error) {
	return a.RuleName.Data, nil
}

func (a *mqlAwsWafRuleFieldtomatchSinglequeryargument) id() (string, error) {
	return a.RuleName.Data, nil
}

func (a *mqlAwsWafRuleFieldtomatchHeadersMatchpattern) id() (string, error) {
	return a.RuleName.Data, nil
}

func (a *mqlAwsWafRuleFieldtomatchJsonbodyMatchpattern) id() (string, error) {
	return a.RuleName.Data, nil
}

func (a *mqlAwsWafRuleFieldtomatchJa3fingerprint) id() (string, error) {
	return a.RuleName.Data, nil
}

func (a *mqlAwsWafRuleStatementXssmatchstatement) id() (string, error) {
	return "aws.waf.rule.xssmatchstatement", nil
}

func (a *mqlAwsWafRuleStatementRegexmatchstatement) id() (string, error) {
	return "aws.waf.rule.regexstatement", nil
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
		mqlStatement, err := createStatementResource(a.MqlRuntime, rule.Statement, rule.Name)
		fmt.Println("mqlStatement:", mqlStatement)
		ruleAction, err := convert.JsonToDict(rule.Action)
		mqlRule, err := CreateResource(a.MqlRuntime, "aws.waf.rule",
			map[string]*llx.RawData{
				"name":      llx.StringDataPtr(rule.Name),
				"priority":  llx.IntData(int64(rule.Priority)),
				"action":    llx.DictData(ruleAction),
				"statement": llx.ResourceData(mqlStatement, "aws.waf.rule.statement"),
			},
		)
		if err != nil {
			return nil, err
		}
		rules = append(rules, mqlRule)
	}
	return rules, nil
}

func createStatementResource(runtime *plugin.Runtime, statement *waftypes.Statement, ruleName *string) (plugin.Resource, error) {
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
	if statement != nil {
		if statement.RegexMatchStatement != nil {
			var fieldToMatch plugin.Resource
			if statement.RegexMatchStatement.FieldToMatch != nil {
				var singleHeader plugin.Resource
				var singleQueryArgument plugin.Resource
				var body plugin.Resource
				var cookie plugin.Resource
				var headerOrder plugin.Resource
				var headers plugin.Resource
				var ja3Fingerprint plugin.Resource
				var jsonBody plugin.Resource
				if statement.RegexMatchStatement.FieldToMatch.SingleHeader != nil {
					singleHeader, err = CreateResource(runtime, "aws.waf.rule.fieldtomatch.singleheader", map[string]*llx.RawData{
						"ruleName": llx.StringDataPtr(ruleName),
						"name":     llx.StringDataPtr(statement.RegexMatchStatement.FieldToMatch.SingleHeader.Name),
					})
					if err != nil {
						return nil, err
					}
				}
				if statement.RegexMatchStatement.FieldToMatch.SingleQueryArgument != nil {
					singleQueryArgument, err = CreateResource(runtime, "aws.waf.rule.fieldtomatch.singlequeryargument", map[string]*llx.RawData{
						"ruleName": llx.StringDataPtr(ruleName),
						"name":     llx.StringDataPtr(statement.RegexMatchStatement.FieldToMatch.SingleQueryArgument.Name),
					})
					if err != nil {
						return nil, err
					}
				}
				if statement.RegexMatchStatement.FieldToMatch.Body != nil {
					body, err = CreateResource(runtime, "aws.waf.rule.fieldtomatch.body", map[string]*llx.RawData{
						"ruleName":         llx.StringDataPtr(ruleName),
						"overSizeHandling": llx.StringData(string(statement.RegexMatchStatement.FieldToMatch.Body.OversizeHandling)),
					})
				}
				if statement.RegexMatchStatement.FieldToMatch.Cookies != nil {
					cookie, err = CreateResource(runtime, "aws.waf.rule.fieldtomatch.cookie", map[string]*llx.RawData{
						"ruleName":         llx.StringDataPtr(ruleName),
						"overSizeHandling": llx.StringData(string(statement.RegexMatchStatement.FieldToMatch.Cookies.OversizeHandling)),
					})
				}
				if statement.RegexMatchStatement.FieldToMatch.HeaderOrder != nil {
					headerOrder, err = CreateResource(runtime, "aws.waf.rule.fieldtomatch.headerOrder", map[string]*llx.RawData{
						"ruleName":         llx.StringDataPtr(ruleName),
						"overSizeHandling": llx.StringData(string(statement.RegexMatchStatement.FieldToMatch.Headers.OversizeHandling)),
					})
				}
				if statement.RegexMatchStatement.FieldToMatch.SingleHeader != nil {
					singleHeader, err = CreateResource(runtime, "aws.waf.rule.fieldtomatch.singleheader", map[string]*llx.RawData{
						"ruleName": llx.StringDataPtr(ruleName),
						"name":     llx.StringDataPtr(statement.RegexMatchStatement.FieldToMatch.SingleHeader.Name),
					})
				}
				if statement.RegexMatchStatement.FieldToMatch.HeaderOrder != nil {
					singleQueryArgument, err = CreateResource(runtime, "aws.waf.rule.fieldtomatch.singlequeryargument", map[string]*llx.RawData{
						"ruleName": llx.StringDataPtr(ruleName),
						"name":     llx.StringDataPtr(statement.RegexMatchStatement.FieldToMatch.SingleQueryArgument.Name),
					})
				}

				if statement.RegexMatchStatement.FieldToMatch.JA3Fingerprint != nil {
					ja3Fingerprint, err = CreateResource(runtime, "aws.waf.rule.fieldtomatch.ja3fingerprint", map[string]*llx.RawData{
						"ruleName":         llx.StringDataPtr(ruleName),
						"fallbackBehavior": llx.StringData(string(statement.RegexMatchStatement.FieldToMatch.JA3Fingerprint.FallbackBehavior)),
					})
				}

				if statement.RegexMatchStatement.FieldToMatch.Headers != nil {
					var matchPattern plugin.Resource
					if statement.RegexMatchStatement.FieldToMatch.JsonBody.MatchPattern != nil {
						includeHeaders := convert.SliceAnyToInterface(statement.RegexMatchStatement.FieldToMatch.Headers.MatchPattern.IncludedHeaders)
						excludeHeaders := convert.SliceAnyToInterface(statement.RegexMatchStatement.FieldToMatch.Headers.MatchPattern.ExcludedHeaders)
						matchPattern, err = CreateResource(runtime, "aws.waf.rule.fieldtomatch.jsonbody.matchpattern", map[string]*llx.RawData{
							"ruleName":       llx.StringDataPtr(ruleName),
							"all":            llx.BoolData(statement.RegexMatchStatement.FieldToMatch.Headers.MatchPattern.All != nil),
							"includeHeaders": llx.ArrayData(includeHeaders, types.String),
							"excludeHeaders": llx.ArrayData(excludeHeaders, types.String),
						})
					}
					headers, err = CreateResource(runtime, "aws.waf.rule.fieldtomatch.headers", map[string]*llx.RawData{
						"ruleName":         llx.StringDataPtr(ruleName),
						"matchPattern":     llx.ResourceData(matchPattern, "aws.waf.rule.fieldtomatch.headers.matchpatern"),
						"overSizeHandling": llx.StringData(string(statement.RegexMatchStatement.FieldToMatch.Headers.OversizeHandling)),
						"matchScope":       llx.StringData(string(statement.RegexMatchStatement.FieldToMatch.Headers.MatchScope)),
					})

				}
				if statement.RegexMatchStatement.FieldToMatch.JsonBody != nil {
					var matchPattern plugin.Resource
					includePathsArray := convert.SliceAnyToInterface(statement.RegexMatchStatement.FieldToMatch.JsonBody.MatchPattern.IncludedPaths)
					if statement.RegexMatchStatement.FieldToMatch.JsonBody.MatchPattern != nil {
						matchPattern, err = CreateResource(runtime, "aws.waf.rule.fieldtomatch.jsonbody.matchpattern", map[string]*llx.RawData{
							"ruleName":     llx.StringDataPtr(ruleName),
							"all":          llx.BoolData(statement.RegexMatchStatement.FieldToMatch.JsonBody.MatchPattern.All != nil),
							"includePaths": llx.ArrayData(includePathsArray, types.String),
						})
						if err != nil {
							return nil, err
						}
					}
					jsonBody, err = CreateResource(runtime, "aws.waf.rule.fieldtomatch.jsonbody", map[string]*llx.RawData{
						"ruleName":                llx.StringDataPtr(ruleName),
						"overSizeHandling":        llx.StringData(string(statement.RegexMatchStatement.FieldToMatch.JsonBody.OversizeHandling)),
						"invalidFallbackBehavior": llx.StringData(string(statement.RegexMatchStatement.FieldToMatch.JsonBody.InvalidFallbackBehavior)),
						"matchScope":              llx.StringData(string(statement.RegexMatchStatement.FieldToMatch.JsonBody.MatchScope)),
						"matchPattern":            llx.ResourceData(matchPattern, "aws.waf.rule.fieldtomatch.jsonbody.matchpattern"),
					})
					if err != nil {
						return nil, err
					}
				}
				fieldToMatch, err = CreateResource(runtime, "aws.waf.rule.fieldtomatch", map[string]*llx.RawData{
					"ruleName":            llx.StringDataPtr(ruleName),
					"queryString":         llx.BoolData(statement.RegexMatchStatement.FieldToMatch.QueryString != nil),
					"method":              llx.BoolData(statement.RegexMatchStatement.FieldToMatch.Method != nil),
					"uriPath":             llx.BoolData(statement.RegexMatchStatement.FieldToMatch.UriPath != nil),
					"allQueryArguments":   llx.BoolData(statement.RegexMatchStatement.FieldToMatch.AllQueryArguments != nil),
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
			}
			regexmatchstatement, err = CreateResource(runtime, "aws.waf.rule.statement.regexmatchstatement", map[string]*llx.RawData{
				"ruleName":     llx.StringDataPtr(ruleName),
				"fieldToMatch": llx.ResourceData(fieldToMatch, "aws.waf.rule.statement.regexmatchstatement.fieldtomatch"),
				"regexString":  llx.StringDataPtr(statement.RegexMatchStatement.RegexString),
			})
			if err != nil {
				return nil, err
			}
		}
		if statement.ByteMatchStatement != nil {
			var fieldToMatch plugin.Resource
			if statement.ByteMatchStatement.FieldToMatch != nil {
				var singleHeader plugin.Resource
				var singleQueryArgument plugin.Resource
				var body plugin.Resource
				var cookie plugin.Resource
				var headerOrder plugin.Resource
				var headers plugin.Resource
				var ja3Fingerprint plugin.Resource
				var jsonBody plugin.Resource
				if statement.ByteMatchStatement.FieldToMatch.SingleHeader != nil {
					singleHeader, err = CreateResource(runtime, "aws.waf.rule.fieldtomatch.singleheader", map[string]*llx.RawData{
						"ruleName": llx.StringDataPtr(ruleName),
						"name":     llx.StringDataPtr(statement.ByteMatchStatement.FieldToMatch.SingleHeader.Name),
					})
					if err != nil {
						return nil, err
					}
				}
				if statement.ByteMatchStatement.FieldToMatch.SingleQueryArgument != nil {
					singleQueryArgument, err = CreateResource(runtime, "aws.waf.rule.fieldtomatch.singlequeryargument", map[string]*llx.RawData{
						"ruleName": llx.StringDataPtr(ruleName),
						"name":     llx.StringDataPtr(statement.ByteMatchStatement.FieldToMatch.SingleQueryArgument.Name),
					})
					if err != nil {
						return nil, err
					}
				}
				if statement.ByteMatchStatement.FieldToMatch.Body != nil {
					body, err = CreateResource(runtime, "aws.waf.rule.fieldtomatch.body", map[string]*llx.RawData{
						"ruleName":         llx.StringDataPtr(ruleName),
						"overSizeHandling": llx.StringData(string(statement.ByteMatchStatement.FieldToMatch.Body.OversizeHandling)),
					})
				}
				if statement.ByteMatchStatement.FieldToMatch.Cookies != nil {
					cookie, err = CreateResource(runtime, "aws.waf.rule.fieldtomatch.cookie", map[string]*llx.RawData{
						"ruleName":         llx.StringDataPtr(ruleName),
						"overSizeHandling": llx.StringData(string(statement.ByteMatchStatement.FieldToMatch.Cookies.OversizeHandling)),
					})
				}
				if statement.ByteMatchStatement.FieldToMatch.HeaderOrder != nil {
					headerOrder, err = CreateResource(runtime, "aws.waf.rule.fieldtomatch.headerOrder", map[string]*llx.RawData{
						"ruleName":         llx.StringDataPtr(ruleName),
						"overSizeHandling": llx.StringData(string(statement.ByteMatchStatement.FieldToMatch.Headers.OversizeHandling)),
					})
				}
				if statement.ByteMatchStatement.FieldToMatch.SingleHeader != nil {
					singleHeader, err = CreateResource(runtime, "aws.waf.rule.fieldtomatch.singleheader", map[string]*llx.RawData{
						"ruleName": llx.StringDataPtr(ruleName),
						"name":     llx.StringDataPtr(statement.ByteMatchStatement.FieldToMatch.SingleHeader.Name),
					})
				}
				if statement.ByteMatchStatement.FieldToMatch.HeaderOrder != nil {
					singleQueryArgument, err = CreateResource(runtime, "aws.waf.rule.fieldtomatch.singlequeryargument", map[string]*llx.RawData{
						"ruleName": llx.StringDataPtr(ruleName),
						"name":     llx.StringDataPtr(statement.ByteMatchStatement.FieldToMatch.SingleQueryArgument.Name),
					})
				}

				if statement.ByteMatchStatement.FieldToMatch.JA3Fingerprint != nil {
					ja3Fingerprint, err = CreateResource(runtime, "aws.waf.rule.fieldtomatch.ja3fingerprint", map[string]*llx.RawData{
						"ruleName":         llx.StringDataPtr(ruleName),
						"fallbackBehavior": llx.StringData(string(statement.ByteMatchStatement.FieldToMatch.JA3Fingerprint.FallbackBehavior)),
					})
				}

				if statement.ByteMatchStatement.FieldToMatch.Headers != nil {
					var matchPattern plugin.Resource
					if statement.ByteMatchStatement.FieldToMatch.JsonBody.MatchPattern != nil {
						includeHeaders := convert.SliceAnyToInterface(statement.ByteMatchStatement.FieldToMatch.Headers.MatchPattern.IncludedHeaders)
						excludeHeaders := convert.SliceAnyToInterface(statement.ByteMatchStatement.FieldToMatch.Headers.MatchPattern.ExcludedHeaders)
						matchPattern, err = CreateResource(runtime, "aws.waf.rule.fieldtomatch.jsonbody.matchpattern", map[string]*llx.RawData{
							"ruleName":       llx.StringDataPtr(ruleName),
							"all":            llx.BoolData(statement.ByteMatchStatement.FieldToMatch.Headers.MatchPattern.All != nil),
							"includeHeaders": llx.ArrayData(includeHeaders, types.String),
							"excludeHeaders": llx.ArrayData(excludeHeaders, types.String),
						})
					}
					headers, err = CreateResource(runtime, "aws.waf.rule.fieldtomatch.headers", map[string]*llx.RawData{
						"ruleName":         llx.StringDataPtr(ruleName),
						"matchPattern":     llx.ResourceData(matchPattern, "aws.waf.rule.fieldtomatch.headers.matchpatern"),
						"overSizeHandling": llx.StringData(string(statement.ByteMatchStatement.FieldToMatch.Headers.OversizeHandling)),
						"matchScope":       llx.StringData(string(statement.ByteMatchStatement.FieldToMatch.Headers.MatchScope)),
					})

				}
				if statement.ByteMatchStatement.FieldToMatch.JsonBody != nil {
					var matchPattern plugin.Resource
					includePathsArray := convert.SliceAnyToInterface(statement.ByteMatchStatement.FieldToMatch.JsonBody.MatchPattern.IncludedPaths)
					if statement.ByteMatchStatement.FieldToMatch.JsonBody.MatchPattern != nil {
						matchPattern, err = CreateResource(runtime, "aws.waf.rule.fieldtomatch.jsonbody.matchpattern", map[string]*llx.RawData{
							"ruleName":     llx.StringDataPtr(ruleName),
							"all":          llx.BoolData(statement.ByteMatchStatement.FieldToMatch.JsonBody.MatchPattern.All != nil),
							"includePaths": llx.ArrayData(includePathsArray, types.String),
						})
						if err != nil {
							return nil, err
						}
					}
					jsonBody, err = CreateResource(runtime, "aws.waf.rule.fieldtomatch.jsonbody", map[string]*llx.RawData{
						"ruleName":                llx.StringDataPtr(ruleName),
						"overSizeHandling":        llx.StringData(string(statement.ByteMatchStatement.FieldToMatch.JsonBody.OversizeHandling)),
						"invalidFallbackBehavior": llx.StringData(string(statement.ByteMatchStatement.FieldToMatch.JsonBody.InvalidFallbackBehavior)),
						"matchScope":              llx.StringData(string(statement.ByteMatchStatement.FieldToMatch.JsonBody.MatchScope)),
						"matchPattern":            llx.ResourceData(matchPattern, "aws.waf.fieldtomatch.jsonbody.matchpattern"),
					})
					if err != nil {
						return nil, err
					}
				}
				fieldToMatch, err = CreateResource(runtime, "aws.waf.rule.fieldtomatch", map[string]*llx.RawData{
					"ruleName":            llx.StringDataPtr(ruleName),
					"queryString":         llx.BoolData(statement.ByteMatchStatement.FieldToMatch.QueryString != nil),
					"method":              llx.BoolData(statement.ByteMatchStatement.FieldToMatch.Method != nil),
					"uriPath":             llx.BoolData(statement.ByteMatchStatement.FieldToMatch.UriPath != nil),
					"allQueryArguments":   llx.BoolData(statement.ByteMatchStatement.FieldToMatch.AllQueryArguments != nil),
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
			}
			bytematchstatement, err = CreateResource(runtime, "aws.waf.rule.statement.bytematchstatement", map[string]*llx.RawData{
				"ruleName":     llx.StringDataPtr(ruleName),
				"fieldToMatch": llx.ResourceData(fieldToMatch, "aws.waf.rule.fieldtomatch"),
				"searchString": llx.StringData(string(statement.ByteMatchStatement.SearchString)),
			})
			if err != nil {
				return nil, err
			}
		}
		if statement.XssMatchStatement != nil {
			var fieldToMatch plugin.Resource
			if statement.XssMatchStatement.FieldToMatch != nil {
				var singleHeader plugin.Resource
				var singleQueryArgument plugin.Resource
				var body plugin.Resource
				var cookie plugin.Resource
				var headerOrder plugin.Resource
				var headers plugin.Resource
				var ja3Fingerprint plugin.Resource
				var jsonBody plugin.Resource
				if statement.XssMatchStatement.FieldToMatch.SingleHeader != nil {
					singleHeader, err = CreateResource(runtime, "aws.waf.rule.fieldtomatch.singleheader", map[string]*llx.RawData{
						"ruleName": llx.StringDataPtr(ruleName),
						"name":     llx.StringDataPtr(statement.XssMatchStatement.FieldToMatch.SingleHeader.Name),
					})
					if err != nil {
						return nil, err
					}
				}
				if statement.XssMatchStatement.FieldToMatch.SingleQueryArgument != nil {
					singleQueryArgument, err = CreateResource(runtime, "aws.waf.rule.fieldtomatch.singlequeryargument", map[string]*llx.RawData{
						"ruleName": llx.StringDataPtr(ruleName),
						"name":     llx.StringDataPtr(statement.XssMatchStatement.FieldToMatch.SingleQueryArgument.Name),
					})
					if err != nil {
						return nil, err
					}
				}
				if statement.XssMatchStatement.FieldToMatch.Body != nil {
					body, err = CreateResource(runtime, "aws.waf.rule.fieldtomatch.body", map[string]*llx.RawData{
						"ruleName":         llx.StringDataPtr(ruleName),
						"overSizeHandling": llx.StringData(string(statement.XssMatchStatement.FieldToMatch.Body.OversizeHandling)),
					})
				}
				if statement.XssMatchStatement.FieldToMatch.Cookies != nil {
					cookie, err = CreateResource(runtime, "aws.waf.rule.fieldtomatch.cookie", map[string]*llx.RawData{
						"ruleName":         llx.StringDataPtr(ruleName),
						"overSizeHandling": llx.StringData(string(statement.XssMatchStatement.FieldToMatch.Cookies.OversizeHandling)),
					})
				}
				if statement.XssMatchStatement.FieldToMatch.HeaderOrder != nil {
					headerOrder, err = CreateResource(runtime, "aws.waf.rule.fieldtomatch.headerOrder", map[string]*llx.RawData{
						"ruleName":         llx.StringDataPtr(ruleName),
						"overSizeHandling": llx.StringData(string(statement.XssMatchStatement.FieldToMatch.Headers.OversizeHandling)),
					})
				}
				if statement.XssMatchStatement.FieldToMatch.SingleHeader != nil {
					singleHeader, err = CreateResource(runtime, "aws.waf.rule.fieldtomatch.singleheader", map[string]*llx.RawData{
						"ruleName": llx.StringDataPtr(ruleName),
						"name":     llx.StringDataPtr(statement.XssMatchStatement.FieldToMatch.SingleHeader.Name),
					})
				}
				if statement.XssMatchStatement.FieldToMatch.HeaderOrder != nil {
					singleQueryArgument, err = CreateResource(runtime, "aws.waf.rule.fieldtomatch.singlequeryargument", map[string]*llx.RawData{
						"ruleName": llx.StringDataPtr(ruleName),
						"name":     llx.StringDataPtr(statement.XssMatchStatement.FieldToMatch.SingleQueryArgument.Name),
					})
				}

				if statement.XssMatchStatement.FieldToMatch.JA3Fingerprint != nil {
					ja3Fingerprint, err = CreateResource(runtime, "aws.waf.rule.fieldtomatch.ja3fingerprint", map[string]*llx.RawData{
						"ruleName":         llx.StringDataPtr(ruleName),
						"fallbackBehavior": llx.StringData(string(statement.XssMatchStatement.FieldToMatch.JA3Fingerprint.FallbackBehavior)),
					})
				}

				if statement.XssMatchStatement.FieldToMatch.Headers != nil {
					var matchPattern plugin.Resource
					if statement.XssMatchStatement.FieldToMatch.JsonBody.MatchPattern != nil {
						includeHeaders := convert.SliceAnyToInterface(statement.XssMatchStatement.FieldToMatch.Headers.MatchPattern.IncludedHeaders)
						excludeHeaders := convert.SliceAnyToInterface(statement.XssMatchStatement.FieldToMatch.Headers.MatchPattern.ExcludedHeaders)
						matchPattern, err = CreateResource(runtime, "aws.waf.rule.fieldtomatch.jsonbody.matchpattern", map[string]*llx.RawData{
							"ruleName":       llx.StringDataPtr(ruleName),
							"all":            llx.BoolData(statement.XssMatchStatement.FieldToMatch.Headers.MatchPattern.All != nil),
							"includeHeaders": llx.ArrayData(includeHeaders, types.String),
							"excludeHeaders": llx.ArrayData(excludeHeaders, types.String),
						})
					}
					headers, err = CreateResource(runtime, "aws.waf.rule.fieldtomatch.headers", map[string]*llx.RawData{
						"ruleName":         llx.StringDataPtr(ruleName),
						"matchPattern":     llx.ResourceData(matchPattern, "aws.waf.rule.fieldtomatch.headers.matchpatern"),
						"overSizeHandling": llx.StringData(string(statement.XssMatchStatement.FieldToMatch.Headers.OversizeHandling)),
						"matchScope":       llx.StringData(string(statement.XssMatchStatement.FieldToMatch.Headers.MatchScope)),
					})

				}
				if statement.XssMatchStatement.FieldToMatch.JsonBody != nil {
					var matchPattern plugin.Resource
					includePathsArray := convert.SliceAnyToInterface(statement.XssMatchStatement.FieldToMatch.JsonBody.MatchPattern.IncludedPaths)
					if statement.XssMatchStatement.FieldToMatch.JsonBody.MatchPattern != nil {
						matchPattern, err = CreateResource(runtime, "aws.waf.rule.fieldtomatch.jsonbody.matchpattern", map[string]*llx.RawData{
							"ruleName":     llx.StringDataPtr(ruleName),
							"all":          llx.BoolData(statement.XssMatchStatement.FieldToMatch.JsonBody.MatchPattern.All != nil),
							"includePaths": llx.ArrayData(includePathsArray, types.String),
						})
						if err != nil {
							return nil, err
						}
					}
					jsonBody, err = CreateResource(runtime, "aws.waf.rule.fieldtomatch.jsonbody", map[string]*llx.RawData{
						"ruleName":                llx.StringDataPtr(ruleName),
						"overSizeHandling":        llx.StringData(string(statement.XssMatchStatement.FieldToMatch.JsonBody.OversizeHandling)),
						"invalidFallbackBehavior": llx.StringData(string(statement.XssMatchStatement.FieldToMatch.JsonBody.InvalidFallbackBehavior)),
						"matchScope":              llx.StringData(string(statement.XssMatchStatement.FieldToMatch.JsonBody.MatchScope)),
						"matchPattern":            llx.ResourceData(matchPattern, "aws.waf.fieldtomatch.jsonbody.matchpattern"),
					})
					if err != nil {
						return nil, err
					}
				}
				fieldToMatch, err = CreateResource(runtime, "aws.waf.rule.fieldtomatch", map[string]*llx.RawData{
					"ruleName":            llx.StringDataPtr(ruleName),
					"queryString":         llx.BoolData(statement.XssMatchStatement.FieldToMatch.QueryString != nil),
					"method":              llx.BoolData(statement.XssMatchStatement.FieldToMatch.Method != nil),
					"uriPath":             llx.BoolData(statement.XssMatchStatement.FieldToMatch.UriPath != nil),
					"allQueryArguments":   llx.BoolData(statement.XssMatchStatement.FieldToMatch.AllQueryArguments != nil),
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
			}
			xssmatchstatement, err = CreateResource(runtime, "aws.waf.rule.statement.xssmatchstatement", map[string]*llx.RawData{
				"ruleName":     llx.StringDataPtr(ruleName),
				"fieldToMatch": llx.ResourceData(fieldToMatch, "aws.waf.rule.fieldtomatch"),
			})
			if err != nil {
				return nil, err
			}
		}
		if statement.SqliMatchStatement != nil {
			var fieldToMatch plugin.Resource
			if statement.SqliMatchStatement.FieldToMatch != nil {
				var singleHeader plugin.Resource
				var singleQueryArgument plugin.Resource
				var body plugin.Resource
				var cookie plugin.Resource
				var headerOrder plugin.Resource
				var headers plugin.Resource
				var ja3Fingerprint plugin.Resource
				var jsonBody plugin.Resource
				if statement.SqliMatchStatement.FieldToMatch.SingleHeader != nil {
					singleHeader, err = CreateResource(runtime, "aws.waf.rule.fieldtomatch.singleheader", map[string]*llx.RawData{
						"ruleName": llx.StringDataPtr(ruleName),
						"name":     llx.StringDataPtr(statement.SqliMatchStatement.FieldToMatch.SingleHeader.Name),
					})
					if err != nil {
						return nil, err
					}
				}
				if statement.SqliMatchStatement.FieldToMatch.SingleQueryArgument != nil {
					singleQueryArgument, err = CreateResource(runtime, "aws.waf.rule.fieldtomatch.singlequeryargument", map[string]*llx.RawData{
						"ruleName": llx.StringDataPtr(ruleName),
						"name":     llx.StringDataPtr(statement.SqliMatchStatement.FieldToMatch.SingleQueryArgument.Name),
					})
					if err != nil {
						return nil, err
					}
				}
				if statement.SqliMatchStatement.FieldToMatch.Body != nil {
					body, err = CreateResource(runtime, "aws.waf.rule.fieldtomatch.body", map[string]*llx.RawData{
						"ruleName":         llx.StringDataPtr(ruleName),
						"overSizeHandling": llx.StringData(string(statement.SqliMatchStatement.FieldToMatch.Body.OversizeHandling)),
					})
				}
				if statement.SqliMatchStatement.FieldToMatch.Cookies != nil {
					cookie, err = CreateResource(runtime, "aws.waf.rule.fieldtomatch.cookie", map[string]*llx.RawData{
						"ruleName":         llx.StringDataPtr(ruleName),
						"overSizeHandling": llx.StringData(string(statement.SqliMatchStatement.FieldToMatch.Cookies.OversizeHandling)),
					})
				}
				if statement.SqliMatchStatement.FieldToMatch.HeaderOrder != nil {
					headerOrder, err = CreateResource(runtime, "aws.waf.rule.fieldtomatch.headerOrder", map[string]*llx.RawData{
						"ruleName":         llx.StringDataPtr(ruleName),
						"overSizeHandling": llx.StringData(string(statement.SqliMatchStatement.FieldToMatch.Headers.OversizeHandling)),
					})
				}
				if statement.SqliMatchStatement.FieldToMatch.SingleHeader != nil {
					singleHeader, err = CreateResource(runtime, "aws.waf.rule.fieldtomatch.singleheader", map[string]*llx.RawData{
						"ruleName": llx.StringDataPtr(ruleName),
						"name":     llx.StringDataPtr(statement.SqliMatchStatement.FieldToMatch.SingleHeader.Name),
					})
				}
				if statement.SqliMatchStatement.FieldToMatch.HeaderOrder != nil {
					singleQueryArgument, err = CreateResource(runtime, "aws.waf.rule.fieldtomatch.singlequeryargument", map[string]*llx.RawData{
						"ruleName": llx.StringDataPtr(ruleName),
						"name":     llx.StringDataPtr(statement.SqliMatchStatement.FieldToMatch.SingleQueryArgument.Name),
					})
				}

				if statement.SqliMatchStatement.FieldToMatch.JA3Fingerprint != nil {
					ja3Fingerprint, err = CreateResource(runtime, "aws.waf.rule.fieldtomatch.ja3fingerprint", map[string]*llx.RawData{
						"ruleName":         llx.StringDataPtr(ruleName),
						"fallbackBehavior": llx.StringData(string(statement.SqliMatchStatement.FieldToMatch.JA3Fingerprint.FallbackBehavior)),
					})
				}

				if statement.SqliMatchStatement.FieldToMatch.Headers != nil {
					var matchPattern plugin.Resource
					if statement.SqliMatchStatement.FieldToMatch.JsonBody.MatchPattern != nil {
						includeHeaders := convert.SliceAnyToInterface(statement.SqliMatchStatement.FieldToMatch.Headers.MatchPattern.IncludedHeaders)
						excludeHeaders := convert.SliceAnyToInterface(statement.SqliMatchStatement.FieldToMatch.Headers.MatchPattern.ExcludedHeaders)
						matchPattern, err = CreateResource(runtime, "aws.waf.rule.fieldtomatch.jsonbody.matchpattern", map[string]*llx.RawData{
							"ruleName":       llx.StringDataPtr(ruleName),
							"all":            llx.BoolData(statement.SqliMatchStatement.FieldToMatch.Headers.MatchPattern.All != nil),
							"includeHeaders": llx.ArrayData(includeHeaders, types.String),
							"excludeHeaders": llx.ArrayData(excludeHeaders, types.String),
						})
					}
					headers, err = CreateResource(runtime, "aws.waf.rule.fieldtomatch.headers", map[string]*llx.RawData{
						"ruleName":         llx.StringDataPtr(ruleName),
						"matchPattern":     llx.ResourceData(matchPattern, "aws.waf.rule.fieldtomatch.headers.matchpatern"),
						"overSizeHandling": llx.StringData(string(statement.SqliMatchStatement.FieldToMatch.Headers.OversizeHandling)),
						"matchScope":       llx.StringData(string(statement.SqliMatchStatement.FieldToMatch.Headers.MatchScope)),
					})

				}
				if statement.SqliMatchStatement.FieldToMatch.JsonBody != nil {
					var matchPattern plugin.Resource
					includePathsArray := convert.SliceAnyToInterface(statement.SqliMatchStatement.FieldToMatch.JsonBody.MatchPattern.IncludedPaths)
					if statement.SqliMatchStatement.FieldToMatch.JsonBody.MatchPattern != nil {
						matchPattern, err = CreateResource(runtime, "aws.waf.rule.fieldtomatch.jsonbody.matchpattern", map[string]*llx.RawData{
							"ruleName":     llx.StringDataPtr(ruleName),
							"all":          llx.BoolData(statement.SqliMatchStatement.FieldToMatch.JsonBody.MatchPattern.All != nil),
							"includePaths": llx.ArrayData(includePathsArray, types.String),
						})
						if err != nil {
							return nil, err
						}
					}
					jsonBody, err = CreateResource(runtime, "aws.waf.rule.fieldtomatch.jsonbody", map[string]*llx.RawData{
						"ruleName":                llx.StringDataPtr(ruleName),
						"overSizeHandling":        llx.StringData(string(statement.SqliMatchStatement.FieldToMatch.JsonBody.OversizeHandling)),
						"invalidFallbackBehavior": llx.StringData(string(statement.SqliMatchStatement.FieldToMatch.JsonBody.InvalidFallbackBehavior)),
						"matchScope":              llx.StringData(string(statement.SqliMatchStatement.FieldToMatch.JsonBody.MatchScope)),
						"matchPattern":            llx.ResourceData(matchPattern, "aws.waf.fieldtomatch.jsonbody.matchpattern"),
					})
					if err != nil {
						return nil, err
					}
				}
				fieldToMatch, err = CreateResource(runtime, "aws.waf.rule.fieldtomatch", map[string]*llx.RawData{
					"ruleName":            llx.StringDataPtr(ruleName),
					"queryString":         llx.BoolData(statement.SqliMatchStatement.FieldToMatch.QueryString != nil),
					"method":              llx.BoolData(statement.SqliMatchStatement.FieldToMatch.Method != nil),
					"uriPath":             llx.BoolData(statement.SqliMatchStatement.FieldToMatch.UriPath != nil),
					"allQueryArguments":   llx.BoolData(statement.SqliMatchStatement.FieldToMatch.AllQueryArguments != nil),
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
			}
			sqlimatchstatement, err = CreateResource(runtime, "aws.waf.rule.statement.sqlimatchstatement", map[string]*llx.RawData{
				"ruleName":         llx.StringDataPtr(ruleName),
				"fieldToMatch":     llx.ResourceData(fieldToMatch, "aws.waf.rule.fieldtomatch"),
				"sensitivityLevel": llx.StringData(string(statement.SqliMatchStatement.SensitivityLevel)),
			})
			if err != nil {
				return nil, err
			}
		}
		if statement.GeoMatchStatement != nil {
			var countryCodes []string
			for _, countryCode := range statement.GeoMatchStatement.CountryCodes {
				countryCodes = append(countryCodes, string(countryCode))
			}
			countryCodesArray := convert.SliceAnyToInterface(countryCodes)
			geomatchstatement, err = CreateResource(runtime, "aws.waf.rule.statement.geomatchstatement", map[string]*llx.RawData{
				"ruleName":     llx.StringDataPtr(ruleName),
				"countryCodes": llx.ArrayData(countryCodesArray, types.String),
			})
			if err != nil {
				return nil, err
			}
		}
		if statement.IPSetReferenceStatement != nil {
			var IPSetForwardedIPConfig plugin.Resource
			if statement.IPSetReferenceStatement.IPSetForwardedIPConfig != nil {
				IPSetForwardedIPConfig, err = CreateResource(runtime, "aws.waf.rule.statement.ipsetreferencestatement.ipsetforwardedipconfig", map[string]*llx.RawData{
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
				"ruleName":               llx.StringDataPtr(ruleName),
				"arn":                    llx.StringDataPtr(statement.IPSetReferenceStatement.ARN),
				"ipSetForwardedIPConfig": llx.ResourceData(IPSetForwardedIPConfig, "aws.waf.rule.statement.ipsetreferencestatement.ipsetforwardedipconfig"),
			})
			if err != nil {
				return nil, err
			}
		}
		if statement.LabelMatchStatement != nil {
			labelmatchstatement, err = CreateResource(runtime, "aws.waf.rule.statement.labelmatchstatement", map[string]*llx.RawData{
				"ruleName": llx.StringDataPtr(ruleName),
				"key":      llx.StringDataPtr(statement.LabelMatchStatement.Key),
				"scope":    llx.StringData(string(statement.LabelMatchStatement.Scope)),
			})
			if err != nil {
				return nil, err
			}
		}
		if statement.ManagedRuleGroupStatement != nil {
			managedrulegroupstatement, err = CreateResource(runtime, "aws.waf.rule.statement.managedrulegroupstatement", map[string]*llx.RawData{
				"ruleName":   llx.StringDataPtr(ruleName),
				"Name":       llx.StringDataPtr(statement.ManagedRuleGroupStatement.Name),
				"VendorName": llx.StringDataPtr(statement.ManagedRuleGroupStatement.VendorName),
			})
			if err != nil {
				return nil, err
			}
		}
		if statement.AndStatement != nil {
			var statements []interface{}
			for _, statement := range statement.AndStatement.Statements {
				andStatementMqlStatement, err := createStatementResource(runtime, &statement, ruleName)
				if err != nil {
					return nil, err
				}
				statements = append(statements, andStatementMqlStatement)
			}
			andStatement, err = CreateResource(runtime, "aws.waf.rule.statement.andstatement", map[string]*llx.RawData{
				"statements": llx.ArrayData(statements, types.ResourceLike),
			})
			if err != nil {
				return nil, err
			}
		}
		if statement.NotStatement != nil {
			var notStatementMqlStatement plugin.Resource
			notStatementMqlStatement, err = createStatementResource(runtime, statement.NotStatement.Statement, ruleName)
			if err != nil {
				return nil, err
			}
			notstatement, err = CreateResource(runtime, "aws.waf.rule.statement.notstatement", map[string]*llx.RawData{
				"statement": llx.ResourceData(notStatementMqlStatement, "aws.waf.rule.statement.notstatement"),
			})
		}
		if statement.OrStatement != nil {
			var statements []interface{}
			for _, statement := range statement.OrStatement.Statements {
				orStatementMqlStatement, err := createStatementResource(runtime, &statement, ruleName)
				if err != nil {
					return nil, err
				}
				statements = append(statements, orStatementMqlStatement)
			}
			orstatement, err = CreateResource(runtime, "aws.waf.rule.statement.orstatement", map[string]*llx.RawData{
				"statements": llx.ArrayData(statements, types.ResourceLike),
			})
		}
		if statement.RateBasedStatement != nil {
			ratebasedstatement, err = CreateResource(runtime, "aws.waf.rule.statement.ratebasedstatement", map[string]*llx.RawData{})
			if err != nil {
				return nil, err
			}
		}
		if statement.RegexPatternSetReferenceStatement != nil {
			regexpatternsetreferencestatement, err = CreateResource(runtime, "aws.waf.rule.statement.regexpatternsetreferencestatement", map[string]*llx.RawData{})
			if err != nil {
				return nil, err
			}
		}
		if statement.RuleGroupReferenceStatement != nil {
			rulegroupreferencestatement, err = CreateResource(runtime, "aws.waf.rule.statement.rulegroupreferencestatement", map[string]*llx.RawData{})
			if err != nil {
				return nil, err
			}
		}
		if statement.SizeConstraintStatement != nil {
			var fieldToMatch plugin.Resource
			if statement.SizeConstraintStatement.FieldToMatch != nil {
				var body plugin.Resource
				var cookie plugin.Resource
				var singleHeader plugin.Resource
				var headerOrder plugin.Resource
				var headers plugin.Resource
				var ja3Fingerprint plugin.Resource
				var jsonBody plugin.Resource
				var singleQueryArgument plugin.Resource
				if statement.SizeConstraintStatement.FieldToMatch.Body != nil {
					body, err = CreateResource(runtime, "aws.waf.rule.statement.sizeconstraintstatement.fieldtomatch.body", map[string]*llx.RawData{
						"ruleName":         llx.StringDataPtr(ruleName),
						"overSizeHandling": llx.StringData(string(statement.SizeConstraintStatement.FieldToMatch.Body.OversizeHandling)),
					})
				}
				if statement.SizeConstraintStatement.FieldToMatch.Cookies != nil {
					cookie, err = CreateResource(runtime, "aws.waf.rule.statement.sizeconstraintstatement.fieldtomatch.cookie", map[string]*llx.RawData{
						"ruleName":         llx.StringDataPtr(ruleName),
						"overSizeHandling": llx.StringData(string(statement.SizeConstraintStatement.FieldToMatch.Cookies.OversizeHandling)),
					})
				}
				if statement.SizeConstraintStatement.FieldToMatch.HeaderOrder != nil {
					headerOrder, err = CreateResource(runtime, "aws.waf.rule.statement.sizeconstraintstatement.fieldtomatch.headerOrder", map[string]*llx.RawData{
						"ruleName":         llx.StringDataPtr(ruleName),
						"overSizeHandling": llx.StringData(string(statement.SizeConstraintStatement.FieldToMatch.Headers.OversizeHandling)),
					})
				}
				if statement.SizeConstraintStatement.FieldToMatch.SingleHeader != nil {
					singleHeader, err = CreateResource(runtime, "aws.waf.rule.statement.sizeconstraintstatement.fieldtomatch.singleheader", map[string]*llx.RawData{
						"name":     llx.StringDataPtr(statement.SizeConstraintStatement.FieldToMatch.SingleHeader.Name),
						"ruleName": llx.StringDataPtr(ruleName),
					})
				}
				if statement.SizeConstraintStatement.FieldToMatch.HeaderOrder != nil {
					singleQueryArgument, err = CreateResource(runtime, "aws.waf.rule.statement.sizeconstraintstatement.fieldtomatch.singlequeryargument", map[string]*llx.RawData{
						"ruleName": llx.StringDataPtr(ruleName),
						"name":     llx.StringDataPtr(statement.SizeConstraintStatement.FieldToMatch.SingleQueryArgument.Name),
					})
				}

				if statement.SizeConstraintStatement.FieldToMatch.JA3Fingerprint != nil {
					ja3Fingerprint, err = CreateResource(runtime, "aws.waf.rule.statement.sizeconstraintstatement.fieldtomatch.ja3fingerprint", map[string]*llx.RawData{
						"ruleName":         llx.StringDataPtr(ruleName),
						"fallbackBehavior": llx.StringData(string(statement.SizeConstraintStatement.FieldToMatch.JA3Fingerprint.FallbackBehavior)),
					})
				}

				if statement.SizeConstraintStatement.FieldToMatch.Headers != nil {
					var matchPattern plugin.Resource
					if statement.SizeConstraintStatement.FieldToMatch.JsonBody.MatchPattern != nil {
						includeHeaders := convert.SliceAnyToInterface(statement.SizeConstraintStatement.FieldToMatch.Headers.MatchPattern.IncludedHeaders)
						excludeHeaders := convert.SliceAnyToInterface(statement.SizeConstraintStatement.FieldToMatch.Headers.MatchPattern.ExcludedHeaders)
						matchPattern, err = CreateResource(runtime, "aws.waf.rule.statement.sizeconstraintstatement.fieldtomatch.jsonbody.matchpattern", map[string]*llx.RawData{
							"ruleName":       llx.StringDataPtr(ruleName),
							"all":            llx.BoolData(statement.SizeConstraintStatement.FieldToMatch.Headers.MatchPattern.All != nil),
							"includeHeaders": llx.ArrayData(includeHeaders, types.String),
							"excludeHeaders": llx.ArrayData(excludeHeaders, types.String),
						})
					}
					headers, err = CreateResource(runtime, "aws.waf.rule.statement.sizeconstraintstatement.fieldtomatch.headers", map[string]*llx.RawData{
						"ruleName":         llx.StringDataPtr(ruleName),
						"matchPattern":     llx.ResourceData(matchPattern, "aws.waf.rule.statement.sizeconstraintstatement.fieldtomatch.headers.matchpatern"),
						"overSizeHandling": llx.StringData(string(statement.SizeConstraintStatement.FieldToMatch.Headers.OversizeHandling)),
						"matchScope":       llx.StringData(string(statement.SizeConstraintStatement.FieldToMatch.Headers.MatchScope)),
					})

				}

				if statement.SizeConstraintStatement.FieldToMatch.JsonBody != nil {
					var matchPattern plugin.Resource
					includePathsArray := convert.SliceAnyToInterface(statement.SizeConstraintStatement.FieldToMatch.JsonBody.MatchPattern.IncludedPaths)
					if statement.SizeConstraintStatement.FieldToMatch.JsonBody.MatchPattern != nil {
						matchPattern, err = CreateResource(runtime, "aws.waf.rule.statement.sizeconstraintstatement.fieldtomatch.jsonbody.matchpattern", map[string]*llx.RawData{
							"ruleName":     llx.StringDataPtr(ruleName),
							"all":          llx.BoolData(statement.SizeConstraintStatement.FieldToMatch.JsonBody.MatchPattern.All != nil),
							"includePaths": llx.ArrayData(includePathsArray, types.String),
						})
						if err != nil {
							return nil, err
						}
					}
					jsonBody, err = CreateResource(runtime, "aws.waf.rule.statement.sizeconstraintstatement.fieldtomatch.jsonbody", map[string]*llx.RawData{
						"ruleName":                llx.StringDataPtr(ruleName),
						"overSizeHandling":        llx.StringData(string(statement.SizeConstraintStatement.FieldToMatch.JsonBody.OversizeHandling)),
						"invalidFallbackBehavior": llx.StringData(string(statement.SizeConstraintStatement.FieldToMatch.JsonBody.InvalidFallbackBehavior)),
						"matchScope":              llx.StringData(string(statement.SizeConstraintStatement.FieldToMatch.JsonBody.MatchScope)),
						"matchPattern":            llx.ResourceData(matchPattern, "aws.waf.rule.statement.sizeconstraintstatement.fieldtomatch.jsonbody.matchpattern"),
					})
					if err != nil {
						return nil, err
					}
				}

				fieldToMatch, err = CreateResource(runtime, "aws.waf.rule.statement.sizeconstraintstatement.fieldtomatch", map[string]*llx.RawData{
					"ruleName":            llx.StringDataPtr(ruleName),
					"method":              llx.BoolData(statement.SizeConstraintStatement.FieldToMatch.Method != nil),
					"queryString":         llx.BoolData(statement.SizeConstraintStatement.FieldToMatch.QueryString != nil),
					"allQueryArguments":   llx.BoolData(statement.SizeConstraintStatement.FieldToMatch.AllQueryArguments != nil),
					"uriPath":             llx.BoolData(statement.SizeConstraintStatement.FieldToMatch.UriPath != nil),
					"body":                llx.ResourceData(body, "aws.waf.rule.statement.sizeconstraintstatement.fieldtomatch.body"),
					"cookie":              llx.ResourceData(cookie, "aws.waf.rule.statement.sizeconstraintstatement.fieldtomatch.cookie"),
					"singleHeader":        llx.ResourceData(singleHeader, "aws.waf.rule.statement.sizeconstraintstatement.fieldtomatch.singleheader"),
					"headerOrder":         llx.ResourceData(headerOrder, "aws.waf.rule.statement.sizeconstraintstatement.fieldToMatch.headerorder"),
					"headers":             llx.ResourceData(headers, "aws.waf.rule.statement.sizeconstraintstatement.fieldToMatch.headers"),
					"ja3Fingerprint":      llx.ResourceData(ja3Fingerprint, "aws.waf.rule.statement.sizeconstraintstatement.fieldToMatch.ja3fingerprint"),
					"jsonBody":            llx.ResourceData(jsonBody, "aws.waf.rule.statement.sizeconstraintstatement.fieldToMatch.jsonbody"),
					"singleQueryArgument": llx.ResourceData(singleQueryArgument, "aws.waf.rule.statement.sizeconstraintstatement.fieldToMatch.singlequeryargument"),
				})
				if err != nil {
					return nil, err
				}
			}
			sizeconstraintstatement, err = CreateResource(runtime, "aws.waf.rule.statement.sizeconstraintstatement", map[string]*llx.RawData{
				"ruleName":           llx.StringDataPtr(ruleName),
				"size":               llx.IntData(statement.SizeConstraintStatement.Size),
				"comparisonOperator": llx.StringData(string(statement.SizeConstraintStatement.ComparisonOperator)),
				"fieldToMatch":       llx.ResourceData(fieldToMatch, "aws.waf.rule.statement.sizeconstraintstatement.fieldtomatch"),
			})
			if err != nil {
				return nil, err
			}
		}
	}
	//fmt.Println(regexmatchstatement)
	//fmt.Println(bytematchstatement)
	//fmt.Println(xssmatchstatement)
	//fmt.Println(sqlimatchstatement)
	//fmt.Println(geomatchstatement)
	//fmt.Println(ipsetreferencestatement)
	//fmt.Println(labelmatchstatement)
	//fmt.Println(managedrulegroupstatement)
	//fmt.Println(notstatement)
	//fmt.Println(orstatement)
	//fmt.Println(ratebasedstatement)
	//fmt.Println(regexpatternsetreferencestatement)
	//fmt.Println(rulegroupreferencestatement)
	//fmt.Println(sizeconstraintstatement)
	var mqlStatement plugin.Resource
	mqlStatementID := uuid.New() // maybe use the rule.name instead?
	mqlStatement, err = CreateResource(runtime, "aws.waf.rule.statement",
		map[string]*llx.RawData{
			"id":                                llx.StringData(mqlStatementID.String()),
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
