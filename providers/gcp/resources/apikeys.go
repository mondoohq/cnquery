// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"fmt"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/gcp/connection"
	"go.mondoo.com/mql/v13/types"

	"google.golang.org/api/apikeys/v2"
	"google.golang.org/api/option"
)

func (g *mqlGcpProject) apiKeys() ([]any, error) {
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
		log.Debug().Str("service", service_apikeys).Str("project", projectId).Msg("gcp service is not enabled, skipping")
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

	var apiKeyItems []*apikeys.V2Key
	if err := apiKeysSvc.Projects.Locations.Keys.List(fmt.Sprintf("projects/%s/locations/global", projectId)).Pages(ctx, func(page *apikeys.V2ListKeysResponse) error {
		apiKeyItems = append(apiKeyItems, page.Keys...)
		return nil
	}); err != nil {
		return nil, err
	}

	mqlKeys := make([]any, 0, len(apiKeyItems))
	for _, k := range apiKeyItems {
		var mqlRestrictions plugin.Resource
		if k.Restrictions != nil {
			var mqlAndroidRestr any
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

			mqlApiTargets := make([]any, 0, len(k.Restrictions.ApiTargets))
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

			var mqlBrowserRest any
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

			var mqlIosRestr any
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

			var mqlServerKeyRestr any
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

func initGcpProjectApiKey(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}

	if len(args) == 0 {
		if args == nil {
			args = make(map[string]*llx.RawData)
		}
		if ids := getAssetIdentifier(runtime); ids != nil {
			args["id"] = llx.StringData(ids.name)
			args["projectId"] = llx.StringData(ids.project)
		} else {
			return nil, nil, errors.New("no asset identifier found")
		}
	}

	obj, err := CreateResource(runtime, "gcp.project", map[string]*llx.RawData{
		"id": args["projectId"],
	})
	if err != nil {
		return nil, nil, err
	}
	proj := obj.(*mqlGcpProject)
	keys := proj.GetApiKeys()
	if keys.Error != nil {
		return nil, nil, keys.Error
	}

	idVal := args["id"].Value.(string)
	for _, k := range keys.Data {
		key := k.(*mqlGcpProjectApiKey)
		if key.Id.Data == idVal {
			return args, key, nil
		}
	}

	return nil, nil, fmt.Errorf("api key %q not found", idVal)
}

func (g *mqlGcpProjectApiKeyRestrictions) id() (string, error) {
	if g.ParentResourcePath.Error != nil {
		return "", g.ParentResourcePath.Error
	}
	return fmt.Sprintf("%s/restrictions", g.ParentResourcePath.Data), nil
}

func (g *mqlGcpProjectApiKeyRestrictions) unrestricted() (bool, error) {
	if g.AndroidKeyRestrictions.Error != nil {
		return false, g.AndroidKeyRestrictions.Error
	}
	if g.BrowserKeyRestrictions.Error != nil {
		return false, g.BrowserKeyRestrictions.Error
	}
	if g.IosKeyRestrictions.Error != nil {
		return false, g.IosKeyRestrictions.Error
	}
	if g.ServerKeyRestrictions.Error != nil {
		return false, g.ServerKeyRestrictions.Error
	}
	if g.ApiTargets.Error != nil {
		return false, g.ApiTargets.Error
	}
	return g.AndroidKeyRestrictions.Data == nil &&
		g.BrowserKeyRestrictions.Data == nil &&
		g.IosKeyRestrictions.Data == nil &&
		g.ServerKeyRestrictions.Data == nil &&
		len(g.ApiTargets.Data) == 0, nil
}

// Always bootstrap through the parent key. apiKeys() populates restrictions
// via CreateResource (which bypasses Init), so any NewResource call here is
// a top-level `gcp.project.apiKey.restrictions` query — partial args would
// pass through to Create and yield a bare stub whose fields all error with
// "cannot convert primitive with NO type information".
func initGcpProjectApiKeyRestrictions(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	ids := getAssetIdentifier(runtime)
	if ids == nil {
		return nil, nil, errors.New("no asset identifier found; gcp.project.apiKey.restrictions requires a gcp-apikey asset context")
	}

	obj, err := NewResource(runtime, "gcp.project.apiKey", map[string]*llx.RawData{
		"id":        llx.StringData(ids.name),
		"projectId": llx.StringData(ids.project),
	})
	if err != nil {
		return nil, nil, err
	}
	key := obj.(*mqlGcpProjectApiKey)
	restrictions := key.GetRestrictions()
	if restrictions.Error != nil {
		return nil, nil, restrictions.Error
	}
	if restrictions.Data == nil {
		// Key has no restrictions. Null out all fields so the Create fallthrough
		// produces a resource whose TValues are StateIsNull|StateIsSet (via
		// RawToTValue's nil path) instead of a bare stub. Matches the pattern
		// in initGcpProjectIamServiceAccount.
		if args == nil {
			args = make(map[string]*llx.RawData)
		}
		args["parentResourcePath"] = llx.NilData
		args["androidKeyRestrictions"] = llx.NilData
		args["apiTargets"] = llx.NilData
		args["browserKeyRestrictions"] = llx.NilData
		args["iosKeyRestrictions"] = llx.NilData
		args["serverKeyRestrictions"] = llx.NilData
		return args, nil, nil
	}
	return args, restrictions.Data, nil
}
