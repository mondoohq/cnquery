// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"fmt"

	binaryauthorization "cloud.google.com/go/binaryauthorization/apiv1"
	"cloud.google.com/go/binaryauthorization/apiv1/binaryauthorizationpb"
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers/gcp/connection"
	"go.mondoo.com/cnquery/v11/types"
	"google.golang.org/api/option"
)

func (g *mqlGcpProject) binaryAuthorization() (*mqlGcpProjectBinaryAuthorizationControl, error) {
	if g.Id.Error != nil {
		return nil, g.Id.Error
	}
	projectId := g.Id.Data

	serviceEnabled, err := g.isServiceEnabled(service_binaryauthorization)
	if err != nil {
		return nil, err
	}
	if !serviceEnabled {
		g.BinaryAuthorization.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)
	credentials, err := conn.Credentials(binaryauthorization.DefaultAuthScopes()...)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	c, err := binaryauthorization.NewSystemPolicyClient(ctx, option.WithCredentials(credentials), option.WithQuotaProject(projectId))
	if err != nil {
		return nil, err
	}

	defer c.Close()

	name := fmt.Sprintf("projects/%s/policy", projectId)
	resp, err := c.GetSystemPolicy(ctx, &binaryauthorizationpb.GetSystemPolicyRequest{
		Name: name,
	})
	if err != nil {
		return nil, err
	}

	var admissionWhitelistPatterns []interface{}
	for _, pattern := range resp.GetAdmissionWhitelistPatterns() {
		admissionWhitelistPatterns = append(admissionWhitelistPatterns, pattern.GetNamePattern())
	}

	clusterAdmissionRules, err := g.toMqlBinaryAuthzAdmissionRules(resp.GetClusterAdmissionRules(), name, "clusterAdmissionRules")
	if err != nil {
		return nil, err
	}

	kubernetesNamespaceAdmissionRules, err := g.toMqlBinaryAuthzAdmissionRules(resp.GetKubernetesNamespaceAdmissionRules(), name, "kubernetesNamespaceAdmissionRules")
	if err != nil {
		return nil, err
	}

	kubernetesServiceAccountAdmissionRules, err := g.toMqlBinaryAuthzAdmissionRules(resp.GetKubernetesServiceAccountAdmissionRules(), name, "kubernetesServiceAccountAdmissionRules")
	if err != nil {
		return nil, err
	}

	istioServiceIdentityAdmissionRules, err := g.toMqlBinaryAuthzAdmissionRules(resp.GetIstioServiceIdentityAdmissionRules(), name, "istioServiceIdentityAdmissionRules")
	if err != nil {
		return nil, err
	}

	defaultAdmissionRule, err := g.toMqlBinaryAuthzAdmissionRule(resp.GetDefaultAdmissionRule(), fmt.Sprintf("%s/defaultAdmissionRule", name))
	if err != nil {
		return nil, err
	}

	updateTime := resp.GetUpdateTime().AsTime()

	policy, err := CreateResource(g.MqlRuntime, "gcp.project.binaryAuthorizationControl.policy", map[string]*llx.RawData{
		"__id":                                   llx.StringData(name),
		"name":                                   llx.StringData(name),
		"admissionWhitelistPatterns":             llx.ArrayData(admissionWhitelistPatterns, types.String),
		"globalPolicyEvaluationMode":             llx.StringData(resp.GetGlobalPolicyEvaluationMode().String()),
		"clusterAdmissionRules":                  llx.MapData(clusterAdmissionRules, types.Resource("gcp.project.binaryAuthorizationControl.admissionRule")),
		"kubernetesNamespaceAdmissionRules":      llx.MapData(kubernetesNamespaceAdmissionRules, types.Resource("gcp.project.binaryAuthorizationControl.admissionRule")),
		"kubernetesServiceAccountAdmissionRules": llx.MapData(kubernetesServiceAccountAdmissionRules, types.Resource("gcp.project.binaryAuthorizationControl.admissionRule")),
		"istioServiceIdentityAdmissionRules":     llx.MapData(istioServiceIdentityAdmissionRules, types.Resource("gcp.project.binaryAuthorizationControl.admissionRule")),
		"defaultAdmissionRule":                   llx.ResourceData(defaultAdmissionRule, "gcp.project.binaryAuthorizationControl.admissionRule"),
		"updated":                                llx.TimeData(updateTime),
	})
	if err != nil {
		return nil, err
	}

	bauthz, err := CreateResource(g.MqlRuntime, "gcp.project.binaryAuthorizationControl", map[string]*llx.RawData{
		"__id":   llx.StringData(fmt.Sprintf("projects/%s/binaryAuthorizationControl", projectId)),
		"policy": llx.ResourceData(policy, "gcp.project.binaryAuthorizationControl.policy"),
	})
	if err != nil {
		return nil, err
	}

	return bauthz.(*mqlGcpProjectBinaryAuthorizationControl), nil
}

func (g *mqlGcpProject) toMqlBinaryAuthzAdmissionRules(rules map[string]*binaryauthorizationpb.AdmissionRule, policyName string, ruleSetName string) (map[string]interface{}, error) {
	mqlRules := make(map[string]interface{})
	for ruleName, rule := range rules {
		mqlId := fmt.Sprintf("%s/%s/%s", policyName, ruleSetName, ruleName)
		mqlRule, err := g.toMqlBinaryAuthzAdmissionRule(rule, mqlId)
		if err != nil {
			return nil, err
		}
		mqlRules[ruleName] = mqlRule
	}
	return mqlRules, nil
}

func (g *mqlGcpProject) toMqlBinaryAuthzAdmissionRule(rule *binaryauthorizationpb.AdmissionRule, mqlId string) (plugin.Resource, error) {
	var requiresAttestationsBy []interface{}
	for _, attestation := range rule.GetRequireAttestationsBy() {
		requiresAttestationsBy = append(requiresAttestationsBy, attestation)
	}
	return CreateResource(g.MqlRuntime, "gcp.project.binaryAuthorizationControl.admissionRule", map[string]*llx.RawData{
		"__id":                  llx.StringData(mqlId),
		"evaluationMode":        llx.StringData(rule.GetEvaluationMode().String()),
		"requireAttestationsBy": llx.ArrayData(requiresAttestationsBy, types.String),
	})
}
