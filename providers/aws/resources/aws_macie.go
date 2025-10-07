// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/macie2"
	"github.com/aws/aws-sdk-go-v2/service/macie2/types"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v12/llx"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/cnquery/v12/providers/aws/connection"
)

func (a *mqlAwsMacie) id() (string, error) {
	return ResourceAwsMacie, nil
}

func (a *mqlAwsMacie) sessions() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getSessions(conn), 5)
	poolOfJobs.Run()

	// check for errors
	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	// get all the results
	for i := range poolOfJobs.Jobs {
		if poolOfJobs.Jobs[i].Result != nil {
			res = append(res, poolOfJobs.Jobs[i].Result.([]any)...)
		}
	}
	return res, nil
}

func (a *mqlAwsMacie) getSessions(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			svc := conn.Macie2(region)
			ctx := context.Background()
			res := []any{}

			session, err := svc.GetMacieSession(ctx, &macie2.GetMacieSessionInput{})
			if err != nil {
				if Is400AccessDeniedError(err) {
					log.Warn().Str("region", region).Msg("error accessing region for AWS API")
					return res, nil
				}
				var notFoundErr *types.ResourceNotFoundException
				if errors.As(err, &notFoundErr) {
					return nil, nil
				}
				return nil, err
			}

			// Get bucket statistics for S3 bucket count
			bucketStats, err := svc.GetBucketStatistics(ctx, &macie2.GetBucketStatisticsInput{})
			var s3BucketCount int
			if err == nil && bucketStats.BucketCount != nil {
				s3BucketCount = int(*bucketStats.BucketCount)
			}

			mqlSession, err := CreateResource(a.MqlRuntime, ResourceAwsMacieSession,
				map[string]*llx.RawData{
					"arn":                        llx.StringData(generateMacieSessionArn(conn.AccountId(), region)),
					"region":                     llx.StringData(region),
					"status":                     llx.StringData(string(session.Status)),
					"createdAt":                  llx.TimeDataPtr(session.CreatedAt),
					"updatedAt":                  llx.TimeDataPtr(session.UpdatedAt),
					"findingPublishingFrequency": llx.StringData(string(session.FindingPublishingFrequency)),
					"serviceRole":                llx.StringDataPtr(session.ServiceRole),
					"s3BucketCount":              llx.IntData(int64(s3BucketCount)),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlSession)
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func (a *mqlAwsMacie) classificationJobs() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getClassificationJobs(conn), 5)
	poolOfJobs.Run()

	// check for errors
	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	// get all the results
	for i := range poolOfJobs.Jobs {
		if poolOfJobs.Jobs[i].Result != nil {
			res = append(res, poolOfJobs.Jobs[i].Result.([]any)...)
		}
	}
	return res, nil
}

func (a *mqlAwsMacie) getClassificationJobs(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			svc := conn.Macie2(region)
			ctx := context.Background()
			res := []any{}

			params := &macie2.ListClassificationJobsInput{}
			paginator := macie2.NewListClassificationJobsPaginator(svc, params)
			for paginator.HasMorePages() {
				jobs, err := paginator.NextPage(ctx)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS API")
						return res, nil
					}
					return nil, err
				}

				for _, job := range jobs.Items {
					jobId := ""
					if job.JobId != nil {
						jobId = *job.JobId
					}
					jobArn := generateClassificationJobArn(conn.AccountId(), region, jobId)
					mqlJob, err := CreateResource(a.MqlRuntime, ResourceAwsMacieClassificationJob,
						map[string]*llx.RawData{
							"arn":       llx.StringData(jobArn),
							"jobId":     llx.StringDataPtr(job.JobId),
							"name":      llx.StringDataPtr(job.Name),
							"region":    llx.StringData(region),
							"status":    llx.StringData(string(job.JobStatus)),
							"jobType":   llx.StringData(string(job.JobType)),
							"createdAt": llx.TimeDataPtr(job.CreatedAt),
						})
					if err != nil {
						return nil, err
					}
					mqlJob.(*mqlAwsMacieClassificationJob).cacheJob = &job
					res = append(res, mqlJob)
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func (a *mqlAwsMacie) findings() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getFindings(conn), 5)
	poolOfJobs.Run()

	// check for errors
	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	// get all the results
	for i := range poolOfJobs.Jobs {
		if poolOfJobs.Jobs[i].Result != nil {
			res = append(res, poolOfJobs.Jobs[i].Result.([]any)...)
		}
	}
	return res, nil
}

func (a *mqlAwsMacie) getFindings(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			svc := conn.Macie2(region)
			ctx := context.Background()
			res := []any{}

			params := &macie2.ListFindingsInput{}
			paginator := macie2.NewListFindingsPaginator(svc, params)
			for paginator.HasMorePages() {
				findings, err := paginator.NextPage(ctx)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS API")
						return res, nil
					}
					return nil, err
				}

				// Get finding details for all finding IDs
				if len(findings.FindingIds) > 0 {
					detailsRes, err := fetchMacieFindings(svc, region, findings.FindingIds, a.MqlRuntime)
					if err != nil {
						return nil, err
					}
					res = append(res, detailsRes...)
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func (a *mqlAwsMacie) customDataIdentifiers() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getCustomDataIdentifiers(conn), 5)
	poolOfJobs.Run()

	// check for errors
	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	// get all the results
	for i := range poolOfJobs.Jobs {
		if poolOfJobs.Jobs[i].Result != nil {
			res = append(res, poolOfJobs.Jobs[i].Result.([]any)...)
		}
	}
	return res, nil
}

func (a *mqlAwsMacie) getCustomDataIdentifiers(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			svc := conn.Macie2(region)
			ctx := context.Background()
			res := []any{}

			params := &macie2.ListCustomDataIdentifiersInput{}
			paginator := macie2.NewListCustomDataIdentifiersPaginator(svc, params)
			for paginator.HasMorePages() {
				identifiers, err := paginator.NextPage(ctx)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS API")
						return res, nil
					}
					return nil, err
				}

				for _, identifier := range identifiers.Items {
					mqlIdentifier, err := CreateResource(a.MqlRuntime, ResourceAwsMacieCustomDataIdentifier,
						map[string]*llx.RawData{
							"id":        llx.StringDataPtr(identifier.Id),
							"arn":       llx.StringDataPtr(identifier.Arn),
							"name":      llx.StringDataPtr(identifier.Name),
							"createdAt": llx.TimeDataPtr(identifier.CreatedAt),
						})
					if err != nil {
						return nil, err
					}
					mqlIdentifier.(*mqlAwsMacieCustomDataIdentifier).cacheIdentifier = &identifier
					res = append(res, mqlIdentifier)
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

// Resource ID implementations
func (a *mqlAwsMacieSession) id() (string, error) {
	return a.Arn.Data, nil
}

// Field implementations for Macie session
func (a *mqlAwsMacieSession) findingPublishingFrequency() (string, error) {
	return a.FindingPublishingFrequency.Data, nil
}

func (a *mqlAwsMacieSession) serviceRole() (string, error) {
	return a.ServiceRole.Data, nil
}

func (a *mqlAwsMacieSession) s3BucketCount() (int64, error) {
	return a.S3BucketCount.Data, nil
}

func (a *mqlAwsMacieClassificationJob) id() (string, error) {
	return a.Arn.Data, nil
}

func (a *mqlAwsMacieFinding) id() (string, error) {
	return a.Id.Data, nil
}

// Field implementations for Macie finding
func (a *mqlAwsMacieFinding) classificationDetails() (any, error) {
	return a.ClassificationDetails.Data, nil
}

func (a *mqlAwsMacieFinding) resourcesAffected() (any, error) {
	return a.ResourcesAffected.Data, nil
}

func (a *mqlAwsMacieCustomDataIdentifier) id() (string, error) {
	return a.Id.Data, nil
}

// Internal cache structures
type mqlAwsMacieClassificationJobInternal struct {
	cacheJob *types.JobSummary
}

type mqlAwsMacieCustomDataIdentifierInternal struct {
	cacheIdentifier *types.CustomDataIdentifierSummary
}

// Populate detailed data for classification job
func (a *mqlAwsMacieClassificationJob) populateJobDetails() error {
	if a.cacheJob != nil {
		return nil
	}

	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	region := a.Region.Data
	jobId := a.JobId.Data

	svc := conn.Macie2(region)
	ctx := context.Background()

	job, err := svc.DescribeClassificationJob(ctx, &macie2.DescribeClassificationJobInput{
		JobId: &jobId,
	})
	if err != nil {
		return err
	}

	// Set optional fields if available
	if job.LastRunTime != nil {
		a.LastRunTime = plugin.TValue[*time.Time]{Data: job.LastRunTime, State: plugin.StateIsSet}
	}
	if job.SamplingPercentage != nil {
		a.SamplingPercentage = plugin.TValue[int64]{Data: int64(*job.SamplingPercentage), State: plugin.StateIsSet}
	}
	if job.S3JobDefinition != nil && job.S3JobDefinition.BucketDefinitions != nil {
		bucketDefs, _ := convert.JsonToDictSlice(job.S3JobDefinition.BucketDefinitions)
		a.BucketDefinitions = plugin.TValue[[]any]{Data: bucketDefs, State: plugin.StateIsSet}
	}
	if job.ScheduleFrequency != nil {
		scheduleFreq, _ := convert.JsonToDict(job.ScheduleFrequency)
		a.ScheduleFrequency = plugin.TValue[any]{Data: scheduleFreq, State: plugin.StateIsSet}
	}
	if job.Statistics != nil {
		stats, _ := convert.JsonToDict(job.Statistics)
		a.Statistics = plugin.TValue[any]{Data: stats, State: plugin.StateIsSet}
	}
	if job.Tags != nil {
		a.Tags = plugin.TValue[map[string]any]{Data: convert.MapToInterfaceMap(job.Tags), State: plugin.StateIsSet}
	}

	return nil
}

// Populate detailed data for custom data identifier
func (a *mqlAwsMacieCustomDataIdentifier) populateIdentifierDetails() error {
	if a.cacheIdentifier == nil {
		return nil
	}

	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	identifierId := a.Id.Data

	// Note: We need to determine the region from somewhere - Macie custom data identifiers are global
	// For now, we'll try us-east-1 as the default region
	svc := conn.Macie2("us-east-1")
	ctx := context.Background()

	identifier, err := svc.GetCustomDataIdentifier(ctx, &macie2.GetCustomDataIdentifierInput{
		Id: &identifierId,
	})
	if err != nil {
		return err
	}

	// Set optional fields if available
	if identifier.Description != nil {
		a.Description = plugin.TValue[string]{Data: *identifier.Description, State: plugin.StateIsSet}
	}
	if identifier.Regex != nil {
		a.Regex = plugin.TValue[string]{Data: *identifier.Regex, State: plugin.StateIsSet}
	}
	if identifier.Keywords != nil {
		keywords := make([]any, len(identifier.Keywords))
		for i, kw := range identifier.Keywords {
			keywords[i] = kw
		}
		a.Keywords = plugin.TValue[[]any]{Data: keywords, State: plugin.StateIsSet}
	}
	if identifier.Tags != nil {
		a.Tags = plugin.TValue[map[string]any]{Data: convert.MapToInterfaceMap(identifier.Tags), State: plugin.StateIsSet}
	}

	return nil
}

// Field implementations for classification job
func (a *mqlAwsMacieClassificationJob) lastRunTime() (*time.Time, error) {
	return nil, a.populateJobDetails()
}

func (a *mqlAwsMacieClassificationJob) samplingPercentage() (int64, error) {
	return 0, a.populateJobDetails()
}

func (a *mqlAwsMacieClassificationJob) bucketDefinitions() ([]any, error) {
	return nil, a.populateJobDetails()
}

func (a *mqlAwsMacieClassificationJob) scheduleFrequency() (any, error) {
	return nil, a.populateJobDetails()
}

func (a *mqlAwsMacieClassificationJob) statistics() (any, error) {
	return nil, a.populateJobDetails()
}

func (a *mqlAwsMacieClassificationJob) tags() (map[string]any, error) {
	return nil, a.populateJobDetails()
}

// Field implementations for custom data identifier
func (a *mqlAwsMacieCustomDataIdentifier) description() (string, error) {
	return "", a.populateIdentifierDetails()
}

func (a *mqlAwsMacieCustomDataIdentifier) regex() (string, error) {
	return "", a.populateIdentifierDetails()
}

func (a *mqlAwsMacieCustomDataIdentifier) keywords() ([]any, error) {
	return nil, a.populateIdentifierDetails()
}

func (a *mqlAwsMacieCustomDataIdentifier) tags() (map[string]any, error) {
	return nil, a.populateIdentifierDetails()
}

// Helper functions
func fetchMacieFindings(svc *macie2.Client, region string, findingIds []string, runtime *plugin.Runtime) ([]any, error) {
	res := []any{}
	ctx := context.Background()

	// Process findings in chunks of 50 (API limit)
	for i := 0; i < len(findingIds); i += 50 {
		end := i + 50
		if end > len(findingIds) {
			end = len(findingIds)
		}
		chunk := findingIds[i:end]

		findingDetails, err := svc.GetFindings(ctx, &macie2.GetFindingsInput{
			FindingIds: chunk,
		})
		if err != nil {
			return nil, err
		}

		for _, finding := range findingDetails.Findings {
			mqlFinding, err := newMqlMacieFinding(runtime, finding, region)
			if err != nil {
				return nil, err
			}
			res = append(res, mqlFinding)
		}
	}
	return res, nil
}

func newMqlMacieFinding(runtime *plugin.Runtime, finding types.Finding, region string) (*mqlAwsMacieFinding, error) {
	var severity any
	if finding.Severity != nil {
		severityDict, _ := convert.JsonToDict(finding.Severity)
		severity = severityDict
	}

	var classificationDetails any
	if finding.ClassificationDetails != nil {
		classificationDict, _ := convert.JsonToDict(finding.ClassificationDetails)
		classificationDetails = classificationDict
	}

	var resourcesAffected any
	if finding.ResourcesAffected != nil {
		resourcesDict, _ := convert.JsonToDict(finding.ResourcesAffected)
		resourcesAffected = resourcesDict
	}

	accountId := ""
	if finding.AccountId != nil {
		accountId = *finding.AccountId
	}
	findingId := ""
	if finding.Id != nil {
		findingId = *finding.Id
	}
	findingArn := generateFindingArn(accountId, region, findingId)
	res, err := CreateResource(runtime, ResourceAwsMacieFinding, map[string]*llx.RawData{
		"id":                    llx.StringDataPtr(finding.Id),
		"arn":                   llx.StringData(findingArn),
		"region":                llx.StringData(region),
		"accountId":             llx.StringDataPtr(finding.AccountId),
		"type":                  llx.StringData(string(finding.Type)),
		"severity":              llx.DictData(severity),
		"category":              llx.StringData(string(finding.Category)),
		"archived":              llx.BoolDataPtr(finding.Archived),
		"count":                 llx.IntData(int64(*finding.Count)),
		"createdAt":             llx.TimeDataPtr(finding.CreatedAt),
		"updatedAt":             llx.TimeDataPtr(finding.UpdatedAt),
		"title":                 llx.StringDataPtr(finding.Title),
		"description":           llx.StringDataPtr(finding.Description),
		"classificationDetails": llx.DictData(classificationDetails),
		"resourcesAffected":     llx.DictData(resourcesAffected),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAwsMacieFinding), nil
}

func generateMacieSessionArn(accountId, region string) string {
	return "arn:aws:macie2:" + region + ":" + accountId + ":session"
}

func generateClassificationJobArn(accountId, region, jobId string) string {
	return "arn:aws:macie2:" + region + ":" + accountId + ":classification-job/" + jobId
}

func generateFindingArn(accountId, region, findingId string) string {
	return "arn:aws:macie2:" + region + ":" + accountId + ":finding/" + findingId
}
