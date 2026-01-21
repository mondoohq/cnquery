// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/aws/aws-sdk-go-v2/service/drs"
	drstypes "github.com/aws/aws-sdk-go-v2/service/drs/types"
	"github.com/aws/smithy-go/transport/http"
	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v12/llx"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/cnquery/v12/providers/aws/connection"
	"go.mondoo.com/cnquery/v12/types"
)

func (a *mqlAwsDrs) id() (string, error) {
	return "aws.drs", nil
}

func (a *mqlAwsDrsSourceServer) id() (string, error) {
	return a.Arn.Data, nil
}

func (a *mqlAwsDrsJob) id() (string, error) {
	return a.Arn.Data, nil
}

func (a *mqlAwsDrsReplicationConfiguration) id() (string, error) {
	return a.SourceServerID.Data, nil
}

func (a *mqlAwsDrsLaunchConfiguration) id() (string, error) {
	return a.SourceServerID.Data, nil
}

func (a *mqlAwsDrs) sourceServers() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getSourceServers(conn), 5)
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

func (a *mqlAwsDrs) getSourceServers(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			svc := conn.Drs(region)
			ctx := context.Background()
			res := []any{}

			paginator := drs.NewDescribeSourceServersPaginator(svc, &drs.DescribeSourceServersInput{})
			for paginator.HasMorePages() {
				page, err := paginator.NextPage(ctx)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS DRS API")
						return res, nil
					}
					if IsDrsNotInitializedError(err) {
						log.Debug().Str("region", region).Msg("DRS not initialized in region")
						return res, nil
					}
					return nil, err
				}

				for _, server := range page.Items {
					mqlServer, err := a.createSourceServerResource(server, region)
					if err != nil {
						return nil, err
					}
					res = append(res, mqlServer)
				}
			}

			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func (a *mqlAwsDrs) createSourceServerResource(server drstypes.SourceServer, region string) (*mqlAwsDrsSourceServer, error) {
	dataReplicationInfo, err := convert.JsonToDict(server.DataReplicationInfo)
	if err != nil {
		return nil, err
	}

	lifeCycle, err := convert.JsonToDict(server.LifeCycle)
	if err != nil {
		return nil, err
	}

	sourceProperties, err := convert.JsonToDict(server.SourceProperties)
	if err != nil {
		return nil, err
	}

	stagingArea, err := convert.JsonToDict(server.StagingArea)
	if err != nil {
		return nil, err
	}

	tags := make(map[string]interface{})
	for k, v := range server.Tags {
		tags[k] = v
	}

	mqlServer, err := CreateResource(a.MqlRuntime, "aws.drs.sourceServer",
		map[string]*llx.RawData{
			"sourceServerID":       llx.StringDataPtr(server.SourceServerID),
			"arn":                  llx.StringDataPtr(server.Arn),
			"dataReplicationInfo":  llx.DictData(dataReplicationInfo),
			"lastLaunchResult":     llx.StringData(string(server.LastLaunchResult)),
			"lifeCycle":            llx.DictData(lifeCycle),
			"sourceProperties":     llx.DictData(sourceProperties),
			"stagingArea":          llx.DictData(stagingArea),
			"replicationDirection": llx.StringData(string(server.ReplicationDirection)),
			"recoveryInstanceId":   llx.StringDataPtr(server.RecoveryInstanceId),
			"tags":                 llx.MapData(tags, types.String),
		})
	if err != nil {
		return nil, err
	}

	return mqlServer.(*mqlAwsDrsSourceServer), nil
}

func (a *mqlAwsDrsSourceServer) replicationConfiguration() (*mqlAwsDrsReplicationConfiguration, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	sourceServerID := a.SourceServerID.Data
	region, err := GetRegionFromArn(a.Arn.Data)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse source server ARN")
	}

	svc := conn.Drs(region)
	ctx := context.Background()

	resp, err := svc.GetReplicationConfiguration(ctx, &drs.GetReplicationConfigurationInput{
		SourceServerID: &sourceServerID,
	})
	if err != nil {
		return nil, err
	}

	stagingAreaTags := make(map[string]interface{})
	for k, v := range resp.StagingAreaTags {
		stagingAreaTags[k] = v
	}

	replicatedDisks := make([]interface{}, 0, len(resp.ReplicatedDisks))
	for _, disk := range resp.ReplicatedDisks {
		diskMap, err := convert.JsonToDict(disk)
		if err != nil {
			return nil, err
		}
		replicatedDisks = append(replicatedDisks, diskMap)
	}

	mqlConfig, err := CreateResource(a.MqlRuntime, "aws.drs.replicationConfiguration",
		map[string]*llx.RawData{
			"sourceServerID":                llx.StringDataPtr(resp.SourceServerID),
			"stagingAreaSubnetId":           llx.StringDataPtr(resp.StagingAreaSubnetId),
			"stagingAreaTags":               llx.MapData(stagingAreaTags, types.String),
			"useDedicatedReplicationServer": llx.BoolDataPtr(resp.UseDedicatedReplicationServer),
			"replicationServerInstanceType": llx.StringDataPtr(resp.ReplicationServerInstanceType),
			"ebsEncryption":                 llx.StringData(string(resp.EbsEncryption)),
			"ebsEncryptionKeyArn":           llx.StringDataPtr(resp.EbsEncryptionKeyArn),
			"replicatedDisks":               llx.ArrayData(replicatedDisks, types.Dict),
			"bandwidthThrottling":           llx.IntData(int64(resp.BandwidthThrottling)),
		})
	if err != nil {
		return nil, err
	}

	return mqlConfig.(*mqlAwsDrsReplicationConfiguration), nil
}

func (a *mqlAwsDrsSourceServer) launchConfiguration() (*mqlAwsDrsLaunchConfiguration, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	sourceServerID := a.SourceServerID.Data
	region, err := GetRegionFromArn(a.Arn.Data)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse source server ARN")
	}

	svc := conn.Drs(region)
	ctx := context.Background()

	resp, err := svc.GetLaunchConfiguration(ctx, &drs.GetLaunchConfigurationInput{
		SourceServerID: &sourceServerID,
	})
	if err != nil {
		return nil, err
	}

	licensing, err := convert.JsonToDict(resp.Licensing)
	if err != nil {
		return nil, err
	}

	launchIntoInstanceProperties, err := convert.JsonToDict(resp.LaunchIntoInstanceProperties)
	if err != nil {
		return nil, err
	}

	mqlConfig, err := CreateResource(a.MqlRuntime, "aws.drs.launchConfiguration",
		map[string]*llx.RawData{
			"sourceServerID":                      llx.StringDataPtr(resp.SourceServerID),
			"targetInstanceTypeRightSizingMethod": llx.StringData(string(resp.TargetInstanceTypeRightSizingMethod)),
			"launchDisposition":                   llx.StringData(string(resp.LaunchDisposition)),
			"copyPrivateIp":                       llx.BoolDataPtr(resp.CopyPrivateIp),
			"copyTags":                            llx.BoolDataPtr(resp.CopyTags),
			"ec2LaunchTemplateID":                 llx.StringDataPtr(resp.Ec2LaunchTemplateID),
			"licensing":                           llx.DictData(licensing),
			"postLaunchEnabled":                   llx.BoolDataPtr(resp.PostLaunchEnabled),
			"launchIntoInstanceProperties":        llx.DictData(launchIntoInstanceProperties),
		})
	if err != nil {
		return nil, err
	}

	return mqlConfig.(*mqlAwsDrsLaunchConfiguration), nil
}

func (a *mqlAwsDrs) jobs() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getJobs(conn), 5)
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

func (a *mqlAwsDrs) getJobs(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			svc := conn.Drs(region)
			ctx := context.Background()
			res := []any{}

			paginator := drs.NewDescribeJobsPaginator(svc, &drs.DescribeJobsInput{})
			for paginator.HasMorePages() {
				page, err := paginator.NextPage(ctx)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS DRS API")
						return res, nil
					}
					if IsDrsNotInitializedError(err) {
						log.Debug().Str("region", region).Msg("DRS not initialized in region")
						return res, nil
					}
					return nil, err
				}

				for _, job := range page.Items {
					mqlJob, err := a.createJobResource(job, region)
					if err != nil {
						return nil, err
					}
					res = append(res, mqlJob)
				}
			}

			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func (a *mqlAwsDrs) createJobResource(job drstypes.Job, region string) (*mqlAwsDrsJob, error) {
	participatingServers := make([]interface{}, 0, len(job.ParticipatingServers))
	for _, server := range job.ParticipatingServers {
		serverMap, err := convert.JsonToDict(server)
		if err != nil {
			return nil, err
		}
		participatingServers = append(participatingServers, serverMap)
	}

	// Use provided ARN or construct one
	jobArn := ""
	if job.Arn != nil && *job.Arn != "" {
		jobArn = *job.Arn
	} else {
		// Construct ARN for the job if not provided
		accountID := ""
		if job.Arn != nil {
			if parsed, err := arn.Parse(*job.Arn); err == nil {
				accountID = parsed.AccountID
			}
		}
		jobArn = fmt.Sprintf("arn:aws:drs:%s:%s:job/%s", region, accountID, *job.JobID)
	}

	// Parse time strings
	var createdAt *time.Time
	if job.CreationDateTime != nil {
		if t, err := time.Parse(time.RFC3339, *job.CreationDateTime); err == nil {
			createdAt = &t
		}
	}
	var endedAt *time.Time
	if job.EndDateTime != nil {
		if t, err := time.Parse(time.RFC3339, *job.EndDateTime); err == nil {
			endedAt = &t
		}
	}

	mqlJob, err := CreateResource(a.MqlRuntime, "aws.drs.job",
		map[string]*llx.RawData{
			"jobID":                llx.StringDataPtr(job.JobID),
			"arn":                  llx.StringData(jobArn),
			"type":                 llx.StringData(string(job.Type)),
			"status":               llx.StringData(string(job.Status)),
			"initiatedBy":          llx.StringData(string(job.InitiatedBy)),
			"createdAt":            llx.TimeDataPtr(createdAt),
			"endedAt":              llx.TimeDataPtr(endedAt),
			"participatingServers": llx.ArrayData(participatingServers, types.Dict),
		})
	if err != nil {
		return nil, err
	}

	return mqlJob.(*mqlAwsDrsJob), nil
}

// IsDrsNotInitializedError checks if the error indicates DRS is not initialized in the region
func IsDrsNotInitializedError(err error) bool {
	if err == nil {
		return false
	}

	var respErr *http.ResponseError
	if errors.As(err, &respErr) {
		// DRS returns UninitializedAccountException when DRS is not initialized in the region
		errMsg := respErr.Error()
		if strings.Contains(errMsg, "UninitializedAccountException") ||
			strings.Contains(errMsg, "not initialized") ||
			strings.Contains(errMsg, "is not enabled") {
			return true
		}
	}
	return false
}
