// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"fmt"
	"strings"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/gcp/connection"
	"go.mondoo.com/mql/v13/types"
	"google.golang.org/api/composer/v1"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
)

// composerServiceDisabled reports whether the error indicates the Cloud
// Composer API is not enabled for the project. A genuine permission denial
// (HTTP 403 without a "not enabled" message) is deliberately not treated as
// disabled so it surfaces to the caller instead of being swallowed.
func composerServiceDisabled(err error) bool {
	gerr, ok := err.(*googleapi.Error)
	if !ok {
		return false
	}
	if strings.Contains(gerr.Message, "not enabled") || strings.Contains(gerr.Message, "has not been used") {
		return true
	}
	return gerr.Code == 404
}

func (g *mqlGcpProject) composer() (*mqlGcpProjectComposerService, error) {
	if g.Id.Error != nil {
		return nil, g.Id.Error
	}
	res, err := CreateResource(g.MqlRuntime, "gcp.project.composerService", map[string]*llx.RawData{
		"projectId": llx.StringData(g.Id.Data),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlGcpProjectComposerService), nil
}

func (g *mqlGcpProjectComposerService) id() (string, error) {
	if g.ProjectId.Error != nil {
		return "", g.ProjectId.Error
	}
	return fmt.Sprintf("gcp.project/%s/composerService", g.ProjectId.Data), nil
}

func (g *mqlGcpProjectComposerService) environments() ([]any, error) {
	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)
	client, err := conn.Client(composer.CloudPlatformScope)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	svc, err := composer.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}

	// Cloud Composer environments are regional and the environments.list API
	// rejects the "-" location wildcard, so enumerate the project's regions
	// and list environments in each.
	computeSvc, err := compute.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}
	regions, err := computeSvc.Regions.List(projectId).Do()
	if err != nil {
		return nil, err
	}

	var res []any
	for _, region := range regions.Items {
		parent := fmt.Sprintf("projects/%s/locations/%s", projectId, region.Name)
		req := svc.Projects.Locations.Environments.List(parent)
		if err := req.Pages(ctx, func(page *composer.ListEnvironmentsResponse) error {
			for _, e := range page.Environments {
				cfg, err := convert.JsonToDict(e.Config)
				if err != nil {
					return err
				}

				var imageVersion string
				if e.Config != nil && e.Config.SoftwareConfig != nil {
					imageVersion = e.Config.SoftwareConfig.ImageVersion
				}

				privateEnvironmentEnabled := false
				webServerAllowedIpRanges := []any{}
				if e.Config != nil {
					if e.Config.PrivateEnvironmentConfig != nil {
						privateEnvironmentEnabled = e.Config.PrivateEnvironmentConfig.EnablePrivateEnvironment
					}
					if e.Config.WebServerNetworkAccessControl != nil {
						for _, r := range e.Config.WebServerNetworkAccessControl.AllowedIpRanges {
							if r != nil && r.Value != "" {
								webServerAllowedIpRanges = append(webServerAllowedIpRanges, r.Value)
							}
						}
					}
				}

				mqlEnv, err := CreateResource(g.MqlRuntime, "gcp.project.composerService.environment", map[string]*llx.RawData{
					"projectId":                 llx.StringData(projectId),
					"name":                      llx.StringData(e.Name),
					"state":                     llx.StringData(e.State),
					"uuid":                      llx.StringData(e.Uuid),
					"createTime":                llx.TimeDataPtr(parseTime(e.CreateTime)),
					"updateTime":                llx.TimeDataPtr(parseTime(e.UpdateTime)),
					"labels":                    llx.MapData(convert.MapToInterfaceMap(e.Labels), types.String),
					"imageVersion":              llx.StringData(imageVersion),
					"config":                    llx.DictData(cfg),
					"privateEnvironmentEnabled": llx.BoolData(privateEnvironmentEnabled),
					"webServerAllowedIpRanges":  llx.ArrayData(webServerAllowedIpRanges, types.String),
				})
				if err != nil {
					return err
				}
				if e.Config != nil && e.Config.EncryptionConfig != nil {
					mqlEnv.(*mqlGcpProjectComposerServiceEnvironment).cacheKmsKeyName = e.Config.EncryptionConfig.KmsKeyName
				}
				res = append(res, mqlEnv)
			}
			return nil
		}); err != nil {
			if composerServiceDisabled(err) {
				// The Cloud Composer API is not enabled for the project; it
				// is disabled project-wide, so no other region would succeed.
				return []any{}, nil
			}
			// Surface transient or server-side errors instead of silently
			// dropping this region's environments from the result set.
			return nil, err
		}
	}

	return res, nil
}

type mqlGcpProjectComposerServiceEnvironmentInternal struct {
	cacheKmsKeyName string
}

func (g *mqlGcpProjectComposerServiceEnvironment) id() (string, error) {
	if g.Name.Error != nil {
		return "", g.Name.Error
	}
	return g.Name.Data, nil
}

func (g *mqlGcpProjectComposerServiceEnvironment) kmsKey() (*mqlGcpProjectKmsServiceKeyringCryptokey, error) {
	return newKmsCryptoKeyRef(g.MqlRuntime, &g.KmsKey, g.cacheKmsKeyName)
}
