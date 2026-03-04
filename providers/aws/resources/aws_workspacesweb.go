// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/workspacesweb"
	workspaceswebtypes "github.com/aws/aws-sdk-go-v2/service/workspacesweb/types"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/mql/v13/providers/aws/connection"
)

// isWorkspacesWebRegionError checks if the error indicates the WorkSpaces Web
// service is not available or reachable in the given region.
func isWorkspacesWebRegionError(err error) bool {
	return Is400AccessDeniedError(err) ||
		IsServiceNotAvailableInRegionError(err) ||
		errors.Is(err, context.DeadlineExceeded)
}

func (a *mqlAwsWorkspacesweb) id() (string, error) {
	return "aws.workspacesweb", nil
}

// Portals

func (a *mqlAwsWorkspacesweb) portals() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getPortals(conn), 5)
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

func (a *mqlAwsWorkspacesweb) getPortals(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("workspacesweb>getPortals>calling aws with region %s", region)
			svc := conn.WorkspacesWeb(region)
			res := []any{}

			var nextToken *string
			for {
				ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
				resp, err := svc.ListPortals(ctx, &workspacesweb.ListPortalsInput{
					NextToken: nextToken,
				})
				cancel()
				if err != nil {
					if isWorkspacesWebRegionError(err) {
						log.Debug().Str("region", region).Msg("error accessing region for AWS WorkSpaces Web portal API")
						return res, nil
					}
					return nil, err
				}
				for _, portal := range resp.Portals {
					mqlPortal, err := newMqlAwsWorkspaceswebPortal(a.MqlRuntime, region, portal)
					if err != nil {
						return nil, err
					}
					res = append(res, mqlPortal)
				}
				if resp.NextToken == nil {
					break
				}
				nextToken = resp.NextToken
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func newMqlAwsWorkspaceswebPortal(runtime *plugin.Runtime, region string, portal workspaceswebtypes.PortalSummary) (*mqlAwsWorkspaceswebPortal, error) {
	res, err := CreateResource(runtime, "aws.workspacesweb.portal",
		map[string]*llx.RawData{
			"__id":                         llx.StringDataPtr(portal.PortalArn),
			"portalArn":                    llx.StringDataPtr(portal.PortalArn),
			"displayName":                  llx.StringDataPtr(portal.DisplayName),
			"portalEndpoint":               llx.StringDataPtr(portal.PortalEndpoint),
			"portalStatus":                 llx.StringData(string(portal.PortalStatus)),
			"authenticationType":           llx.StringData(string(portal.AuthenticationType)),
			"browserType":                  llx.StringData(string(portal.BrowserType)),
			"instanceType":                 llx.StringData(string(portal.InstanceType)),
			"rendererType":                 llx.StringData(string(portal.RendererType)),
			"browserSettingsArn":           llx.StringDataPtr(portal.BrowserSettingsArn),
			"networkSettingsArn":           llx.StringDataPtr(portal.NetworkSettingsArn),
			"userSettingsArn":              llx.StringDataPtr(portal.UserSettingsArn),
			"trustStoreArn":                llx.StringDataPtr(portal.TrustStoreArn),
			"ipAccessSettingsArn":          llx.StringDataPtr(portal.IpAccessSettingsArn),
			"userAccessLoggingSettingsArn": llx.StringDataPtr(portal.UserAccessLoggingSettingsArn),
			"dataProtectionSettingsArn":    llx.StringDataPtr(portal.DataProtectionSettingsArn),
			"maxConcurrentSessions":        llx.IntDataDefault(portal.MaxConcurrentSessions, 0),
			"creationDate":                 llx.TimeDataPtr(portal.CreationDate),
			"region":                       llx.StringData(region),
		})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAwsWorkspaceswebPortal), nil
}

func (a *mqlAwsWorkspaceswebPortal) id() (string, error) {
	return a.PortalArn.Data, nil
}

// User Access Logging Settings

func (a *mqlAwsWorkspacesweb) userAccessLoggingSettings() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getUserAccessLoggingSettings(conn), 5)
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

func (a *mqlAwsWorkspacesweb) getUserAccessLoggingSettings(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("workspacesweb>getUserAccessLoggingSettings>calling aws with region %s", region)
			svc := conn.WorkspacesWeb(region)
			res := []any{}

			var nextToken *string
			for {
				ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
				resp, err := svc.ListUserAccessLoggingSettings(ctx, &workspacesweb.ListUserAccessLoggingSettingsInput{
					NextToken: nextToken,
				})
				cancel()
				if err != nil {
					if isWorkspacesWebRegionError(err) {
						log.Debug().Str("region", region).Msg("error accessing region for AWS WorkSpaces Web user access logging settings API")
						return res, nil
					}
					return nil, err
				}
				for _, setting := range resp.UserAccessLoggingSettings {
					mqlSetting, err := newMqlAwsWorkspaceswebUserAccessLoggingSetting(a.MqlRuntime, region, setting)
					if err != nil {
						return nil, err
					}
					res = append(res, mqlSetting)
				}
				if resp.NextToken == nil {
					break
				}
				nextToken = resp.NextToken
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func newMqlAwsWorkspaceswebUserAccessLoggingSetting(runtime *plugin.Runtime, region string, setting workspaceswebtypes.UserAccessLoggingSettingsSummary) (*mqlAwsWorkspaceswebUserAccessLoggingSetting, error) {
	res, err := CreateResource(runtime, "aws.workspacesweb.userAccessLoggingSetting",
		map[string]*llx.RawData{
			"__id":                         llx.StringDataPtr(setting.UserAccessLoggingSettingsArn),
			"userAccessLoggingSettingsArn": llx.StringDataPtr(setting.UserAccessLoggingSettingsArn),
			"kinesisStreamArn":             llx.StringDataPtr(setting.KinesisStreamArn),
			"region":                       llx.StringData(region),
		})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAwsWorkspaceswebUserAccessLoggingSetting), nil
}

func (a *mqlAwsWorkspaceswebUserAccessLoggingSetting) id() (string, error) {
	return a.UserAccessLoggingSettingsArn.Data, nil
}
