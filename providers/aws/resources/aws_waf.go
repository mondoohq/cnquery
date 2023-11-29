package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/wafv2"
	waftypes "github.com/aws/aws-sdk-go-v2/service/wafv2/types"

	//"github.com/aws/aws-sdk-go/aws"
	"go.mondoo.com/cnquery/v9/llx"
	"go.mondoo.com/cnquery/v9/providers/aws/connection"
)

func (a *mqlAwsWaf) id() (string, error) {
	return "aws.waf", nil
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
			mqlAcl, err := CreateResource(a.MqlRuntime, "aws.waf.acl",
				map[string]*llx.RawData{
					"id":          llx.StringDataPtr(acl.Id),
					"arn":         llx.StringDataPtr(acl.ARN),
					"name":        llx.StringDataPtr(acl.Name),
					"description": llx.StringDataPtr(acl.Description),
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

func (a *mqlAwsWaf) rules() ([]interface{}, error) {
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
	params := &wafv2.ListManagedRuleSetsInput{Scope: scope}
	for nextMarker != nil {
		aclsRes, err := svc.ListManagedRuleSets(ctx, params)
		if err != nil {
			return nil, err
		}
		nextMarker = aclsRes.NextMarker
		if aclsRes.NextMarker != nil {
			params.NextMarker = nextMarker
		}

		for _, rule := range aclsRes.ManagedRuleSets {
			mqlAcl, err := CreateResource(a.MqlRuntime, "aws.waf.rule",
				map[string]*llx.RawData{
					"id":          llx.StringDataPtr(rule.Id),
					"arn":         llx.StringDataPtr(rule.ARN),
					"name":        llx.StringDataPtr(rule.Name),
					"description": llx.StringDataPtr(rule.Description),
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
