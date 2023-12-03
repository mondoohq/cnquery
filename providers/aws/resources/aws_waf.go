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
		var sqlimatchstatement plugin.Resource
		var xssmatchstatement plugin.Resource
		var bytematchstatement plugin.Resource
		var regexmatchstatement plugin.Resource
		var geomatchstatement plugin.Resource
		var ipsetreferencestatement plugin.Resource
		var labelmatchstatement plugin.Resource
		var managedrulegroupstatement plugin.Resource
		var notstatement plugin.Resource
		var orstatement plugin.Resource
		var ratebasedstatement plugin.Resource
		var regexpatternsetreferencestatement plugin.Resource
		var rulegroupreferencestatement plugin.Resource
		var sizeconstraintstatement plugin.Resource
		if rule.Statement != nil {
			if rule.Statement.RegexMatchStatement != nil {
				var fieldToMatch plugin.Resource
				if rule.Statement.RegexMatchStatement.FieldToMatch != nil {
					var singleHeader plugin.Resource
					if rule.Statement.RegexMatchStatement.FieldToMatch.SingleHeader != nil {
						singleHeader, err = CreateResource(a.MqlRuntime, "aws.waf.rule.statement.regexmatchstatement.fieldtomatch.singleheader", map[string]*llx.RawData{
							"name": llx.StringDataPtr(rule.Statement.RegexMatchStatement.FieldToMatch.SingleHeader.Name),
						})
						if err != nil {
							return nil, err
						}
					}
					var singleQueryArgument plugin.Resource
					if rule.Statement.RegexMatchStatement.FieldToMatch.SingleQueryArgument != nil {
						singleQueryArgument, err = CreateResource(a.MqlRuntime, "aws.waf.rule.statement.regexmatchstatement.fieldtomatch.singlequeryargument", map[string]*llx.RawData{
							"name": llx.StringDataPtr(rule.Statement.RegexMatchStatement.FieldToMatch.SingleQueryArgument.Name),
						})
						if err != nil {
							return nil, err
						}
					}
					var queryString bool
					if rule.Statement.RegexMatchStatement.FieldToMatch.QueryString != nil {
						queryString = true
					} else {
						queryString = false
					}
					fieldToMatch, err = CreateResource(a.MqlRuntime, "aws.waf.rule.statement.regexmatchstatement.fieldtomatch", map[string]*llx.RawData{
						"singleHeader":        llx.ResourceData(singleHeader, "aws.waf.rule.statement.regexmatchstatement.fieldtomatch.singleheader"),
						"singleQueryArgument": llx.ResourceData(singleQueryArgument, "aws.waf.rule.statement.regexmatchstatement.fieldtomatch.singlequeryargument"),
						"queryString":         llx.BoolData(queryString),
					})
					if err != nil {
						return nil, err
					}
				}
				regexmatchstatement, err = CreateResource(a.MqlRuntime, "aws.waf.rule.statement.regexmatchstatement", map[string]*llx.RawData{
					"fieldToMatch": llx.ResourceData(fieldToMatch, "aws.waf.rule.statement.regexmatchstatement.fieldtomatch"),
				})
				if err != nil {
					return nil, err
				}
			}
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
					var singleQueryArgument plugin.Resource
					if rule.Statement.ByteMatchStatement.FieldToMatch.SingleQueryArgument != nil {
						singleQueryArgument, err = CreateResource(a.MqlRuntime, "aws.waf.rule.statement.bytematchstatement.fieldtomatch.singlequeryargument", map[string]*llx.RawData{
							"name": llx.StringDataPtr(rule.Statement.ByteMatchStatement.FieldToMatch.SingleQueryArgument.Name),
						})
						if err != nil {
							return nil, err
						}
					}
					var queryString bool
					if rule.Statement.ByteMatchStatement.FieldToMatch.QueryString != nil {
						queryString = true
					} else {
						queryString = false
					}
					fieldToMatch, err = CreateResource(a.MqlRuntime, "aws.waf.rule.statement.bytematchstatement.fieldtomatch", map[string]*llx.RawData{
						"singleHeader":        llx.ResourceData(singleHeader, "aws.waf.rule.statement.bytematchstatement.fieldtomatch.singleheader"),
						"singleQueryArgument": llx.ResourceData(singleQueryArgument, "aws.waf.rule.statement.bytematchstatement.fieldtomatch.singlequeryargument"),
						"queryString":         llx.BoolData(queryString),
					})
					if err != nil {
						return nil, err
					}
				}
				bytematchstatement, err = CreateResource(a.MqlRuntime, "aws.waf.rule.statement.bytematchstatement", map[string]*llx.RawData{
					"fieldToMatch": llx.ResourceData(fieldToMatch, "aws.waf.rule.statement.bytematchstatement.fieldtomatch"),
					"searchString": llx.StringData(string(rule.Statement.ByteMatchStatement.SearchString)),
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
					var singleQueryArgument plugin.Resource
					if rule.Statement.XssMatchStatement.FieldToMatch.SingleQueryArgument != nil {
						singleQueryArgument, err = CreateResource(a.MqlRuntime, "aws.waf.rule.statement.xssmatchstatement.fieldtomatch.singlequeryargument", map[string]*llx.RawData{
							"name": llx.StringDataPtr(rule.Statement.XssMatchStatement.FieldToMatch.SingleQueryArgument.Name),
						})
						if err != nil {
							return nil, err
						}
					}
					var queryString bool
					if rule.Statement.XssMatchStatement.FieldToMatch.QueryString != nil {
						queryString = true
					} else {
						queryString = false
					}
					fieldToMatch, err = CreateResource(a.MqlRuntime, "aws.waf.rule.statement.xssmatchstatement.fieldtomatch", map[string]*llx.RawData{
						"singleHeader":        llx.ResourceData(singleHeader, "aws.waf.rule.statement.xssmatchstatement.fieldtomatch.singleheader"),
						"singleQueryArgument": llx.ResourceData(singleQueryArgument, "aws.waf.rule.statement.xssmatchstatement.fieldtomatch.singlequeryargument"),
						"queryString":         llx.BoolData(queryString),
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
					var singleQueryArgument plugin.Resource
					if rule.Statement.SqliMatchStatement.FieldToMatch.SingleQueryArgument != nil {
						singleQueryArgument, err = CreateResource(a.MqlRuntime, "aws.waf.rule.statement.sqlimatchstatement.fieldtomatch.singlequeryargument", map[string]*llx.RawData{
							"name": llx.StringDataPtr(rule.Statement.SqliMatchStatement.FieldToMatch.SingleQueryArgument.Name),
						})
						if err != nil {
							return nil, err
						}
					}
					var queryString bool
					if rule.Statement.SqliMatchStatement.FieldToMatch.QueryString != nil {
						queryString = true
					} else {
						queryString = false
					}
					fieldToMatch, err = CreateResource(a.MqlRuntime, "aws.waf.rule.statement.sqlimatchstatement.fieldtomatch", map[string]*llx.RawData{
						"singleHeader":        llx.ResourceData(singleHeader, "aws.waf.rule.statement.sqlimatchstatement.fieldtomatch.singleheader"),
						"singleQueryArgument": llx.ResourceData(singleQueryArgument, "aws.waf.rule.statement.sqlimatchstatement.fieldtomatch.singlequeryargument"),
						"queryString":         llx.BoolData(queryString),
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
			if rule.Statement.GeoMatchStatement != nil {
				var countryCodes []string
				for _, countryCode := range rule.Statement.GeoMatchStatement.CountryCodes {
					countryCodes = append(countryCodes, string(countryCode))
				}
				countryCodesArray := convert.SliceAnyToInterface(countryCodes)
				geomatchstatement, err = CreateResource(a.MqlRuntime, "aws.waf.rule.statement.geomatchstatement", map[string]*llx.RawData{
					"countryCodes": llx.ArrayData(countryCodesArray, types.String),
				})
				if err != nil {
					return nil, err
				}
			}
			if rule.Statement.IPSetReferenceStatement != nil {
				ipsetreferencestatement, err = CreateResource(a.MqlRuntime, "aws.waf.rule.statement.ipsetreferencestatement", map[string]*llx.RawData{})
				if err != nil {
					return nil, err
				}
			}
			if rule.Statement.LabelMatchStatement != nil {
				labelmatchstatement, err = CreateResource(a.MqlRuntime, "aws.waf.rule.statement.labelmatchstatement", map[string]*llx.RawData{
					"key":   llx.StringDataPtr(rule.Statement.LabelMatchStatement.Key),
					"scope": llx.StringData(string(rule.Statement.LabelMatchStatement.Scope)),
				})
				if err != nil {
					return nil, err
				}
			}
			if rule.Statement.ManagedRuleGroupStatement != nil {
				managedrulegroupstatement, err = CreateResource(a.MqlRuntime, "aws.waf.rule.statement.managedrulegroupstatement", map[string]*llx.RawData{})
				if err != nil {
					return nil, err
				}
			}
			if rule.Statement.NotStatement != nil {
				notstatement, err = CreateResource(a.MqlRuntime, "aws.waf.rule.statement.notstatement", map[string]*llx.RawData{})
				if err != nil {
					return nil, err
				}
			}
			if rule.Statement.OrStatement != nil {
				orstatement, err = CreateResource(a.MqlRuntime, "aws.waf.rule.statement.orstatement", map[string]*llx.RawData{})
				if err != nil {
					return nil, err
				}
			}
			if rule.Statement.RateBasedStatement != nil {
				ratebasedstatement, err = CreateResource(a.MqlRuntime, "aws.waf.rule.statement.ratebasedstatement", map[string]*llx.RawData{})
				if err != nil {
					return nil, err
				}
			}
			if rule.Statement.RegexPatternSetReferenceStatement != nil {
				regexpatternsetreferencestatement, err = CreateResource(a.MqlRuntime, "aws.waf.rule.statement.regexpatternsetreferencestatement", map[string]*llx.RawData{})
				if err != nil {
					return nil, err
				}
			}
			if rule.Statement.RuleGroupReferenceStatement != nil {
				rulegroupreferencestatement, err = CreateResource(a.MqlRuntime, "aws.waf.rule.statement.rulegroupreferencestatement", map[string]*llx.RawData{})
				if err != nil {
					return nil, err
				}
			}
			if rule.Statement.SizeConstraintStatement != nil {
				var fieldToMatch plugin.Resource
				if rule.Statement.SizeConstraintStatement.FieldToMatch != nil {
					var body plugin.Resource
					var cookie plugin.Resource
					var singleHeader plugin.Resource
					//var allQueryArguments plugin.Resource
					var headerOrder plugin.Resource
					//var headers plugin.Resource
					var ja3Fingerprint plugin.Resource
					var jsonBody plugin.Resource
					//var queryString plugin.Resource
					var singleQueryArgument plugin.Resource
					if rule.Statement.SizeConstraintStatement.FieldToMatch.Body != nil {
						body, err = CreateResource(a.MqlRuntime, "aws.waf.rule.statement.sizeconstraintstatement.fieldtomatch.body", map[string]*llx.RawData{
							"overSizeHandling": llx.StringData(string(rule.Statement.SizeConstraintStatement.FieldToMatch.Body.OversizeHandling)),
						})
					}
					if rule.Statement.SizeConstraintStatement.FieldToMatch.Cookies != nil {
						cookie, err = CreateResource(a.MqlRuntime, "aws.waf.rule.statement.sizeconstraintstatement", map[string]*llx.RawData{
							"overSizeHandling": llx.StringData(string(rule.Statement.SizeConstraintStatement.FieldToMatch.Body.OversizeHandling)),
						})
					}
					if rule.Statement.SizeConstraintStatement.FieldToMatch.HeaderOrder != nil {
						headerOrder, err = CreateResource(a.MqlRuntime, "aws.waf.rule.statement.sizeconstraintstatement.fieldtomatch.headerOrder", map[string]*llx.RawData{
							"OverSizeHandling": llx.StringDataPtr(rule.Statement.SizeConstraintStatement.FieldToMatch.SingleHeader.Name),
						})
					}
					if rule.Statement.SizeConstraintStatement.FieldToMatch.SingleHeader != nil {
						singleHeader, err = CreateResource(a.MqlRuntime, "aws.waf.rule.statement.sizeconstraintstatement.fieldtomatch.singleheader", map[string]*llx.RawData{
							"name": llx.StringDataPtr(rule.Statement.SizeConstraintStatement.FieldToMatch.SingleHeader.Name),
						})
					}
					if rule.Statement.SizeConstraintStatement.FieldToMatch.HeaderOrder != nil {
						singleQueryArgument, err = CreateResource(a.MqlRuntime, "aws.waf.rule.statement.sizeconstraintstatement.fieldtomatch.singlequeryargument", map[string]*llx.RawData{
							"name": llx.StringDataPtr(rule.Statement.SizeConstraintStatement.FieldToMatch.SingleQueryArgument.Name),
						})
					}

					if rule.Statement.SizeConstraintStatement.FieldToMatch.JA3Fingerprint != nil {
						ja3Fingerprint, err = CreateResource(a.MqlRuntime, "aws.waf.rule.statement.sizeconstraintstatement.fieldtomatch.ja3fingerprint", map[string]*llx.RawData{
							"fallbackBehavior": llx.StringData(string(rule.Statement.SizeConstraintStatement.FieldToMatch.JA3Fingerprint.FallbackBehavior)),
						})
					}

					if rule.Statement.SizeConstraintStatement.FieldToMatch.JsonBody != nil {
						var matchPattern plugin.Resource
						includePathsArray := convert.SliceAnyToInterface(rule.Statement.SizeConstraintStatement.FieldToMatch.JsonBody.MatchPattern.IncludedPaths)
						if rule.Statement.SizeConstraintStatement.FieldToMatch.JsonBody.MatchPattern != nil {
							matchPattern, err = CreateResource(a.MqlRuntime, "aws.waf.rule.statement.sizeconstraintstatement.fieldtomatch.jsonbody.matchpattern", map[string]*llx.RawData{
								"all":          llx.BoolData(rule.Statement.SizeConstraintStatement.FieldToMatch.JsonBody.MatchPattern.All != nil),
								"includePaths": llx.ArrayData(includePathsArray, types.String),
							})
							if err != nil {
								return nil, err
							}
						}
						jsonBody, err = CreateResource(a.MqlRuntime, "aws.waf.rule.statement.sizeconstraintstatement.fieldtomatch.jsonbody", map[string]*llx.RawData{
							"overSizeHandling":        llx.StringData(string(rule.Statement.SizeConstraintStatement.FieldToMatch.JsonBody.OversizeHandling)),
							"invalidFallbackBehavior": llx.StringData(string(rule.Statement.SizeConstraintStatement.FieldToMatch.JsonBody.InvalidFallbackBehavior)),
							"matchPattern":            llx.ResourceData(matchPattern, "aws.waf.rule.statement.sizeconstraintstatement.fieldtomatch.jsonbody.matchpattern"),
						})
						if err != nil {
							return nil, err
						}
					}

					fieldToMatch, err = CreateResource(a.MqlRuntime, "aws.waf.rule.statement.sizeconstraintstatement.fieldtomatch", map[string]*llx.RawData{
						"method":       llx.BoolData(rule.Statement.SizeConstraintStatement.FieldToMatch.Method != nil),
						"uriPath":      llx.BoolData(rule.Statement.SizeConstraintStatement.FieldToMatch.UriPath != nil),
						"body":         llx.ResourceData(body, "aws.waf.rule.statement.sizeconstraintstatement.fieldtomatch.body"),
						"cookie":       llx.ResourceData(cookie, "aws.waf.rule.statement.sizeconstraintstatement.fieldtomatch.cookie"),
						"singleHeader": llx.ResourceData(singleHeader, "aws.waf.rule.statement.sizeconstraintstatement.fieldtomatch.singleheader"),
						//"allQueryArguments":   llx.ResourceData(allQueryArguments, "aws.waf.rule.statement.sizeconstraintstatement.fifieldToMatch.allqueryarguments"),
						"headerOrder": llx.ResourceData(headerOrder, "aws.waf.rule.statement.sizeconstraintstatement.fifieldToMatch.headerorder"),
						//"headers":             llx.ResourceData(headers, "aws.waf.rule.statement.sizeconstraintstatement.fifieldToMatch.headers"),
						"ja3Fingerprint": llx.ResourceData(ja3Fingerprint, "aws.waf.rule.statement.sizeconstraintstatement.fifieldToMatch.ja3fingerprint"),
						"jsonBody":       llx.ResourceData(jsonBody, "aws.waf.rule.statement.sizeconstraintstatement.fifieldToMatch.jsonbody"),
						//"queryString":         llx.ResourceData(queryString, "aws.waf.rule.statement.sizeconstraintstatement.fifieldToMatch.querystring"),
						"singleQueryArgument": llx.ResourceData(singleQueryArgument, "aws.waf.rule.statement.sizeconstraintstatement.fifieldToMatch.singlequeryargument"),
					})
					if err != nil {
						return nil, err
					}
				}
				sizeconstraintstatement, err = CreateResource(a.MqlRuntime, "aws.waf.rule.statement.sizeconstraintstatement", map[string]*llx.RawData{
					"size":               llx.IntData(rule.Statement.SizeConstraintStatement.Size),
					"comparisonOperator": llx.StringData(string(rule.Statement.SizeConstraintStatement.ComparisonOperator)),
					"fieldToMatch":       llx.ResourceData(fieldToMatch, "aws.waf.rule.statement.sizeconstraintstatement.fieldtomatch"),
				})
				if err != nil {
					return nil, err
				}
			}
		}
		fmt.Println(regexmatchstatement)
		fmt.Println(bytematchstatement)
		fmt.Println(xssmatchstatement)
		fmt.Println(sqlimatchstatement)
		fmt.Println(geomatchstatement)
		fmt.Println(ipsetreferencestatement)
		fmt.Println(labelmatchstatement)
		fmt.Println(managedrulegroupstatement)
		fmt.Println(notstatement)
		fmt.Println(orstatement)
		fmt.Println(ratebasedstatement)
		fmt.Println(regexpatternsetreferencestatement)
		fmt.Println(rulegroupreferencestatement)
		fmt.Println(sizeconstraintstatement)
		var mqlStatement plugin.Resource
		mqlStatementID := uuid.New() // maybe use the rule.name instead?
		mqlStatement, err = CreateResource(a.MqlRuntime, "aws.waf.rule.statement",
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
				"rateBasedStatement":                llx.ResourceData(ratebasedstatement, "aws.waf.rule.statement.ratebasedstatement"),
				"regexPatternSetReferenceStatement": llx.ResourceData(regexpatternsetreferencestatement, "aws.waf.rule.statement.regexpatternsetreferencestatement"),
				"ruleGroupReferenceStatement":       llx.ResourceData(rulegroupreferencestatement, "aws.waf.rule.statement.rulegroupreferencestatement"),
				"sizeConstraintStatement":           llx.ResourceData(sizeconstraintstatement, "aws.waf.rule.statement.sizeconstraintstatement"),
			},
		)
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

func (a *mqlAwsWafRuleStatementSqlimatchstatement) id() (string, error) {
	return "aws.waf.rule.sqlimatchstatement", nil
}

func (a *mqlAwsWafRuleStatementSqlimatchstatementFieldtomatch) id() (string, error) {
	return "aws.waf.rule.sqlimatchstatement.fieldtomatch", nil
}

func (a *mqlAwsWafRuleStatementSqlimatchstatementFieldtomatchSingleheader) id() (string, error) {
	return a.Name.Data, nil
}

func (a *mqlAwsWafRuleStatementSqlimatchstatementFieldtomatchSinglequeryargument) id() (string, error) {
	return a.Name.Data, nil
}

func (a *mqlAwsWafRuleStatementBytematchstatement) id() (string, error) {
	return "aws.waf.rule.bytematchstatement", nil
}

func (a *mqlAwsWafRuleStatementBytematchstatementFieldtomatch) id() (string, error) {
	return "aws.waf.rule.bytematchstatement.fieldtomatch", nil
}

func (a *mqlAwsWafRuleStatementBytematchstatementFieldtomatchSingleheader) id() (string, error) {
	return a.Name.Data, nil
}

func (a *mqlAwsWafRuleStatementBytematchstatementFieldtomatchSinglequeryargument) id() (string, error) {
	return a.Name.Data, nil
}

func (a *mqlAwsWafRuleStatementXssmatchstatement) id() (string, error) {
	return "aws.waf.rule.xssmatchstatement", nil
}

func (a *mqlAwsWafRuleStatementXssmatchstatementFieldtomatch) id() (string, error) {
	return "aws.waf.rule.sqlimatchstatement.fieldtomatch", nil
}

func (a *mqlAwsWafRuleStatementXssmatchstatementFieldtomatchSingleheader) id() (string, error) {
	return a.Name.Data, nil
}

func (a *mqlAwsWafRuleStatementXssmatchstatementFieldtomatchSinglequeryargument) id() (string, error) {
	return a.Name.Data, nil
}

func (a *mqlAwsWafRuleStatementRegexmatchstatement) id() (string, error) {
	return "aws.waf.rule.regexstatement", nil
}

func (a *mqlAwsWafRuleStatementRegexmatchstatementFieldtomatch) id() (string, error) {
	return "aws.waf.rule.regexstatement.fieldtomatch", nil
}

func (a *mqlAwsWafRuleStatementRegexmatchstatementFieldtomatchSingleheader) id() (string, error) {
	return a.Name.Data, nil
}

func (a *mqlAwsWafRuleStatementRegexmatchstatementFieldtomatchSinglequeryargument) id() (string, error) {
	return a.Name.Data, nil
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
