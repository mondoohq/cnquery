// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go-v2/service/bedrockagentcorecontrol"
	bacctypes "github.com/aws/aws-sdk-go-v2/service/bedrockagentcorecontrol/types"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/mql/v13/providers/aws/connection"
)

// --- AgentCore namespace ---

func (a *mqlAwsBedrock) agentCore() (*mqlAwsBedrockAgentCore, error) {
	res, err := CreateResource(a.MqlRuntime, "aws.bedrock.agentCore", map[string]*llx.RawData{
		"__id": llx.StringData("aws.bedrock.agentCore"),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAwsBedrockAgentCore), nil
}

func (a *mqlAwsBedrockAgentCore) id() (string, error) {
	return "aws.bedrock.agentCore", nil
}

// collectJobs runs a region-fanned job pool and flattens the []any results.
func (a *mqlAwsBedrockAgentCore) collectJobs(jobs []*jobpool.Job) ([]any, error) {
	poolOfJobs := jobpool.CreatePool(jobs, 5)
	poolOfJobs.Run()
	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	res := []any{}
	for i := range poolOfJobs.Jobs {
		if poolOfJobs.Jobs[i].Result != nil {
			res = append(res, poolOfJobs.Jobs[i].Result.([]any)...)
		}
	}
	return res, nil
}

// agentCoreRegionTasks builds one job per region, invoking fn with a
// region-scoped context. Region/permission issues are logged and skipped so a
// single inaccessible region doesn't fail the whole query.
func (a *mqlAwsBedrockAgentCore) agentCoreRegionTasks(conn *connection.AwsConnection, fn func(ctx context.Context, region string) ([]any, error)) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}
	for _, region := range regions {
		region := region
		f := func() (jobpool.JobResult, error) {
			ctx := context.Background()
			res, err := fn(ctx, region)
			if err != nil {
				if Is400AccessDeniedError(err) {
					log.Warn().Str("region", region).Msg("error accessing region for AWS AgentCore API")
					return []any{}, nil
				}
				if IsServiceNotAvailableInRegionError(err) {
					log.Debug().Str("region", region).Msg("bedrock agentcore is not available in region")
					return []any{}, nil
				}
				return nil, err
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

// --- Gateways ---

func (a *mqlAwsBedrockAgentCore) gateways() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	return a.collectJobs(a.agentCoreRegionTasks(conn, func(ctx context.Context, region string) ([]any, error) {
		svc := conn.BedrockAgentCoreControl(region)
		res := []any{}
		paginator := bedrockagentcorecontrol.NewListGatewaysPaginator(svc, &bedrockagentcorecontrol.ListGatewaysInput{})
		for paginator.HasMorePages() {
			page, err := paginator.NextPage(ctx)
			if err != nil {
				return nil, err
			}
			for _, gw := range page.Items {
				gatewayId := convert.ToValue(gw.GatewayId)
				mqlGw, err := CreateResource(a.MqlRuntime, "aws.bedrock.agentCore.gateway", map[string]*llx.RawData{
					"__id":           llx.StringData(region + "/" + gatewayId),
					"name":           llx.StringDataPtr(gw.Name),
					"region":         llx.StringData(region),
					"status":         llx.StringData(string(gw.Status)),
					"description":    llx.StringDataPtr(gw.Description),
					"protocolType":   llx.StringData(string(gw.ProtocolType)),
					"authorizerType": llx.StringData(string(gw.AuthorizerType)),
					"createdAt":      llx.TimeDataPtr(gw.CreatedAt),
					"updatedAt":      llx.TimeDataPtr(gw.UpdatedAt),
				})
				if err != nil {
					return nil, err
				}
				mqlGwRes := mqlGw.(*mqlAwsBedrockAgentCoreGateway)
				mqlGwRes.cacheRegion = region
				mqlGwRes.cacheGatewayId = gatewayId
				res = append(res, mqlGwRes)
			}
		}
		return res, nil
	}))
}

type mqlAwsBedrockAgentCoreGatewayInternal struct {
	cacheRegion    string
	cacheGatewayId string
	fetchLock      sync.Mutex
	fetched        bool
	detail         *bedrockagentcorecontrol.GetGatewayOutput
}

func (a *mqlAwsBedrockAgentCoreGateway) id() (string, error) {
	return a.Region.Data + "/" + a.cacheGatewayId, nil
}

func (a *mqlAwsBedrockAgentCoreGateway) fetchDetail() (*bedrockagentcorecontrol.GetGatewayOutput, error) {
	if a.fetched {
		return a.detail, nil
	}
	a.fetchLock.Lock()
	defer a.fetchLock.Unlock()
	if a.fetched {
		return a.detail, nil
	}
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	svc := conn.BedrockAgentCoreControl(a.cacheRegion)
	ctx := context.Background()
	gatewayId := a.cacheGatewayId
	detail, err := svc.GetGateway(ctx, &bedrockagentcorecontrol.GetGatewayInput{GatewayIdentifier: &gatewayId})
	if err != nil {
		return nil, err
	}
	a.detail = detail
	a.fetched = true
	return a.detail, nil
}

func (a *mqlAwsBedrockAgentCoreGateway) arn() (string, error) {
	detail, err := a.fetchDetail()
	if err != nil {
		return "", err
	}
	if detail == nil {
		return "", nil
	}
	return convert.ToValue(detail.GatewayArn), nil
}

func (a *mqlAwsBedrockAgentCoreGateway) gatewayUrl() (string, error) {
	detail, err := a.fetchDetail()
	if err != nil {
		return "", err
	}
	if detail == nil {
		return "", nil
	}
	return convert.ToValue(detail.GatewayUrl), nil
}

func (a *mqlAwsBedrockAgentCoreGateway) roleArn() (string, error) {
	detail, err := a.fetchDetail()
	if err != nil {
		return "", err
	}
	if detail == nil {
		return "", nil
	}
	return convert.ToValue(detail.RoleArn), nil
}

func (a *mqlAwsBedrockAgentCoreGateway) iamRole() (*mqlAwsIamRole, error) {
	detail, err := a.fetchDetail()
	if err != nil {
		return nil, err
	}
	if detail == nil || detail.RoleArn == nil || *detail.RoleArn == "" {
		a.IamRole.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	res, err := NewResource(a.MqlRuntime, "aws.iam.role",
		map[string]*llx.RawData{"arn": llx.StringDataPtr(detail.RoleArn)})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAwsIamRole), nil
}

func (a *mqlAwsBedrockAgentCoreGateway) kmsKey() (*mqlAwsKmsKey, error) {
	detail, err := a.fetchDetail()
	if err != nil {
		return nil, err
	}
	if detail == nil || detail.KmsKeyArn == nil || *detail.KmsKeyArn == "" {
		a.KmsKey.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	res, err := NewResource(a.MqlRuntime, "aws.kms.key",
		map[string]*llx.RawData{"arn": llx.StringDataPtr(detail.KmsKeyArn)})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAwsKmsKey), nil
}

func (a *mqlAwsBedrockAgentCoreGateway) workloadIdentityArn() (string, error) {
	detail, err := a.fetchDetail()
	if err != nil {
		return "", err
	}
	if detail == nil || detail.WorkloadIdentityDetails == nil {
		return "", nil
	}
	return convert.ToValue(detail.WorkloadIdentityDetails.WorkloadIdentityArn), nil
}

func (a *mqlAwsBedrockAgentCoreGateway) workloadIdentity() (*mqlAwsBedrockAgentCoreWorkloadIdentity, error) {
	detail, err := a.fetchDetail()
	if err != nil {
		return nil, err
	}
	if detail == nil || detail.WorkloadIdentityDetails == nil || detail.WorkloadIdentityDetails.WorkloadIdentityArn == nil {
		a.WorkloadIdentity.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	res, err := NewResource(a.MqlRuntime, "aws.bedrock.agentCore.workloadIdentity",
		map[string]*llx.RawData{"arn": llx.StringDataPtr(detail.WorkloadIdentityDetails.WorkloadIdentityArn)})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAwsBedrockAgentCoreWorkloadIdentity), nil
}

func (a *mqlAwsBedrockAgentCoreGateway) targets() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	svc := conn.BedrockAgentCoreControl(a.cacheRegion)
	ctx := context.Background()
	gatewayId := a.cacheGatewayId
	region := a.Region.Data
	res := []any{}
	paginator := bedrockagentcorecontrol.NewListGatewayTargetsPaginator(svc, &bedrockagentcorecontrol.ListGatewayTargetsInput{
		GatewayIdentifier: &gatewayId,
	})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			if Is400AccessDeniedError(err) {
				return res, nil
			}
			return nil, err
		}
		for _, t := range page.Items {
			mqlTarget, err := CreateResource(a.MqlRuntime, "aws.bedrock.agentCore.gatewayTarget", map[string]*llx.RawData{
				"__id":        llx.StringData(region + "/" + gatewayId + "/target/" + convert.ToValue(t.TargetId)),
				"targetId":    llx.StringDataPtr(t.TargetId),
				"gatewayId":   llx.StringData(gatewayId),
				"name":        llx.StringDataPtr(t.Name),
				"region":      llx.StringData(region),
				"status":      llx.StringData(string(t.Status)),
				"description": llx.StringDataPtr(t.Description),
				"createdAt":   llx.TimeDataPtr(t.CreatedAt),
				"updatedAt":   llx.TimeDataPtr(t.UpdatedAt),
			})
			if err != nil {
				return nil, err
			}
			mqlTargetRes := mqlTarget.(*mqlAwsBedrockAgentCoreGatewayTarget)
			mqlTargetRes.cacheRegion = region
			res = append(res, mqlTargetRes)
		}
	}
	return res, nil
}

// --- Gateway targets ---

type mqlAwsBedrockAgentCoreGatewayTargetInternal struct {
	cacheRegion string
	fetchLock   sync.Mutex
	fetched     bool
	detail      *bedrockagentcorecontrol.GetGatewayTargetOutput
}

func (a *mqlAwsBedrockAgentCoreGatewayTarget) id() (string, error) {
	return a.Region.Data + "/" + a.GatewayId.Data + "/target/" + a.TargetId.Data, nil
}

func (a *mqlAwsBedrockAgentCoreGatewayTarget) fetchDetail() (*bedrockagentcorecontrol.GetGatewayTargetOutput, error) {
	if a.fetched {
		return a.detail, nil
	}
	a.fetchLock.Lock()
	defer a.fetchLock.Unlock()
	if a.fetched {
		return a.detail, nil
	}
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	svc := conn.BedrockAgentCoreControl(a.cacheRegion)
	ctx := context.Background()
	gatewayId := a.GatewayId.Data
	targetId := a.TargetId.Data
	detail, err := svc.GetGatewayTarget(ctx, &bedrockagentcorecontrol.GetGatewayTargetInput{
		GatewayIdentifier: &gatewayId,
		TargetId:          &targetId,
	})
	if err != nil {
		return nil, err
	}
	a.detail = detail
	a.fetched = true
	return a.detail, nil
}

func (a *mqlAwsBedrockAgentCoreGatewayTarget) targetConfiguration() (any, error) {
	detail, err := a.fetchDetail()
	if err != nil {
		return nil, err
	}
	if detail == nil {
		return nil, nil
	}
	return convert.JsonToDict(detail.TargetConfiguration)
}

func (a *mqlAwsBedrockAgentCoreGatewayTarget) credentialProviderConfigurations() ([]any, error) {
	detail, err := a.fetchDetail()
	if err != nil {
		return nil, err
	}
	if detail == nil {
		return nil, nil
	}
	return convert.JsonToDictSlice(detail.CredentialProviderConfigurations)
}

// --- Agent runtimes ---

func (a *mqlAwsBedrockAgentCore) runtimes() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	return a.collectJobs(a.agentCoreRegionTasks(conn, func(ctx context.Context, region string) ([]any, error) {
		svc := conn.BedrockAgentCoreControl(region)
		res := []any{}
		paginator := bedrockagentcorecontrol.NewListAgentRuntimesPaginator(svc, &bedrockagentcorecontrol.ListAgentRuntimesInput{})
		for paginator.HasMorePages() {
			page, err := paginator.NextPage(ctx)
			if err != nil {
				return nil, err
			}
			for _, rt := range page.AgentRuntimes {
				mqlRt, err := CreateResource(a.MqlRuntime, "aws.bedrock.agentCore.runtime", map[string]*llx.RawData{
					"__id":        llx.StringDataPtr(rt.AgentRuntimeArn),
					"id":          llx.StringDataPtr(rt.AgentRuntimeId),
					"arn":         llx.StringDataPtr(rt.AgentRuntimeArn),
					"name":        llx.StringDataPtr(rt.AgentRuntimeName),
					"region":      llx.StringData(region),
					"version":     llx.StringDataPtr(rt.AgentRuntimeVersion),
					"status":      llx.StringData(string(rt.Status)),
					"description": llx.StringDataPtr(rt.Description),
					"updatedAt":   llx.TimeDataPtr(rt.LastUpdatedAt),
				})
				if err != nil {
					return nil, err
				}
				mqlRtRes := mqlRt.(*mqlAwsBedrockAgentCoreRuntime)
				mqlRtRes.cacheRegion = region
				res = append(res, mqlRtRes)
			}
		}
		return res, nil
	}))
}

type mqlAwsBedrockAgentCoreRuntimeInternal struct {
	cacheRegion string
	fetchLock   sync.Mutex
	fetched     bool
	detail      *bedrockagentcorecontrol.GetAgentRuntimeOutput
}

func (a *mqlAwsBedrockAgentCoreRuntime) id() (string, error) {
	return a.Arn.Data, nil
}

func (a *mqlAwsBedrockAgentCoreRuntime) fetchDetail() (*bedrockagentcorecontrol.GetAgentRuntimeOutput, error) {
	if a.fetched {
		return a.detail, nil
	}
	a.fetchLock.Lock()
	defer a.fetchLock.Unlock()
	if a.fetched {
		return a.detail, nil
	}
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	svc := conn.BedrockAgentCoreControl(a.cacheRegion)
	ctx := context.Background()
	runtimeId := a.Id.Data
	detail, err := svc.GetAgentRuntime(ctx, &bedrockagentcorecontrol.GetAgentRuntimeInput{AgentRuntimeId: &runtimeId})
	if err != nil {
		return nil, err
	}
	a.detail = detail
	a.fetched = true
	return a.detail, nil
}

func (a *mqlAwsBedrockAgentCoreRuntime) roleArn() (string, error) {
	detail, err := a.fetchDetail()
	if err != nil {
		return "", err
	}
	if detail == nil {
		return "", nil
	}
	return convert.ToValue(detail.RoleArn), nil
}

func (a *mqlAwsBedrockAgentCoreRuntime) iamRole() (*mqlAwsIamRole, error) {
	detail, err := a.fetchDetail()
	if err != nil {
		return nil, err
	}
	if detail == nil || detail.RoleArn == nil || *detail.RoleArn == "" {
		a.IamRole.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	res, err := NewResource(a.MqlRuntime, "aws.iam.role",
		map[string]*llx.RawData{"arn": llx.StringDataPtr(detail.RoleArn)})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAwsIamRole), nil
}

func (a *mqlAwsBedrockAgentCoreRuntime) networkMode() (string, error) {
	detail, err := a.fetchDetail()
	if err != nil {
		return "", err
	}
	if detail == nil || detail.NetworkConfiguration == nil {
		return "", nil
	}
	return string(detail.NetworkConfiguration.NetworkMode), nil
}

func (a *mqlAwsBedrockAgentCoreRuntime) authorizerType() (string, error) {
	detail, err := a.fetchDetail()
	if err != nil {
		return "", err
	}
	if detail == nil || detail.AuthorizerConfiguration == nil {
		return "", nil
	}
	// AuthorizerConfiguration is a union; a custom JWT authorizer is the only
	// inbound authorizer type currently expressible on a runtime.
	if _, ok := detail.AuthorizerConfiguration.(*bacctypes.AuthorizerConfigurationMemberCustomJWTAuthorizer); ok {
		return "CUSTOM_JWT", nil
	}
	return "", nil
}

func (a *mqlAwsBedrockAgentCoreRuntime) environmentVariables() (map[string]any, error) {
	detail, err := a.fetchDetail()
	if err != nil {
		return nil, err
	}
	if detail == nil {
		return nil, nil
	}
	return convert.MapToInterfaceMap(detail.EnvironmentVariables), nil
}

func (a *mqlAwsBedrockAgentCoreRuntime) endpoints() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	svc := conn.BedrockAgentCoreControl(a.cacheRegion)
	ctx := context.Background()
	runtimeId := a.Id.Data
	region := a.Region.Data
	res := []any{}
	paginator := bedrockagentcorecontrol.NewListAgentRuntimeEndpointsPaginator(svc, &bedrockagentcorecontrol.ListAgentRuntimeEndpointsInput{
		AgentRuntimeId: &runtimeId,
	})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			if Is400AccessDeniedError(err) {
				return res, nil
			}
			return nil, err
		}
		for _, ep := range page.RuntimeEndpoints {
			mqlEp, err := CreateResource(a.MqlRuntime, "aws.bedrock.agentCore.runtimeEndpoint", map[string]*llx.RawData{
				"__id":            llx.StringDataPtr(ep.AgentRuntimeEndpointArn),
				"id":              llx.StringDataPtr(ep.Id),
				"arn":             llx.StringDataPtr(ep.AgentRuntimeEndpointArn),
				"agentRuntimeArn": llx.StringDataPtr(ep.AgentRuntimeArn),
				"name":            llx.StringDataPtr(ep.Name),
				"region":          llx.StringData(region),
				"status":          llx.StringData(string(ep.Status)),
				"description":     llx.StringDataPtr(ep.Description),
				"liveVersion":     llx.StringDataPtr(ep.LiveVersion),
				"targetVersion":   llx.StringDataPtr(ep.TargetVersion),
				"createdAt":       llx.TimeDataPtr(ep.CreatedAt),
				"updatedAt":       llx.TimeDataPtr(ep.LastUpdatedAt),
			})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlEp)
		}
	}
	return res, nil
}

func (a *mqlAwsBedrockAgentCoreRuntimeEndpoint) id() (string, error) {
	return a.Arn.Data, nil
}

// --- Memory ---

func (a *mqlAwsBedrockAgentCore) memories() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	return a.collectJobs(a.agentCoreRegionTasks(conn, func(ctx context.Context, region string) ([]any, error) {
		svc := conn.BedrockAgentCoreControl(region)
		res := []any{}
		paginator := bedrockagentcorecontrol.NewListMemoriesPaginator(svc, &bedrockagentcorecontrol.ListMemoriesInput{})
		for paginator.HasMorePages() {
			page, err := paginator.NextPage(ctx)
			if err != nil {
				return nil, err
			}
			for _, m := range page.Memories {
				mqlMem, err := CreateResource(a.MqlRuntime, "aws.bedrock.agentCore.memory", map[string]*llx.RawData{
					"__id":      llx.StringDataPtr(m.Arn),
					"id":        llx.StringDataPtr(m.Id),
					"arn":       llx.StringDataPtr(m.Arn),
					"region":    llx.StringData(region),
					"status":    llx.StringData(string(m.Status)),
					"createdAt": llx.TimeDataPtr(m.CreatedAt),
					"updatedAt": llx.TimeDataPtr(m.UpdatedAt),
				})
				if err != nil {
					return nil, err
				}
				res = append(res, mqlMem)
			}
		}
		return res, nil
	}))
}

func (a *mqlAwsBedrockAgentCoreMemory) id() (string, error) {
	return a.Arn.Data, nil
}

// --- Browsers ---

func (a *mqlAwsBedrockAgentCore) browsers() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	return a.collectJobs(a.agentCoreRegionTasks(conn, func(ctx context.Context, region string) ([]any, error) {
		svc := conn.BedrockAgentCoreControl(region)
		res := []any{}
		paginator := bedrockagentcorecontrol.NewListBrowsersPaginator(svc, &bedrockagentcorecontrol.ListBrowsersInput{})
		for paginator.HasMorePages() {
			page, err := paginator.NextPage(ctx)
			if err != nil {
				return nil, err
			}
			for _, b := range page.BrowserSummaries {
				mqlBrowser, err := CreateResource(a.MqlRuntime, "aws.bedrock.agentCore.browser", map[string]*llx.RawData{
					"__id":        llx.StringDataPtr(b.BrowserArn),
					"id":          llx.StringDataPtr(b.BrowserId),
					"arn":         llx.StringDataPtr(b.BrowserArn),
					"name":        llx.StringDataPtr(b.Name),
					"region":      llx.StringData(region),
					"status":      llx.StringData(string(b.Status)),
					"description": llx.StringDataPtr(b.Description),
					"createdAt":   llx.TimeDataPtr(b.CreatedAt),
					"updatedAt":   llx.TimeDataPtr(b.LastUpdatedAt),
				})
				if err != nil {
					return nil, err
				}
				res = append(res, mqlBrowser)
			}
		}
		return res, nil
	}))
}

func (a *mqlAwsBedrockAgentCoreBrowser) id() (string, error) {
	return a.Arn.Data, nil
}

// --- Code interpreters ---

func (a *mqlAwsBedrockAgentCore) codeInterpreters() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	return a.collectJobs(a.agentCoreRegionTasks(conn, func(ctx context.Context, region string) ([]any, error) {
		svc := conn.BedrockAgentCoreControl(region)
		res := []any{}
		paginator := bedrockagentcorecontrol.NewListCodeInterpretersPaginator(svc, &bedrockagentcorecontrol.ListCodeInterpretersInput{})
		for paginator.HasMorePages() {
			page, err := paginator.NextPage(ctx)
			if err != nil {
				return nil, err
			}
			for _, ci := range page.CodeInterpreterSummaries {
				mqlCi, err := CreateResource(a.MqlRuntime, "aws.bedrock.agentCore.codeInterpreter", map[string]*llx.RawData{
					"__id":        llx.StringDataPtr(ci.CodeInterpreterArn),
					"id":          llx.StringDataPtr(ci.CodeInterpreterId),
					"arn":         llx.StringDataPtr(ci.CodeInterpreterArn),
					"name":        llx.StringDataPtr(ci.Name),
					"region":      llx.StringData(region),
					"status":      llx.StringData(string(ci.Status)),
					"description": llx.StringDataPtr(ci.Description),
					"createdAt":   llx.TimeDataPtr(ci.CreatedAt),
					"updatedAt":   llx.TimeDataPtr(ci.LastUpdatedAt),
				})
				if err != nil {
					return nil, err
				}
				res = append(res, mqlCi)
			}
		}
		return res, nil
	}))
}

func (a *mqlAwsBedrockAgentCoreCodeInterpreter) id() (string, error) {
	return a.Arn.Data, nil
}

// --- Identity: OAuth2 credential providers ---

func (a *mqlAwsBedrockAgentCore) oauth2CredentialProviders() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	return a.collectJobs(a.agentCoreRegionTasks(conn, func(ctx context.Context, region string) ([]any, error) {
		svc := conn.BedrockAgentCoreControl(region)
		res := []any{}
		var nextToken *string
		for {
			page, err := svc.ListOauth2CredentialProviders(ctx, &bedrockagentcorecontrol.ListOauth2CredentialProvidersInput{
				NextToken: nextToken,
			})
			if err != nil {
				return nil, err
			}
			for _, p := range page.CredentialProviders {
				mqlProvider, err := CreateResource(a.MqlRuntime, "aws.bedrock.agentCore.oauth2CredentialProvider", map[string]*llx.RawData{
					"__id":      llx.StringDataPtr(p.CredentialProviderArn),
					"arn":       llx.StringDataPtr(p.CredentialProviderArn),
					"name":      llx.StringDataPtr(p.Name),
					"region":    llx.StringData(region),
					"vendor":    llx.StringData(string(p.CredentialProviderVendor)),
					"createdAt": llx.TimeDataPtr(p.CreatedTime),
					"updatedAt": llx.TimeDataPtr(p.LastUpdatedTime),
				})
				if err != nil {
					return nil, err
				}
				res = append(res, mqlProvider)
			}
			if page.NextToken == nil {
				break
			}
			nextToken = page.NextToken
		}
		return res, nil
	}))
}

func (a *mqlAwsBedrockAgentCoreOauth2CredentialProvider) id() (string, error) {
	return a.Arn.Data, nil
}

// --- Identity: API-key credential providers ---

func (a *mqlAwsBedrockAgentCore) apiKeyCredentialProviders() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	return a.collectJobs(a.agentCoreRegionTasks(conn, func(ctx context.Context, region string) ([]any, error) {
		svc := conn.BedrockAgentCoreControl(region)
		res := []any{}
		paginator := bedrockagentcorecontrol.NewListApiKeyCredentialProvidersPaginator(svc, &bedrockagentcorecontrol.ListApiKeyCredentialProvidersInput{})
		for paginator.HasMorePages() {
			page, err := paginator.NextPage(ctx)
			if err != nil {
				return nil, err
			}
			for _, p := range page.CredentialProviders {
				mqlProvider, err := CreateResource(a.MqlRuntime, "aws.bedrock.agentCore.apiKeyCredentialProvider", map[string]*llx.RawData{
					"__id":      llx.StringDataPtr(p.CredentialProviderArn),
					"arn":       llx.StringDataPtr(p.CredentialProviderArn),
					"name":      llx.StringDataPtr(p.Name),
					"region":    llx.StringData(region),
					"createdAt": llx.TimeDataPtr(p.CreatedTime),
					"updatedAt": llx.TimeDataPtr(p.LastUpdatedTime),
				})
				if err != nil {
					return nil, err
				}
				res = append(res, mqlProvider)
			}
		}
		return res, nil
	}))
}

func (a *mqlAwsBedrockAgentCoreApiKeyCredentialProvider) id() (string, error) {
	return a.Arn.Data, nil
}

// --- Identity: workload identities ---

func (a *mqlAwsBedrockAgentCore) workloadIdentities() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	return a.collectJobs(a.agentCoreRegionTasks(conn, func(ctx context.Context, region string) ([]any, error) {
		svc := conn.BedrockAgentCoreControl(region)
		res := []any{}
		paginator := bedrockagentcorecontrol.NewListWorkloadIdentitiesPaginator(svc, &bedrockagentcorecontrol.ListWorkloadIdentitiesInput{})
		for paginator.HasMorePages() {
			page, err := paginator.NextPage(ctx)
			if err != nil {
				return nil, err
			}
			for _, w := range page.WorkloadIdentities {
				mqlIdentity, err := CreateResource(a.MqlRuntime, "aws.bedrock.agentCore.workloadIdentity", map[string]*llx.RawData{
					"__id":   llx.StringDataPtr(w.WorkloadIdentityArn),
					"arn":    llx.StringDataPtr(w.WorkloadIdentityArn),
					"name":   llx.StringDataPtr(w.Name),
					"region": llx.StringData(region),
				})
				if err != nil {
					return nil, err
				}
				res = append(res, mqlIdentity)
			}
		}
		return res, nil
	}))
}

// initAwsBedrockAgentCoreWorkloadIdentity resolves a workload identity from its
// ARN. The region and name are both encoded in the ARN
// (arn:aws:bedrock-agentcore:<region>:<account>:workload-identity-directory/default/workload-identity/<name>),
// so a single ARN is enough to populate the resource without an API call.
func initAwsBedrockAgentCoreWorkloadIdentity(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 1 {
		return args, nil, nil
	}
	if args["arn"] == nil {
		return nil, nil, errors.New("arn required to resolve aws.bedrock.agentCore.workloadIdentity")
	}
	arnVal := args["arn"].Value.(string)

	if args["region"] == nil {
		if region, err := GetRegionFromArn(arnVal); err == nil && region != "" {
			args["region"] = llx.StringData(region)
		}
	}
	if args["name"] == nil {
		if idx := strings.LastIndex(arnVal, "/"); idx >= 0 && idx < len(arnVal)-1 {
			args["name"] = llx.StringData(arnVal[idx+1:])
		}
	}
	return args, nil, nil
}

func (a *mqlAwsBedrockAgentCoreWorkloadIdentity) id() (string, error) {
	return a.Arn.Data, nil
}
