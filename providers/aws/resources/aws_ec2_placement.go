// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/mql/v13/providers/aws/connection"
	"go.mondoo.com/mql/v13/types"
)

// Placement Groups

func (a *mqlAwsEc2) placementGroups() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getPlacementGroups(conn), 5)
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

func (a *mqlAwsEc2) getPlacementGroups(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}
	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			svc := conn.Ec2(region)
			ctx := context.Background()
			res := []any{}

			resp, err := svc.DescribePlacementGroups(ctx, &ec2.DescribePlacementGroupsInput{})
			if err != nil {
				if Is400AccessDeniedError(err) {
					log.Warn().Str("region", region).Msg("error accessing region for AWS API")
					return res, nil
				}
				return nil, err
			}

			for _, pg := range resp.PlacementGroups {
				tags := ec2TagsToMap(pg.Tags)
				if conn.Filters.General.MatchesExcludeTags(tags) {
					continue
				}
				arn := fmt.Sprintf("arn:aws:ec2:%s:%s:placement-group/%s", region, conn.AccountId(), convert.ToValue(pg.GroupName))

				var partitionCount int64
				if pg.PartitionCount != nil {
					partitionCount = int64(*pg.PartitionCount)
				}

				mqlPg, err := CreateResource(a.MqlRuntime, "aws.ec2.placementGroup",
					map[string]*llx.RawData{
						"__id":           llx.StringData(arn),
						"name":           llx.StringDataPtr(pg.GroupName),
						"id":             llx.StringDataPtr(pg.GroupId),
						"arn":            llx.StringData(arn),
						"region":         llx.StringData(region),
						"strategy":       llx.StringData(string(pg.Strategy)),
						"state":          llx.StringData(string(pg.State)),
						"partitionCount": llx.IntData(partitionCount),
						"groupId":        llx.StringDataPtr(pg.GroupId),
						"tags":           llx.MapData(toInterfaceMap(tags), types.String),
					})
				if err != nil {
					return nil, err
				}
				res = append(res, mqlPg)
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func (a *mqlAwsEc2PlacementGroup) id() (string, error) {
	return a.Arn.Data, nil
}

// Capacity Reservations

func (a *mqlAwsEc2) capacityReservations() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getCapacityReservations(conn), 5)
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

func (a *mqlAwsEc2) getCapacityReservations(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}
	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			svc := conn.Ec2(region)
			ctx := context.Background()
			res := []any{}

			paginator := ec2.NewDescribeCapacityReservationsPaginator(svc, &ec2.DescribeCapacityReservationsInput{})
			for paginator.HasMorePages() {
				page, err := paginator.NextPage(ctx)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS API")
						return res, nil
					}
					return nil, err
				}

				for _, cr := range page.CapacityReservations {
					tags := ec2TagsToMap(cr.Tags)
					if conn.Filters.General.MatchesExcludeTags(tags) {
						continue
					}
					arn := fmt.Sprintf("arn:aws:ec2:%s:%s:capacity-reservation/%s", region, conn.AccountId(), convert.ToValue(cr.CapacityReservationId))

					mqlCr, err := CreateResource(a.MqlRuntime, "aws.ec2.capacityReservation",
						map[string]*llx.RawData{
							"__id":                   llx.StringData(arn),
							"id":                     llx.StringDataPtr(cr.CapacityReservationId),
							"arn":                    llx.StringData(arn),
							"region":                 llx.StringData(region),
							"instanceType":           llx.StringDataPtr(cr.InstanceType),
							"instancePlatform":       llx.StringData(string(cr.InstancePlatform)),
							"availabilityZone":       llx.StringDataPtr(cr.AvailabilityZone),
							"totalInstanceCount":     llx.IntDataDefault(cr.TotalInstanceCount, 0),
							"availableInstanceCount": llx.IntDataDefault(cr.AvailableInstanceCount, 0),
							"state":                  llx.StringData(string(cr.State)),
							"instanceMatchCriteria":  llx.StringData(string(cr.InstanceMatchCriteria)),
							"endDateType":            llx.StringData(string(cr.EndDateType)),
							"tenancy":                llx.StringData(string(cr.Tenancy)),
							"ebsOptimized":           llx.BoolData(convert.ToValue(cr.EbsOptimized)),
							"ephemeralStorage":       llx.BoolData(convert.ToValue(cr.EphemeralStorage)),
							"createdAt":              llx.TimeDataPtr(cr.CreateDate),
							"tags":                   llx.MapData(toInterfaceMap(tags), types.String),
						})
					if err != nil {
						return nil, err
					}
					res = append(res, mqlCr)
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func (a *mqlAwsEc2CapacityReservation) id() (string, error) {
	return a.Arn.Data, nil
}

// Instance Connect Endpoints

func (a *mqlAwsEc2) instanceConnectEndpoints() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getInstanceConnectEndpoints(conn), 5)
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

func (a *mqlAwsEc2) getInstanceConnectEndpoints(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}
	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			svc := conn.Ec2(region)
			ctx := context.Background()
			res := []any{}

			paginator := ec2.NewDescribeInstanceConnectEndpointsPaginator(svc, &ec2.DescribeInstanceConnectEndpointsInput{})
			for paginator.HasMorePages() {
				page, err := paginator.NextPage(ctx)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS API")
						return res, nil
					}
					if IsServiceNotAvailableInRegionError(err) {
						return res, nil
					}
					return nil, err
				}

				for _, ep := range page.InstanceConnectEndpoints {
					tags := ec2TagsToMap(ep.Tags)
					if conn.Filters.General.MatchesExcludeTags(tags) {
						continue
					}

					sgIds := make([]any, 0, len(ep.SecurityGroupIds))
					for _, sg := range ep.SecurityGroupIds {
						sgIds = append(sgIds, sg)
					}

					mqlEp, err := CreateResource(a.MqlRuntime, "aws.ec2.instanceConnectEndpoint",
						map[string]*llx.RawData{
							"__id":             llx.StringDataPtr(ep.InstanceConnectEndpointArn),
							"id":               llx.StringDataPtr(ep.InstanceConnectEndpointId),
							"arn":              llx.StringDataPtr(ep.InstanceConnectEndpointArn),
							"region":           llx.StringData(region),
							"state":            llx.StringData(string(ep.State)),
							"subnetId":         llx.StringDataPtr(ep.SubnetId),
							"vpcId":            llx.StringDataPtr(ep.VpcId),
							"securityGroupIds": llx.ArrayData(sgIds, types.String),
							"preserveClientIp": llx.BoolData(convert.ToValue(ep.PreserveClientIp)),
							"dnsName":          llx.StringDataPtr(ep.DnsName),
							"fipsDnsName":      llx.StringDataPtr(ep.FipsDnsName),
							"createdAt":        llx.TimeDataPtr(ep.CreatedAt),
							"tags":             llx.MapData(toInterfaceMap(tags), types.String),
						})
					if err != nil {
						return nil, err
					}
					res = append(res, mqlEp)
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func (a *mqlAwsEc2InstanceConnectEndpoint) id() (string, error) {
	return a.Arn.Data, nil
}
