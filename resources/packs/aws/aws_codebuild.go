package aws

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/codebuild"
	"github.com/aws/aws-sdk-go-v2/service/codebuild/types"
	aws_provider "go.mondoo.com/cnquery/motor/providers/aws"
	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/resources/library/jobpool"
	"go.mondoo.com/cnquery/resources/packs/core"
)

func (c *mqlAwsCodebuild) id() (string, error) {
	return "aws.codebuild", nil
}

func (c *mqlAwsCodebuild) GetProjects() ([]interface{}, error) {
	provider, err := awsProvider(c.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(c.getProjects(provider), 5)
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

func (t *mqlAwsCodebuild) getProjects(provider *aws_provider.Provider) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := provider.GetRegions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			svc := provider.Codebuild(regionVal)
			ctx := context.Background()

			res := []interface{}{}
			params := &codebuild.ListProjectsInput{}
			nextToken := aws.String("no_token_to_start_with")
			for nextToken != nil {
				projects, err := svc.ListProjects(ctx, params)
				if err != nil {
					return nil, err
				}

				for _, project := range projects.Projects {
					mqlProject, err := t.MotorRuntime.CreateResource("aws.codebuild.project",
						"name", project,
						"region", regionVal,
					)
					if err != nil {
						return nil, err
					}
					res = append(res, mqlProject)
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

func (c *mqlAwsCodebuildProject) id() (string, error) {
	return c.Name()
}

func (c *mqlAwsCodebuildProject) init(args *resources.Args) (*resources.Args, AwsCodebuildProject, error) {
	if len(*args) > 2 {
		return args, nil, nil
	}

	if (*args)["name"] == nil && (*args)["region"] == nil {
		return nil, nil, errors.New("name and region required to fetch codebuild project")
	}

	name := (*args)["name"].(string)
	region := (*args)["region"].(string)
	provider, err := awsProvider(c.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, nil, err
	}
	svc := provider.Codebuild(region)
	ctx := context.Background()
	projectDetails, err := svc.BatchGetProjects(ctx, &codebuild.BatchGetProjectsInput{Names: []string{name}})
	if err != nil {
		return nil, nil, err
	}
	if len(projectDetails.Projects) == 0 {
		return nil, nil, errors.New("aws codebuild project not found")
	}

	project := projectDetails.Projects[0]
	jsonEnv, err := core.JsonToDict(project.Environment)
	if err != nil {
		return nil, nil, err
	}
	jsonSource, err := core.JsonToDict(project.Source)
	if err != nil {
		return nil, nil, err
	}
	(*args)["arn"] = core.ToString(project.Arn)
	(*args)["description"] = core.ToString(project.Description)
	(*args)["environment"] = jsonEnv
	(*args)["source"] = jsonSource
	(*args)["tags"] = cbTagsToMap(project.Tags)
	return args, nil, nil
}

func cbTagsToMap(tags []types.Tag) map[string]interface{} {
	tagsMap := make(map[string]interface{})

	if len(tags) > 0 {
		for i := range tags {
			tag := tags[i]
			tagsMap[core.ToString(tag.Key)] = core.ToString(tag.Value)
		}
	}

	return tagsMap
}
