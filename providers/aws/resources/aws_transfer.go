// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"sync"

	"github.com/aws/aws-sdk-go-v2/service/transfer"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/mql/v13/providers/aws/connection"
)

func (a *mqlAwsTransfer) id() (string, error) {
	return "aws.transfer", nil
}

func (a *mqlAwsTransfer) servers() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getServers(conn), 5)
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

func (a *mqlAwsTransfer) getServers(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("transfer>getServers>calling aws with region %s", region)

			svc := conn.Transfer(region)
			ctx := context.Background()
			res := []any{}

			paginator := transfer.NewListServersPaginator(svc, &transfer.ListServersInput{})
			for paginator.HasMorePages() {
				page, err := paginator.NextPage(ctx)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS API")
						return res, nil
					}
					if IsServiceNotAvailableInRegionError(err) {
						log.Warn().Str("region", region).Msg("Transfer Family is not available in region")
						return res, nil
					}
					return nil, err
				}

				for _, server := range page.Servers {
					mqlServer, err := CreateResource(a.MqlRuntime, "aws.transfer.server",
						map[string]*llx.RawData{
							"__id":         llx.StringDataPtr(server.Arn),
							"arn":          llx.StringDataPtr(server.Arn),
							"serverId":     llx.StringDataPtr(server.ServerId),
							"region":       llx.StringData(region),
							"endpointType": llx.StringData(string(server.EndpointType)),
							"state":        llx.StringData(string(server.State)),
							"userCount":    llx.IntData(int64(convert.ToValue(server.UserCount))),
						})
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

func (a *mqlAwsTransferServer) id() (string, error) {
	return a.Arn.Data, nil
}

type mqlAwsTransferServerInternal struct {
	fetched  bool
	lock     sync.Mutex
	descResp *transfer.DescribeServerOutput
}

func (a *mqlAwsTransferServer) fetchDetail() (*transfer.DescribeServerOutput, error) {
	if a.fetched {
		return a.descResp, nil
	}
	a.lock.Lock()
	defer a.lock.Unlock()
	if a.fetched {
		return a.descResp, nil
	}

	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	region := a.Region.Data
	svc := conn.Transfer(region)
	ctx := context.Background()

	serverId := a.ServerId.Data
	resp, err := svc.DescribeServer(ctx, &transfer.DescribeServerInput{
		ServerId: &serverId,
	})
	if err != nil {
		return nil, err
	}
	a.fetched = true
	a.descResp = resp
	return resp, nil
}

func (a *mqlAwsTransferServer) domain() (string, error) {
	resp, err := a.fetchDetail()
	if err != nil {
		return "", err
	}
	return string(resp.Server.Domain), nil
}

func (a *mqlAwsTransferServer) identityProviderType() (string, error) {
	resp, err := a.fetchDetail()
	if err != nil {
		return "", err
	}
	return string(resp.Server.IdentityProviderType), nil
}

func (a *mqlAwsTransferServer) loggingRole() (*mqlAwsIamRole, error) {
	resp, err := a.fetchDetail()
	if err != nil {
		return nil, err
	}
	if resp.Server.LoggingRole == nil || *resp.Server.LoggingRole == "" {
		a.LoggingRole.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	mqlRole, err := NewResource(a.MqlRuntime, "aws.iam.role",
		map[string]*llx.RawData{
			"arn": llx.StringDataPtr(resp.Server.LoggingRole),
		})
	if err != nil {
		return nil, err
	}
	return mqlRole.(*mqlAwsIamRole), nil
}

func (a *mqlAwsTransferServer) protocols() ([]any, error) {
	resp, err := a.fetchDetail()
	if err != nil {
		return nil, err
	}
	protocols := make([]any, 0, len(resp.Server.Protocols))
	for _, p := range resp.Server.Protocols {
		protocols = append(protocols, string(p))
	}
	return protocols, nil
}

func (a *mqlAwsTransferServer) ipAddressType() (string, error) {
	resp, err := a.fetchDetail()
	if err != nil {
		return "", err
	}
	return string(resp.Server.IpAddressType), nil
}

func (a *mqlAwsTransferServer) securityPolicyName() (string, error) {
	resp, err := a.fetchDetail()
	if err != nil {
		return "", err
	}
	return convert.ToValue(resp.Server.SecurityPolicyName), nil
}

func (a *mqlAwsTransferServer) certificate() (*mqlAwsAcmCertificate, error) {
	resp, err := a.fetchDetail()
	if err != nil {
		return nil, err
	}
	certArn := convert.ToValue(resp.Server.Certificate)
	if certArn == "" {
		a.Certificate.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	mqlCert, err := NewResource(a.MqlRuntime, "aws.acm.certificate",
		map[string]*llx.RawData{
			"arn": llx.StringData(certArn),
		})
	if err != nil {
		return nil, err
	}
	return mqlCert.(*mqlAwsAcmCertificate), nil
}

func (a *mqlAwsTransferServer) tags() (map[string]any, error) {
	resp, err := a.fetchDetail()
	if err != nil {
		return nil, err
	}
	tags := make(map[string]any)
	for _, tag := range resp.Server.Tags {
		tags[convert.ToValue(tag.Key)] = convert.ToValue(tag.Value)
	}
	return tags, nil
}

func (a *mqlAwsTransferServer) structuredLogDestinations() ([]any, error) {
	resp, err := a.fetchDetail()
	if err != nil {
		return nil, err
	}
	destinations := make([]any, 0, len(resp.Server.StructuredLogDestinations))
	for _, d := range resp.Server.StructuredLogDestinations {
		destinations = append(destinations, d)
	}
	return destinations, nil
}
