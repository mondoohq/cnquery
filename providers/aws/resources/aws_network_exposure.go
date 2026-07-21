// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/types"
)

// openIngressRulesFromSecurityGroups returns the ingress rules across the given
// security groups that permit inbound traffic from the internet.
func openIngressRulesFromSecurityGroups(sgs *plugin.TValue[[]any]) ([]any, error) {
	if sgs == nil {
		return []any{}, nil
	}
	if sgs.Error != nil {
		return nil, sgs.Error
	}
	rules := []any{}
	for _, s := range sgs.Data {
		sg, ok := s.(*mqlAwsEc2Securitygroup)
		if !ok {
			continue
		}
		perms := sg.GetIpPermissions()
		if perms.Error != nil {
			return nil, perms.Error
		}
		for _, p := range perms.Data {
			perm, ok := p.(*mqlAwsEc2SecuritygroupIppermission)
			if !ok {
				continue
			}
			public := perm.GetIncludesPublicSource()
			if public.Error != nil {
				return nil, public.Error
			}
			if public.Data {
				rules = append(rules, perm)
			}
		}
	}
	return rules, nil
}

// buildNetworkExposure creates a shared aws.network.exposure from a resource's
// public-access toggle and its attached security groups. internetReachable is
// true only when the resource is publicly accessible and a security group opens
// it.
func buildNetworkExposure(runtime *plugin.Runtime, id string, publiclyAccessible bool, sgs *plugin.TValue[[]any]) (*mqlAwsNetworkExposure, error) {
	openRules, err := openIngressRulesFromSecurityGroups(sgs)
	if err != nil {
		return nil, err
	}
	sgAllows := len(openRules) > 0

	res, err := CreateResource(runtime, "aws.network.exposure", map[string]*llx.RawData{
		"__id":                       llx.StringData(id),
		"internetReachable":          llx.BoolData(publiclyAccessible && sgAllows),
		"publiclyAccessible":         llx.BoolData(publiclyAccessible),
		"securityGroupAllowsIngress": llx.BoolData(sgAllows),
		"openIngressRules":           llx.ArrayData(openRules, types.Resource("aws.ec2.securitygroup.ippermission")),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAwsNetworkExposure), nil
}

func (a *mqlAwsRdsDbinstance) exposure() (*mqlAwsNetworkExposure, error) {
	arn := a.GetArn()
	if arn.Error != nil {
		return nil, arn.Error
	}
	publiclyAccessible := a.GetPubliclyAccessible()
	if publiclyAccessible.Error != nil {
		return nil, publiclyAccessible.Error
	}
	return buildNetworkExposure(a.MqlRuntime, arn.Data+"/exposure", publiclyAccessible.Data, a.GetSecurityGroups())
}

func (a *mqlAwsDocumentdbInstance) exposure() (*mqlAwsNetworkExposure, error) {
	arn := a.GetArn()
	if arn.Error != nil {
		return nil, arn.Error
	}
	publiclyAccessible := a.GetPubliclyAccessible()
	if publiclyAccessible.Error != nil {
		return nil, publiclyAccessible.Error
	}
	return buildNetworkExposure(a.MqlRuntime, arn.Data+"/exposure", publiclyAccessible.Data, a.GetSecurityGroups())
}

func (a *mqlAwsElbLoadbalancer) exposure() (*mqlAwsNetworkExposure, error) {
	arn := a.GetArn()
	if arn.Error != nil {
		return nil, arn.Error
	}
	scheme := a.GetScheme()
	if scheme.Error != nil {
		return nil, scheme.Error
	}
	return buildNetworkExposure(a.MqlRuntime, arn.Data+"/exposure", scheme.Data == "internet-facing", a.GetSecurityGroups())
}

func (a *mqlAwsRdsDbcluster) exposure() (*mqlAwsNetworkExposure, error) {
	arn := a.GetArn()
	if arn.Error != nil {
		return nil, arn.Error
	}
	publiclyAccessible := a.GetPubliclyAccessible()
	if publiclyAccessible.Error != nil {
		return nil, publiclyAccessible.Error
	}
	return buildNetworkExposure(a.MqlRuntime, arn.Data+"/exposure", publiclyAccessible.Data, a.GetSecurityGroups())
}

func (a *mqlAwsRedshiftCluster) exposure() (*mqlAwsNetworkExposure, error) {
	arn := a.GetArn()
	if arn.Error != nil {
		return nil, arn.Error
	}
	publiclyAccessible := a.GetPubliclyAccessible()
	if publiclyAccessible.Error != nil {
		return nil, publiclyAccessible.Error
	}
	return buildNetworkExposure(a.MqlRuntime, arn.Data+"/exposure", publiclyAccessible.Data, a.GetSecurityGroups())
}

func (a *mqlAwsMqBroker) exposure() (*mqlAwsNetworkExposure, error) {
	arn := a.GetArn()
	if arn.Error != nil {
		return nil, arn.Error
	}
	publiclyAccessible := a.GetPubliclyAccessible()
	if publiclyAccessible.Error != nil {
		return nil, publiclyAccessible.Error
	}
	return buildNetworkExposure(a.MqlRuntime, arn.Data+"/exposure", publiclyAccessible.Data, a.GetSecurityGroups())
}

func (a *mqlAwsDmsReplicationInstance) exposure() (*mqlAwsNetworkExposure, error) {
	arn := a.GetArn()
	if arn.Error != nil {
		return nil, arn.Error
	}
	publiclyAccessible := a.GetPubliclyAccessible()
	if publiclyAccessible.Error != nil {
		return nil, publiclyAccessible.Error
	}
	return buildNetworkExposure(a.MqlRuntime, arn.Data+"/exposure", publiclyAccessible.Data, a.GetSecurityGroups())
}

func (a *mqlAwsMskCluster) exposure() (*mqlAwsNetworkExposure, error) {
	arn := a.GetArn()
	if arn.Error != nil {
		return nil, arn.Error
	}
	publicAccess := a.GetPublicAccess()
	if publicAccess.Error != nil {
		return nil, publicAccess.Error
	}
	return buildNetworkExposure(a.MqlRuntime, arn.Data+"/exposure", publicAccess.Data, a.GetSecurityGroups())
}

// buildVpcOnlyExposure builds a network exposure for a service that has no
// public endpoint (it is only reachable inside its VPC). publiclyAccessible and
// internetReachable are therefore always false; the value is in surfacing any
// attached security group that opens the resource to a public source.
func buildVpcOnlyExposure(a interface {
	GetArn() *plugin.TValue[string]
	GetSecurityGroups() *plugin.TValue[[]any]
}, runtime *plugin.Runtime) (*mqlAwsNetworkExposure, error) {
	arn := a.GetArn()
	if arn.Error != nil {
		return nil, arn.Error
	}
	return buildNetworkExposure(runtime, arn.Data+"/exposure", false, a.GetSecurityGroups())
}

func (a *mqlAwsDocumentdbCluster) exposure() (*mqlAwsNetworkExposure, error) {
	return buildVpcOnlyExposure(a, a.MqlRuntime)
}

func (a *mqlAwsElasticacheCluster) exposure() (*mqlAwsNetworkExposure, error) {
	return buildVpcOnlyExposure(a, a.MqlRuntime)
}

func (a *mqlAwsElasticacheServerlessCache) exposure() (*mqlAwsNetworkExposure, error) {
	return buildVpcOnlyExposure(a, a.MqlRuntime)
}

func (a *mqlAwsMemorydbCluster) exposure() (*mqlAwsNetworkExposure, error) {
	return buildVpcOnlyExposure(a, a.MqlRuntime)
}

func (a *mqlAwsNeptuneCluster) exposure() (*mqlAwsNetworkExposure, error) {
	return buildVpcOnlyExposure(a, a.MqlRuntime)
}
