// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/networkfirewall"
	nftypes "github.com/aws/aws-sdk-go-v2/service/networkfirewall/types"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/mql/v13/providers/aws/connection"
)

func (a *mqlAwsNetworkfirewall) id() (string, error) {
	return "aws.networkfirewall", nil
}

func (a *mqlAwsNetworkfirewall) firewalls() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getFirewalls(conn), 5)
	poolOfJobs.Run()

	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	for i := range poolOfJobs.Jobs {
		if poolOfJobs.Jobs[i].Result != nil {
			res = append(res, poolOfJobs.Jobs[i].Result.([]any)...)
		}
	}
	return res, nil
}

func (a *mqlAwsNetworkfirewall) getFirewalls(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			svc := conn.NetworkFirewall(region)
			ctx := context.Background()

			res := []any{}
			paginator := networkfirewall.NewListFirewallsPaginator(svc, &networkfirewall.ListFirewallsInput{})
			for paginator.HasMorePages() {
				page, err := paginator.NextPage(ctx)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS API")
						return res, nil
					}
					return nil, err
				}
				for _, fw := range page.Firewalls {
					// DescribeFirewall to get full details
					detail, err := svc.DescribeFirewall(ctx, &networkfirewall.DescribeFirewallInput{
						FirewallArn: fw.FirewallArn,
					})
					if err != nil {
						if Is400AccessDeniedError(err) {
							continue
						}
						return nil, err
					}
					f := detail.Firewall
					subnetMappings := make([]any, 0, len(f.SubnetMappings))
					for _, sm := range f.SubnetMappings {
						d, err := convert.JsonToDict(sm)
						if err != nil {
							log.Warn().Err(err).Msg("failed to convert subnet mapping")
							continue
						}
						subnetMappings = append(subnetMappings, d)
					}
					var encConfig any
					if f.EncryptionConfiguration != nil {
						var encErr error
						encConfig, encErr = convert.JsonToDict(f.EncryptionConfiguration)
						if encErr != nil {
							log.Warn().Err(encErr).Msg("failed to convert encryption configuration")
						}
					}
					tags := nfTagsToMap(f.Tags)

					mqlFirewall, err := CreateResource(a.MqlRuntime, "aws.networkfirewall.firewall",
						map[string]*llx.RawData{
							"arn":                            llx.StringDataPtr(f.FirewallArn),
							"name":                           llx.StringDataPtr(f.FirewallName),
							"description":                    llx.StringDataPtr(f.Description),
							"region":                         llx.StringData(region),
							"deleteProtection":               llx.BoolData(f.DeleteProtection),
							"subnetChangeProtection":         llx.BoolData(f.SubnetChangeProtection),
							"firewallPolicyChangeProtection": llx.BoolData(f.FirewallPolicyChangeProtection),
							"firewallPolicyArn":              llx.StringDataPtr(f.FirewallPolicyArn),
							"subnetMappings":                 llx.ArrayData(subnetMappings, "dict"),
							"encryptionConfiguration":        llx.DictData(encConfig),
							"tags":                           llx.MapData(tags, "string"),
						})
					if err != nil {
						return nil, err
					}
					mqlFw := mqlFirewall.(*mqlAwsNetworkfirewallFirewall)
					mqlFw.cacheVpcId = f.VpcId
					res = append(res, mqlFirewall)
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

type mqlAwsNetworkfirewallFirewallInternal struct {
	cacheVpcId *string
}

func (a *mqlAwsNetworkfirewallFirewall) id() (string, error) {
	return a.Arn.Data, a.Arn.Error
}

func (a *mqlAwsNetworkfirewallFirewall) vpc() (*mqlAwsVpc, error) {
	if a.cacheVpcId == nil || *a.cacheVpcId == "" {
		a.Vpc.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	mqlVpc, err := NewResource(a.MqlRuntime, "aws.vpc",
		map[string]*llx.RawData{"id": llx.StringDataPtr(a.cacheVpcId)})
	if err != nil {
		return nil, err
	}
	return mqlVpc.(*mqlAwsVpc), nil
}

func (a *mqlAwsNetworkfirewallFirewall) policy() (*mqlAwsNetworkfirewallPolicy, error) {
	if a.FirewallPolicyArn.Error != nil {
		return nil, a.FirewallPolicyArn.Error
	}
	policyArn := a.FirewallPolicyArn.Data
	if policyArn == "" {
		a.Policy.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}

	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	if a.Region.Error != nil {
		return nil, a.Region.Error
	}
	region := a.Region.Data
	svc := conn.NetworkFirewall(region)
	ctx := context.Background()

	resp, err := svc.DescribeFirewallPolicy(ctx, &networkfirewall.DescribeFirewallPolicyInput{
		FirewallPolicyArn: &policyArn,
	})
	if err != nil {
		return nil, err
	}

	return networkfirewallPolicyToMql(a.MqlRuntime, resp.FirewallPolicyResponse, resp.FirewallPolicy, region)
}

func (a *mqlAwsNetworkfirewall) policies() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getPolicies(conn), 5)
	poolOfJobs.Run()

	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	for i := range poolOfJobs.Jobs {
		if poolOfJobs.Jobs[i].Result != nil {
			res = append(res, poolOfJobs.Jobs[i].Result.([]any)...)
		}
	}
	return res, nil
}

func (a *mqlAwsNetworkfirewall) getPolicies(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			svc := conn.NetworkFirewall(region)
			ctx := context.Background()

			res := []any{}
			paginator := networkfirewall.NewListFirewallPoliciesPaginator(svc, &networkfirewall.ListFirewallPoliciesInput{})
			for paginator.HasMorePages() {
				page, err := paginator.NextPage(ctx)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS API")
						return res, nil
					}
					return nil, err
				}
				for _, pm := range page.FirewallPolicies {
					detail, err := svc.DescribeFirewallPolicy(ctx, &networkfirewall.DescribeFirewallPolicyInput{
						FirewallPolicyArn: pm.Arn,
					})
					if err != nil {
						if Is400AccessDeniedError(err) {
							continue
						}
						return nil, err
					}
					mqlPolicy, err := networkfirewallPolicyToMql(a.MqlRuntime, detail.FirewallPolicyResponse, detail.FirewallPolicy, region)
					if err != nil {
						return nil, err
					}
					res = append(res, mqlPolicy)
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func networkfirewallPolicyToMql(runtime *plugin.Runtime, policyResp *nftypes.FirewallPolicyResponse, policy *nftypes.FirewallPolicy, region string) (*mqlAwsNetworkfirewallPolicy, error) {
	statelessRuleGroupRefs, err := convert.JsonToDictSlice(policy.StatelessRuleGroupReferences)
	if err != nil {
		log.Warn().Err(err).Msg("failed to convert stateless rule group references")
	}
	statefulRuleGroupRefs, err := convert.JsonToDictSlice(policy.StatefulRuleGroupReferences)
	if err != nil {
		log.Warn().Err(err).Msg("failed to convert stateful rule group references")
	}
	var statefulEngineOpts any
	if policy.StatefulEngineOptions != nil {
		var optErr error
		statefulEngineOpts, optErr = convert.JsonToDict(policy.StatefulEngineOptions)
		if optErr != nil {
			log.Warn().Err(optErr).Msg("failed to convert stateful engine options")
		}
	}
	tags := nfTagsToMap(policyResp.Tags)

	mqlPolicy, err := CreateResource(runtime, "aws.networkfirewall.policy",
		map[string]*llx.RawData{
			"arn":                             llx.StringDataPtr(policyResp.FirewallPolicyArn),
			"name":                            llx.StringDataPtr(policyResp.FirewallPolicyName),
			"description":                     llx.StringDataPtr(policyResp.Description),
			"region":                          llx.StringData(region),
			"statelessDefaultActions":         llx.ArrayData(llx.TArr2Raw(policy.StatelessDefaultActions), "string"),
			"statelessFragmentDefaultActions": llx.ArrayData(llx.TArr2Raw(policy.StatelessFragmentDefaultActions), "string"),
			"statelessRuleGroupReferences":    llx.ArrayData(statelessRuleGroupRefs, "dict"),
			"statefulDefaultActions":          llx.ArrayData(llx.TArr2Raw(policy.StatefulDefaultActions), "string"),
			"statefulRuleGroupReferences":     llx.ArrayData(statefulRuleGroupRefs, "dict"),
			"statefulEngineOptions":           llx.DictData(statefulEngineOpts),
			"tags":                            llx.MapData(tags, "string"),
		})
	if err != nil {
		return nil, err
	}
	return mqlPolicy.(*mqlAwsNetworkfirewallPolicy), nil
}

func (a *mqlAwsNetworkfirewallPolicy) id() (string, error) {
	return a.Arn.Data, a.Arn.Error
}

func (a *mqlAwsNetworkfirewall) ruleGroups() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getRuleGroups(conn), 5)
	poolOfJobs.Run()

	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	for i := range poolOfJobs.Jobs {
		if poolOfJobs.Jobs[i].Result != nil {
			res = append(res, poolOfJobs.Jobs[i].Result.([]any)...)
		}
	}
	return res, nil
}

func (a *mqlAwsNetworkfirewall) getRuleGroups(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			svc := conn.NetworkFirewall(region)
			ctx := context.Background()

			res := []any{}
			paginator := networkfirewall.NewListRuleGroupsPaginator(svc, &networkfirewall.ListRuleGroupsInput{})
			for paginator.HasMorePages() {
				page, err := paginator.NextPage(ctx)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS API")
						return res, nil
					}
					return nil, err
				}
				for _, rg := range page.RuleGroups {
					detail, err := svc.DescribeRuleGroup(ctx, &networkfirewall.DescribeRuleGroupInput{
						RuleGroupArn: rg.Arn,
					})
					if err != nil {
						if Is400AccessDeniedError(err) {
							continue
						}
						return nil, err
					}
					resp := detail.RuleGroupResponse
					var rules any
					if detail.RuleGroup != nil {
						var rulesErr error
						rules, rulesErr = convert.JsonToDict(detail.RuleGroup)
						if rulesErr != nil {
							log.Warn().Err(rulesErr).Msg("failed to convert rule group")
						}
					}
					tags := nfTagsToMap(resp.Tags)

					mqlRuleGroup, err := CreateResource(a.MqlRuntime, "aws.networkfirewall.rulegroup",
						map[string]*llx.RawData{
							"arn":         llx.StringDataPtr(resp.RuleGroupArn),
							"name":        llx.StringDataPtr(resp.RuleGroupName),
							"description": llx.StringDataPtr(resp.Description),
							"region":      llx.StringData(region),
							"capacity":    llx.IntDataDefault(resp.Capacity, 0),
							"type":        llx.StringData(string(resp.Type)),
							"rules":       llx.DictData(rules),
							"tags":        llx.MapData(tags, "string"),
						})
					if err != nil {
						return nil, err
					}
					res = append(res, mqlRuleGroup)
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func (a *mqlAwsNetworkfirewallRulegroup) id() (string, error) {
	return a.Arn.Data, a.Arn.Error
}

func nfTagsToMap(tags []nftypes.Tag) map[string]any {
	m := make(map[string]any, len(tags))
	for _, t := range tags {
		if t.Key != nil && t.Value != nil {
			m[*t.Key] = *t.Value
		}
	}
	return m
}
