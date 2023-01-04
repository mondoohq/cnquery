package gcp

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
			var mqlAndroidRestr resources.ResourceType
			if k.Restrictions.AndroidKeyRestrictions != nil {
				mqlAllowedApps := make([]resources.ResourceType, 0, len(k.Restrictions.AndroidKeyRestrictions.AllowedApplications))
				for _, a := range k.Restrictions.AndroidKeyRestrictions.AllowedApplications {
					mqlAllowedApp, err := g.MotorRuntime.CreateResource("gcp.project.apiKey.restrictions.androidKeyRestrictions.application",
						"id", fmt.Sprintf("%s/androidRestrictions/allowedApplications/%s", k.Name, a.PackageName),
						"packageName", a.PackageName,
						"sha1Fingerprint", a.Sha1Fingerprint,
					)
					if err != nil {
						return nil, err
					}
					mqlAllowedApps = append(mqlAllowedApps, mqlAllowedApp)
				}

				mqlAndroidRestr, err = g.MotorRuntime.CreateResource("gcp.project.apiKey.restrictions.androidKeyRestrictions",
					"id", fmt.Sprintf("%s/androidRestrictions", k.Name),
					"allowedApplications", mqlAllowedApps,
				)
				if err != nil {
					return nil, err
				}
			}

			mqlApiTargets := make([]interface{}, 0, len(k.Restrictions.ApiTargets))
			for i, a := range k.Restrictions.ApiTargets {
				mqlApiTarget, err := g.MotorRuntime.CreateResource("gcp.project.apiKey.restrictions.apiTarget",
					"id", fmt.Sprintf("%s/apiTargets/%d", k.Name, i),
					"methods", core.StrSliceToInterface(a.Methods),
					"service", a.Service,
				)
				if err != nil {
					return nil, err
				}
				mqlApiTargets = append(mqlApiTargets, mqlApiTarget)
			}

			var mqlBrowserRest resources.ResourceType
			if k.Restrictions.BrowserKeyRestrictions != nil {
				mqlBrowserRest, err = g.MotorRuntime.CreateResource("gcp.project.apiKey.restrictions.browserKeyRestrictions",
					"id", fmt.Sprintf("%s/browserKeyRestrictions", k.Name),
					"allowedReferrers", core.StrSliceToInterface(k.Restrictions.BrowserKeyRestrictions.AllowedReferrers),
				)
				if err != nil {
					return nil, err
				}
			}

			var mqlIosRestr resources.ResourceType
			if k.Restrictions.IosKeyRestrictions != nil {
				mqlIosRestr, err = g.MotorRuntime.CreateResource("gcp.project.apiKey.restrictions.iosKeyRestrictions",
					"id", fmt.Sprintf("%s/iosKeyRestrictions", k.Name),
					"allowedBundleIds", core.StrSliceToInterface(k.Restrictions.IosKeyRestrictions.AllowedBundleIds),
				)
				if err != nil {
					return nil, err
				}
			}

			var mqlServerKeyRestr resources.ResourceType
			if k.Restrictions.ServerKeyRestrictions != nil {
				mqlServerKeyRestr, err = g.MotorRuntime.CreateResource("gcp.project.apiKey.restrictions.serverKeyRestrictions",
					"id", fmt.Sprintf("%s/serverKeyRestrictions", k.Name),
					"allowedIps", core.StrSliceToInterface(k.Restrictions.ServerKeyRestrictions.AllowedIps),
				)
				if err != nil {
					return nil, err
				}
			}

			mqlRestrictions, err = g.MotorRuntime.CreateResource("gcp.project.apiKey.restrictions",
				"parentResourcePath", k.Name,
				"androidKeyRestrictions", mqlAndroidRestr,
				"apiTargets", mqlApiTargets,
				"browserKeyRestrictions", mqlBrowserRest,
				"iosKeyRestrictions", mqlIosRestr,
				"serverKeyRestrictions", mqlServerKeyRestr,
			)
			if err != nil {
				return nil, err
			}
		}

		mqlKey, err := g.MotorRuntime.CreateResource("gcp.project.apiKey",
			"projectId", projectId,
			"resourcePath", k.Name,
			"annotations", core.StrMapToInterface(k.Annotations),
			"created", parseTime(k.CreateTime),
			"deleted", parseTime(k.DeleteTime),
			"displayName", k.DisplayName,
			"keyString", k.KeyString,
			"name", parseResourceName(k.Name),
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

func (g *mqlGcpProjectApiKeyRestrictionsAndroidKeyRestrictions) id() (string, error) {
	return g.Id()
}

func (g *mqlGcpProjectApiKeyRestrictionsAndroidKeyRestrictionsApplication) id() (string, error) {
	return g.Id()
}

func (g *mqlGcpProjectApiKeyRestrictionsApiTarget) id() (string, error) {
	return g.Id()
}

func (g *mqlGcpProjectApiKeyRestrictionsBrowserKeyRestrictions) id() (string, error) {
	return g.Id()
}

func (g *mqlGcpProjectApiKeyRestrictionsIosKeyRestrictions) id() (string, error) {
	return g.Id()
}

func (g *mqlGcpProjectApiKeyRestrictionsServerKeyRestrictions) id() (string, error) {
	return g.Id()
}
