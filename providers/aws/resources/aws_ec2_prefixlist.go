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

func (a *mqlAwsEc2ManagedPrefixList) id() (string, error) {
	return a.Arn.Data, nil
}

func (a *mqlAwsEc2ManagedPrefixListEntry) id() (string, error) {
	return a.Cidr.Data, nil
}

type mqlAwsEc2ManagedPrefixListInternal struct {
	region string
}

func (a *mqlAwsEc2) managedPrefixLists() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getManagedPrefixLists(conn), 5)
	poolOfJobs.Run()
	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	for i := range poolOfJobs.Jobs {
		res = append(res, poolOfJobs.Jobs[i].Result.([]any)...)
	}
	return res, nil
}

func (a *mqlAwsEc2) getManagedPrefixLists(conn *connection.AwsConnection) []*jobpool.Job {
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

			paginator := ec2.NewDescribeManagedPrefixListsPaginator(svc, &ec2.DescribeManagedPrefixListsInput{})
			for paginator.HasMorePages() {
				page, err := paginator.NextPage(ctx)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS API")
						return res, nil
					}
					return nil, err
				}
				for _, pl := range page.PrefixLists {
					if conn.Filters.General.MatchesExcludeTags(ec2TagsToMap(pl.Tags)) {
						continue
					}

					var version int64
					if pl.Version != nil {
						version = *pl.Version
					}
					var maxEntries int64
					if pl.MaxEntries != nil {
						maxEntries = int64(*pl.MaxEntries)
					}

					plArn := convert.ToValue(pl.PrefixListArn)
					if plArn == "" {
						plArn = fmt.Sprintf("arn:aws:ec2:%s:%s:prefix-list/%s", region, conn.AccountId(), convert.ToValue(pl.PrefixListId))
					}

					mqlPl, err := CreateResource(a.MqlRuntime, ResourceAwsEc2ManagedPrefixList,
						map[string]*llx.RawData{
							"id":            llx.StringData(convert.ToValue(pl.PrefixListId)),
							"arn":           llx.StringData(plArn),
							"name":          llx.StringData(convert.ToValue(pl.PrefixListName)),
							"region":        llx.StringData(region),
							"state":         llx.StringData(string(pl.State)),
							"addressFamily": llx.StringData(convert.ToValue(pl.AddressFamily)),
							"maxEntries":    llx.IntData(maxEntries),
							"version":       llx.IntData(version),
							"ownerId":       llx.StringData(convert.ToValue(pl.OwnerId)),
							"tags":          llx.MapData(toInterfaceMap(ec2TagsToMap(pl.Tags)), types.String),
						})
					if err != nil {
						return nil, err
					}
					mqlPl.(*mqlAwsEc2ManagedPrefixList).region = region
					res = append(res, mqlPl)
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func (a *mqlAwsEc2ManagedPrefixList) entries() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	svc := conn.Ec2(a.region)
	ctx := context.Background()

	plId := a.Id.Data
	entries := []any{}

	paginator := ec2.NewGetManagedPrefixListEntriesPaginator(svc, &ec2.GetManagedPrefixListEntriesInput{
		PrefixListId: &plId,
	})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			if Is400AccessDeniedError(err) {
				return entries, nil
			}
			return nil, err
		}
		for _, entry := range page.Entries {
			cidr := convert.ToValue(entry.Cidr)
			mqlEntry, err := CreateResource(a.MqlRuntime, ResourceAwsEc2ManagedPrefixListEntry,
				map[string]*llx.RawData{
					"__id":        llx.StringData(plId + "/" + cidr),
					"cidr":        llx.StringData(cidr),
					"description": llx.StringData(convert.ToValue(entry.Description)),
				})
			if err != nil {
				return nil, err
			}
			entries = append(entries, mqlEntry)
		}
	}
	return entries, nil
}
