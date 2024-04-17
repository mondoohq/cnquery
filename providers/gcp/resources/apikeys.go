// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"fmt"

	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v11/providers/gcp/connection"
	"go.mondoo.com/cnquery/v11/types"

	"google.golang.org/api/apikeys/v2"
	"google.golang.org/api/option"
)

func (g *mqlGcpProject) apiKeys() ([]interface{}, error) {
	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)

	if g.Id.Error != nil {
		return nil, g.Id.Error
	}
	projectId := g.Id.Data

	serviceEnabled, err := g.isServiceEnabled(service_apikeys)
	if err != nil {
		return nil, err
	}
	if !serviceEnabled {
		return nil, nil
	}

	client, err := conn.Client(apikeys.CloudPlatformReadOnlyScope)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()

	apiKeysSvc, err := apikeys.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}

	keys, err := apiKeysSvc.Projects.Locations.Keys.List(fmt.Sprintf("projects/%s/locations/global", projectId)).Do()
	if err != nil {
		return nil, err
	}

	mqlKeys := make([]interface{}, 0, len(keys.Keys))
	for _, k := range keys.Keys {
		var mqlRestrictions plugin.Resource
		if k.Restrictions != nil {
			var mqlAndroidRestr interface{}
			if k.Restrictions.AndroidKeyRestrictions != nil {

				type mqlAllowedApp struct {
					PackageName     string `json:"packageName"`
					Sha1Fingerprint string `json:"sha1Fingerprint"`
				}
				type mqlAndroidKeyRestrictions struct {
					AllowedApplications []mqlAllowedApp `json:"allowedApplications"`
				}

				androidRestrictions := mqlAndroidKeyRestrictions{}
				for _, a := range k.Restrictions.AndroidKeyRestrictions.AllowedApplications {
					androidRestrictions.AllowedApplications = append(androidRestrictions.AllowedApplications, mqlAllowedApp{
						PackageName:     a.PackageName,
						Sha1Fingerprint: a.Sha1Fingerprint,
					})
				}

				mqlAndroidRestr, err = convert.JsonToDict(androidRestrictions)
				if err != nil {
					return nil, err
				}
			}

			mqlApiTargets := make([]interface{}, 0, len(k.Restrictions.ApiTargets))
			if k.Restrictions.ApiTargets != nil {
				type mqlApiTarget struct {
					Service string   `json:"service"`
					Methods []string `json:"methods"`
				}

				for _, a := range k.Restrictions.ApiTargets {
					target, err := convert.JsonToDict(mqlApiTarget{
						Service: a.Service,
						Methods: a.Methods,
					})
					if err != nil {
						return nil, err
					}
					mqlApiTargets = append(mqlApiTargets, target)
				}
			}

			var mqlBrowserRest interface{}
			if k.Restrictions.BrowserKeyRestrictions != nil {
				type mqlBrowserKeyRestrictions struct {
					AllowedReferrers []string `json:"allowedReferrers"`
				}

				mqlBrowserRest, err = convert.JsonToDict(mqlBrowserKeyRestrictions{
					AllowedReferrers: k.Restrictions.BrowserKeyRestrictions.AllowedReferrers,
				})
				if err != nil {
					return nil, err
				}
			}

			var mqlIosRestr interface{}
			if k.Restrictions.IosKeyRestrictions != nil {
				type mqlIosKeyRestrictions struct {
					AllowedBundleIds []string `json:"allowedBundleIds"`
				}

				mqlIosRestr, err = convert.JsonToDict(mqlIosKeyRestrictions{
					AllowedBundleIds: k.Restrictions.IosKeyRestrictions.AllowedBundleIds,
				})
				if err != nil {
					return nil, err
				}
			}

			var mqlServerKeyRestr interface{}
			if k.Restrictions.ServerKeyRestrictions != nil {
				type mqlServerKeyRestrictions struct {
					AllowedIps []string `json:"allowedIps"`
				}

				mqlServerKeyRestr, err = convert.JsonToDict(mqlServerKeyRestrictions{
					AllowedIps: k.Restrictions.ServerKeyRestrictions.AllowedIps,
				})
				if err != nil {
					return nil, err
				}
			}

			mqlRestrictions, err = CreateResource(g.MqlRuntime, "gcp.project.apiKey.restrictions", map[string]*llx.RawData{
				"parentResourcePath":     llx.StringData(k.Name),
				"androidKeyRestrictions": llx.DictData(mqlAndroidRestr),
				"browserKeyRestrictions": llx.DictData(mqlBrowserRest),
				"iosKeyRestrictions":     llx.DictData(mqlIosRestr),
				"serverKeyRestrictions":  llx.DictData(mqlServerKeyRestr),
				"apiTargets":             llx.ArrayData(mqlApiTargets, types.Dict),
			})
			if err != nil {
				return nil, err
			}
		}

		mqlKey, err := CreateResource(g.MqlRuntime, "gcp.project.apiKey", map[string]*llx.RawData{
			"projectId":    llx.StringData(projectId),
			"id":           llx.StringData(parseResourceName(k.Name)),
			"name":         llx.StringData(k.DisplayName),
			"resourcePath": llx.StringData(k.Name),
			"annotations":  llx.MapData(convert.MapToInterfaceMap(k.Annotations), types.String),
			"created":      llx.TimeDataPtr(parseTime(k.CreateTime)),
			"deleted":      llx.TimeDataPtr(parseTime(k.DeleteTime)),
			"keyString":    llx.StringData(k.KeyString),
			"restrictions": llx.ResourceData(mqlRestrictions, "gcp.project.apiKey.restrictions"),
			"updated":      llx.TimeDataPtr(parseTime(k.UpdateTime)),
		})
		if err != nil {
			return nil, err
		}
		mqlKeys = append(mqlKeys, mqlKey)
	}
	return mqlKeys, nil
}

func (g *mqlGcpProjectApiKey) id() (string, error) {
	return g.ResourcePath.Data, g.ResourcePath.Error
}

func (g *mqlGcpProjectApiKeyRestrictions) id() (string, error) {
	if g.ParentResourcePath.Error != nil {
		return "", g.ParentResourcePath.Error
	}
	return fmt.Sprintf("%s/restrictions", g.ParentResourcePath.Data), nil
}
