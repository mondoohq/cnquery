// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/dsql"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/mql/v13/providers/aws/connection"
)

func (a *mqlAwsDsql) id() (string, error) {
	return "aws.dsql", nil
}

func (a *mqlAwsDsql) clusters() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getClusters(conn), 5)
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

func (a *mqlAwsDsql) getClusters(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("dsql>getClusters>calling aws with region %s", region)

			svc := conn.Dsql(region)
			ctx := context.Background()
			res := []any{}

			paginator := dsql.NewListClustersPaginator(svc, &dsql.ListClustersInput{})
			for paginator.HasMorePages() {
				page, err := paginator.NextPage(ctx)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Debug().Str("region", region).Msg("error accessing region for AWS API")
						return res, nil
					}
					if IsServiceNotAvailableInRegionError(err) {
						log.Debug().Str("region", region).Msg("dsql service not available in region")
						return res, nil
					}
					return nil, err
				}
				for _, cluster := range page.Clusters {
					mqlCluster, err := CreateResource(a.MqlRuntime, "aws.dsql.cluster",
						map[string]*llx.RawData{
							"__id":       llx.StringDataPtr(cluster.Arn),
							"arn":        llx.StringDataPtr(cluster.Arn),
							"identifier": llx.StringDataPtr(cluster.Identifier),
							"region":     llx.StringData(region),
						})
					if err != nil {
						return nil, err
					}
					res = append(res, mqlCluster)
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

type mqlAwsDsqlClusterInternal struct {
	detailOnce sync.Once
	detailErr  error
	detail     *dsql.GetClusterOutput
}

func (a *mqlAwsDsqlCluster) fetchDetail() (*dsql.GetClusterOutput, error) {
	a.detailOnce.Do(func() {
		conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
		svc := conn.Dsql(a.Region.Data)
		ctx := context.Background()
		identifier := a.Identifier.Data
		a.detail, a.detailErr = svc.GetCluster(ctx, &dsql.GetClusterInput{
			Identifier: &identifier,
		})
	})
	return a.detail, a.detailErr
}

func (a *mqlAwsDsqlCluster) status() (string, error) {
	detail, err := a.fetchDetail()
	if err != nil {
		return "", err
	}
	return string(detail.Status), nil
}

func (a *mqlAwsDsqlCluster) createdAt() (*time.Time, error) {
	detail, err := a.fetchDetail()
	if err != nil {
		return nil, err
	}
	return detail.CreationTime, nil
}

func (a *mqlAwsDsqlCluster) deletionProtectionEnabled() (bool, error) {
	detail, err := a.fetchDetail()
	if err != nil {
		return false, err
	}
	return convert.ToValue(detail.DeletionProtectionEnabled), nil
}

func (a *mqlAwsDsqlCluster) endpoint() (string, error) {
	detail, err := a.fetchDetail()
	if err != nil {
		return "", err
	}
	return convert.ToValue(detail.Endpoint), nil
}

func (a *mqlAwsDsqlCluster) encryptionStatus() (string, error) {
	detail, err := a.fetchDetail()
	if err != nil {
		return "", err
	}
	if detail.EncryptionDetails == nil {
		return "", nil
	}
	return string(detail.EncryptionDetails.EncryptionStatus), nil
}

func (a *mqlAwsDsqlCluster) encryptionType() (string, error) {
	detail, err := a.fetchDetail()
	if err != nil {
		return "", err
	}
	if detail.EncryptionDetails == nil {
		return "", nil
	}
	return string(detail.EncryptionDetails.EncryptionType), nil
}

func (a *mqlAwsDsqlCluster) kmsKey() (*mqlAwsKmsKey, error) {
	detail, err := a.fetchDetail()
	if err != nil {
		return nil, err
	}
	if detail.EncryptionDetails == nil || detail.EncryptionDetails.KmsKeyArn == nil || *detail.EncryptionDetails.KmsKeyArn == "" {
		a.KmsKey.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	mqlKey, err := NewResource(a.MqlRuntime, ResourceAwsKmsKey,
		map[string]*llx.RawData{
			"arn": llx.StringDataPtr(detail.EncryptionDetails.KmsKeyArn),
		})
	if err != nil {
		return nil, err
	}
	return mqlKey.(*mqlAwsKmsKey), nil
}

func (a *mqlAwsDsqlCluster) multiRegionPeers() ([]any, error) {
	detail, err := a.fetchDetail()
	if err != nil {
		return nil, err
	}
	if detail.MultiRegionProperties == nil {
		return []any{}, nil
	}
	return convert.SliceAnyToInterface(detail.MultiRegionProperties.Clusters), nil
}

func (a *mqlAwsDsqlCluster) witnessRegion() (string, error) {
	detail, err := a.fetchDetail()
	if err != nil {
		return "", err
	}
	if detail.MultiRegionProperties == nil {
		return "", nil
	}
	return convert.ToValue(detail.MultiRegionProperties.WitnessRegion), nil
}

func (a *mqlAwsDsqlCluster) tags() (map[string]any, error) {
	detail, err := a.fetchDetail()
	if err != nil {
		return nil, err
	}
	tags := make(map[string]any, len(detail.Tags))
	for k, v := range detail.Tags {
		tags[k] = v
	}
	return tags, nil
}
