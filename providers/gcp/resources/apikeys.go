// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"fmt"

	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/resources/packs/core"
	"google.golang.org/api/apikeys/v2"
	"google.golang.org/api/option"
)

func (g *mqlGcpProject) GetApiKeys() ([]interface{}, error) {
	provider, err := gcpProvider(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	projectId, err := g.Id()
	if err != nil {
		return nil, err
	}

	client, err := provider.Client(apikeys.CloudPlatformReadOnlyScope)
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
		var mqlRestrictions resources.ResourceType
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

				mqlAndroidRestr, err = core.JsonToDict(androidRestrictions)
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
					target, err := core.JsonToDict(mqlApiTarget{
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

				mqlBrowserRest, err = core.JsonToDict(mqlBrowserKeyRestrictions{
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

				mqlIosRestr, err = core.JsonToDict(mqlIosKeyRestrictions{
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

				mqlServerKeyRestr, err = core.JsonToDict(mqlServerKeyRestrictions{
					AllowedIps: k.Restrictions.ServerKeyRestrictions.AllowedIps,
				})
				if err != nil {
					return nil, err
				}
			}

			mqlRestrictions, err = g.MotorRuntime.CreateResource("gcp.project.apiKey.restrictions",
				"parentResourcePath", k.Name,
				"androidKeyRestrictions", mqlAndroidRestr,
				"browserKeyRestrictions", mqlBrowserRest,
				"iosKeyRestrictions", mqlIosRestr,
				"serverKeyRestrictions", mqlServerKeyRestr,
				"apiTargets", mqlApiTargets,
			)
			if err != nil {
				return nil, err
			}
		}

		mqlKey, err := g.MotorRuntime.CreateResource("gcp.project.apiKey",
			"projectId", projectId,
			"id", parseResourceName(k.Name),
			"name", k.DisplayName,
			"resourcePath", k.Name,
			"annotations", core.StrMapToInterface(k.Annotations),
			"created", parseTime(k.CreateTime),
			"deleted", parseTime(k.DeleteTime),
			"keyString", k.KeyString,
			"restrictions", mqlRestrictions,
			"updated", parseTime(k.UpdateTime),
		)
		if err != nil {
			return nil, err
		}
		mqlKeys = append(mqlKeys, mqlKey)
	}
	return mqlKeys, nil
}

func (g *mqlGcpProjectApiKey) id() (string, error) {
	return g.ResourcePath()
}

func (g *mqlGcpProjectApiKeyRestrictions) id() (string, error) {
	parent, err := g.ParentResourcePath()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/restrictions", parent), nil
}
