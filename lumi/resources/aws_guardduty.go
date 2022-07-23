package resources

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/guardduty"
	"github.com/aws/aws-sdk-go-v2/service/guardduty/types"
	"go.mondoo.io/mondoo/lumi"
	"go.mondoo.io/mondoo/lumi/library/jobpool"
	aws_transport "go.mondoo.io/mondoo/motor/transports/aws"
)

func (g *lumiAwsGuardduty) id() (string, error) {
	return "aws.guardduty", nil
}

func (g *lumiAwsGuardduty) GetDetectors() ([]interface{}, error) {
	at, err := awstransport(g.MotorRuntime.Motor.Transport)
	if err != nil {
		return nil, err
	}
	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(g.getDetectors(at), 5)
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

func (g *lumiAwsGuarddutyDetector) id() (string, error) {
	return g.Id()
}

func (g *lumiAwsGuardduty) getDetectors(at *aws_transport.Transport) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := at.GetRegions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			svc := at.Guardduty(regionVal)
			ctx := context.Background()

			res := []interface{}{}
			params := &guardduty.ListDetectorsInput{}

			nextToken := aws.String("no_token_to_start_with")
			for nextToken != nil {
				detectors, err := svc.ListDetectors(ctx, params)
				if err != nil {
					return nil, err
				}

				for _, id := range detectors.DetectorIds {
					lumiCluster, err := g.MotorRuntime.CreateResource("aws.guardduty.detector",
						"id", id,
						"region", regionVal,
					)
					if err != nil {
						return nil, err
					}
					res = append(res, lumiCluster)
				}
				nextToken = detectors.NextToken
				if detectors.NextToken != nil {
					params.NextToken = nextToken
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func (g *lumiAwsGuarddutyDetector) GetUnarchivedFindings() ([]interface{}, error) {
	id, err := g.Id()
	if err != nil {
		return nil, err
	}
	region, err := g.Region()
	if err != nil {
		return nil, err
	}
	at, err := awstransport(g.MotorRuntime.Motor.Transport)
	if err != nil {
		return nil, err
	}
	svc := at.Guardduty(region)
	ctx := context.Background()

	findings, err := svc.ListFindings(ctx, &guardduty.ListFindingsInput{
		DetectorId: &id,
		FindingCriteria: &types.FindingCriteria{
			Criterion: map[string]types.Condition{
				"service.archived": {
					Equals: []string{"false"},
				},
			},
		},
	})
	if err != nil {
		return nil, err
	}
	findingDetails, err := svc.GetFindings(ctx, &guardduty.GetFindingsInput{FindingIds: findings.FindingIds, DetectorId: &id})
	if err != nil {
		return nil, err
	}
	return jsonToDictSlice(findingDetails.Findings)
}

func (g *lumiAwsGuarddutyDetector) init(args *lumi.Args) (*lumi.Args, AwsGuarddutyDetector, error) {
	if len(*args) > 2 {
		return args, nil, nil
	}

	if (*args)["id"] == nil && (*args)["region"] == nil {
		return nil, nil, errors.New("name and region required to fetch codebuild project")
	}

	id := (*args)["id"].(string)
	region := (*args)["region"].(string)
	at, err := awstransport(g.MotorRuntime.Motor.Transport)
	if err != nil {
		return nil, nil, err
	}
	svc := at.Guardduty(region)
	ctx := context.Background()
	detector, err := svc.GetDetector(ctx, &guardduty.GetDetectorInput{DetectorId: &id})
	if err != nil {
		return nil, nil, err
	}

	(*args)["status"] = string(detector.Status)
	(*args)["findingPublishingFrequency"] = string(detector.FindingPublishingFrequency)
	return args, nil, nil
}
