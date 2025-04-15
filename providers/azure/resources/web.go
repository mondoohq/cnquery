// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/rs/zerolog/log"

	"go.mondoo.com/cnquery/v12/llx"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v12/providers/azure/connection"
	"go.mondoo.com/cnquery/v12/types"

	web "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appservice/armappservice"
)

var majorVersionRegex = regexp.MustCompile(`^(\d+)`)

func isPlatformEol(platform string, version string) bool {
	if version == "" {
		return false
	}
	if platform != "node" {
		return false
	}
	m := majorVersionRegex.FindString(version)
	val, err := strconv.Atoi(m)
	if err != nil {
		log.Error().
			Err(err).
			Str("platform", platform).
			Str("version", version).
			Msg("could not parse the azure webapp version")
		return false
	}

	if val < 10 || val == 11 {
		return true
	}
	return false
}

type AzureWebAppStackRuntime struct {
	Name          string    `json:"name,omitempty"`
	ID            string    `json:"id,omitempty"`
	Os            string    `json:"os,omitempty"`
	MajorVersion  string    `json:"majorVersion,omitempty"`
	MinorVersion  string    `json:"minorVersion,omitempty"`
	IsDeprecated  bool      `json:"isDeprecated,omitempty"`
	IsHidden      bool      `json:"isHidden,omitempty"`
	IsDefault     bool      `json:"isDefault,omitempty"`
	EndOfLifeDate time.Time `json:"endOfLifeDate,omitempty"`
}

func (a *mqlAzureSubscriptionWebService) id() (string, error) {
	return "azure.subscription.web/" + a.SubscriptionId.Data, nil
}

func initAzureSubscriptionWebService(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 0 {
		return args, nil, nil
	}

	conn, ok := runtime.Connection.(*connection.AzureConnection)
	if !ok {
		return nil, nil, errors.New("invalid connection provided, it is not an Azure connection")
	}
	args["subscriptionId"] = llx.StringData(conn.SubId())

	return args, nil, nil
}

func (a *mqlAzureSubscriptionWebServiceAppsite) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionWebServiceAppsiteauthsettings) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionWebServiceAppsiteconfig) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionWebServiceAppsite) diagnosticSettings() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	return getDiagnosticSettings(a.Id.Data, a.MqlRuntime, conn)
}

func (a *mqlAzureSubscriptionWebService) apps() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	subId := a.SubscriptionId.Data
	client, err := web.NewWebAppsClient(subId, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}
	pager := client.NewListPager(&web.WebAppsClientListOptions{})
	res := []any{}
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, entry := range page.Value {
			properties, err := convert.JsonToDict(entry.Properties)
			if err != nil {
				return nil, err
			}

			identity, err := convert.JsonToDict(entry.Identity)
			if err != nil {
				return nil, err
			}

			mqlAzure, err := CreateResource(a.MqlRuntime, "azure.subscription.webService.appsite",
				map[string]*llx.RawData{
					"id":         llx.StringDataPtr(entry.ID),
					"name":       llx.StringDataPtr(entry.Name),
					"location":   llx.StringDataPtr(entry.Location),
					"tags":       llx.MapData(convert.PtrMapStrToInterface(entry.Tags), types.String),
					"type":       llx.StringDataPtr(entry.Type),
					"kind":       llx.StringDataPtr(entry.Kind),
					"properties": llx.DictData(properties),
					"identity":   llx.DictData(identity),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlAzure)
		}
	}

	return res, nil
}

func (a *mqlAzureSubscriptionWebService) availableRuntimes() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	subId := a.SubscriptionId.Data
	client, err := web.NewProviderClient(subId, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	res := []any{}
	mapIDs := map[string]struct{}{}
	pager := client.NewGetWebAppStacksPager(&web.ProviderClientGetWebAppStacksOptions{
		StackOsType: convert.ToPtr(web.Enum19All),
	})
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, entry := range page.Value {
			majorVersions := entry.Properties.MajorVersions
			stackName := convert.ToValue(entry.Name)
			for _, major := range majorVersions {
				majorText := convert.ToValue(major.DisplayText)
				for _, minor := range major.MinorVersions {
					minorText := convert.ToValue(minor.DisplayText)
					if minor.StackSettings == nil {
						log.Debug().
							Str("stack_name", stackName).
							Str("major_version", majorText).
							Str("minor_version", minorText).
							Msg("no stack settings, skipping")
						continue
					}

					var os string
					var settings *web.WebAppRuntimeSettings
					switch convert.ToValue(entry.Properties.PreferredOs) {
					case "linux":
						settings = minor.StackSettings.LinuxRuntimeSettings
						os = "linux"
					case "windows":
						settings = minor.StackSettings.WindowsRuntimeSettings
						os = "windows"
					}

					if settings == nil {
						log.Debug().
							Str("stack_name", stackName).
							Str("major_version", majorText).
							Str("preferred_os", string(convert.ToValue(entry.Properties.PreferredOs))).
							Interface("stack_settings", minor.StackSettings).
							Msg("unknown runtime settings, skipping")
						continue
					}

					runtimeVersion := convert.ToValue(settings.RuntimeVersion)
					if runtimeVersion == "" {
						// some app runtimes like java doesn't return a runtime version, so we try
						// to build it like "stackName|minorVersion"
						runtimeVersion = strings.ToUpper(stackName + "|" + convert.ToValue(minor.Value))
					} else if stackName == "dotnet" {
						// dotnet doesn't format the runtime the same way like the rest of the runtimes
						// so we try to format it to match what the Azure portal shows
						dotNet := ".NET"
						if strings.Contains(convert.ToValue(major.Value), "asp") {
							dotNet = "ASP.NET"
						}
						runtimeVersion = strings.ToUpper(dotNet + "|" + convert.ToValue(minor.Value))
					}

					deprecated := convert.ToValue(settings.IsDeprecated) ||
						isPlatformEol(convert.ToValue(entry.Name), convert.ToValue(major.Value))

					id := strings.Join([]string{"azure.subscription", subId,
						"webService.appRuntimeStack", os, runtimeVersion,
					}, "/")

					if _, exist := mapIDs[id]; exist {
						continue
					}
					mapIDs[id] = struct{}{}

					resource, err := NewResource(a.MqlRuntime, "azure.subscription.webService.appRuntimeStack",
						map[string]*llx.RawData{
							"__id":           llx.StringData(id),
							"name":           llx.StringData(stackName),
							"preferredOs":    llx.StringData(os),
							"runtimeVersion": llx.StringData(runtimeVersion),
							"deprecated":     llx.BoolData(deprecated),
							"autoUpdate":     llx.BoolDataPtr(settings.IsAutoUpdate),
							"hidden":         llx.BoolDataPtr(settings.IsHidden),
							"endOfLifeDate":  llx.TimeDataPtr(settings.EndOfLifeDate),
							"majorVersion":   llx.StringDataPtr(major.Value),
							"minorVersion":   llx.StringDataPtr(minor.Value),
						})
					if err != nil {
						return nil, err
					}
					res = append(res, resource)
				}
			}
		}
	}

	return res, nil
}

func (a *mqlAzureSubscriptionWebServiceAppsite) configuration() (*mqlAzureSubscriptionWebServiceAppsiteconfig, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	id := a.Id.Data

	resourceID, err := ParseResourceID(id)
	if err != nil {
		return nil, err
	}

	site, err := resourceID.Component("sites")
	if err != nil {
		return nil, err
	}

	client, err := web.NewWebAppsClient(resourceID.SubscriptionID, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	configuration, err := client.GetConfiguration(ctx, resourceID.ResourceGroup, site, &web.WebAppsClientGetConfigurationOptions{})
	if err != nil {
		return nil, err
	}

	entry := configuration
	properties, err := convert.JsonToDict(entry.Properties)
	if err != nil {
		return nil, err
	}

	res, err := CreateResource(a.MqlRuntime, "azure.subscription.webService.appsiteconfig",
		map[string]*llx.RawData{
			"id":         llx.StringDataPtr(entry.ID),
			"name":       llx.StringDataPtr(entry.Name),
			"kind":       llx.StringDataPtr(entry.Kind),
			"type":       llx.StringDataPtr(entry.Type),
			"properties": llx.DictData(properties),
		})
	if err != nil {
		return nil, err
	}

	return res.(*mqlAzureSubscriptionWebServiceAppsiteconfig), nil
}

func (a *mqlAzureSubscriptionWebServiceAppsite) authenticationSettings() (*mqlAzureSubscriptionWebServiceAppsiteauthsettings, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	id := a.Id.Data

	resourceID, err := ParseResourceID(id)
	if err != nil {
		return nil, err
	}

	site, err := resourceID.Component("sites")
	if err != nil {
		return nil, err
	}

	client, err := web.NewWebAppsClient(resourceID.SubscriptionID, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	configuration, err := client.GetAuthSettings(ctx, resourceID.ResourceGroup, site, &web.WebAppsClientGetAuthSettingsOptions{})
	if err != nil {
		return nil, err
	}
	properties, err := convert.JsonToDict(configuration.Properties)
	if err != nil {
		return nil, err
	}

	res, err := CreateResource(a.MqlRuntime, "azure.subscription.webService.appsiteauthsettings",
		map[string]*llx.RawData{
			"id":         llx.StringDataPtr(configuration.ID),
			"name":       llx.StringDataPtr(configuration.Name),
			"kind":       llx.StringDataPtr(configuration.Kind),
			"type":       llx.StringDataPtr(configuration.Type),
			"properties": llx.DictData(properties),
		})
	if err != nil {
		return nil, err
	}

	return res.(*mqlAzureSubscriptionWebServiceAppsiteauthsettings), nil
}

func (a *mqlAzureSubscriptionWebServiceAppsite) metadata() (any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	id := a.Id.Data

	resourceID, err := ParseResourceID(id)
	if err != nil {
		return nil, err
	}

	site, err := resourceID.Component("sites")
	if err != nil {
		return nil, err
	}

	client, err := web.NewWebAppsClient(resourceID.SubscriptionID, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	metadata, err := client.ListMetadata(ctx, resourceID.ResourceGroup, site, &web.WebAppsClientListMetadataOptions{})
	if err != nil {
		return nil, err
	}

	res := make(map[string]any)

	for k := range metadata.Properties {
		res[k] = convert.ToValue(metadata.Properties[k])
	}

	return res, nil
}

func (a *mqlAzureSubscriptionWebServiceAppsite) ftp() (*mqlAzureSubscriptionWebServiceAppsiteBasicPublishingCredentialsPolicies, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	id := a.Id.Data
	resourceID, err := ParseResourceID(id)
	if err != nil {
		return nil, err
	}
	client, err := web.NewWebAppsClient(resourceID.SubscriptionID, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	site, err := resourceID.Component("sites")
	if err != nil {
		return nil, err
	}

	response, err := client.GetFtpAllowed(ctx, resourceID.ResourceGroup, site, nil)
	if err != nil {
		return nil, err
	}

	args := map[string]*llx.RawData{
		"id":   llx.StringDataPtr(response.ID),
		"name": llx.StringDataPtr(response.Name),
		"type": llx.StringDataPtr(response.Type),
	}
	if response.Properties != nil {
		args["allow"] = llx.BoolDataPtr(response.Properties.Allow)
	}
	mqlResource, err := CreateResource(a.MqlRuntime, "azure.subscription.webService.appsite.basicPublishingCredentialsPolicies", args)
	if err != nil {
		return nil, err
	}
	return mqlResource.(*mqlAzureSubscriptionWebServiceAppsiteBasicPublishingCredentialsPolicies), nil
}

func (a *mqlAzureSubscriptionWebServiceAppsiteBasicPublishingCredentialsPolicies) id() (string, error) {
	return fmt.Sprintf("%s/%s", a.Id.Data, a.Name.Data), nil
}

func (a *mqlAzureSubscriptionWebServiceAppsite) scm() (*mqlAzureSubscriptionWebServiceAppsiteBasicPublishingCredentialsPolicies, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	id := a.Id.Data
	resourceID, err := ParseResourceID(id)
	if err != nil {
		return nil, err
	}
	client, err := web.NewWebAppsClient(resourceID.SubscriptionID, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	site, err := resourceID.Component("sites")
	if err != nil {
		return nil, err
	}

	response, err := client.GetScmAllowed(ctx, resourceID.ResourceGroup, site, nil)
	if err != nil {
		return nil, err
	}

	args := map[string]*llx.RawData{
		"id":   llx.StringDataPtr(response.ID),
		"name": llx.StringDataPtr(response.Name),
		"type": llx.StringDataPtr(response.Type),
	}
	if response.Properties != nil {
		args["allow"] = llx.BoolDataPtr(response.Properties.Allow)
	}
	mqlResource, err := CreateResource(a.MqlRuntime, "azure.subscription.webService.appsite.basicPublishingCredentialsPolicies", args)
	if err != nil {
		return nil, err
	}
	return mqlResource.(*mqlAzureSubscriptionWebServiceAppsiteBasicPublishingCredentialsPolicies), nil
}

func (a *mqlAzureSubscriptionWebServiceAppsite) functions() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	id := a.Id.Data
	resourceID, err := ParseResourceID(id)
	if err != nil {
		return nil, err
	}
	client, err := web.NewWebAppsClient(resourceID.SubscriptionID, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	site, err := resourceID.Component("sites")
	if err != nil {
		return nil, err
	}

	pager := client.NewListFunctionsPager(resourceID.ResourceGroup, site, &web.WebAppsClientListFunctionsOptions{})
	res := []any{}

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, entry := range page.Value {
			props, err := convert.JsonToDict(entry.Properties)
			if err != nil {
				return nil, err
			}
			mqlAzure, err := CreateResource(a.MqlRuntime, "azure.subscription.webService.function",
				map[string]*llx.RawData{
					"id":         llx.StringDataPtr(entry.ID),
					"name":       llx.StringDataPtr(entry.Name),
					"type":       llx.StringDataPtr(entry.Type),
					"kind":       llx.StringDataPtr(entry.Kind),
					"properties": llx.AnyData(props),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlAzure)
		}
	}
	return res, nil
}

func (a *mqlAzureSubscriptionWebServiceFunction) id() (string, error) {
	return a.id()
}

func (a *mqlAzureSubscriptionWebServiceAppsite) connectionSettings() (any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	id := a.Id.Data

	resourceID, err := ParseResourceID(id)
	if err != nil {
		return nil, err
	}

	site, err := resourceID.Component("sites")
	if err != nil {
		return nil, err
	}

	client, err := web.NewWebAppsClient(resourceID.SubscriptionID, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	settings, err := client.ListConnectionStrings(ctx, resourceID.ResourceGroup, site, &web.WebAppsClientListConnectionStringsOptions{})
	if err != nil {
		return nil, err
	}

	res := make(map[string]any)

	for k := range settings.Properties {
		value, err := convert.JsonToDict(settings.Properties[k])
		if err != nil {
			return nil, err
		}

		res[k] = value
	}

	return res, nil
}

// TODO: check here if we can use cached stuff (and how)
func (a *mqlAzureSubscriptionWebServiceAppsite) stack() (any, error) {
	configPlugin := a.GetConfiguration()
	if configPlugin.Error != nil {
		return nil, configPlugin.Error
	}

	config := configPlugin.Data
	// read configuration into go struct
	configProperties := config.Properties.Data

	// unmarshal into go struct for easier access
	data, err := json.Marshal(configProperties)
	if err != nil {
		return nil, err
	}

	var properties web.SiteConfig
	err = json.Unmarshal(data, &properties)
	if err != nil {
		return nil, err
	}
	// get metadata
	metadataPlugin := a.GetMetadata()
	if metadataPlugin.Error != nil {
		return nil, metadataPlugin.Error
	}
	metadata := metadataPlugin.Data

	runtime := AzureWebAppStackRuntime{
		Os: "windows",
	}

	if properties.LinuxFxVersion == nil && properties.WindowsFxVersion == nil {
		return nil, errors.New("could not determine stack version")
	}

	if properties.LinuxFxVersion != nil && len(*properties.LinuxFxVersion) > 0 {
		runtime.Os = "linux"
		runtime.ID = *properties.LinuxFxVersion

		fxversion := strings.Split(*properties.LinuxFxVersion, "|")
		runtime.Name = strings.ToLower(fxversion[0])
		runtime.MinorVersion = strings.ToLower(fxversion[1])
	} else {
		metadata, ok := metadata.(map[string]any)
		if !ok {
			return nil, nil // see behavior below
		}

		// read runtime from metadata, it works completely different than on linux
		// NOTE: also take care of the runtime version for dotnet.
		stack, ok := metadata["CURRENT_STACK"].(string)
		if !ok {
			// This doesn't seem to be consistently set
			// https://stackoverflow.com/questions/63950946/azure-app-service-get-stack-settings-via-api#comment113188903_63987100
			// https://github.com/mondoohq/installer/issues/157
			return nil, nil
		}
		version := ""
		switch stack {
		case "dotnet":
			stack = "aspnet" // needs to be overwritten (do not ask)
			version = convert.ToValue(properties.NetFrameworkVersion)
		case "dotnetcore":
			// NOTE: this will always return v4.0 no matter what you set (which is definitely wrong for dotnetcore)
			// The UI let you select different version, but in configure it does not show the version again
			// therefore we have no way than to hardcode the value for now
			version = "3.1"
		case "php":
			version = convert.ToValue(properties.PhpVersion)
		case "python":
			version = convert.ToValue(properties.PythonVersion)
		case "node":
			version = convert.ToValue(properties.NodeVersion)
		case "powershell":
			version = convert.ToValue(properties.PowerShellVersion)
		case "java":
			version = convert.ToValue(properties.JavaVersion)
		case "javaContainer":
			version = convert.ToValue(properties.JavaContainerVersion)
		}

		runtime.Name = strings.ToLower(stack)
		runtime.ID = strings.ToUpper(stack) + "|" + version
		runtime.MinorVersion = version
	}

	// fetch available runtimes and check if they are included
	// if they are included, leverage their additional properties
	// if they are not included they are either eol or custom
	obj, err := CreateResource(a.MqlRuntime, "azure.subscription.webService", map[string]*llx.RawData{})
	if err != nil {
		return nil, err
	}
	azureWeb := obj.(*mqlAzureSubscriptionWebService)
	runtimesPlugin := azureWeb.GetAvailableRuntimes()
	if runtimesPlugin.Error != nil {
		return nil, runtimesPlugin.Error
	}

	runtimes := runtimesPlugin.Data
	var match map[string]any

	for i := range runtimes {
		rt := runtimes[i]
		hashmap := rt.(map[string]any)
		if (hashmap["name"] == runtime.Name && hashmap["minorVersion"] == runtime.MinorVersion) || hashmap["id"] == runtime.ID {
			match = hashmap
		}
	}

	if match != nil {
		if match["isDefault"] != nil {
			runtime.IsDefault = match["isDefault"].(bool)
		}
		if match["isDeprecated"] != nil {
			runtime.IsDeprecated = match["isDeprecated"].(bool)
		}
	} else {
		// NOTE: it is only deprecated if a version number is provided, otherwise it is going to auto-update
		if len(runtime.MinorVersion) > 0 {
			runtime.IsDeprecated = true
		}
	}

	return convert.JsonToDict(runtime)
}

func (a *mqlAzureSubscriptionWebServiceAppsite) applicationSettings() (any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	id := a.Id.Data

	resourceID, err := ParseResourceID(id)
	if err != nil {
		return nil, err
	}

	site, err := resourceID.Component("sites")
	if err != nil {
		return nil, err
	}

	client, err := web.NewWebAppsClient(resourceID.SubscriptionID, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	settings, err := client.ListApplicationSettings(ctx, resourceID.ResourceGroup, site, &web.WebAppsClientListApplicationSettingsOptions{})
	if err != nil {
		return nil, err
	}

	res := make(map[string]any)

	for k := range settings.Properties {
		res[k] = convert.ToValue(settings.Properties[k])
	}

	return res, nil
}
