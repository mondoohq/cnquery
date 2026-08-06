// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws/arn"
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

// securityGroupCount reports how many security groups are attached, and is used
// to tell "no security group opens this" apart from "security groups do not
// constrain this resource at all".
func securityGroupCount(sgs *plugin.TValue[[]any]) int {
	if sgs == nil {
		return 0
	}
	return len(sgs.Data)
}

// buildNetworkExposure creates a shared aws.network.exposure from a resource's
// public-access toggle and its attached security groups.
//
// internetReachable requires the resource to be publicly accessible, and -- when
// security groups apply to it -- for one of them to open it. When no security
// group is attached, security groups are not what stands between the resource
// and the internet, so publiclyAccessible alone decides.
//
// That distinction matters most for load balancers: Network Load Balancers only
// support security groups when one was attached at creation (they cannot be
// added later) and Gateway Load Balancers never support them, so most NLBs
// report an empty list. Requiring sgAllows there made every internet-facing NLB
// -- including one with a world-open listener -- report internetReachable:false.
// It is also the safe direction when the group list could not be enumerated.
func buildNetworkExposure(runtime *plugin.Runtime, id string, publiclyAccessible bool, sgs *plugin.TValue[[]any]) (*mqlAwsNetworkExposure, error) {
	openRules, err := openIngressRulesFromSecurityGroups(sgs)
	if err != nil {
		return nil, err
	}
	sgAllows := len(openRules) > 0
	sgsApply := securityGroupCount(sgs) > 0

	internetReachable := publiclyAccessible
	if sgsApply {
		internetReachable = publiclyAccessible && sgAllows
	}

	res, err := CreateResource(runtime, "aws.network.exposure", map[string]*llx.RawData{
		"__id":                       llx.StringData(id),
		"internetReachable":          llx.BoolData(internetReachable),
		"publiclyAccessible":         llx.BoolData(publiclyAccessible),
		"securityGroupAllowsIngress": llx.BoolData(sgAllows),
		"openIngressRules":           llx.ArrayData(openRules, types.Resource("aws.ec2.securitygroup.ippermission")),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAwsNetworkExposure), nil
}

// buildNetworkExposureFromGroups is buildNetworkExposure for callers holding
// already-resolved security groups rather than a lazily-fetched field.
func buildNetworkExposureFromGroups(runtime *plugin.Runtime, id string, publiclyAccessible bool, sgs []any) (*mqlAwsNetworkExposure, error) {
	return buildNetworkExposure(runtime, id, publiclyAccessible, &plugin.TValue[[]any]{
		Data:  sgs,
		State: plugin.StateIsSet,
	})
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

// exposure reports the internet exposure of the cluster's Kubernetes API server
// endpoint. publicAccessCidrs can narrow a public endpoint to specific source
// ranges; it is left on the cluster rather than folded in here, so a caller sees
// both the verdict and the allow-list.
func (a *mqlAwsEksCluster) exposure() (*mqlAwsNetworkExposure, error) {
	arn := a.GetArn()
	if arn.Error != nil {
		return nil, arn.Error
	}
	publicAccess := a.GetEndpointPublicAccess()
	if publicAccess.Error != nil {
		return nil, publicAccess.Error
	}
	return buildNetworkExposure(a.MqlRuntime, arn.Data+"/exposure", publicAccess.Data, a.GetClusterSecurityGroups())
}

// exposure reports the internet exposure of tasks the service runs.
//
// Only awsvpc networking carries a public-IP assignment and task-level security
// groups. A service using host or bridge networking has neither, and its
// reachability is decided by the container instance, so exposure is null rather
// than a verdict about the wrong thing.
func (a *mqlAwsEcsService) exposure() (*mqlAwsNetworkExposure, error) {
	arn := a.GetArn()
	if arn.Error != nil {
		return nil, arn.Error
	}

	netConfig := a.GetNetworkConfiguration()
	if netConfig.Error != nil {
		return nil, netConfig.Error
	}
	if netConfig.Data == nil {
		a.Exposure.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	vpcConfig := netConfig.Data.GetAwsVpcConfiguration()
	if vpcConfig.Error != nil {
		return nil, vpcConfig.Error
	}
	if vpcConfig.Data == nil {
		a.Exposure.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}

	assignPublicIp := vpcConfig.Data.GetAssignPublicIp()
	if assignPublicIp.Error != nil {
		return nil, assignPublicIp.Error
	}
	sgIds := vpcConfig.Data.GetSecurityGroups()
	if sgIds.Error != nil {
		return nil, sgIds.Error
	}

	sgs, err := resolveSecurityGroupsByIdInArnScope(a.MqlRuntime, arn.Data, sgIds.Data)
	if err != nil {
		return nil, err
	}
	return buildNetworkExposureFromGroups(a.MqlRuntime, arn.Data+"/exposure",
		strings.EqualFold(assignPublicIp.Data, "ENABLED"), sgs)
}

// resolveSecurityGroupsByIdInArnScope turns bare security group IDs into typed
// resources, taking the region and account from a sibling ARN. Services that
// store group IDs rather than ARNs have no other way to name the scope.
func resolveSecurityGroupsByIdInArnScope(runtime *plugin.Runtime, scopeArn string, sgIds []any) ([]any, error) {
	parsed, err := arn.Parse(scopeArn)
	if err != nil {
		return nil, err
	}
	handler := securityGroupIdHandler{}
	arns := make([]string, 0, len(sgIds))
	for _, raw := range sgIds {
		sgId, ok := raw.(string)
		if !ok || sgId == "" {
			continue
		}
		arns = append(arns, fmt.Sprintf(securityGroupArnPattern, parsed.Region, parsed.AccountID, sgId))
	}
	handler.setSecurityGroupArns(arns)
	return handler.newSecurityGroupResources(runtime)
}

// exposure reports the internet exposure of the App Runner service endpoint.
// App Runner does not gate inbound traffic with security groups, so the
// public-accessibility setting alone decides.
func (a *mqlAwsApprunnerService) exposure() (*mqlAwsNetworkExposure, error) {
	arn := a.GetArn()
	if arn.Error != nil {
		return nil, arn.Error
	}
	publiclyAccessible := a.GetIsPubliclyAccessible()
	if publiclyAccessible.Error != nil {
		return nil, publiclyAccessible.Error
	}
	return buildNetworkExposure(a.MqlRuntime, arn.Data+"/exposure", publiclyAccessible.Data, nil)
}

// exposure reports the internet exposure of the notebook instance. Both inputs
// live on the details sub-resource, which is a separate API call, so this is the
// one exposure accessor that can cost a fetch.
func (a *mqlAwsSagemakerNotebookinstance) exposure() (*mqlAwsNetworkExposure, error) {
	arn := a.GetArn()
	if arn.Error != nil {
		return nil, arn.Error
	}
	details := a.GetDetails()
	if details.Error != nil {
		return nil, details.Error
	}
	if details.Data == nil {
		a.Exposure.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	directInternetAccess := details.Data.GetDirectInternetAccess()
	if directInternetAccess.Error != nil {
		return nil, directInternetAccess.Error
	}
	return buildNetworkExposure(a.MqlRuntime, arn.Data+"/exposure",
		directInternetAccess.Data, details.Data.GetSecurityGroups())
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
