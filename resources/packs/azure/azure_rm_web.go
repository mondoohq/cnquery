package azure

import (
	"context"
	"encoding/json"
	"errors"
	"regexp"
	"strconv"
	"strings"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/web/mgmt/web"
	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/resources/packs/core"
)

func (a *lumiAzurermWeb) id() (string, error) {
	return "azurerm.web", nil
}

func (a *lumiAzurermWeb) GetApps() ([]interface{}, error) {
	at, err := azuretransport(a.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	authorizer, err := at.Authorizer()
	if err != nil {
		return nil, err
	}

	client := web.NewAppsClient(at.SubscriptionID())
	client.Authorizer = authorizer

	webapps, err := client.List(ctx)
	if err != nil {
		return nil, err
	}

	res := []interface{}{}
	for i := range webapps.Values() {
		entry := webapps.Values()[i]

		properties, err := core.JsonToDict(entry.SiteProperties)
		if err != nil {
			return nil, err
		}

		identity, err := core.JsonToDict(entry.Identity)
		if err != nil {
			return nil, err
		}

		lumiAzure, err := a.MotorRuntime.CreateResource("azurerm.web.appsite",
			"id", core.ToString(entry.ID),
			"name", core.ToString(entry.Name),
			"location", core.ToString(entry.Location),
			"tags", azureTagsToInterface(entry.Tags),
			"type", core.ToString(entry.Type),
			"kind", core.ToString(entry.Kind),
			"properties", properties,
			"identity", identity,
		)
		if err != nil {
			return nil, err
		}
		res = append(res, lumiAzure)
	}

	return res, nil
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

// all runtimes that are returned here are not EOL and are supported
func (a *lumiAzurermWeb) GetAvailableRuntimes() ([]interface{}, error) {
	at, err := azuretransport(a.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	authorizer, err := at.Authorizer()
	if err != nil {
		return nil, err
	}

	client := web.NewProviderClient(at.SubscriptionID())
	client.Authorizer = authorizer

	res := []interface{}{}
	// NOTE: we do not return a lumi resource since stacks do not have their own proper id in azure

	// fetch all windows stacks
	// NOTE: 💥 This api is one of the worst I've ever seen and I understand the az client team why they maintain a hardcoded list
	// - behaves completely different for linux and windows
	// - even for windows, its output is different for different runtimes
	// - versions are unreliable, on linux it includes only maintained runtimes 🎉 on windows it behaves different for each runtime
	// - some entries have minor versions, some dont, on linux all major version include at least one minor version
	// - some devs at microsoft seem to be unsure if node.js is supported for windows, this api show all version even unmainted ones, the ui does not support nodejs for windows at all
	stacks, err := client.GetAvailableStacks(ctx, "windows")
	if err != nil {
		return nil, err
	}

	for i := range stacks.Values() {
		entry := stacks.Values()[i]

		majorVersions := *entry.ApplicationStack.MajorVersions
		for j := range majorVersions {
			majorVersion := majorVersions[j]

			// NOTE: yes, not all major versions include minor versions
			minorVersions := *majorVersion.MinorVersions

			// special handling for dotnet and aspdotnet
			if len(minorVersions) == 0 {

				// NOTE: for dotnet, it seems the runtime is using the display version to create a stack
				// BUT: the stack itself reports the runtime version, therefore we need it to match the stacks
				runtimeVersion := core.ToString(majorVersion.RuntimeVersion)
				// for dotnet, no runtime version is returned, therefore we need to use the display version
				if len(runtimeVersion) == 0 {
					runtimeVersion = core.ToString(majorVersion.DisplayVersion)
				}

				runtime := AzureWebAppStackRuntime{
					Name: core.ToString(entry.Name),

					ID:           strings.ToUpper(core.ToString(entry.Name)) + "|" + runtimeVersion,
					Os:           "windows",
					MajorVersion: core.ToString(majorVersion.DisplayVersion),
					IsDeprecated: core.ToBool(majorVersion.IsDeprecated),
					IsHidden:     core.ToBool(majorVersion.IsHidden),
					IsDefault:    core.ToBool(majorVersion.IsDefault),
				}
				properties, err := core.JsonToDict(runtime)
				if err != nil {
					return nil, err
				}
				res = append(res, properties)
			}

			for l := range minorVersions {
				minorVersion := minorVersions[l]

				runtime := AzureWebAppStackRuntime{
					Name:         core.ToString(entry.Name),
					ID:           strings.ToUpper(core.ToString(entry.Name)) + "|" + core.ToString(minorVersion.RuntimeVersion),
					Os:           "windows",
					MinorVersion: core.ToString(minorVersion.DisplayVersion),
					MajorVersion: core.ToString(majorVersion.DisplayVersion),
					IsDeprecated: core.ToBool(majorVersion.IsDeprecated) || isPlatformEol(core.ToString(entry.Name), core.ToString(minorVersion.RuntimeVersion)),
					IsHidden:     core.ToBool(majorVersion.IsHidden),
					IsDefault:    core.ToBool(majorVersion.IsDefault),
				}

				properties, err := core.JsonToDict(runtime)
				if err != nil {
					return nil, err
				}
				res = append(res, properties)
			}
		}
	}

	// fetch all linux stacks
	stacks, err = client.GetAvailableStacks(ctx, "linux")
	if err != nil {
		return nil, err
	}

	for i := range stacks.Values() {
		entry := stacks.Values()[i]

		majorVersions := *entry.ApplicationStack.MajorVersions
		for j := range majorVersions {
			majorVersion := majorVersions[j]

			minorVersions := *majorVersion.MinorVersions
			for l := range minorVersions {
				minorVersion := minorVersions[l]
				runtime := AzureWebAppStackRuntime{
					Name:         core.ToString(entry.Name),
					ID:           core.ToString(minorVersion.RuntimeVersion),
					Os:           "linux",
					MinorVersion: core.ToString(minorVersion.DisplayVersion),
					MajorVersion: core.ToString(majorVersion.DisplayVersion),
					IsDeprecated: core.ToBool(majorVersion.IsDeprecated),
					IsHidden:     core.ToBool(majorVersion.IsHidden),
					IsDefault:    core.ToBool(majorVersion.IsDefault),
				}

				properties, err := core.JsonToDict(runtime)
				if err != nil {
					return nil, err
				}
				res = append(res, properties)
			}
		}
	}

	return res, nil
}

func (a *lumiAzurermWebAppsite) id() (string, error) {
	return a.Id()
}

func (a *lumiAzurermWebAppsite) GetConfiguration() (interface{}, error) {
	at, err := azuretransport(a.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	// id is a azure resource od
	id, err := a.Id()
	if err != nil {
		return nil, err
	}

	resourceID, err := at.ParseResourceID(id)
	if err != nil {
		return nil, err
	}

	site, err := resourceID.Component("sites")
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	authorizer, err := at.Authorizer()
	if err != nil {
		return nil, err
	}

	client := web.NewAppsClient(resourceID.SubscriptionID)
	client.Authorizer = authorizer

	configuration, err := client.GetConfiguration(ctx, resourceID.ResourceGroup, site)
	if err != nil {
		return nil, err
	}

	entry := configuration
	properties, err := core.JsonToDict(entry.SiteConfig)
	if err != nil {
		return nil, err
	}

	return a.MotorRuntime.CreateResource("azurerm.web.appsiteconfig",
		"id", core.ToString(entry.ID),
		"name", core.ToString(entry.Name),
		"kind", core.ToString(entry.Kind),
		"type", core.ToString(entry.Type),
		"properties", properties,
	)
}

func (a *lumiAzurermWebAppsite) GetAuthenticationSettings() (interface{}, error) {
	at, err := azuretransport(a.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	// id is a azure resource od
	id, err := a.Id()
	if err != nil {
		return nil, err
	}

	resourceID, err := at.ParseResourceID(id)
	if err != nil {
		return nil, err
	}

	site, err := resourceID.Component("sites")
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	authorizer, err := at.Authorizer()
	if err != nil {
		return nil, err
	}

	client := web.NewAppsClient(resourceID.SubscriptionID)
	client.Authorizer = authorizer

	authSettings, err := client.GetAuthSettings(ctx, resourceID.ResourceGroup, site)
	if err != nil {
		return nil, err
	}

	entry := authSettings
	properties, err := core.JsonToDict(entry.SiteAuthSettingsProperties)
	if err != nil {
		return nil, err
	}

	return a.MotorRuntime.CreateResource("azurerm.web.appsiteauthsettings",
		"id", core.ToString(entry.ID),
		"name", core.ToString(entry.Name),
		"kind", core.ToString(entry.Kind),
		"type", core.ToString(entry.Type),
		"properties", properties,
	)
}

func (a *lumiAzurermWebAppsite) GetApplicationSettings() (interface{}, error) {
	at, err := azuretransport(a.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	// id is a azure resource od
	id, err := a.Id()
	if err != nil {
		return nil, err
	}

	resourceID, err := at.ParseResourceID(id)
	if err != nil {
		return nil, err
	}

	site, err := resourceID.Component("sites")
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	authorizer, err := at.Authorizer()
	if err != nil {
		return nil, err
	}

	client := web.NewAppsClient(resourceID.SubscriptionID)
	client.Authorizer = authorizer

	settings, err := client.ListApplicationSettings(ctx, resourceID.ResourceGroup, site)
	if err != nil {
		return nil, err
	}

	res := make(map[string](interface{}))

	for k := range settings.Properties {
		res[k] = core.ToString(settings.Properties[k])
	}

	return res, nil
}

func (a *lumiAzurermWebAppsite) GetMetadata() (interface{}, error) {
	at, err := azuretransport(a.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	// id is a azure resource od
	id, err := a.Id()
	if err != nil {
		return nil, err
	}

	resourceID, err := at.ParseResourceID(id)
	if err != nil {
		return nil, err
	}

	site, err := resourceID.Component("sites")
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	authorizer, err := at.Authorizer()
	if err != nil {
		return nil, err
	}

	client := web.NewAppsClient(resourceID.SubscriptionID)
	client.Authorizer = authorizer

	metadata, err := client.ListMetadata(ctx, resourceID.ResourceGroup, site)
	if err != nil {
		return nil, err
	}

	res := make(map[string](interface{}))

	for k := range metadata.Properties {
		res[k] = core.ToString(metadata.Properties[k])
	}

	return res, nil
}

func (a *lumiAzurermWebAppsite) GetConnectionSettings() (interface{}, error) {
	at, err := azuretransport(a.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	// id is a azure resource od
	id, err := a.Id()
	if err != nil {
		return nil, err
	}

	resourceID, err := at.ParseResourceID(id)
	if err != nil {
		return nil, err
	}

	site, err := resourceID.Component("sites")
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	authorizer, err := at.Authorizer()
	if err != nil {
		return nil, err
	}

	client := web.NewAppsClient(resourceID.SubscriptionID)
	client.Authorizer = authorizer

	settings, err := client.ListConnectionStrings(ctx, resourceID.ResourceGroup, site)
	if err != nil {
		return nil, err
	}

	res := make(map[string](interface{}))

	for k := range settings.Properties {

		value, err := core.JsonToDict(settings.Properties[k])
		if err != nil {
			return nil, err
		}

		res[k] = value
	}

	return res, nil
}

func (a *lumiAzurermWebAppsite) GetStack() (map[string]interface{}, error) {
	config, err := a.Configuration()
	if err != nil {
		return nil, err
	}

	// read confguration into go struct
	configProperties, err := config.Properties()
	if err != nil {
		return nil, err
	}

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
	metadataRaw, err := a.Metadata()
	if err != nil {
		return nil, err
	}

	runtime := AzureWebAppStackRuntime{
		Os: "windows",
	}

	// LOL Fact: WindowsFxVersion is never set :-)
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
		metadata, ok := metadataRaw.(map[string]interface{})
		if !ok {
			return nil, nil // see behavior below
		}

		// read runtime from metadata, YES its works completely different than on linux
		// NOTE: also take care of the runtime version for dotnet. This API and webapp runtime
		// handling in specific is a complete 💥.
		stack, ok := metadata["CURRENT_STACK"].(string)
		if !ok {
			// This doesn't seem to be consistently set
			// https://stackoverflow.com/questions/63950946/azure-app-service-get-stack-settings-via-api#comment113188903_63987100
			// https://github.com/mondoohq/client/issues/157
			return nil, nil
		}
		version := ""
		switch stack {
		case "dotnet":
			stack = "aspnet" // needs to be overwritten (do not ask)
			version = core.ToString(properties.NetFrameworkVersion)
		case "dotnetcore":
			// NOTE: this will always return v4.0 no matter what you set (which is definitly wrong for dotnetcore)
			// The UI let you select different version, but in confgure it does not show the version again
			// therefore we have no way than to hardcode the value for now
			version = "3.1"
		case "php":
			version = core.ToString(properties.PhpVersion)
		case "python":
			version = core.ToString(properties.PythonVersion)
		case "node":
			version = core.ToString(properties.NodeVersion)
		case "powershell":
			version = core.ToString(properties.PowerShellVersion)
		case "java":
			version = core.ToString(properties.JavaVersion)
		case "javaContainer":
			version = core.ToString(properties.JavaContainerVersion)
		}

		runtime.Name = strings.ToLower(stack)
		runtime.ID = strings.ToUpper(stack) + "|" + version
		runtime.MinorVersion = version
	}

	// fetch available runtimes and check if they are included
	// if they are included, leverage their additional properties
	// if they are not included they are either eol or custom
	obj, err := a.MotorRuntime.CreateResource("azurerm.web")
	if err != nil {
		return nil, err
	}
	azureWeb := obj.(AzurermWeb)
	runtimes, err := azureWeb.AvailableRuntimes()
	if err != nil {
		return nil, err
	}

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

	return core.JsonToDict(runtime)
}

func (a *lumiAzurermWebAppsiteconfig) id() (string, error) {
	return a.Id()
}

func (a *lumiAzurermWebAppsiteauthsettings) id() (string, error) {
	return a.Id()
}
