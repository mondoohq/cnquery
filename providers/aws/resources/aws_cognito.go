// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentity"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	cognitoidentityprovidertypes "github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/mql/v13/providers/aws/connection"
)

func (a *mqlAwsCognito) id() (string, error) {
	return "aws.cognito", nil
}

func (a *mqlAwsCognito) userPools() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getUserPools(conn), 5)
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

func (a *mqlAwsCognito) getUserPools(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("cognito>getUserPools>calling aws with region %s", region)

			svc := conn.CognitoIdentityProvider(region)
			ctx := context.Background()
			res := []any{}

			var nextToken *string
			for {
				resp, err := svc.ListUserPools(ctx, &cognitoidentityprovider.ListUserPoolsInput{
					MaxResults: aws.Int32(60),
					NextToken:  nextToken,
				})
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS API")
						return res, nil
					}
					if IsServiceNotAvailableInRegionError(err) {
						log.Debug().Str("region", region).Msg("cognito idp service not available in region")
						return res, nil
					}
					return nil, err
				}

				for _, pool := range resp.UserPools {
					poolArn := "arn:aws:cognito-idp:" + region + ":" + conn.AccountId() + ":userpool/" + convert.ToValue(pool.Id)

					mqlPool, err := CreateResource(a.MqlRuntime, "aws.cognito.userPool",
						map[string]*llx.RawData{
							"__id":      llx.StringData(poolArn),
							"arn":       llx.StringData(poolArn),
							"id":        llx.StringDataPtr(pool.Id),
							"name":      llx.StringDataPtr(pool.Name),
							"region":    llx.StringData(region),
							"status":    llx.StringData(string(pool.Status)),
							"createdAt": llx.TimeDataPtr(pool.CreationDate),
							"updatedAt": llx.TimeDataPtr(pool.LastModifiedDate),
						})
					if err != nil {
						return nil, err
					}
					res = append(res, mqlPool)
				}

				if resp.NextToken == nil {
					break
				}
				nextToken = resp.NextToken
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

// Internal caching for DescribeUserPool results
type mqlAwsCognitoUserPoolInternal struct {
	descFetched bool
	descData    *cognitoidentityprovider.DescribeUserPoolOutput
	descLock    sync.Mutex
}

func (a *mqlAwsCognitoUserPool) fetchDescribeUserPool() (*cognitoidentityprovider.DescribeUserPoolOutput, error) {
	if a.descFetched {
		return a.descData, nil
	}
	a.descLock.Lock()
	defer a.descLock.Unlock()

	if a.descFetched {
		return a.descData, nil
	}

	poolId := a.Id.Data
	region := a.Region.Data

	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	svc := conn.CognitoIdentityProvider(region)
	ctx := context.Background()

	resp, err := svc.DescribeUserPool(ctx, &cognitoidentityprovider.DescribeUserPoolInput{
		UserPoolId: &poolId,
	})
	if err != nil {
		log.Warn().Str("userPoolId", poolId).Err(err).Msg("could not describe Cognito user pool")
		a.descFetched = true
		a.descData = nil
		return nil, nil
	}

	a.descFetched = true
	a.descData = resp
	return resp, nil
}

func (a *mqlAwsCognitoUserPool) deletionProtection() (bool, error) {
	resp, err := a.fetchDescribeUserPool()
	if err != nil || resp == nil || resp.UserPool == nil {
		return false, err
	}
	return resp.UserPool.DeletionProtection == cognitoidentityprovidertypes.DeletionProtectionTypeActive, nil
}

func (a *mqlAwsCognitoUserPool) mfaConfiguration() (string, error) {
	poolId := a.Id.Data
	region := a.Region.Data

	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	svc := conn.CognitoIdentityProvider(region)
	ctx := context.Background()

	resp, err := svc.GetUserPoolMfaConfig(ctx, &cognitoidentityprovider.GetUserPoolMfaConfigInput{
		UserPoolId: &poolId,
	})
	if err != nil {
		return "", err
	}
	return string(resp.MfaConfiguration), nil
}

func (a *mqlAwsCognitoUserPool) passwordPolicy() (any, error) {
	resp, err := a.fetchDescribeUserPool()
	if err != nil || resp == nil || resp.UserPool == nil || resp.UserPool.Policies == nil {
		return nil, err
	}
	return convert.JsonToDict(resp.UserPool.Policies.PasswordPolicy)
}

func (a *mqlAwsCognitoUserPool) advancedSecurityMode() (string, error) {
	resp, err := a.fetchDescribeUserPool()
	if err != nil || resp == nil || resp.UserPool == nil || resp.UserPool.UserPoolAddOns == nil {
		return "", err
	}
	return string(resp.UserPool.UserPoolAddOns.AdvancedSecurityMode), nil
}

func (a *mqlAwsCognitoUserPool) tags() (map[string]any, error) {
	resp, err := a.fetchDescribeUserPool()
	if err != nil || resp == nil || resp.UserPool == nil {
		return nil, err
	}
	return convert.MapToInterfaceMap(resp.UserPool.UserPoolTags), nil
}

// Identity Pools (Federated Identities)

func (a *mqlAwsCognito) identityPools() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getIdentityPools(conn), 5)
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

func (a *mqlAwsCognito) getIdentityPools(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("cognito>getIdentityPools>calling aws with region %s", region)

			svc := conn.CognitoIdentity(region)
			ctx := context.Background()
			res := []any{}

			var nextToken *string
			for {
				resp, err := svc.ListIdentityPools(ctx, &cognitoidentity.ListIdentityPoolsInput{
					MaxResults: aws.Int32(60),
					NextToken:  nextToken,
				})
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS API")
						return res, nil
					}
					if IsServiceNotAvailableInRegionError(err) {
						log.Debug().Str("region", region).Msg("cognito identity service not available in region")
						return res, nil
					}
					return nil, err
				}

				for _, pool := range resp.IdentityPools {
					poolId := convert.ToValue(pool.IdentityPoolId)
					poolArn := "arn:aws:cognito-identity:" + region + ":" + conn.AccountId() + ":identitypool/" + poolId

					mqlPool, err := CreateResource(a.MqlRuntime, "aws.cognito.identityPool",
						map[string]*llx.RawData{
							"__id":   llx.StringData(poolArn),
							"id":     llx.StringData(poolId),
							"name":   llx.StringDataPtr(pool.IdentityPoolName),
							"region": llx.StringData(region),
						})
					if err != nil {
						return nil, err
					}
					res = append(res, mqlPool)
				}

				if resp.NextToken == nil {
					break
				}
				nextToken = resp.NextToken
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

type mqlAwsCognitoIdentityPoolInternal struct {
	descCache *cognitoidentity.DescribeIdentityPoolOutput
	descDone  bool
	descLock  sync.Mutex
}

func (a *mqlAwsCognitoIdentityPool) describe() (*cognitoidentity.DescribeIdentityPoolOutput, error) {
	if a.descDone {
		return a.descCache, nil
	}
	a.descLock.Lock()
	defer a.descLock.Unlock()

	if a.descDone {
		return a.descCache, nil
	}

	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	svc := conn.CognitoIdentity(a.Region.Data)
	ctx := context.Background()
	poolId := a.Id.Data

	resp, err := svc.DescribeIdentityPool(ctx, &cognitoidentity.DescribeIdentityPoolInput{
		IdentityPoolId: &poolId,
	})
	if err != nil {
		if Is400AccessDeniedError(err) {
			a.descDone = true
			return nil, nil
		}
		return nil, err
	}

	a.descCache = resp
	a.descDone = true
	return resp, nil
}

func (a *mqlAwsCognitoIdentityPool) allowUnauthenticatedIdentities() (bool, error) {
	resp, err := a.describe()
	if err != nil || resp == nil {
		return false, err
	}
	return resp.AllowUnauthenticatedIdentities, nil
}

func (a *mqlAwsCognitoIdentityPool) allowClassicFlow() (bool, error) {
	resp, err := a.describe()
	if err != nil || resp == nil {
		return false, err
	}
	if resp.AllowClassicFlow == nil {
		return false, nil
	}
	return *resp.AllowClassicFlow, nil
}

func (a *mqlAwsCognitoIdentityPool) developerProviderName() (string, error) {
	resp, err := a.describe()
	if err != nil || resp == nil {
		return "", err
	}
	return convert.ToValue(resp.DeveloperProviderName), nil
}

func (a *mqlAwsCognitoIdentityPool) openIdConnectProviderArns() ([]any, error) {
	resp, err := a.describe()
	if err != nil || resp == nil {
		return nil, err
	}
	var arns []any
	for _, arn := range resp.OpenIdConnectProviderARNs {
		arns = append(arns, arn)
	}
	return arns, nil
}

func (a *mqlAwsCognitoIdentityPool) samlProviderArns() ([]any, error) {
	resp, err := a.describe()
	if err != nil || resp == nil {
		return nil, err
	}
	var arns []any
	for _, arn := range resp.SamlProviderARNs {
		arns = append(arns, arn)
	}
	return arns, nil
}

func (a *mqlAwsCognitoIdentityPool) supportedLoginProviders() (map[string]any, error) {
	resp, err := a.describe()
	if err != nil || resp == nil {
		return nil, err
	}
	return convert.MapToInterfaceMap(resp.SupportedLoginProviders), nil
}

func (a *mqlAwsCognitoIdentityPool) tags() (map[string]any, error) {
	resp, err := a.describe()
	if err != nil || resp == nil {
		return nil, err
	}
	return convert.MapToInterfaceMap(resp.IdentityPoolTags), nil
}
