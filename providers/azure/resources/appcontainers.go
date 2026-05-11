// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	apps "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appcontainers/armappcontainers/v3"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/azure/connection"
	"go.mondoo.com/mql/v13/types"
)

type mqlAzureSubscriptionContainerAppServiceContainerAppInternal struct {
	cacheRGAndName struct {
		resourceGroup string
		name          string
	}
	revisionsLock      sync.Mutex
	revisionsFetched   atomic.Bool
	revisionsCache     []any
	authConfigsLock    sync.Mutex
	authConfigsFetched atomic.Bool
	authConfigsCache   []any
}

type mqlAzureSubscriptionContainerAppServiceManagedEnvironmentInternal struct {
	cacheRGAndName struct {
		resourceGroup string
		name          string
	}
	componentsLock      sync.Mutex
	componentsFetched   atomic.Bool
	componentsCache     []any
	certificatesLock    sync.Mutex
	certificatesFetched atomic.Bool
	certificatesCache   []any
}

func (a *mqlAzureSubscriptionContainerAppService) id() (string, error) {
	return "azure.subscription.containerAppService/" + a.SubscriptionId.Data, nil
}

func (a *mqlAzureSubscriptionContainerAppServiceManagedEnvironment) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionContainerAppServiceManagedEnvironmentDaprComponent) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionContainerAppServiceManagedEnvironmentCertificate) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionContainerAppServiceContainerApp) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionContainerAppServiceContainerAppContainer) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionContainerAppServiceContainerAppRevision) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionContainerAppServiceContainerAppAuthConfig) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionContainerAppServiceJob) id() (string, error) {
	return a.Id.Data, nil
}

func initAzureSubscriptionContainerAppService(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
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

// resourceGroupAndName parses an ARM resource ID and returns the resource
// group plus the leaf component for `leafKey` (e.g. "managedEnvironments",
// "containerApps"). Returns empty strings when the ID is malformed or the
// leaf key is absent.
func resourceGroupAndName(id, leafKey string) (resourceGroup, name string) {
	parsed, err := ParseResourceID(id)
	if err != nil {
		return "", ""
	}
	if leaf, lerr := parsed.Component(leafKey); lerr == nil {
		name = leaf
	}
	return parsed.ResourceGroup, name
}

func imagePinnedByDigest(image *string) bool {
	if image == nil {
		return false
	}
	return strings.Contains(*image, "@sha256:")
}

func acaClientOptions(conn *connection.AzureConnection) *arm.ClientOptions {
	return &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	}
}

// ----------------- managed environments -----------------

func (a *mqlAzureSubscriptionContainerAppService) managedEnvironments() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	subId := a.SubscriptionId.Data

	envClient, err := apps.NewManagedEnvironmentsClient(subId, conn.Token(), acaClientOptions(conn))
	if err != nil {
		return nil, err
	}

	res := []any{}
	pager := envClient.NewListBySubscriptionPager(nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, entry := range page.Value {
			mqlEnv, err := acaManagedEnvironmentToMQL(a.MqlRuntime, entry)
			if err != nil {
				return nil, err
			}
			res = append(res, mqlEnv)
		}
	}
	return res, nil
}

func acaManagedEnvironmentToMQL(runtime *plugin.Runtime, entry *apps.ManagedEnvironment) (plugin.Resource, error) {
	props := entry.Properties

	var staticIp, defaultDomain, provisioningState, kind string
	var zoneRedundant *bool
	// Default to false rather than leaving the pointer nil. An env without
	// a VnetConfiguration has no internal LB by definition; returning null
	// would break audits that check `internalLoadBalancerEnabled == false`.
	internalLB := false
	vnet := map[string]any{}
	workloadProfiles := []any{}
	peerAuth := map[string]any{}
	peerTraffic := map[string]any{}
	customDomain := map[string]any{}
	var logDest, logCustomerId string

	if entry.Kind != nil {
		kind = *entry.Kind
	}

	if props != nil {
		if props.StaticIP != nil {
			staticIp = *props.StaticIP
		}
		if props.DefaultDomain != nil {
			defaultDomain = *props.DefaultDomain
		}
		if props.ProvisioningState != nil {
			provisioningState = string(*props.ProvisioningState)
		}
		zoneRedundant = props.ZoneRedundant

		if props.VnetConfiguration != nil {
			d, err := convert.JsonToDict(props.VnetConfiguration)
			if err != nil {
				return nil, err
			}
			vnet = d
			if props.VnetConfiguration.Internal != nil {
				internalLB = *props.VnetConfiguration.Internal
			}
		}
		if len(props.WorkloadProfiles) > 0 {
			d, err := convert.JsonToDictSlice(props.WorkloadProfiles)
			if err != nil {
				return nil, err
			}
			workloadProfiles = d
		}
		if props.PeerAuthentication != nil {
			d, err := convert.JsonToDict(props.PeerAuthentication)
			if err != nil {
				return nil, err
			}
			peerAuth = d
		}
		if props.PeerTrafficConfiguration != nil {
			d, err := convert.JsonToDict(props.PeerTrafficConfiguration)
			if err != nil {
				return nil, err
			}
			peerTraffic = d
		}
		if props.CustomDomainConfiguration != nil {
			d, err := convert.JsonToDict(props.CustomDomainConfiguration)
			if err != nil {
				return nil, err
			}
			customDomain = d
		}
		if props.AppLogsConfiguration != nil {
			if props.AppLogsConfiguration.Destination != nil {
				logDest = *props.AppLogsConfiguration.Destination
			}
			if props.AppLogsConfiguration.LogAnalyticsConfiguration != nil &&
				props.AppLogsConfiguration.LogAnalyticsConfiguration.CustomerID != nil {
				logCustomerId = *props.AppLogsConfiguration.LogAnalyticsConfiguration.CustomerID
			}
		}
	}

	mqlEnv, err := CreateResource(runtime, "azure.subscription.containerAppService.managedEnvironment",
		map[string]*llx.RawData{
			"id":                          llx.StringDataPtr(entry.ID),
			"name":                        llx.StringDataPtr(entry.Name),
			"location":                    llx.StringDataPtr(entry.Location),
			"tags":                        llx.MapData(convert.PtrMapStrToInterface(entry.Tags), types.String),
			"provisioningState":           llx.StringData(provisioningState),
			"kind":                        llx.StringData(kind),
			"vnetConfiguration":           llx.DictData(vnet),
			"workloadProfiles":            llx.ArrayData(workloadProfiles, types.Dict),
			"staticIp":                    llx.StringData(staticIp),
			"defaultDomain":               llx.StringData(defaultDomain),
			"internalLoadBalancerEnabled": llx.BoolData(internalLB),
			"zoneRedundant":               llx.BoolDataPtr(zoneRedundant),
			"peerAuthentication":          llx.DictData(peerAuth),
			"peerTrafficConfiguration":    llx.DictData(peerTraffic),
			"customDomainConfiguration":   llx.DictData(customDomain),
			"logAnalyticsDestination":     llx.StringData(logDest),
			"logAnalyticsCustomerId":      llx.StringData(logCustomerId),
		})
	if err != nil {
		return nil, err
	}

	envRes := mqlEnv.(*mqlAzureSubscriptionContainerAppServiceManagedEnvironment)
	rg, name := resourceGroupAndName(envRes.Id.Data, "managedEnvironments")
	envRes.cacheRGAndName.resourceGroup = rg
	envRes.cacheRGAndName.name = name
	return envRes, nil
}

func (a *mqlAzureSubscriptionContainerAppServiceManagedEnvironment) daprComponents() ([]any, error) {
	if a.componentsFetched.Load() {
		return a.componentsCache, nil
	}
	a.componentsLock.Lock()
	defer a.componentsLock.Unlock()
	if a.componentsFetched.Load() {
		return a.componentsCache, nil
	}

	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	subId := conn.SubId()

	rg, envName := a.cacheRGAndName.resourceGroup, a.cacheRGAndName.name
	if rg == "" || envName == "" {
		rg, envName = resourceGroupAndName(a.Id.Data, "managedEnvironments")
	}

	client, err := apps.NewDaprComponentsClient(subId, conn.Token(), acaClientOptions(conn))
	if err != nil {
		return nil, err
	}

	res := []any{}
	pager := client.NewListPager(rg, envName, nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, entry := range page.Value {
			var componentType, version string
			var ignoreErrors *bool
			secretNames := []any{}
			scopes := []any{}
			secretsConfigured := false
			if entry.Properties != nil {
				if entry.Properties.ComponentType != nil {
					componentType = *entry.Properties.ComponentType
				}
				if entry.Properties.Version != nil {
					version = *entry.Properties.Version
				}
				ignoreErrors = entry.Properties.IgnoreErrors
				for _, sec := range entry.Properties.Secrets {
					if sec != nil && sec.Name != nil {
						secretNames = append(secretNames, *sec.Name)
						secretsConfigured = true
					}
				}
				for _, scope := range entry.Properties.Scopes {
					if scope != nil {
						scopes = append(scopes, *scope)
					}
				}
			}

			mqlComp, err := CreateResource(a.MqlRuntime, "azure.subscription.containerAppService.managedEnvironment.daprComponent",
				map[string]*llx.RawData{
					"id":                llx.StringDataPtr(entry.ID),
					"name":              llx.StringDataPtr(entry.Name),
					"componentType":     llx.StringData(componentType),
					"version":           llx.StringData(version),
					"secretsConfigured": llx.BoolData(secretsConfigured),
					"secretNames":       llx.ArrayData(secretNames, types.String),
					"scopes":            llx.ArrayData(scopes, types.String),
					"ignoreErrors":      llx.BoolDataPtr(ignoreErrors),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlComp)
		}
	}

	a.componentsCache = res
	a.componentsFetched.Store(true)
	return res, nil
}

func (a *mqlAzureSubscriptionContainerAppServiceManagedEnvironment) certificates() ([]any, error) {
	if a.certificatesFetched.Load() {
		return a.certificatesCache, nil
	}
	a.certificatesLock.Lock()
	defer a.certificatesLock.Unlock()
	if a.certificatesFetched.Load() {
		return a.certificatesCache, nil
	}

	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	subId := conn.SubId()

	rg, envName := a.cacheRGAndName.resourceGroup, a.cacheRGAndName.name
	if rg == "" || envName == "" {
		rg, envName = resourceGroupAndName(a.Id.Data, "managedEnvironments")
	}

	client, err := apps.NewCertificatesClient(subId, conn.Token(), acaClientOptions(conn))
	if err != nil {
		return nil, err
	}

	res := []any{}
	pager := client.NewListPager(rg, envName, nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, entry := range page.Value {
			var subject, thumbprint, provisioningState string
			var issueDate, notAfter *time.Time
			var validPtr *bool
			if entry.Properties != nil {
				p := entry.Properties
				if p.SubjectName != nil {
					subject = *p.SubjectName
				}
				if p.Thumbprint != nil {
					thumbprint = *p.Thumbprint
				}
				if p.ProvisioningState != nil {
					provisioningState = string(*p.ProvisioningState)
				}
				issueDate = p.IssueDate
				notAfter = p.ExpirationDate
				validPtr = p.Valid
			}
			valid := false
			if validPtr != nil {
				valid = *validPtr
			} else if notAfter != nil {
				valid = notAfter.After(time.Now())
			}

			mqlCert, err := CreateResource(a.MqlRuntime, "azure.subscription.containerAppService.managedEnvironment.certificate",
				map[string]*llx.RawData{
					"id":                llx.StringDataPtr(entry.ID),
					"name":              llx.StringDataPtr(entry.Name),
					"subjectName":       llx.StringData(subject),
					"thumbprint":        llx.StringData(thumbprint),
					"issueDate":         llx.TimeDataPtr(issueDate),
					"notAfter":          llx.TimeDataPtr(notAfter),
					"valid":             llx.BoolData(valid),
					"provisioningState": llx.StringData(provisioningState),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlCert)
		}
	}

	a.certificatesCache = res
	a.certificatesFetched.Store(true)
	return res, nil
}

// ----------------- container apps -----------------

func (a *mqlAzureSubscriptionContainerAppService) containerApps() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	subId := a.SubscriptionId.Data

	client, err := apps.NewContainerAppsClient(subId, conn.Token(), acaClientOptions(conn))
	if err != nil {
		return nil, err
	}

	res := []any{}
	pager := client.NewListBySubscriptionPager(nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, entry := range page.Value {
			mqlApp, err := acaContainerAppToMQL(a.MqlRuntime, entry)
			if err != nil {
				return nil, err
			}
			res = append(res, mqlApp)
		}
	}
	return res, nil
}

func acaContainerAppToMQL(runtime *plugin.Runtime, entry *apps.ContainerApp) (plugin.Resource, error) {
	props := entry.Properties

	var managedEnvironmentId, provisioningState, latestRevName, latestRevFqdn, revisionMode, workloadProfile string
	var ingressEnabled bool
	var ingressExternal *bool
	var targetPort *int32
	var transport, clientCertMode, stickySessions string
	httpsOnly := true
	corsOrigins := []any{}
	ipRules := []any{}
	registries := []any{}
	registryUsesIdentity := false
	secretNames := []any{}
	identity := map[string]any{}
	scaleRules := []any{}
	volumes := []any{}
	var minReplicas, maxReplicas *int32
	containerSpecs := []*apps.Container{}
	initContainerSpecs := []*apps.Container{}

	if props != nil {
		if props.EnvironmentID != nil {
			managedEnvironmentId = *props.EnvironmentID
		} else if props.ManagedEnvironmentID != nil {
			managedEnvironmentId = *props.ManagedEnvironmentID
		}
		if props.ProvisioningState != nil {
			provisioningState = string(*props.ProvisioningState)
		}
		if props.LatestRevisionName != nil {
			latestRevName = *props.LatestRevisionName
		}
		if props.LatestRevisionFqdn != nil {
			latestRevFqdn = *props.LatestRevisionFqdn
		}
		if props.WorkloadProfileName != nil {
			workloadProfile = *props.WorkloadProfileName
		}

		if props.Configuration != nil {
			cfg := props.Configuration
			if cfg.ActiveRevisionsMode != nil {
				revisionMode = string(*cfg.ActiveRevisionsMode)
			}
			if cfg.Ingress != nil {
				ingressEnabled = true
				ing := cfg.Ingress
				ingressExternal = ing.External
				targetPort = ing.TargetPort
				if ing.Transport != nil {
					transport = string(*ing.Transport)
				}
				if ing.AllowInsecure != nil {
					httpsOnly = !*ing.AllowInsecure
				}
				if ing.ClientCertificateMode != nil {
					clientCertMode = string(*ing.ClientCertificateMode)
				}
				if ing.StickySessions != nil && ing.StickySessions.Affinity != nil {
					stickySessions = string(*ing.StickySessions.Affinity)
				}
				if ing.CorsPolicy != nil {
					for _, o := range ing.CorsPolicy.AllowedOrigins {
						if o != nil {
							corsOrigins = append(corsOrigins, *o)
						}
					}
				}
				if len(ing.IPSecurityRestrictions) > 0 {
					d, err := convert.JsonToDictSlice(ing.IPSecurityRestrictions)
					if err != nil {
						return nil, err
					}
					ipRules = d
				}
			}
			if len(cfg.Registries) > 0 {
				d, err := convert.JsonToDictSlice(cfg.Registries)
				if err != nil {
					return nil, err
				}
				registries = d
				for _, r := range cfg.Registries {
					if r != nil && r.Identity != nil && *r.Identity != "" {
						registryUsesIdentity = true
					}
				}
			}
			for _, sec := range cfg.Secrets {
				if sec != nil && sec.Name != nil {
					secretNames = append(secretNames, *sec.Name)
				}
			}
		}

		if props.Template != nil {
			tpl := props.Template
			if tpl.Scale != nil {
				minReplicas = tpl.Scale.MinReplicas
				maxReplicas = tpl.Scale.MaxReplicas
				if len(tpl.Scale.Rules) > 0 {
					d, err := convert.JsonToDictSlice(tpl.Scale.Rules)
					if err != nil {
						return nil, err
					}
					scaleRules = d
				}
			}
			containerSpecs = tpl.Containers
			// apps.InitContainer has the same shape as apps.Container minus Probes
			// (the SDK's InitContainer type does not expose probes — Container Apps
			// does not run probes on init containers).
			for _, ic := range tpl.InitContainers {
				if ic == nil {
					continue
				}
				initContainerSpecs = append(initContainerSpecs, &apps.Container{
					Args:         ic.Args,
					Command:      ic.Command,
					Env:          ic.Env,
					Image:        ic.Image,
					Name:         ic.Name,
					Resources:    ic.Resources,
					VolumeMounts: ic.VolumeMounts,
				})
			}
			if len(tpl.Volumes) > 0 {
				d, err := convert.JsonToDictSlice(tpl.Volumes)
				if err != nil {
					return nil, err
				}
				volumes = d
			}
		}
	}

	if entry.Identity != nil {
		d, err := convert.JsonToDict(entry.Identity)
		if err != nil {
			return nil, err
		}
		identity = d
	}

	mqlApp, err := CreateResource(runtime, "azure.subscription.containerAppService.containerApp",
		map[string]*llx.RawData{
			"id":                       llx.StringDataPtr(entry.ID),
			"name":                     llx.StringDataPtr(entry.Name),
			"location":                 llx.StringDataPtr(entry.Location),
			"tags":                     llx.MapData(convert.PtrMapStrToInterface(entry.Tags), types.String),
			"provisioningState":        llx.StringData(provisioningState),
			"managedEnvironmentId":     llx.StringData(managedEnvironmentId),
			"revisionMode":             llx.StringData(revisionMode),
			"latestRevisionName":       llx.StringData(latestRevName),
			"latestRevisionFqdn":       llx.StringData(latestRevFqdn),
			"ingressEnabled":           llx.BoolData(ingressEnabled),
			"ingressExternal":          llx.BoolDataPtr(ingressExternal),
			"targetPort":               llx.IntDataDefault(targetPort, 0),
			"transport":                llx.StringData(transport),
			"httpsOnly":                llx.BoolData(httpsOnly),
			"stickySessions":           llx.StringData(stickySessions),
			"clientCertificateMode":    llx.StringData(clientCertMode),
			"corsAllowedOrigins":       llx.ArrayData(corsOrigins, types.String),
			"ipSecurityRestrictions":   llx.ArrayData(ipRules, types.Dict),
			"workloadProfileName":      llx.StringData(workloadProfile),
			"minReplicas":              llx.IntDataDefault(minReplicas, 0),
			"maxReplicas":              llx.IntDataDefault(maxReplicas, 0),
			"scaleRules":               llx.ArrayData(scaleRules, types.Dict),
			"identity":                 llx.DictData(identity),
			"registries":               llx.ArrayData(registries, types.Dict),
			"registryAuthUsesIdentity": llx.BoolData(registryUsesIdentity),
			"secretNames":              llx.ArrayData(secretNames, types.String),
			"volumes":                  llx.ArrayData(volumes, types.Dict),
		})
	if err != nil {
		return nil, err
	}

	appRes := mqlApp.(*mqlAzureSubscriptionContainerAppServiceContainerApp)
	rg, name := resourceGroupAndName(appRes.Id.Data, "containerApps")
	appRes.cacheRGAndName.resourceGroup = rg
	appRes.cacheRGAndName.name = name

	containers, err := acaContainersToMQL(runtime, appRes.Id.Data, containerSpecs, "containers")
	if err != nil {
		return nil, err
	}
	initContainers, err := acaContainersToMQL(runtime, appRes.Id.Data, initContainerSpecs, "initContainers")
	if err != nil {
		return nil, err
	}
	appRes.Containers = plugin.TValue[[]any]{Data: containers, State: plugin.StateIsSet}
	appRes.InitContainers = plugin.TValue[[]any]{Data: initContainers, State: plugin.StateIsSet}

	return appRes, nil
}

func acaContainersToMQL(runtime *plugin.Runtime, parentId string, specs []*apps.Container, segment string) ([]any, error) {
	res := []any{}
	for _, c := range specs {
		if c == nil {
			continue
		}
		var name, image, memory string
		if c.Name != nil {
			name = *c.Name
		}
		if c.Image != nil {
			image = *c.Image
		}

		var cpu *float64
		if c.Resources != nil {
			cpu = c.Resources.CPU
			if c.Resources.Memory != nil {
				memory = *c.Resources.Memory
			}
		}

		command := []any{}
		for _, s := range c.Command {
			if s != nil {
				command = append(command, *s)
			}
		}
		args := []any{}
		for _, s := range c.Args {
			if s != nil {
				args = append(args, *s)
			}
		}

		env := []any{}
		if len(c.Env) > 0 {
			d, err := convert.JsonToDictSlice(c.Env)
			if err != nil {
				return nil, err
			}
			env = d
		}
		probes := []any{}
		if len(c.Probes) > 0 {
			d, err := convert.JsonToDictSlice(c.Probes)
			if err != nil {
				return nil, err
			}
			probes = d
		}
		mounts := []any{}
		if len(c.VolumeMounts) > 0 {
			d, err := convert.JsonToDictSlice(c.VolumeMounts)
			if err != nil {
				return nil, err
			}
			mounts = d
		}

		mqlContainer, err := CreateResource(runtime, "azure.subscription.containerAppService.containerApp.container",
			map[string]*llx.RawData{
				"id":           llx.StringData(parentId + "/" + segment + "/" + name),
				"name":         llx.StringData(name),
				"image":        llx.StringData(image),
				"imagePinned":  llx.BoolData(imagePinnedByDigest(c.Image)),
				"command":      llx.ArrayData(command, types.String),
				"args":         llx.ArrayData(args, types.String),
				"env":          llx.ArrayData(env, types.Dict),
				"cpuCores":     llx.FloatData(convert.ToValue(cpu)),
				"memory":       llx.StringData(memory),
				"probes":       llx.ArrayData(probes, types.Dict),
				"volumeMounts": llx.ArrayData(mounts, types.Dict),
			})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlContainer)
	}
	return res, nil
}

// containers/initContainers are populated eagerly when the app is created.
// These methods only run if MQL ever reaches them with State unset.
func (a *mqlAzureSubscriptionContainerAppServiceContainerApp) containers() ([]any, error) {
	return []any{}, nil
}

func (a *mqlAzureSubscriptionContainerAppServiceContainerApp) initContainers() ([]any, error) {
	return []any{}, nil
}

func (a *mqlAzureSubscriptionContainerAppServiceContainerApp) revisions() ([]any, error) {
	if a.revisionsFetched.Load() {
		return a.revisionsCache, nil
	}
	a.revisionsLock.Lock()
	defer a.revisionsLock.Unlock()
	if a.revisionsFetched.Load() {
		return a.revisionsCache, nil
	}

	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	subId := conn.SubId()

	rg, appName := a.cacheRGAndName.resourceGroup, a.cacheRGAndName.name
	if rg == "" || appName == "" {
		rg, appName = resourceGroupAndName(a.Id.Data, "containerApps")
	}

	client, err := apps.NewContainerAppsRevisionsClient(subId, conn.Token(), acaClientOptions(conn))
	if err != nil {
		return nil, err
	}

	res := []any{}
	pager := client.NewListRevisionsPager(rg, appName, nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, entry := range page.Value {
			var active *bool
			var trafficWeight, replicas *int32
			var provisioningState, healthState string
			var createdTime, lastActiveTime *time.Time
			if entry.Properties != nil {
				p := entry.Properties
				active = p.Active
				trafficWeight = p.TrafficWeight
				replicas = p.Replicas
				if p.ProvisioningState != nil {
					provisioningState = string(*p.ProvisioningState)
				}
				if p.HealthState != nil {
					healthState = string(*p.HealthState)
				}
				createdTime = p.CreatedTime
				lastActiveTime = p.LastActiveTime
			}
			mqlRev, err := CreateResource(a.MqlRuntime, "azure.subscription.containerAppService.containerApp.revision",
				map[string]*llx.RawData{
					"id":                llx.StringDataPtr(entry.ID),
					"name":              llx.StringDataPtr(entry.Name),
					"active":            llx.BoolDataPtr(active),
					"trafficWeight":     llx.IntDataDefault(trafficWeight, 0),
					"replicas":          llx.IntDataDefault(replicas, 0),
					"provisioningState": llx.StringData(provisioningState),
					"healthState":       llx.StringData(healthState),
					"createdTime":       llx.TimeDataPtr(createdTime),
					"lastActiveTime":    llx.TimeDataPtr(lastActiveTime),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlRev)
		}
	}

	a.revisionsCache = res
	a.revisionsFetched.Store(true)
	return res, nil
}

func (a *mqlAzureSubscriptionContainerAppServiceContainerApp) authConfigs() ([]any, error) {
	if a.authConfigsFetched.Load() {
		return a.authConfigsCache, nil
	}
	a.authConfigsLock.Lock()
	defer a.authConfigsLock.Unlock()
	if a.authConfigsFetched.Load() {
		return a.authConfigsCache, nil
	}

	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	subId := conn.SubId()

	rg, appName := a.cacheRGAndName.resourceGroup, a.cacheRGAndName.name
	if rg == "" || appName == "" {
		rg, appName = resourceGroupAndName(a.Id.Data, "containerApps")
	}

	client, err := apps.NewContainerAppsAuthConfigsClient(subId, conn.Token(), acaClientOptions(conn))
	if err != nil {
		return nil, err
	}

	res := []any{}
	pager := client.NewListByContainerAppPager(rg, appName, nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			// Container apps without auth configured surface as 404
			// AuthConfigNotFound from the list endpoint, not as an empty
			// page. Match the ErrorCode specifically so a 404 from a
			// different cause (e.g. the container app was deleted
			// mid-query, or a future API change) still surfaces as a
			// real error.
			var rerr *azcore.ResponseError
			if errors.As(err, &rerr) && rerr.StatusCode == http.StatusNotFound && rerr.ErrorCode == "AuthConfigNotFound" {
				a.authConfigsCache = res
				a.authConfigsFetched.Store(true)
				return res, nil
			}
			return nil, err
		}
		for _, entry := range page.Value {
			enabled := false
			var unauth string
			providers := []any{}
			login := map[string]any{}
			httpSettings := map[string]any{}
			if entry.Properties != nil {
				p := entry.Properties
				if p.Platform != nil && p.Platform.Enabled != nil {
					enabled = *p.Platform.Enabled
				}
				if p.GlobalValidation != nil && p.GlobalValidation.UnauthenticatedClientAction != nil {
					unauth = string(*p.GlobalValidation.UnauthenticatedClientAction)
				}
				if p.IdentityProviders != nil {
					if p.IdentityProviders.AzureActiveDirectory != nil {
						providers = append(providers, "azureActiveDirectory")
					}
					if p.IdentityProviders.GitHub != nil {
						providers = append(providers, "github")
					}
					if p.IdentityProviders.Google != nil {
						providers = append(providers, "google")
					}
					if p.IdentityProviders.Facebook != nil {
						providers = append(providers, "facebook")
					}
					if p.IdentityProviders.Twitter != nil {
						providers = append(providers, "twitter")
					}
					if p.IdentityProviders.Apple != nil {
						providers = append(providers, "apple")
					}
					for k := range p.IdentityProviders.CustomOpenIDConnectProviders {
						providers = append(providers, "customOpenIdConnect:"+k)
					}
				}
				if p.Login != nil {
					d, err := convert.JsonToDict(p.Login)
					if err != nil {
						return nil, err
					}
					login = d
				}
				if p.HTTPSettings != nil {
					d, err := convert.JsonToDict(p.HTTPSettings)
					if err != nil {
						return nil, err
					}
					httpSettings = d
				}
			}

			mqlAuth, err := CreateResource(a.MqlRuntime, "azure.subscription.containerAppService.containerApp.authConfig",
				map[string]*llx.RawData{
					"id":                          llx.StringDataPtr(entry.ID),
					"name":                        llx.StringDataPtr(entry.Name),
					"enabled":                     llx.BoolData(enabled),
					"unauthenticatedClientAction": llx.StringData(unauth),
					"identityProviders":           llx.ArrayData(providers, types.String),
					"login":                       llx.DictData(login),
					"httpSettings":                llx.DictData(httpSettings),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlAuth)
		}
	}

	a.authConfigsCache = res
	a.authConfigsFetched.Store(true)
	return res, nil
}

// ----------------- jobs -----------------

func (a *mqlAzureSubscriptionContainerAppService) jobs() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	subId := a.SubscriptionId.Data

	client, err := apps.NewJobsClient(subId, conn.Token(), acaClientOptions(conn))
	if err != nil {
		return nil, err
	}

	res := []any{}
	pager := client.NewListBySubscriptionPager(nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, entry := range page.Value {
			var managedEnvId, provisioningState, triggerType, cron, workloadProfile string
			var replicaTimeout, replicaRetry *int32
			eventTrigger := map[string]any{}
			containers := []any{}
			registries := []any{}
			registryUsesIdentity := false
			secretNames := []any{}
			identity := map[string]any{}

			if entry.Properties != nil {
				p := entry.Properties
				if p.EnvironmentID != nil {
					managedEnvId = *p.EnvironmentID
				}
				if p.ProvisioningState != nil {
					provisioningState = string(*p.ProvisioningState)
				}
				if p.WorkloadProfileName != nil {
					workloadProfile = *p.WorkloadProfileName
				}
				if p.Configuration != nil {
					cfg := p.Configuration
					if cfg.TriggerType != nil {
						triggerType = string(*cfg.TriggerType)
					}
					replicaTimeout = cfg.ReplicaTimeout
					replicaRetry = cfg.ReplicaRetryLimit
					if cfg.ScheduleTriggerConfig != nil && cfg.ScheduleTriggerConfig.CronExpression != nil {
						cron = *cfg.ScheduleTriggerConfig.CronExpression
					}
					if cfg.EventTriggerConfig != nil {
						d, err := convert.JsonToDict(cfg.EventTriggerConfig)
						if err != nil {
							return nil, err
						}
						eventTrigger = d
					}
					if len(cfg.Registries) > 0 {
						d, err := convert.JsonToDictSlice(cfg.Registries)
						if err != nil {
							return nil, err
						}
						registries = d
						for _, r := range cfg.Registries {
							if r != nil && r.Identity != nil && *r.Identity != "" {
								registryUsesIdentity = true
							}
						}
					}
					for _, sec := range cfg.Secrets {
						if sec != nil && sec.Name != nil {
							secretNames = append(secretNames, *sec.Name)
						}
					}
				}
				if p.Template != nil && len(p.Template.Containers) > 0 {
					d, err := convert.JsonToDictSlice(p.Template.Containers)
					if err != nil {
						return nil, err
					}
					containers = d
				}
			}
			if entry.Identity != nil {
				d, err := convert.JsonToDict(entry.Identity)
				if err != nil {
					return nil, err
				}
				identity = d
			}

			mqlJob, err := CreateResource(a.MqlRuntime, "azure.subscription.containerAppService.job",
				map[string]*llx.RawData{
					"id":                       llx.StringDataPtr(entry.ID),
					"name":                     llx.StringDataPtr(entry.Name),
					"location":                 llx.StringDataPtr(entry.Location),
					"tags":                     llx.MapData(convert.PtrMapStrToInterface(entry.Tags), types.String),
					"managedEnvironmentId":     llx.StringData(managedEnvId),
					"provisioningState":        llx.StringData(provisioningState),
					"triggerType":              llx.StringData(triggerType),
					"cronExpression":           llx.StringData(cron),
					"eventTriggerConfig":       llx.DictData(eventTrigger),
					"replicaTimeoutSeconds":    llx.IntDataDefault(replicaTimeout, 0),
					"replicaRetryLimit":        llx.IntDataDefault(replicaRetry, 0),
					"identity":                 llx.DictData(identity),
					"containers":               llx.ArrayData(containers, types.Dict),
					"workloadProfileName":      llx.StringData(workloadProfile),
					"registries":               llx.ArrayData(registries, types.Dict),
					"registryAuthUsesIdentity": llx.BoolData(registryUsesIdentity),
					"secretNames":              llx.ArrayData(secretNames, types.String),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlJob)
		}
	}
	return res, nil
}
