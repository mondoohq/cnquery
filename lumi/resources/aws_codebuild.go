package resources

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/codebuild"
	"go.mondoo.io/mondoo/lumi"
	"go.mondoo.io/mondoo/lumi/library/jobpool"
)

func (c *lumiAwsCodebuild) id() (string, error) {
	return "aws.codebuild", nil
}

func (c *lumiAwsCodebuild) GetProjects() ([]interface{}, error) {
	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(c.getProjects(), 5)
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

func (t *lumiAwsCodebuild) getProjects() []*jobpool.Job {
	var tasks = make([]*jobpool.Job, 0)
	at, err := awstransport(t.Runtime.Motor.Transport)
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}
	regions, err := at.GetRegions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {

			svc := at.Codebuild(regionVal)
			ctx := context.Background()

			res := []interface{}{}
			params := &codebuild.ListProjectsInput{}
			nextToken := aws.String("no_token_to_start_with")
			for nextToken != nil {
				projects, err := svc.ListProjectsRequest(params).Send(ctx)
				if err != nil {
					return nil, err
				}

				for _, project := range projects.Projects {
					lumiProject, err := t.Runtime.CreateResource("aws.codebuild.project",
						"name", project,
						"region", regionVal,
					)
					if err != nil {
						return nil, err
					}
					res = append(res, lumiProject)
				}
				nextToken = projects.NextToken
				if projects.NextToken != nil {
					params.NextToken = nextToken
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}
func (c *lumiAwsCodebuildProject) id() (string, error) {
	return c.Name()
}

func (c *lumiAwsCodebuildProject) init(args *lumi.Args) (*lumi.Args, AwsCodebuildProject, error) {
	if len(*args) > 2 {
		return args, nil, nil
	}

	if (*args)["name"] == nil && (*args)["region"] == nil {
		return nil, nil, errors.New("name and region required to fetch codebuild project")
	}

	name := (*args)["name"].(string)
	region := (*args)["region"].(string)
	at, err := awstransport(c.Runtime.Motor.Transport)
	if err != nil {
		return nil, nil, err
	}
	svc := at.Codebuild(region)
	ctx := context.Background()
	projectDetails, err := svc.BatchGetProjectsRequest(&codebuild.BatchGetProjectsInput{Names: []string{name}}).Send(ctx)
	if err != nil {
		return nil, nil, err
	}
	if len(projectDetails.Projects) == 0 {
		return nil, nil, errors.New("aws codebuild project not found")
	}

	project := projectDetails.Projects[0]
	jsonEnv, err := jsonToDict(project.Environment)
	if err != nil {
		return nil, nil, err
	}
	jsonSource, err := jsonToDict(project.Source)
	if err != nil {
		return nil, nil, err
	}
	(*args)["arn"] = toString(project.Arn)
	(*args)["description"] = toString(project.Description)
	(*args)["environment"] = jsonEnv
	(*args)["source"] = jsonSource
	return args, nil, nil
}
