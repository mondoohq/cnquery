// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"encoding/json"
	"errors"
	"regexp"
	"strconv"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/rs/zerolog/log"

	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v11/providers/azure/connection"
	"go.mondoo.com/cnquery/v11/types"

	web "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appservice/armappservice"
)

var majorVersionRegex = regexp.MustCompile(`^(\d+)`)

func isPlatformEol(platform string, version string) bool {
	m := majorVersionRegex.FindString(version)
	if platform == "node" {

		val, err := strconv.Atoi(m)
		if err != nil {
			log.Error().Err(err).Str("platform", platform).Str("version", version).Msg("could not parse the azure webapp version")
			return false
		}

		if val < 10 || val == 11 {
			return true
		}
	}
	return false
}

type AzureWebAppStackRuntime struct {
	Name         string `json:"name,omitempty"`
	ID           string `json:"id,omitempty"`
	Os           string `json:"os,omitempty"`
	MajorVersion string `json:"majorVersion,omitempty"`
	MinorVersion string `json:"minorVersion,omitempty"`
	IsDeprecated bool   `json:"isDeprecated,omitempty"`
	IsHidden     bool   `json:"isHidden,omitempty"`
	IsDefault    bool   `json:"isDefault,omitempty"`
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

func (a *mqlAzureSubscriptionWebServiceAppsite) diagnosticSettings() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	return getDiagnosticSettings(a.Id.Data, a.MqlRuntime, conn)
}

func (a *mqlAzureSubscriptionWebService) apps() ([]interface{}, error) {
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
	res := []interface{}{}
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

func (a *mqlAzureSubscriptionWebService) availableRuntimes() ([]interface{}, error) {
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

	res := []interface{}{}
	windows := web.Enum15Windows
	// NOTE: we do not return MQL resource since stacks do not have their own proper id in azure
	windowsPager := client.NewGetAvailableStacksPager(&web.ProviderClientGetAvailableStacksOptions{OSTypeSelected: &windows})
	for windowsPager.More() {
		page, err := windowsPager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, entry := range page.Value {

			majorVersions := entry.Properties.MajorVersions
			for j := range majorVersions {
				majorVersion := majorVersions[j]

				// NOTE: yes, not all major versions include minor versions
				minorVersions := majorVersion.MinorVersions

				// special handling for dotnet and aspdotnet
				if len(minorVersions) == 0 {

					// NOTE: for dotnet, it seems the runtime is using the display version to create a stack
					// BUT: the stack itself reports the runtime version, therefore we need it to match the stacks
					runtimeVersion := convert.ToString(majorVersion.RuntimeVersion)
					// for dotnet, no runtime version is returned, therefore we need to use the display version
					if len(runtimeVersion) == 0 {
						runtimeVersion = convert.ToString(majorVersion.DisplayVersion)
					}

					runtime := AzureWebAppStackRuntime{
						Name: convert.ToString(entry.Name),

						ID:           strings.ToUpper(convert.ToString(entry.Name)) + "|" + runtimeVersion,
						Os:           "windows",
						MajorVersion: convert.ToString(majorVersion.DisplayVersion),
						IsDeprecated: convert.ToBool(majorVersion.IsDeprecated),
						IsHidden:     convert.ToBool(majorVersion.IsHidden),
						IsDefault:    convert.ToBool(majorVersion.IsDefault),
					}
					properties, err := convert.JsonToDict(runtime)
					if err != nil {
						return nil, err
					}
					res = append(res, properties)
				}

				for l := range minorVersions {
					minorVersion := minorVersions[l]

					runtime := AzureWebAppStackRuntime{
						Name:         convert.ToString(entry.Name),
						ID:           strings.ToUpper(convert.ToString(entry.Name)) + "|" + convert.ToString(minorVersion.RuntimeVersion),
						Os:           "windows",
						MinorVersion: convert.ToString(minorVersion.DisplayVersion),
						MajorVersion: convert.ToString(majorVersion.DisplayVersion),
						IsDeprecated: convert.ToBool(majorVersion.IsDeprecated) || isPlatformEol(convert.ToString(entry.Name), convert.ToString(minorVersion.RuntimeVersion)),
						IsHidden:     convert.ToBool(majorVersion.IsHidden),
						IsDefault:    convert.ToBool(majorVersion.IsDefault),
					}

					properties, err := convert.JsonToDict(runtime)
					if err != nil {
						return nil, err
					}
					res = append(res, properties)
				}
			}
		}
	}

	linux := web.Enum15Linux
	// fetch all linux stacks
	linuxPager := client.NewGetAvailableStacksPager(&web.ProviderClientGetAvailableStacksOptions{OSTypeSelected: &linux})
	for linuxPager.More() {
		page, err := linuxPager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, entry := range page.Value {

			majorVersions := entry.Properties.MajorVersions
			for j := range majorVersions {
				majorVersion := majorVersions[j]

				minorVersions := majorVersion.MinorVersions
				for l := range minorVersions {
					minorVersion := minorVersions[l]
					runtime := AzureWebAppStackRuntime{
						Name:         convert.ToString(entry.Name),
						ID:           convert.ToString(minorVersion.RuntimeVersion),
						Os:           "linux",
						MinorVersion: convert.ToString(minorVersion.DisplayVersion),
						MajorVersion: convert.ToString(majorVersion.DisplayVersion),
						IsDeprecated: convert.ToBool(majorVersion.IsDeprecated),
						IsHidden:     convert.ToBool(majorVersion.IsHidden),
						IsDefault:    convert.ToBool(majorVersion.IsDefault),
					}

					properties, err := convert.JsonToDict(runtime)
					if err != nil {
						return nil, err
					}
					res = append(res, properties)
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

func (a *mqlAzureSubscriptionWebServiceAppsite) metadata() (interface{}, error) {
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

	res := make(map[string]interface{})

	for k := range metadata.Properties {
		res[k] = convert.ToString(metadata.Properties[k])
	}

	return res, nil
}

func (a *mqlAzureSubscriptionWebServiceAppsite) functions() ([]interface{}, error) {
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
	res := []interface{}{}

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

func (a *mqlAzureSubscriptionWebServiceAppsite) connectionSettings() (interface{}, error) {
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

	res := make(map[string]interface{})

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
func (a *mqlAzureSubscriptionWebServiceAppsite) stack() (interface{}, error) {
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
		metadata, ok := metadata.(map[string]interface{})
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
			version = convert.ToString(properties.NetFrameworkVersion)
		case "dotnetcore":
			// NOTE: this will always return v4.0 no matter what you set (which is definitely wrong for dotnetcore)
			// The UI let you select different version, but in configure it does not show the version again
			// therefore we have no way than to hardcode the value for now
			version = "3.1"
		case "php":
			version = convert.ToString(properties.PhpVersion)
		case "python":
			version = convert.ToString(properties.PythonVersion)
		case "node":
			version = convert.ToString(properties.NodeVersion)
		case "powershell":
			version = convert.ToString(properties.PowerShellVersion)
		case "java":
			version = convert.ToString(properties.JavaVersion)
		case "javaContainer":
			version = convert.ToString(properties.JavaContainerVersion)
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
	var match map[string]interface{}

	for i := range runtimes {
		rt := runtimes[i]
		hashmap := rt.(map[string]interface{})
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

func (a *mqlAzureSubscriptionWebServiceAppsite) applicationSettings() (interface{}, error) {
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

	res := make(map[string]interface{})

	for k := range settings.Properties {
		res[k] = convert.ToString(settings.Properties[k])
	}

	return res, nil
}
