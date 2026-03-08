// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/aws/aws-sdk-go-v2/service/sagemaker"
	"github.com/aws/smithy-go/transport/http"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/mql/v13/providers/aws/connection"

	"go.mondoo.com/mql/v13/types"
)

func (a *mqlAwsSagemaker) id() (string, error) {
	return ResourceAwsSagemaker, nil
}

func (a *mqlAwsSagemaker) endpoints() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getEndpoints(conn), 5)
	poolOfJobs.Run()

	// check for errors
	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	// get all the results
	for i := range poolOfJobs.Jobs {
		res = append(res, poolOfJobs.Jobs[i].Result.([]any)...)
	}

	return res, nil
}

func (a *mqlAwsSagemaker) getEndpoints(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			svc := conn.Sagemaker(region)
			ctx := context.Background()
			res := []any{}

			params := &sagemaker.ListEndpointsInput{}
			paginator := sagemaker.NewListEndpointsPaginator(svc, params)
			for paginator.HasMorePages() {
				endpoints, err := paginator.NextPage(ctx)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS API")
						return res, nil
					}
					return nil, err
				}

				for _, endpoint := range endpoints.Endpoints {
					tags, err := getSagemakerTags(ctx, svc, endpoint.EndpointArn)
					if err != nil {
						return nil, err
					}

					if conn.Filters.General.IsFilteredOutByTags(mapStringInterfaceToStringString(tags)) {
						log.Debug().Interface("endpoint", endpoint.EndpointArn).Msg("skipping sagemaker endpoint due to filters")
						continue
					}

					mqlEndpoint, err := CreateResource(a.MqlRuntime, ResourceAwsSagemakerEndpoint,
						map[string]*llx.RawData{
							"arn":            llx.StringDataPtr(endpoint.EndpointArn),
							"name":           llx.StringDataPtr(endpoint.EndpointName),
							"region":         llx.StringData(region),
							"tags":           llx.MapData(tags, types.String),
							"createdAt":      llx.TimeDataPtr(endpoint.CreationTime),
							"lastModifiedAt": llx.TimeDataPtr(endpoint.LastModifiedTime),
							"status":         llx.StringData(string(endpoint.EndpointStatus)),
						})
					if err != nil {
						return nil, err
					}
					res = append(res, mqlEndpoint)
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func (a *mqlAwsSagemakerEndpoint) config() (map[string]any, error) {
	name := a.Name.Data
	region := a.Region.Data
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	svc := conn.Sagemaker(region)
	ctx := context.Background()
	config, err := svc.DescribeEndpointConfig(ctx, &sagemaker.DescribeEndpointConfigInput{EndpointConfigName: &name})
	if err != nil {
		return nil, err
	}
	return convert.JsonToDict(config)
}

func (a *mqlAwsSagemaker) notebookInstances() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getNotebookInstances(conn), 5)
	poolOfJobs.Run()

	// check for errors
	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	// get all the results
	for i := range poolOfJobs.Jobs {
		res = append(res, poolOfJobs.Jobs[i].Result.([]any)...)
	}

	return res, nil
}

func (a *mqlAwsSagemaker) getNotebookInstances(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			svc := conn.Sagemaker(region)
			ctx := context.Background()
			res := []any{}

			params := &sagemaker.ListNotebookInstancesInput{}
			paginator := sagemaker.NewListNotebookInstancesPaginator(svc, params)
			for paginator.HasMorePages() {
				notebookInstances, err := paginator.NextPage(ctx)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS API")
						return res, nil
					}
					return nil, err
				}
				for _, instance := range notebookInstances.NotebookInstances {
					tags, err := getSagemakerTags(ctx, svc, instance.NotebookInstanceArn)
					if err != nil {
						return nil, err
					}

					if conn.Filters.General.IsFilteredOutByTags(mapStringInterfaceToStringString(tags)) {
						log.Debug().Interface("notebook", instance.NotebookInstanceArn).Msg("skipping sagemaker notebook instance due to filters")
						continue
					}

					mqlEndpoint, err := CreateResource(a.MqlRuntime, ResourceAwsSagemakerNotebookinstance,
						map[string]*llx.RawData{
							"arn":            llx.StringData(convert.ToValue(instance.NotebookInstanceArn)),
							"name":           llx.StringData(convert.ToValue(instance.NotebookInstanceName)),
							"region":         llx.StringData(region),
							"tags":           llx.MapData(tags, types.String),
							"createdAt":      llx.TimeDataPtr(instance.CreationTime),
							"lastModifiedAt": llx.TimeDataPtr(instance.LastModifiedTime),
							"status":         llx.StringData(string(instance.NotebookInstanceStatus)),
							"url":            llx.StringDataPtr(instance.Url),
						})
					if err != nil {
						return nil, err
					}
					res = append(res, mqlEndpoint)
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func initAwsSagemakerNotebookinstance(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}

	if len(args) == 0 {
		if ids := getAssetIdentifier(runtime); ids != nil {
			args["name"] = llx.StringData(ids.name)
			args["arn"] = llx.StringData(ids.arn)
		}
	}

	if args["arn"] == nil {
		return nil, nil, errors.New("arn required to fetch sagemaker notebookinstance")
	}

	obj, err := CreateResource(runtime, "aws.sagemaker", map[string]*llx.RawData{})
	if err != nil {
		return nil, nil, err
	}
	sm := obj.(*mqlAwsSagemaker)

	rawResources := sm.GetNotebookInstances()
	if rawResources.Error != nil {
		return nil, nil, rawResources.Error
	}

	arnVal := args["arn"].Value.(string)
	for _, rawResource := range rawResources.Data {
		ni := rawResource.(*mqlAwsSagemakerNotebookinstance)
		if ni.Arn.Data == arnVal {
			return args, ni, nil
		}
	}
	return nil, nil, errors.New("sagemaker notebookinstance does not exist")
}

func (a *mqlAwsSagemakerNotebookinstance) details() (*mqlAwsSagemakerNotebookinstancedetails, error) {
	name := a.Name.Data
	region := a.Region.Data

	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	svc := conn.Sagemaker(region)
	ctx := context.Background()
	instanceDetails, err := svc.DescribeNotebookInstance(ctx, &sagemaker.DescribeNotebookInstanceInput{NotebookInstanceName: &name})
	if err != nil {
		return nil, err
	}
	args := map[string]*llx.RawData{
		"arn":                  llx.StringDataPtr(instanceDetails.NotebookInstanceArn),
		"directInternetAccess": llx.BoolData(string(instanceDetails.DirectInternetAccess) == "Enabled"),
		"rootAccess":           llx.BoolData(string(instanceDetails.RootAccess) == "Enabled"),
	}
	if instanceDetails.InstanceMetadataServiceConfiguration != nil {
		args["minimumInstanceMetadataServiceVersion"] = llx.StringDataPtr(instanceDetails.InstanceMetadataServiceConfiguration.MinimumInstanceMetadataServiceVersion)
	} else {
		args["minimumInstanceMetadataServiceVersion"] = llx.StringData("1")
	}

	mqlInstanceDetails, err := CreateResource(a.MqlRuntime, "aws.sagemaker.notebookinstancedetails", args)
	if err != nil {
		return nil, err
	}
	details := mqlInstanceDetails.(*mqlAwsSagemakerNotebookinstancedetails)
	details.cacheKmsKey = instanceDetails.KmsKeyId
	details.cacheSubnetId = instanceDetails.SubnetId
	details.region = region
	return details, nil
}

type mqlAwsSagemakerNotebookinstancedetailsInternal struct {
	cacheKmsKey   *string
	cacheSubnetId *string
	region        string
}

func (a *mqlAwsSagemakerNotebookinstancedetails) kmsKey() (*mqlAwsKmsKey, error) {
	if a.cacheKmsKey != nil && *a.cacheKmsKey != "" {
		mqlKeyResource, err := NewResource(a.MqlRuntime, "aws.kms.key",
			map[string]*llx.RawData{"arn": llx.StringData(convert.ToValue(a.cacheKmsKey))},
		)
		if err != nil {
			return nil, err
		}
		return mqlKeyResource.(*mqlAwsKmsKey), nil
	}
	a.KmsKey.State = plugin.StateIsNull | plugin.StateIsSet
	return nil, nil
}

func (a *mqlAwsSagemakerNotebookinstancedetails) subnet() (*mqlAwsVpcSubnet, error) {
	if a.cacheSubnetId != nil && *a.cacheSubnetId != "" {
		conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
		arn := fmt.Sprintf(subnetArnPattern, a.region, conn.AccountId(), *a.cacheSubnetId)
		res, err := NewResource(a.MqlRuntime, ResourceAwsVpcSubnet, map[string]*llx.RawData{"arn": llx.StringData(arn)})
		if err != nil {
			return nil, err
		}
		return res.(*mqlAwsVpcSubnet), nil
	}
	a.Subnet.State = plugin.StateIsNull | plugin.StateIsSet
	return nil, nil
}

func (a *mqlAwsSagemakerEndpoint) id() (string, error) {
	return a.Arn.Data, nil
}

func (a *mqlAwsSagemakerNotebookinstance) id() (string, error) {
	return a.Arn.Data, nil
}

func (a *mqlAwsSagemakerNotebookinstancedetails) id() (string, error) {
	return a.Arn.Data, nil
}

func getSagemakerTags(ctx context.Context, svc *sagemaker.Client, arn *string) (map[string]any, error) {
	resp, err := svc.ListTags(ctx, &sagemaker.ListTagsInput{ResourceArn: arn})
	var respErr *http.ResponseError
	if err != nil {
		if errors.As(err, &respErr) {
			if respErr.HTTPStatusCode() == 404 {
				return nil, nil
			}
		}
		return nil, err
	}
	tags := make(map[string]any)
	for _, t := range resp.Tags {
		if t.Key != nil && t.Value != nil {
			tags[*t.Key] = *t.Value
		}
	}
	return tags, nil
}

// ---- Models ----

func (a *mqlAwsSagemaker) models() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getModels(conn), 5)
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

func (a *mqlAwsSagemaker) getModels(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}
	for _, region := range regions {
		region := region
		f := func() (jobpool.JobResult, error) {
			svc := conn.Sagemaker(region)
			ctx := context.Background()
			res := []any{}
			paginator := sagemaker.NewListModelsPaginator(svc, &sagemaker.ListModelsInput{})
			for paginator.HasMorePages() {
				page, err := paginator.NextPage(ctx)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS API")
						return res, nil
					}
					return nil, err
				}
				for _, model := range page.Models {
					tags, err := getSagemakerTags(ctx, svc, model.ModelArn)
					if err != nil {
						return nil, err
					}
					if conn.Filters.General.IsFilteredOutByTags(mapStringInterfaceToStringString(tags)) {
						continue
					}
					mqlModel, err := CreateResource(a.MqlRuntime, ResourceAwsSagemakerModel,
						map[string]*llx.RawData{
							"arn":       llx.StringDataPtr(model.ModelArn),
							"name":      llx.StringDataPtr(model.ModelName),
							"region":    llx.StringData(region),
							"createdAt": llx.TimeDataPtr(model.CreationTime),
							"tags":      llx.MapData(tags, types.String),
						})
					if err != nil {
						return nil, err
					}
					m := mqlModel.(*mqlAwsSagemakerModel)
					m.region = region
					m.accountID = conn.AccountId()
					res = append(res, mqlModel)
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

type mqlAwsSagemakerModelInternal struct {
	securityGroupIdHandler
	cacheRoleArn    *string
	cacheVpcId      *string
	cacheSubnetIds  []string
	region          string
	accountID       string
	fetched         bool
	fetchedDetails  *sagemaker.DescribeModelOutput
	lock            sync.Mutex
}

func (a *mqlAwsSagemakerModel) id() (string, error) {
	return a.Arn.Data, nil
}

func (a *mqlAwsSagemakerModel) fetchDetails() (*sagemaker.DescribeModelOutput, error) {
	if a.fetched {
		return a.fetchedDetails, nil
	}
	a.lock.Lock()
	defer a.lock.Unlock()
	if a.fetched {
		return a.fetchedDetails, nil
	}
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	svc := conn.Sagemaker(a.region)
	name := a.Name.Data
	resp, err := svc.DescribeModel(context.Background(), &sagemaker.DescribeModelInput{ModelName: &name})
	if err != nil {
		return nil, err
	}
	a.cacheRoleArn = resp.ExecutionRoleArn
	if resp.VpcConfig != nil {
		if len(resp.VpcConfig.Subnets) > 0 {
			a.cacheSubnetIds = resp.VpcConfig.Subnets
		}
		if len(resp.VpcConfig.SecurityGroupIds) > 0 {
			sgs := make([]string, len(resp.VpcConfig.SecurityGroupIds))
			for i, id := range resp.VpcConfig.SecurityGroupIds {
				sgs[i] = NewSecurityGroupArn(a.region, a.accountID, id)
			}
			a.setSecurityGroupArns(sgs)
		}
	}
	a.fetchedDetails = resp
	a.fetched = true
	return resp, nil
}

func (a *mqlAwsSagemakerModel) containers() ([]any, error) {
	resp, err := a.fetchDetails()
	if err != nil {
		return nil, err
	}
	containers := make([]any, 0)
	if resp.PrimaryContainer != nil {
		d, err := convert.JsonToDict(resp.PrimaryContainer)
		if err != nil {
			return nil, err
		}
		containers = append(containers, d)
	}
	for _, c := range resp.Containers {
		d, err := convert.JsonToDict(c)
		if err != nil {
			return nil, err
		}
		containers = append(containers, d)
	}
	return containers, nil
}

func (a *mqlAwsSagemakerModel) enableNetworkIsolation() (bool, error) {
	resp, err := a.fetchDetails()
	if err != nil {
		return false, err
	}
	if resp.EnableNetworkIsolation != nil {
		return *resp.EnableNetworkIsolation, nil
	}
	return false, nil
}

func (a *mqlAwsSagemakerModel) iamRole() (*mqlAwsIamRole, error) {
	_, err := a.fetchDetails()
	if err != nil {
		return nil, err
	}
	if a.cacheRoleArn == nil || *a.cacheRoleArn == "" {
		a.IamRole.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	res, err := NewResource(a.MqlRuntime, "aws.iam.role", map[string]*llx.RawData{"arn": llx.StringDataPtr(a.cacheRoleArn)})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAwsIamRole), nil
}

func (a *mqlAwsSagemakerModel) subnets() ([]any, error) {
	_, err := a.fetchDetails()
	if err != nil {
		return nil, err
	}
	if len(a.cacheSubnetIds) == 0 {
		a.Subnets.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	subnets := make([]any, 0, len(a.cacheSubnetIds))
	for _, subnetId := range a.cacheSubnetIds {
		subnetArn := fmt.Sprintf(subnetArnPattern, a.region, conn.AccountId(), subnetId)
		res, err := NewResource(a.MqlRuntime, ResourceAwsVpcSubnet, map[string]*llx.RawData{"arn": llx.StringData(subnetArn)})
		if err != nil {
			return nil, err
		}
		subnets = append(subnets, res)
	}
	return subnets, nil
}

func (a *mqlAwsSagemakerModel) securityGroups() ([]any, error) {
	_, err := a.fetchDetails()
	if err != nil {
		return nil, err
	}
	return a.newSecurityGroupResources(a.MqlRuntime)
}

// ---- Domains ----

func (a *mqlAwsSagemaker) domains() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getDomains(conn), 5)
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

func (a *mqlAwsSagemaker) getDomains(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}
	for _, region := range regions {
		region := region
		f := func() (jobpool.JobResult, error) {
			svc := conn.Sagemaker(region)
			ctx := context.Background()
			res := []any{}
			paginator := sagemaker.NewListDomainsPaginator(svc, &sagemaker.ListDomainsInput{})
			for paginator.HasMorePages() {
				page, err := paginator.NextPage(ctx)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS API")
						return res, nil
					}
					return nil, err
				}
				for _, domain := range page.Domains {
					tags, err := getSagemakerTags(ctx, svc, domain.DomainArn)
					if err != nil {
						return nil, err
					}
					if conn.Filters.General.IsFilteredOutByTags(mapStringInterfaceToStringString(tags)) {
						continue
					}
					mqlDomain, err := CreateResource(a.MqlRuntime, ResourceAwsSagemakerDomain,
						map[string]*llx.RawData{
							"arn":       llx.StringDataPtr(domain.DomainArn),
							"name":      llx.StringDataPtr(domain.DomainName),
							"domainId":  llx.StringDataPtr(domain.DomainId),
							"region":    llx.StringData(region),
							"status":    llx.StringData(string(domain.Status)),
							"createdAt": llx.TimeDataPtr(domain.CreationTime),
							"tags":      llx.MapData(tags, types.String),
						})
					if err != nil {
						return nil, err
					}
					d := mqlDomain.(*mqlAwsSagemakerDomain)
					d.region = region
					d.accountID = conn.AccountId()
					res = append(res, mqlDomain)
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

type mqlAwsSagemakerDomainInternal struct {
	securityGroupIdHandler
	cacheVpcId      *string
	cacheSubnetIds  []string
	cacheKmsKeyId   *string
	region          string
	accountID       string
	fetched         bool
	fetchedDetails  *sagemaker.DescribeDomainOutput
	lock            sync.Mutex
}

func (a *mqlAwsSagemakerDomain) id() (string, error) {
	return a.Arn.Data, nil
}

func (a *mqlAwsSagemakerDomain) fetchDomainDetails() (*sagemaker.DescribeDomainOutput, error) {
	if a.fetched {
		return a.fetchedDetails, nil
	}
	a.lock.Lock()
	defer a.lock.Unlock()
	if a.fetched {
		return a.fetchedDetails, nil
	}
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	svc := conn.Sagemaker(a.region)
	domainId := a.DomainId.Data
	resp, err := svc.DescribeDomain(context.Background(), &sagemaker.DescribeDomainInput{DomainId: &domainId})
	if err != nil {
		return nil, err
	}
	a.cacheVpcId = resp.VpcId
	a.cacheKmsKeyId = resp.KmsKeyId
	if len(resp.SubnetIds) > 0 {
		a.cacheSubnetIds = resp.SubnetIds
	}
	if resp.DefaultUserSettings != nil && len(resp.DefaultUserSettings.SecurityGroups) > 0 {
		sgs := make([]string, len(resp.DefaultUserSettings.SecurityGroups))
		for i, id := range resp.DefaultUserSettings.SecurityGroups {
			sgs[i] = NewSecurityGroupArn(a.region, a.accountID, id)
		}
		a.setSecurityGroupArns(sgs)
	}
	a.fetchedDetails = resp
	a.fetched = true
	return resp, nil
}

func (a *mqlAwsSagemakerDomain) authMode() (string, error) {
	resp, err := a.fetchDomainDetails()
	if err != nil {
		return "", err
	}
	return string(resp.AuthMode), nil
}

func (a *mqlAwsSagemakerDomain) appNetworkAccessType() (string, error) {
	resp, err := a.fetchDomainDetails()
	if err != nil {
		return "", err
	}
	return string(resp.AppNetworkAccessType), nil
}

func (a *mqlAwsSagemakerDomain) defaultUserSettings() (map[string]any, error) {
	resp, err := a.fetchDomainDetails()
	if err != nil {
		return nil, err
	}
	if resp.DefaultUserSettings == nil {
		return nil, nil
	}
	return convert.JsonToDict(resp.DefaultUserSettings)
}

func (a *mqlAwsSagemakerDomain) vpc() (*mqlAwsVpc, error) {
	_, err := a.fetchDomainDetails()
	if err != nil {
		return nil, err
	}
	if a.cacheVpcId == nil || *a.cacheVpcId == "" {
		a.Vpc.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	res, err := NewResource(a.MqlRuntime, "aws.vpc", map[string]*llx.RawData{"id": llx.StringDataPtr(a.cacheVpcId)})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAwsVpc), nil
}

func (a *mqlAwsSagemakerDomain) subnets() ([]any, error) {
	_, err := a.fetchDomainDetails()
	if err != nil {
		return nil, err
	}
	if len(a.cacheSubnetIds) == 0 {
		a.Subnets.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	subnets := make([]any, 0, len(a.cacheSubnetIds))
	for _, subnetId := range a.cacheSubnetIds {
		subnetArn := fmt.Sprintf(subnetArnPattern, a.region, conn.AccountId(), subnetId)
		res, err := NewResource(a.MqlRuntime, ResourceAwsVpcSubnet, map[string]*llx.RawData{"arn": llx.StringData(subnetArn)})
		if err != nil {
			return nil, err
		}
		subnets = append(subnets, res)
	}
	return subnets, nil
}

func (a *mqlAwsSagemakerDomain) securityGroups() ([]any, error) {
	_, err := a.fetchDomainDetails()
	if err != nil {
		return nil, err
	}
	return a.newSecurityGroupResources(a.MqlRuntime)
}

func (a *mqlAwsSagemakerDomain) kmsKey() (*mqlAwsKmsKey, error) {
	_, err := a.fetchDomainDetails()
	if err != nil {
		return nil, err
	}
	if a.cacheKmsKeyId == nil || *a.cacheKmsKeyId == "" {
		a.KmsKey.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	res, err := NewResource(a.MqlRuntime, "aws.kms.key", map[string]*llx.RawData{"arn": llx.StringDataPtr(a.cacheKmsKeyId)})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAwsKmsKey), nil
}

// ---- User Profiles ----

func (a *mqlAwsSagemaker) userProfiles() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getUserProfiles(conn), 5)
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

func (a *mqlAwsSagemaker) getUserProfiles(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}
	for _, region := range regions {
		region := region
		f := func() (jobpool.JobResult, error) {
			svc := conn.Sagemaker(region)
			ctx := context.Background()
			res := []any{}
			paginator := sagemaker.NewListUserProfilesPaginator(svc, &sagemaker.ListUserProfilesInput{})
			for paginator.HasMorePages() {
				page, err := paginator.NextPage(ctx)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS API")
						return res, nil
					}
					return nil, err
				}
				for _, up := range page.UserProfiles {
					arn := fmt.Sprintf("arn:aws:sagemaker:%s:%s:user-profile/%s/%s",
						region, conn.AccountId(),
						convert.ToValue(up.DomainId), convert.ToValue(up.UserProfileName))
					tags, err := getSagemakerTags(ctx, svc, &arn)
					if err != nil {
						return nil, err
					}
					if conn.Filters.General.IsFilteredOutByTags(mapStringInterfaceToStringString(tags)) {
						continue
					}
					mqlUP, err := CreateResource(a.MqlRuntime, ResourceAwsSagemakerUserProfile,
						map[string]*llx.RawData{
							"arn":       llx.StringData(arn),
							"name":      llx.StringDataPtr(up.UserProfileName),
							"domainId":  llx.StringDataPtr(up.DomainId),
							"region":    llx.StringData(region),
							"status":    llx.StringData(string(up.Status)),
							"createdAt": llx.TimeDataPtr(up.CreationTime),
							"tags":      llx.MapData(tags, types.String),
						})
					if err != nil {
						return nil, err
					}
					u := mqlUP.(*mqlAwsSagemakerUserProfile)
					u.region = region
					u.accountID = conn.AccountId()
					res = append(res, mqlUP)
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

type mqlAwsSagemakerUserProfileInternal struct {
	securityGroupIdHandler
	cacheRoleArn *string
	region       string
	accountID    string
	fetched      bool
	lock         sync.Mutex
}

func (a *mqlAwsSagemakerUserProfile) id() (string, error) {
	return a.Arn.Data, nil
}

func (a *mqlAwsSagemakerUserProfile) fetchUserProfileDetails() error {
	if a.fetched {
		return nil
	}
	a.lock.Lock()
	defer a.lock.Unlock()
	if a.fetched {
		return nil
	}
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	svc := conn.Sagemaker(a.region)
	domainId := a.DomainId.Data
	name := a.Name.Data
	resp, err := svc.DescribeUserProfile(context.Background(), &sagemaker.DescribeUserProfileInput{
		DomainId:        &domainId,
		UserProfileName: &name,
	})
	if err != nil {
		return err
	}
	if resp.UserSettings != nil {
		a.cacheRoleArn = resp.UserSettings.ExecutionRole
		if len(resp.UserSettings.SecurityGroups) > 0 {
			sgs := make([]string, len(resp.UserSettings.SecurityGroups))
			for i, id := range resp.UserSettings.SecurityGroups {
				sgs[i] = NewSecurityGroupArn(a.region, a.accountID, id)
			}
			a.setSecurityGroupArns(sgs)
		}
	}
	a.fetched = true
	return nil
}

func (a *mqlAwsSagemakerUserProfile) iamRole() (*mqlAwsIamRole, error) {
	if err := a.fetchUserProfileDetails(); err != nil {
		return nil, err
	}
	if a.cacheRoleArn == nil || *a.cacheRoleArn == "" {
		a.IamRole.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	res, err := NewResource(a.MqlRuntime, "aws.iam.role", map[string]*llx.RawData{"arn": llx.StringDataPtr(a.cacheRoleArn)})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAwsIamRole), nil
}

func (a *mqlAwsSagemakerUserProfile) securityGroups() ([]any, error) {
	if err := a.fetchUserProfileDetails(); err != nil {
		return nil, err
	}
	return a.newSecurityGroupResources(a.MqlRuntime)
}

// ---- Training Jobs ----

func (a *mqlAwsSagemaker) trainingJobs() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getTrainingJobs(conn), 5)
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

func (a *mqlAwsSagemaker) getTrainingJobs(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}
	for _, region := range regions {
		region := region
		f := func() (jobpool.JobResult, error) {
			svc := conn.Sagemaker(region)
			ctx := context.Background()
			res := []any{}
			paginator := sagemaker.NewListTrainingJobsPaginator(svc, &sagemaker.ListTrainingJobsInput{})
			for paginator.HasMorePages() {
				page, err := paginator.NextPage(ctx)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS API")
						return res, nil
					}
					return nil, err
				}
				for _, job := range page.TrainingJobSummaries {
					tags, err := getSagemakerTags(ctx, svc, job.TrainingJobArn)
					if err != nil {
						return nil, err
					}
					if conn.Filters.General.IsFilteredOutByTags(mapStringInterfaceToStringString(tags)) {
						continue
					}
					mqlJob, err := CreateResource(a.MqlRuntime, ResourceAwsSagemakerTrainingJob,
						map[string]*llx.RawData{
							"arn":            llx.StringDataPtr(job.TrainingJobArn),
							"name":           llx.StringDataPtr(job.TrainingJobName),
							"region":         llx.StringData(region),
							"status":         llx.StringData(string(job.TrainingJobStatus)),
							"createdAt":      llx.TimeDataPtr(job.CreationTime),
							"lastModifiedAt": llx.TimeDataPtr(job.LastModifiedTime),
							"tags":           llx.MapData(tags, types.String),
						})
					if err != nil {
						return nil, err
					}
					j := mqlJob.(*mqlAwsSagemakerTrainingJob)
					j.region = region
					j.accountID = conn.AccountId()
					res = append(res, mqlJob)
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

type mqlAwsSagemakerTrainingJobInternal struct {
	securityGroupIdHandler
	cacheRoleArn   *string
	cacheVpcId     *string
	cacheSubnetIds []string
	cacheKmsKeyId  *string
	region         string
	accountID      string
	fetched        bool
	fetchedDetails *sagemaker.DescribeTrainingJobOutput
	lock           sync.Mutex
}

func (a *mqlAwsSagemakerTrainingJob) id() (string, error) {
	return a.Arn.Data, nil
}

func (a *mqlAwsSagemakerTrainingJob) fetchTrainingJobDetails() (*sagemaker.DescribeTrainingJobOutput, error) {
	if a.fetched {
		return a.fetchedDetails, nil
	}
	a.lock.Lock()
	defer a.lock.Unlock()
	if a.fetched {
		return a.fetchedDetails, nil
	}
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	svc := conn.Sagemaker(a.region)
	name := a.Name.Data
	resp, err := svc.DescribeTrainingJob(context.Background(), &sagemaker.DescribeTrainingJobInput{TrainingJobName: &name})
	if err != nil {
		return nil, err
	}
	a.cacheRoleArn = resp.RoleArn
	if resp.VpcConfig != nil {
		if len(resp.VpcConfig.Subnets) > 0 {
			a.cacheSubnetIds = resp.VpcConfig.Subnets
		}
		if len(resp.VpcConfig.SecurityGroupIds) > 0 {
			sgs := make([]string, len(resp.VpcConfig.SecurityGroupIds))
			for i, id := range resp.VpcConfig.SecurityGroupIds {
				sgs[i] = NewSecurityGroupArn(a.region, a.accountID, id)
			}
			a.setSecurityGroupArns(sgs)
		}
	}
	if resp.OutputDataConfig != nil {
		a.cacheKmsKeyId = resp.OutputDataConfig.KmsKeyId
	}
	a.fetchedDetails = resp
	a.fetched = true
	return resp, nil
}

func (a *mqlAwsSagemakerTrainingJob) algorithmSpecification() (map[string]any, error) {
	resp, err := a.fetchTrainingJobDetails()
	if err != nil {
		return nil, err
	}
	if resp.AlgorithmSpecification == nil {
		return nil, nil
	}
	return convert.JsonToDict(resp.AlgorithmSpecification)
}

func (a *mqlAwsSagemakerTrainingJob) inputDataConfig() ([]any, error) {
	resp, err := a.fetchTrainingJobDetails()
	if err != nil {
		return nil, err
	}
	return convert.JsonToDictSlice(resp.InputDataConfig)
}

func (a *mqlAwsSagemakerTrainingJob) outputDataConfig() (map[string]any, error) {
	resp, err := a.fetchTrainingJobDetails()
	if err != nil {
		return nil, err
	}
	if resp.OutputDataConfig == nil {
		return nil, nil
	}
	return convert.JsonToDict(resp.OutputDataConfig)
}

func (a *mqlAwsSagemakerTrainingJob) enableNetworkIsolation() (bool, error) {
	resp, err := a.fetchTrainingJobDetails()
	if err != nil {
		return false, err
	}
	if resp.EnableNetworkIsolation != nil {
		return *resp.EnableNetworkIsolation, nil
	}
	return false, nil
}

func (a *mqlAwsSagemakerTrainingJob) enableInterContainerTrafficEncryption() (bool, error) {
	resp, err := a.fetchTrainingJobDetails()
	if err != nil {
		return false, err
	}
	if resp.EnableInterContainerTrafficEncryption != nil {
		return *resp.EnableInterContainerTrafficEncryption, nil
	}
	return false, nil
}

func (a *mqlAwsSagemakerTrainingJob) iamRole() (*mqlAwsIamRole, error) {
	_, err := a.fetchTrainingJobDetails()
	if err != nil {
		return nil, err
	}
	if a.cacheRoleArn == nil || *a.cacheRoleArn == "" {
		a.IamRole.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	res, err := NewResource(a.MqlRuntime, "aws.iam.role", map[string]*llx.RawData{"arn": llx.StringDataPtr(a.cacheRoleArn)})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAwsIamRole), nil
}

func (a *mqlAwsSagemakerTrainingJob) subnets() ([]any, error) {
	_, err := a.fetchTrainingJobDetails()
	if err != nil {
		return nil, err
	}
	if len(a.cacheSubnetIds) == 0 {
		a.Subnets.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	subnets := make([]any, 0, len(a.cacheSubnetIds))
	for _, subnetId := range a.cacheSubnetIds {
		subnetArn := fmt.Sprintf(subnetArnPattern, a.region, conn.AccountId(), subnetId)
		res, err := NewResource(a.MqlRuntime, ResourceAwsVpcSubnet, map[string]*llx.RawData{"arn": llx.StringData(subnetArn)})
		if err != nil {
			return nil, err
		}
		subnets = append(subnets, res)
	}
	return subnets, nil
}

func (a *mqlAwsSagemakerTrainingJob) securityGroups() ([]any, error) {
	_, err := a.fetchTrainingJobDetails()
	if err != nil {
		return nil, err
	}
	return a.newSecurityGroupResources(a.MqlRuntime)
}

func (a *mqlAwsSagemakerTrainingJob) kmsKey() (*mqlAwsKmsKey, error) {
	_, err := a.fetchTrainingJobDetails()
	if err != nil {
		return nil, err
	}
	if a.cacheKmsKeyId == nil || *a.cacheKmsKeyId == "" {
		a.KmsKey.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	res, err := NewResource(a.MqlRuntime, "aws.kms.key", map[string]*llx.RawData{"arn": llx.StringDataPtr(a.cacheKmsKeyId)})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAwsKmsKey), nil
}

// ---- Model Package Groups ----

func (a *mqlAwsSagemaker) modelPackageGroups() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getModelPackageGroups(conn), 5)
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

func (a *mqlAwsSagemaker) getModelPackageGroups(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}
	for _, region := range regions {
		region := region
		f := func() (jobpool.JobResult, error) {
			svc := conn.Sagemaker(region)
			ctx := context.Background()
			res := []any{}
			paginator := sagemaker.NewListModelPackageGroupsPaginator(svc, &sagemaker.ListModelPackageGroupsInput{})
			for paginator.HasMorePages() {
				page, err := paginator.NextPage(ctx)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS API")
						return res, nil
					}
					return nil, err
				}
				for _, group := range page.ModelPackageGroupSummaryList {
					tags, err := getSagemakerTags(ctx, svc, group.ModelPackageGroupArn)
					if err != nil {
						return nil, err
					}
					if conn.Filters.General.IsFilteredOutByTags(mapStringInterfaceToStringString(tags)) {
						continue
					}
					mqlGroup, err := CreateResource(a.MqlRuntime, ResourceAwsSagemakerModelPackageGroup,
						map[string]*llx.RawData{
							"arn":       llx.StringDataPtr(group.ModelPackageGroupArn),
							"name":      llx.StringDataPtr(group.ModelPackageGroupName),
							"region":    llx.StringData(region),
							"status":    llx.StringData(string(group.ModelPackageGroupStatus)),
							"createdAt": llx.TimeDataPtr(group.CreationTime),
							"tags":      llx.MapData(tags, types.String),
						})
					if err != nil {
						return nil, err
					}
					g := mqlGroup.(*mqlAwsSagemakerModelPackageGroup)
					g.region = region
					res = append(res, mqlGroup)
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

type mqlAwsSagemakerModelPackageGroupInternal struct {
	region string
}

func (a *mqlAwsSagemakerModelPackageGroup) id() (string, error) {
	return a.Arn.Data, nil
}

func (a *mqlAwsSagemakerModelPackageGroup) description() (string, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	svc := conn.Sagemaker(a.region)
	name := a.Name.Data
	resp, err := svc.DescribeModelPackageGroup(context.Background(), &sagemaker.DescribeModelPackageGroupInput{ModelPackageGroupName: &name})
	if err != nil {
		return "", err
	}
	return convert.ToValue(resp.ModelPackageGroupDescription), nil
}

func (a *mqlAwsSagemakerModelPackageGroup) modelPackages() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	svc := conn.Sagemaker(a.region)
	name := a.Name.Data
	ctx := context.Background()

	res := []any{}
	paginator := sagemaker.NewListModelPackagesPaginator(svc, &sagemaker.ListModelPackagesInput{
		ModelPackageGroupName: &name,
	})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, pkg := range page.ModelPackageSummaryList {
			mqlPkg, err := CreateResource(a.MqlRuntime, ResourceAwsSagemakerModelPackage,
				map[string]*llx.RawData{
					"arn":            llx.StringDataPtr(pkg.ModelPackageArn),
					"name":           llx.StringDataPtr(pkg.ModelPackageName),
					"status":         llx.StringData(string(pkg.ModelPackageStatus)),
					"approvalStatus": llx.StringData(string(pkg.ModelApprovalStatus)),
					"description":    llx.StringDataPtr(pkg.ModelPackageDescription),
					"createdAt":      llx.TimeDataPtr(pkg.CreationTime),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlPkg)
		}
	}
	return res, nil
}

func (a *mqlAwsSagemakerModelPackage) id() (string, error) {
	return a.Arn.Data, nil
}
