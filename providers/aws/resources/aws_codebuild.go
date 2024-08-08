// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/codebuild"
	cbtypes "github.com/aws/aws-sdk-go-v2/service/codebuild/types"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/cnquery/v11/providers/aws/connection"
	"go.mondoo.com/cnquery/v11/types"
)

func (a *mqlAwsCodebuild) id() (string, error) {
	return "aws.codebuild", nil
}

func (a *mqlAwsCodebuild) projects() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(a.getProjects(conn), 5)
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

func (a *mqlAwsCodebuild) getProjects(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			svc := conn.Codebuild(regionVal)
			ctx := context.Background()

			res := []interface{}{}
			params := &codebuild.ListProjectsInput{}
			nextToken := aws.String("no_token_to_start_with")
			for nextToken != nil {
				projects, err := svc.ListProjects(ctx, params)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", regionVal).Msg("error accessing region for AWS API")
						return res, nil
					}
					return nil, err
				}

				for _, project := range projects.Projects {
					mqlProject, err := CreateResource(a.MqlRuntime, "aws.codebuild.project",
						map[string]*llx.RawData{
							"name":   llx.StringData(project),
							"region": llx.StringData(regionVal),
						})
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

func (a *mqlAwsCodebuildProject) id() (string, error) {
	return a.Name.Data, nil
}

func initAwsCodebuildProject(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}

	if args["name"] == nil && args["region"] == nil {
		return nil, nil, errors.New("name and region required to fetch codebuild project")
	}

	name := args["name"].Value.(string)
	region := args["region"].Value.(string)
	conn := runtime.Connection.(*connection.AwsConnection)

	svc := conn.Codebuild(region)
	ctx := context.Background()
	projectDetails, err := svc.BatchGetProjects(ctx, &codebuild.BatchGetProjectsInput{Names: []string{name}})
	if err != nil {
		return nil, nil, err
	}
	if len(projectDetails.Projects) == 0 {
		return nil, nil, errors.New("aws codebuild project not found")
	}

	project := projectDetails.Projects[0]
	jsonEnv, err := convert.JsonToDict(project.Environment)
	if err != nil {
		return nil, nil, err
	}
	jsonSource, err := convert.JsonToDict(project.Source)
	if err != nil {
		return nil, nil, err
	}
	args["arn"] = llx.StringData(convert.ToString(project.Arn))
	args["description"] = llx.StringData(convert.ToString(project.Description))
	args["environment"] = llx.MapData(jsonEnv, types.String)
	args["source"] = llx.MapData(jsonSource, types.String)
	args["tags"] = llx.MapData(cbTagsToMap(project.Tags), types.String)
	return args, nil, nil
}

func cbTagsToMap(tags []cbtypes.Tag) map[string]interface{} {
	tagsMap := make(map[string]interface{})

	if len(tags) > 0 {
		for i := range tags {
			tag := tags[i]
			tagsMap[convert.ToString(tag.Key)] = convert.ToString(tag.Value)
		}
	}

	return tagsMap
}
