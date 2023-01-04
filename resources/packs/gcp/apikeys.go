package gcp

import (
	"context"
	"fmt"

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

	keys, err := apiKeysSvc.Projects.Locations.Keys.List(fmt.Sprintf("projects/%s/locations/-", projectId)).Do()
	if err != nil {
		return nil, err
	}

	mqlKeys := make([]interface{}, 0, len(keys.Keys))
	for _, k := range keys.Keys {
		mqlKey, err := g.MotorRuntime.CreateResource("gcp.project.apiKey",
			"projectId", projectId,
			"resourcePath", k.Name,
			"annotations", core.StrMapToInterface(k.Annotations),
			"created", parseTime(k.CreateTime),
			"deleted", parseTime(k.DeleteTime),
			"displayName", k.DisplayName,
			"keyString", k.KeyString,
			"name", parseResourceName(k.Name),
			"restrictions", nil, // TODO
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

func (g *mqlGcpProjectApiKeyRestriction) id() (string, error) {
	parent, err := g.ParentResourcePath()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/restriction", parent), nil
}

func (g *mqlGcpProjectApiKeyRestrictionAndroidKeyRestrictions) id() (string, error) {
	return g.Id()
}

func (g *mqlGcpProjectApiKeyRestrictionAndroidKeyRestrictionsApplication) id() (string, error) {
	return g.Id()
}

func (g *mqlGcpProjectApiKeyRestrictionApiTarget) id() (string, error) {
	return g.Id()
}

func (g *mqlGcpProjectApiKeyRestrictionBrowserKeyRestrictions) id() (string, error) {
	return g.Id()
}

func (g *mqlGcpProjectApiKeyRestrictionIosKeyRestrictions) id() (string, error) {
	return g.Id()
}

func (g *mqlGcpProjectApiKeyRestrictionServerKeyRestrictions) id() (string, error) {
	return g.Id()
}
