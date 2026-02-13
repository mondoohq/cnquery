// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go-v2/service/codebuild"
	cbtypes "github.com/aws/aws-sdk-go-v2/service/codebuild/types"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/mql/v13/providers/aws/connection"
	"go.mondoo.com/mql/v13/types"
)

func (a *mqlAwsCodebuild) id() (string, error) {
	return "aws.codebuild", nil
}

func (a *mqlAwsCodebuild) projects() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getProjects(conn), 5)
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

func (a *mqlAwsCodebuild) getProjects(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			svc := conn.Codebuild(region)
			ctx := context.Background()

			res := []any{}
			params := &codebuild.ListProjectsInput{}
			paginator := codebuild.NewListProjectsPaginator(svc, params)
			for paginator.HasMorePages() {
				projects, err := paginator.NextPage(ctx)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS API")
						return res, nil
					}
					return nil, err
				}

				for _, project := range projects.Projects {
					mqlProject, err := CreateResource(a.MqlRuntime, "aws.codebuild.project",
						map[string]*llx.RawData{
							"name":   llx.StringData(project),
							"region": llx.StringData(region),
						})
					if err != nil {
						return nil, err
					}
					res = append(res, mqlProject)
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
	args["arn"] = llx.StringDataPtr(project.Arn)
	args["description"] = llx.StringDataPtr(project.Description)
	args["environment"] = llx.MapData(jsonEnv, types.String)
	args["source"] = llx.MapData(jsonSource, types.String)
	args["tags"] = llx.MapData(cbTagsToMap(project.Tags), types.String)
	return args, nil, nil
}

func cbTagsToMap(tags []cbtypes.Tag) map[string]any {
	tagsMap := make(map[string]any)

	if len(tags) > 0 {
		for i := range tags {
			tag := tags[i]
			tagsMap[convert.ToValue(tag.Key)] = convert.ToValue(tag.Value)
		}
	}

	return tagsMap
}
