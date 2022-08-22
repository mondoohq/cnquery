package aws

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go-v2/service/elasticsearchservice"
	"github.com/aws/smithy-go/transport/http"
	aws_provider "go.mondoo.io/mondoo/motor/providers/aws"
	"go.mondoo.io/mondoo/resources"
	"go.mondoo.io/mondoo/resources/library/jobpool"
	"go.mondoo.io/mondoo/resources/packs/core"
)

func (e *mqlAwsEs) id() (string, error) {
	return "aws.es", nil
}

func (e *mqlAwsEs) GetDomains() ([]interface{}, error) {
	provider, err := awsProvider(e.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(e.getDomains(provider), 5)
	poolOfJobs.Run()

	// check for errors
	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	// get all the results
	for i := range poolOfJobs.Jobs {
		res = append(res, poolOfJobs.Jobs[i].Result.([]interface{})...)
	}

	return res, nil
}

func (e *mqlAwsEs) getDomains(provider *aws_provider.Provider) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := provider.GetRegions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			svc := provider.Es(regionVal)
			ctx := context.Background()
			res := []interface{}{}

			domains, err := svc.ListDomainNames(ctx, &elasticsearchservice.ListDomainNamesInput{})
			if err != nil {
				return nil, err
			}

			for _, domain := range domains.DomainNames {
				// note: the api returns name and region here, so we just use that.
				// the arn is not returned until we get to the describe call
				mqlDomain, err := e.MotorRuntime.CreateResource("aws.es.domain",
					"name", core.ToString(domain.DomainName),
					"region", regionVal,
				)
				if err != nil {
					return nil, err
				}
				res = append(res, mqlDomain)
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func (a *mqlAwsEsDomain) init(args *resources.Args) (*resources.Args, AwsEsDomain, error) {
	if len(*args) > 2 {
		return args, nil, nil
	}

	if (*args)["name"] == nil || (*args)["region"] == nil {
		return nil, nil, errors.New("name and region required to fetch es domain")
	}

	name := (*args)["name"].(string)
	region := (*args)["region"].(string)
	provider, err := awsProvider(a.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, nil, err
	}
	svc := provider.Es(region)
	ctx := context.Background()
	domainDetails, err := svc.DescribeElasticsearchDomain(ctx, &elasticsearchservice.DescribeElasticsearchDomainInput{DomainName: &name})
	if err != nil {
		return nil, nil, err
	}
	tags, err := getESTags(ctx, svc, domainDetails.DomainStatus.ARN)
	if err != nil {
		return nil, nil, err
	}
	(*args)["encryptionAtRestEnabled"] = core.ToBool(domainDetails.DomainStatus.EncryptionAtRestOptions.Enabled)
	(*args)["nodeToNodeEncryptionEnabled"] = core.ToBool(domainDetails.DomainStatus.NodeToNodeEncryptionOptions.Enabled)
	(*args)["endpoint"] = core.ToString(domainDetails.DomainStatus.Endpoint)
	(*args)["arn"] = core.ToString(domainDetails.DomainStatus.ARN)
	(*args)["tags"] = tags
	return args, nil, nil
}

func (e *mqlAwsEsDomain) id() (string, error) {
	return e.Arn()
}

func getESTags(ctx context.Context, svc *elasticsearchservice.Client, arn *string) (map[string]interface{}, error) {
	resp, err := svc.ListTags(ctx, &elasticsearchservice.ListTagsInput{ARN: arn})
	var respErr *http.ResponseError
	if err != nil {
		if errors.As(err, &respErr) {
			if respErr.HTTPStatusCode() == 404 {
				return nil, nil
			}
		}
		return nil, err
	}
	tags := make(map[string]interface{})
	for _, t := range resp.TagList {
		tags[*t.Key] = *t.Value
	}
	return tags, nil
}
