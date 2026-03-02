// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/workdocs"
	workdocstypes "github.com/aws/aws-sdk-go-v2/service/workdocs/types"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/mql/v13/providers/aws/connection"
)

func (a *mqlAwsWorkdocs) id() (string, error) {
	return "aws.workdocs", nil
}

func (a *mqlAwsWorkdocsUser) id() (string, error) {
	return a.Region.Data + "/" + a.Id.Data, nil
}

func (a *mqlAwsWorkdocs) users() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getUsers(conn), 5)
	poolOfJobs.Run()

	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	for i := range poolOfJobs.Jobs {
		res = append(res, poolOfJobs.Jobs[i].Result.([]any)...)
	}

	return res, nil
}

func (a *mqlAwsWorkdocs) getUsers(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			svc := conn.WorkDocs(region)
			ctx := context.Background()
			res := []any{}

			params := &workdocs.DescribeUsersInput{
				Fields:  aws.String("STORAGE_METADATA"),
				Include: workdocstypes.UserFilterTypeAll,
			}
			paginator := workdocs.NewDescribeUsersPaginator(svc, params)
			for paginator.HasMorePages() {
				page, err := paginator.NextPage(ctx)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS WorkDocs API")
						return res, nil
					}
					return nil, err
				}
				for _, user := range page.Users {
					var storageAllocatedInBytes int64
					var storageUtilizedInBytes int64
					var storageType string
					if user.Storage != nil {
						if user.Storage.StorageUtilizedInBytes != nil {
							storageUtilizedInBytes = *user.Storage.StorageUtilizedInBytes
						}
						if user.Storage.StorageRule != nil {
							if user.Storage.StorageRule.StorageAllocatedInBytes != nil {
								storageAllocatedInBytes = *user.Storage.StorageRule.StorageAllocatedInBytes
							}
							storageType = string(user.Storage.StorageRule.StorageType)
						}
					}

					mqlUser, err := CreateResource(a.MqlRuntime, "aws.workdocs.user",
						map[string]*llx.RawData{
							"id":                      llx.StringDataPtr(user.Id),
							"username":                llx.StringDataPtr(user.Username),
							"emailAddress":            llx.StringDataPtr(user.EmailAddress),
							"givenName":               llx.StringDataPtr(user.GivenName),
							"surname":                 llx.StringDataPtr(user.Surname),
							"status":                  llx.StringData(string(user.Status)),
							"userType":                llx.StringData(string(user.Type)),
							"createdTimestamp":        llx.TimeDataPtr(user.CreatedTimestamp),
							"modifiedTimestamp":       llx.TimeDataPtr(user.ModifiedTimestamp),
							"timeZoneId":              llx.StringDataPtr(user.TimeZoneId),
							"locale":                  llx.StringData(string(user.Locale)),
							"organizationId":          llx.StringDataPtr(user.OrganizationId),
							"storageAllocatedInBytes": llx.IntData(storageAllocatedInBytes),
							"storageUtilizedInBytes":  llx.IntData(storageUtilizedInBytes),
							"storageType":             llx.StringData(storageType),
							"recycleBinFolderId":      llx.StringDataPtr(user.RecycleBinFolderId),
							"rootFolderId":            llx.StringDataPtr(user.RootFolderId),
							"region":                  llx.StringData(region),
						},
					)
					if err != nil {
						return nil, err
					}
					res = append(res, mqlUser)
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}
