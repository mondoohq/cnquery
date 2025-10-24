// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
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
	AutoUpdate    bool      `json:"autoUpdate,omitempty"`
	EndOfLifeDate time.Time `json:"endOfLifeDate,omitempty"`
}

func createWebAppResourceFromSite(runtime *plugin.Runtime, resourceType string, site *web.Site) (any, error) {
	if site == nil {
		return nil, errors.New("site cannot be nil")
	}

	properties := map[string]any{}
	if site.Properties != nil {
		var err error
		properties, err = convert.JsonToDict(site.Properties)
		if err != nil {
			return nil, err
		}
	}

	identity := map[string]any{}
	if site.Identity != nil {
		var err error
		identity, err = convert.JsonToDict(site.Identity)
		if err != nil {
			return nil, err
		}
	}

	return CreateResource(runtime, resourceType,
		map[string]*llx.RawData{
			"id":         llx.StringDataPtr(site.ID),
			"name":       llx.StringDataPtr(site.Name),
			"location":   llx.StringDataPtr(site.Location),
			"tags":       llx.MapData(convert.PtrMapStrToInterface(site.Tags), types.String),
			"type":       llx.StringDataPtr(site.Type),
			"kind":       llx.StringDataPtr(site.Kind),
			"properties": llx.DictData(properties),
			"identity":   llx.DictData(identity),
		})
}

type runtimeStackDescriptor struct {
	Name         string
	MinorVersion string
	ID           string
	AutoUpdate   bool
	IsDeprecated bool
}

func runtimeStackDescriptorFromEntry(entry any) (*runtimeStackDescriptor, bool) {
	switch v := entry.(type) {
	case map[string]any:
		return runtimeStackDescriptorFromMap(v), true
	case *mqlAzureSubscriptionWebServiceAppRuntimeStack:
		return runtimeStackDescriptorFromResource(v), true
	default:
		return nil, false
	}
}

func runtimeStackDescriptorFromMap(values map[string]any) *runtimeStackDescriptor {
	descriptor := &runtimeStackDescriptor{}
	if values == nil {
		return descriptor
	}
	if name, ok := values["name"].(string); ok {
		descriptor.Name = strings.ToLower(name)
	}
	if minor, ok := values["minorVersion"].(string); ok {
		descriptor.MinorVersion = strings.ToLower(minor)
	}
	if id, ok := values["id"].(string); ok {
		descriptor.ID = id
	}
	if autoUpdate, ok := values["autoUpdate"].(bool); ok {
		descriptor.AutoUpdate = autoUpdate
	}
	if isDeprecated, ok := values["isDeprecated"].(bool); ok {
		descriptor.IsDeprecated = isDeprecated
	}
	return descriptor
}

func runtimeStackDescriptorFromResource(runtime *mqlAzureSubscriptionWebServiceAppRuntimeStack) *runtimeStackDescriptor {
	descriptor := &runtimeStackDescriptor{}
	if runtime == nil {
		return descriptor
	}
	var runtimeName string
	if name, ok := stringFromTValue(&runtime.Name); ok {
		runtimeName = name
		descriptor.Name = strings.ToLower(name)
	}
	if minor, ok := stringFromTValue(&runtime.MinorVersion); ok {
		descriptor.MinorVersion = strings.ToLower(minor)
	}
	subscriptionID := ""
	if runtime.MqlRuntime != nil {
		if conn, ok := runtime.MqlRuntime.Connection.(*connection.AzureConnection); ok {
			subscriptionID = conn.SubId()
		}
	}
	if subscriptionID != "" && runtimeName != "" {
		descriptor.ID = fmt.Sprintf("%s/%s", subscriptionID, runtimeName)
	} else if id, ok := stringFromTValue(&runtime.RuntimeVersion); ok {
		descriptor.ID = id
	}
	if autoUpdate, ok := boolFromTValue(&runtime.AutoUpdate); ok {
		descriptor.AutoUpdate = autoUpdate
	}
	if isDeprecated, ok := boolFromTValue(&runtime.Deprecated); ok {
		descriptor.IsDeprecated = isDeprecated
	}
	return descriptor
}

func stringFromTValue(tv *plugin.TValue[string]) (string, bool) {
	if tv == nil || !tv.IsSet() || tv.IsNull() {
		return "", false
	}
	return tv.Data, true
}

func boolFromTValue(tv *plugin.TValue[bool]) (bool, bool) {
	if tv == nil || !tv.IsSet() || tv.IsNull() {
		return false, false
	}
	return tv.Data, true
}

func computeWebAppStack(runtime *plugin.Runtime, config *mqlAzureSubscriptionWebServiceAppsiteconfig, metadata any) (any, error) {
	if config == nil {
		return nil, errors.New("web app configuration is nil")
	}

	configProperties := config.Properties.Data

	data, err := json.Marshal(configProperties)
	if err != nil {
		return nil, err
	}

	var properties web.SiteConfig
	err = json.Unmarshal(data, &properties)
	if err != nil {
		return nil, err
	}

	runtimeInfo := AzureWebAppStackRuntime{
		Os: "windows",
	}

	if properties.LinuxFxVersion == nil && properties.WindowsFxVersion == nil {
		return nil, errors.New("could not determine stack version")
	}

	if properties.LinuxFxVersion != nil && len(*properties.LinuxFxVersion) > 0 {
		runtimeInfo.Os = "linux"
		runtimeInfo.ID = *properties.LinuxFxVersion

		fxversion := strings.Split(*properties.LinuxFxVersion, "|")
		runtimeInfo.Name = strings.ToLower(fxversion[0])
		runtimeInfo.MinorVersion = strings.ToLower(fxversion[1])
	} else {
		metadataMap, ok := metadata.(map[string]any)
		if !ok {
			return nil, nil
		}

		stack, ok := metadataMap["CURRENT_STACK"].(string)
		if !ok {
			return nil, nil
		}
		version := ""
		switch stack {
		case "dotnet":
			stack = "aspnet"
			version = convert.ToValue(properties.NetFrameworkVersion)
		case "dotnetcore":
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

		runtimeInfo.Name = strings.ToLower(stack)
		runtimeInfo.ID = strings.ToUpper(stack) + "|" + version
		runtimeInfo.MinorVersion = version
	}

	obj, err := CreateResource(runtime, ResourceAzureSubscriptionWebService, map[string]*llx.RawData{})
	if err != nil {
		return nil, err
	}
	azureWeb := obj.(*mqlAzureSubscriptionWebService)
	runtimesPlugin := azureWeb.GetAvailableRuntimes()
	if runtimesPlugin.Error != nil {
		return nil, runtimesPlugin.Error
	}

	runtimes := runtimesPlugin.Data
	var match *runtimeStackDescriptor

	for _, rt := range runtimes {
		descriptor, ok := runtimeStackDescriptorFromEntry(rt)
		if !ok || descriptor == nil {
			continue
		}
		sameStack := descriptor.Name != "" && strings.EqualFold(descriptor.Name, runtimeInfo.Name) &&
			strings.EqualFold(descriptor.MinorVersion, runtimeInfo.MinorVersion)
		sameID := descriptor.ID != "" && strings.EqualFold(descriptor.ID, runtimeInfo.ID)
		if sameStack || sameID {
			match = descriptor
		}
	}

	if match != nil {
		runtimeInfo.AutoUpdate = match.AutoUpdate
		runtimeInfo.IsDeprecated = match.IsDeprecated
	} else {
		if len(runtimeInfo.MinorVersion) > 0 {
			runtimeInfo.IsDeprecated = true
		}
	}

	return convert.JsonToDict(runtimeInfo)
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

func (a *mqlAzureSubscriptionWebServiceAppsite) slots() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	resourceID, err := ParseResourceID(a.Id.Data)
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

	pager := client.NewListSlotsPager(resourceID.ResourceGroup, site, &web.WebAppsClientListSlotsOptions{})
	res := []any{}

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, entry := range page.Value {
			slotResource, err := createWebAppResourceFromSite(a.MqlRuntime, ResourceAzureSubscriptionWebServiceAppslot, entry)
			if err != nil {
				return nil, err
			}
			res = append(res, slotResource)
		}
	}

	return res, nil
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

			mqlAzure, err := CreateResource(a.MqlRuntime, ResourceAzureSubscriptionWebServiceAppsite,
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

					id := strings.Join([]string{
						ResourceAzureSubscription, subId,
						"webService.appRuntimeStack", os, runtimeVersion,
					}, "/")

					if _, exist := mapIDs[id]; exist {
						continue
					}
					mapIDs[id] = struct{}{}

					resource, err := NewResource(a.MqlRuntime, ResourceAzureSubscriptionWebServiceAppRuntimeStack,
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

	res, err := CreateResource(a.MqlRuntime, ResourceAzureSubscriptionWebServiceAppsiteconfig,
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

	res, err := CreateResource(a.MqlRuntime, ResourceAzureSubscriptionWebServiceAppsiteauthsettings,
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
	mqlResource, err := CreateResource(a.MqlRuntime, ResourceAzureSubscriptionWebServiceAppsiteBasicPublishingCredentialsPolicies, args)
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
	mqlResource, err := CreateResource(a.MqlRuntime, ResourceAzureSubscriptionWebServiceAppsiteBasicPublishingCredentialsPolicies, args)
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
			var respErr *azcore.ResponseError
			if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
				return res, nil
			}
			return nil, err
		}
		for _, entry := range page.Value {
			props, err := convert.JsonToDict(entry.Properties)
			if err != nil {
				return nil, err
			}
			mqlAzure, err := CreateResource(a.MqlRuntime, ResourceAzureSubscriptionWebServiceFunction,
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

func (a *mqlAzureSubscriptionWebServiceAppslot) webAppsClient() (*connection.AzureConnection, context.Context, *ResourceID, *web.WebAppsClient, string, string, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()

	resourceID, err := ParseResourceID(a.Id.Data)
	if err != nil {
		return nil, nil, nil, nil, "", "", err
	}

	site, err := resourceID.Component("sites")
	if err != nil {
		return nil, nil, nil, nil, "", "", err
	}

	slot, err := resourceID.Component("slots")
	if err != nil {
		return nil, nil, nil, nil, "", "", err
	}

	client, err := web.NewWebAppsClient(resourceID.SubscriptionID, conn.Token(), &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, nil, nil, nil, "", "", err
	}

	return conn, ctx, resourceID, client, site, slot, nil
}

func (a *mqlAzureSubscriptionWebServiceAppslot) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionWebServiceAppslot) diagnosticSettings() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	return getDiagnosticSettings(a.Id.Data, a.MqlRuntime, conn)
}

func (a *mqlAzureSubscriptionWebServiceAppslot) parent() (*mqlAzureSubscriptionWebServiceAppsite, error) {
	_, ctx, resourceID, client, site, _, err := a.webAppsClient()
	if err != nil {
		return nil, err
	}

	response, err := client.Get(ctx, resourceID.ResourceGroup, site, &web.WebAppsClientGetOptions{})
	if err != nil {
		return nil, err
	}

	parentResource, err := createWebAppResourceFromSite(a.MqlRuntime, ResourceAzureSubscriptionWebServiceAppsite, &response.Site)
	if err != nil {
		return nil, err
	}

	return parentResource.(*mqlAzureSubscriptionWebServiceAppsite), nil
}

func (a *mqlAzureSubscriptionWebServiceAppslot) configuration() (*mqlAzureSubscriptionWebServiceAppsiteconfig, error) {
	_, ctx, resourceID, client, site, slot, err := a.webAppsClient()
	if err != nil {
		return nil, err
	}

	configuration, err := client.GetConfigurationSlot(ctx, resourceID.ResourceGroup, site, slot, &web.WebAppsClientGetConfigurationSlotOptions{})
	if err != nil {
		return nil, err
	}

	properties := map[string]any{}
	if configuration.Properties != nil {
		props, err := convert.JsonToDict(configuration.Properties)
		if err != nil {
			return nil, err
		}
		properties = props
	}

	res, err := CreateResource(a.MqlRuntime, ResourceAzureSubscriptionWebServiceAppsiteconfig,
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

	return res.(*mqlAzureSubscriptionWebServiceAppsiteconfig), nil
}

func (a *mqlAzureSubscriptionWebServiceAppslot) authenticationSettings() (*mqlAzureSubscriptionWebServiceAppsiteauthsettings, error) {
	_, ctx, resourceID, client, site, slot, err := a.webAppsClient()
	if err != nil {
		return nil, err
	}

	configuration, err := client.GetAuthSettingsSlot(ctx, resourceID.ResourceGroup, site, slot, &web.WebAppsClientGetAuthSettingsSlotOptions{})
	if err != nil {
		return nil, err
	}

	properties := map[string]any{}
	if configuration.Properties != nil {
		props, err := convert.JsonToDict(configuration.Properties)
		if err != nil {
			return nil, err
		}
		properties = props
	}

	res, err := CreateResource(a.MqlRuntime, ResourceAzureSubscriptionWebServiceAppsiteauthsettings,
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

func (a *mqlAzureSubscriptionWebServiceAppslot) metadata() (any, error) {
	_, ctx, resourceID, client, site, slot, err := a.webAppsClient()
	if err != nil {
		return nil, err
	}

	metadata, err := client.ListMetadataSlot(ctx, resourceID.ResourceGroup, site, slot, &web.WebAppsClientListMetadataSlotOptions{})
	if err != nil {
		return nil, err
	}

	res := make(map[string]any)

	for k := range metadata.Properties {
		res[k] = convert.ToValue(metadata.Properties[k])
	}

	return res, nil
}

func (a *mqlAzureSubscriptionWebServiceAppslot) applicationSettings() (any, error) {
	_, ctx, resourceID, client, site, slot, err := a.webAppsClient()
	if err != nil {
		return nil, err
	}

	settings, err := client.ListApplicationSettingsSlot(ctx, resourceID.ResourceGroup, site, slot, &web.WebAppsClientListApplicationSettingsSlotOptions{})
	if err != nil {
		return nil, err
	}

	res := make(map[string]any)

	for k := range settings.Properties {
		res[k] = convert.ToValue(settings.Properties[k])
	}

	return res, nil
}

func (a *mqlAzureSubscriptionWebServiceAppslot) connectionSettings() (any, error) {
	_, ctx, resourceID, client, site, slot, err := a.webAppsClient()
	if err != nil {
		return nil, err
	}

	settings, err := client.ListConnectionStringsSlot(ctx, resourceID.ResourceGroup, site, slot, &web.WebAppsClientListConnectionStringsSlotOptions{})
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

func (a *mqlAzureSubscriptionWebServiceAppslot) stack() (any, error) {
	configPlugin := a.GetConfiguration()
	if configPlugin.Error != nil {
		return nil, configPlugin.Error
	}
	config := configPlugin.Data

	metadataPlugin := a.GetMetadata()
	if metadataPlugin.Error != nil {
		return nil, metadataPlugin.Error
	}
	metadata := metadataPlugin.Data

	return computeWebAppStack(a.MqlRuntime, config, metadata)
}

func (a *mqlAzureSubscriptionWebServiceAppslot) functions() ([]any, error) {
	_, ctx, resourceID, client, site, slot, err := a.webAppsClient()
	if err != nil {
		return nil, err
	}

	pager := client.NewListInstanceFunctionsSlotPager(resourceID.ResourceGroup, site, slot, &web.WebAppsClientListInstanceFunctionsSlotOptions{})
	res := []any{}

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			var respErr *azcore.ResponseError
			if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
				return res, nil
			}
			return nil, err
		}
		for _, entry := range page.Value {
			props, err := convert.JsonToDict(entry.Properties)
			if err != nil {
				return nil, err
			}
			mqlAzure, err := CreateResource(a.MqlRuntime, ResourceAzureSubscriptionWebServiceFunction,
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

func (a *mqlAzureSubscriptionWebServiceAppslot) ftp() (*mqlAzureSubscriptionWebServiceAppsiteBasicPublishingCredentialsPolicies, error) {
	_, ctx, resourceID, client, site, slot, err := a.webAppsClient()
	if err != nil {
		return nil, err
	}

	response, err := client.GetFtpAllowedSlot(ctx, resourceID.ResourceGroup, site, slot, &web.WebAppsClientGetFtpAllowedSlotOptions{})
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

	mqlResource, err := CreateResource(a.MqlRuntime, ResourceAzureSubscriptionWebServiceAppsiteBasicPublishingCredentialsPolicies, args)
	if err != nil {
		return nil, err
	}

	return mqlResource.(*mqlAzureSubscriptionWebServiceAppsiteBasicPublishingCredentialsPolicies), nil
}

func (a *mqlAzureSubscriptionWebServiceAppslot) scm() (*mqlAzureSubscriptionWebServiceAppsiteBasicPublishingCredentialsPolicies, error) {
	_, ctx, resourceID, client, site, slot, err := a.webAppsClient()
	if err != nil {
		return nil, err
	}

	response, err := client.GetScmAllowedSlot(ctx, resourceID.ResourceGroup, site, slot, &web.WebAppsClientGetScmAllowedSlotOptions{})
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

	mqlResource, err := CreateResource(a.MqlRuntime, ResourceAzureSubscriptionWebServiceAppsiteBasicPublishingCredentialsPolicies, args)
	if err != nil {
		return nil, err
	}

	return mqlResource.(*mqlAzureSubscriptionWebServiceAppsiteBasicPublishingCredentialsPolicies), nil
}

func (a *mqlAzureSubscriptionWebServiceFunction) id() (string, error) {
	return a.Id.Data, nil
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

	metadataPlugin := a.GetMetadata()
	if metadataPlugin.Error != nil {
		return nil, metadataPlugin.Error
	}
	metadata := metadataPlugin.Data

	return computeWebAppStack(a.MqlRuntime, config, metadata)
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

func (a *mqlAzureSubscriptionWebServiceAppsite) privateEndpointConnections() ([]any, error) {
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

	pager := client.NewGetPrivateEndpointConnectionListPager(resourceID.ResourceGroup, site, &web.WebAppsClientGetPrivateEndpointConnectionListOptions{})
	res := []any{}

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, entry := range page.Value {
			if entry == nil {
				continue
			}

			privateEndpoint := map[string]*llx.RawData{
				"id":   llx.StringDataPtr(entry.ID),
				"name": llx.StringDataPtr(entry.Name),
				"type": llx.StringDataPtr(entry.Type),
			}

			if entry.Properties != nil {
				props := entry.Properties
				propsMap, err := convert.JsonToDict(props)
				if err != nil {
					return nil, err
				}

				privateEndpoint["properties"] = llx.DictData(propsMap)

				if len(props.IPAddresses) > 0 {
					privateEndpoint["ipAddresses"] = llx.ArrayData(convert.SliceStrPtrToInterface(props.IPAddresses), types.String)
				}
				if props.PrivateEndpoint != nil {
					privateEndpoint["privateEndpointId"] = llx.StringDataPtr(props.PrivateEndpoint.ID)
				}
				if props.PrivateLinkServiceConnectionState != nil {
					stateArgs := map[string]*llx.RawData{
						"actionsRequired": llx.StringDataPtr(props.PrivateLinkServiceConnectionState.ActionsRequired),
						"description":     llx.StringDataPtr(props.PrivateLinkServiceConnectionState.Description),
						"status":          llx.StringDataPtr(props.PrivateLinkServiceConnectionState.Status),
					}
					stateRes, err := CreateResource(a.MqlRuntime, ResourceAzureSubscriptionPrivateEndpointConnectionConnectionState, stateArgs)
					if err != nil {
						return nil, err
					}
					privateEndpoint["privateLinkServiceConnectionState"] = llx.ResourceData(stateRes, ResourceAzureSubscriptionPrivateEndpointConnectionConnectionState)
				}
				if props.ProvisioningState != nil {
					privateEndpoint["provisioningState"] = llx.StringData(string(*props.ProvisioningState))
				}
			}

			mqlRes, err := CreateResource(a.MqlRuntime, ResourceAzureSubscriptionPrivateEndpointConnection, privateEndpoint)
			if err != nil {
				return nil, err
			}

			res = append(res, mqlRes)
		}
	}

	return res, nil
}

func (a *mqlAzureSubscriptionWebService) hostingEnvironments() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	id := a.GetSubscriptionId().Data

	client, err := web.NewEnvironmentsClient(id, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	pager := client.NewListPager(&web.EnvironmentsClientListOptions{})
	res := []any{}

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, entry := range page.Value {
			if entry == nil {
				continue
			}

			args := map[string]*llx.RawData{
				"id":       llx.StringDataPtr(entry.ID),
				"name":     llx.StringDataPtr(entry.Name),
				"type":     llx.StringDataPtr(entry.Type),
				"kind":     llx.StringDataPtr(entry.Kind),
				"location": llx.StringDataPtr(entry.Location),
				"tags":     llx.MapData(convert.PtrMapStrToInterface(entry.Tags), types.String),
			}

			if entry.Properties != nil {
				props := entry.Properties

				// Convert properties to dict
				propsDict, err := convert.JsonToDict(props)
				if err != nil {
					return nil, err
				}
				args["properties"] = llx.DictData(propsDict)

				args["dnsSuffix"] = llx.StringDataPtr(props.DNSSuffix)
				args["multiSize"] = llx.StringDataPtr(props.MultiSize)
				args["suspended"] = llx.BoolDataPtr(props.Suspended)
				args["hasLinuxWorkers"] = llx.BoolDataPtr(props.HasLinuxWorkers)
				args["zoneRedundant"] = llx.BoolDataPtr(props.ZoneRedundant)
				args["userWhitelistedIpRanges"] = llx.ArrayData(convert.SliceStrPtrToInterface(props.UserWhitelistedIPRanges), types.String)

				// Handle enum fields (need to convert to string)
				if props.Status != nil {
					args["status"] = llx.StringData(string(*props.Status))
				}
				if props.InternalLoadBalancingMode != nil {
					args["internalLoadBalancingMode"] = llx.StringData(string(*props.InternalLoadBalancingMode))
				}
				if props.ProvisioningState != nil {
					args["provisioningState"] = llx.StringData(string(*props.ProvisioningState))
				}
				args["maximumNumberOfMachines"] = llx.IntDataPtr(props.MaximumNumberOfMachines)
				args["multiRoleCount"] = llx.IntDataPtr(props.MultiRoleCount)

				args["frontEndScaleFactor"] = llx.IntDataPtr(props.FrontEndScaleFactor)
				args["ipsslAddressCount"] = llx.IntDataPtr(props.IpsslAddressCount)
				args["dedicatedHostCount"] = llx.IntDataPtr(props.DedicatedHostCount)

				if props.VirtualNetwork != nil {
					vnArgs := map[string]*llx.RawData{
						"id":     llx.StringDataPtr(props.VirtualNetwork.ID),
						"name":   llx.StringDataPtr(props.VirtualNetwork.Name),
						"type":   llx.StringDataPtr(props.VirtualNetwork.Type),
						"subnet": llx.StringDataPtr(props.VirtualNetwork.Subnet),
					}
					vnRes, err := CreateResource(a.MqlRuntime, ResourceAzureSubscriptionWebServiceHostingEnvironmentVirtualNetwork, vnArgs)
					if err != nil {
						return nil, err
					}
					args["virtualNetwork"] = llx.ResourceData(vnRes, vnRes.MqlName())
				}

				items := []any{}
				for _, setting := range props.ClusterSettings {
					if setting == nil {
						continue
					}
					dict, err := convert.JsonToDict(setting)
					if err != nil {
						return nil, err
					}
					items = append(items, dict)
				}
				args["clusterSettings"] = llx.ArrayData(items, types.Dict)
			}

			mqlRes, err := CreateResource(a.MqlRuntime, ResourceAzureSubscriptionWebServiceHostingEnvironment, args)
			if err != nil {
				return nil, err
			}

			res = append(res, mqlRes)
		}
	}

	return res, nil
}
