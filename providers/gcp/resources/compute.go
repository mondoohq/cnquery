package resources

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/types"
	"google.golang.org/api/cloudresourcemanager/v3"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/iam/v1"
	"google.golang.org/api/option"
)

// func (g *mqlGcpProjectComputeService) init(args *resources.Args) (*resources.Args, GcpProjectComputeService, error) {
// 	if len(*args) > 2 {
// 		return args, nil, nil
// 	}

// 	provider, err := gcpProvider(g.MotorRuntime.Motor.Provider)
// 	if err != nil {
// 		return nil, nil, err
// 	}

// 	projectId := provider.ResourceID()
// 	(*args)["projectId"] = projectId

// 	return args, nil, nil
// }

func (g *mqlGcpProject) compute() (*mqlGcpProjectComputeService, error) {
	if err := g.Id.Error; err != nil {
		return nil, err
	}

	res, err := CreateResource(g.MqlRuntime, "gcp.project.computeService", map[string]*llx.RawData{
		"projectId": llx.StringData(g.Id.Data),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlGcpProjectComputeService), nil
}

func (g *mqlGcpProjectComputeService) id() (string, error) {
	if err := g.ProjectId.Error; err != nil {
		return "", err
	}
	return g.ProjectId.Data + "/gcp.project.computeService", nil
}

func (g *mqlGcpProjectComputeServiceRegion) id() (string, error) {
	if err := g.Name.Error; err != nil {
		return "", err
	}
	return "gcp.project.computeService.region/" + g.Name.Data, nil
}

// func (g *mqlGcpProjectComputeServiceRegion) init(args *resources.Args) (*resources.Args, GcpProjectComputeServiceRegion, error) {
// 	if len(*args) > 2 {
// 		return args, nil, nil
// 	}

// 	provider, err := gcpProvider(g.MotorRuntime.Motor.Provider)
// 	if err != nil {
// 		return nil, nil, err
// 	}

// 	projectId := provider.ResourceID()
// 	(*args)["projectId"] = projectId

// 	return args, nil, nil
// }

func (g *mqlGcpProjectComputeService) regions() ([]interface{}, error) {
	if err := g.ProjectId.Error; err != nil {
		return nil, err
	}
	projectId := g.ProjectId.Data

	provider, err := gcpProvider(g.MqlRuntime.Connection)
	if err != nil {
		return nil, err
	}

	client, err := provider.Client(cloudresourcemanager.CloudPlatformReadOnlyScope, iam.CloudPlatformScope, compute.CloudPlatformScope)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()

	computeSvc, err := compute.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}

	req, err := computeSvc.Regions.List(projectId).Do()
	if err != nil {
		return nil, err
	}
	res := make([]interface{}, 0, len(req.Items))
	for _, r := range req.Items {
		mqlRegion, err := newMqlRegion(g.MqlRuntime, r)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlRegion)
	}

	return res, nil
}

func (g *mqlGcpProjectComputeServiceZone) id() (string, error) {
	if err := g.Id.Error; err != nil {
		return "", err
	}
	return "gcp.project.computeService.zone/" + g.Id.Data, nil
}

func (g *mqlGcpProjectComputeService) zones() ([]interface{}, error) {
	if err := g.ProjectId.Error; err != nil {
		return nil, err
	}
	projectId := g.ProjectId.Data

	provider, err := gcpProvider(g.MqlRuntime.Connection)
	if err != nil {
		return nil, err
	}

	client, err := provider.Client(cloudresourcemanager.CloudPlatformReadOnlyScope)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()

	computeSvc, err := compute.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}

	res := []interface{}{}
	req := computeSvc.Zones.List(projectId)
	if err := req.Pages(ctx, func(page *compute.ZoneList) error {
		for _, zone := range page.Items {
			mqlZone, err := CreateResource(g.MqlRuntime, "gcp.project.computeService.zone", map[string]*llx.RawData{
				"id":          llx.StringData(strconv.FormatInt(int64(zone.Id), 10)),
				"name":        llx.StringData(zone.Name),
				"description": llx.StringData(zone.Description),
				"status":      llx.StringData(zone.Status),
				"created":     llx.TimeDataPtr(parseTime(zone.CreationTimestamp)),
			})
			if err != nil {
				return err
			}
			res = append(res, mqlZone)
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return res, nil
}

func (g *mqlGcpProjectComputeServiceMachineType) id() (string, error) {
	if err := g.Id.Error; err != nil {
		return "", err
	}

	if err := g.ProjectId.Error; err != nil {
		return "", err
	}

	return "gcp.project.computeService.machineType/" + g.ProjectId.Data + "/" + g.Id.Data, nil
}

// func (g *mqlGcpProjectComputeServiceMachineType) GetZone() (interface{}, error) {
// 	// NOTE: this should never be called since we add the zone during construction of the resource
// 	return nil, errors.New("not implemented")
// }

func newMqlMachineType(runtime *plugin.Runtime, entry *compute.MachineType, projectId string, zone *mqlGcpProjectComputeServiceZone) (*mqlGcpProjectComputeServiceMachineType, error) {
	res, err := CreateResource(runtime, "gcp.project.computeService.machineType", map[string]*llx.RawData{
		"id":                           llx.StringData(strconv.FormatInt(int64(entry.Id), 10)),
		"projectId":                    llx.StringData(projectId),
		"name":                         llx.StringData(entry.Name),
		"description":                  llx.StringData(entry.Description),
		"guestCpus":                    llx.IntData(entry.GuestCpus),
		"isSharedCpu":                  llx.BoolData(entry.IsSharedCpu),
		"maximumPersistentDisks":       llx.IntData(entry.MaximumPersistentDisks),
		"maximumPersistentDisksSizeGb": llx.IntData(entry.MaximumPersistentDisksSizeGb),
		"memoryMb":                     llx.IntData(entry.MemoryMb),
		"created":                      llx.TimeDataPtr(parseTime(entry.CreationTimestamp)),
		"zone":                         llx.ResourceData(zone, "gcp.project.computeService.zone"),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlGcpProjectComputeServiceMachineType), nil
}

// func (g *mqlGcpProjectComputeService) GetMachineTypes() ([]interface{}, error) {
// 	projectId, err := g.ProjectId()
// 	if err != nil {
// 		return nil, err
// 	}

// 	// get list of zones first since we need this for all entries
// 	zones, err := g.Zones()
// 	if err != nil {
// 		return nil, err
// 	}

// 	provider, err := gcpProvider(g.MotorRuntime.Motor.Provider)
// 	if err != nil {
// 		return nil, err
// 	}

// 	client, err := provider.Client(cloudresourcemanager.CloudPlatformReadOnlyScope, iam.CloudPlatformScope, compute.CloudPlatformScope)
// 	if err != nil {
// 		return nil, err
// 	}

// 	ctx := context.Background()

// 	computeSvc, err := compute.NewService(ctx, option.WithHTTPClient(client))
// 	if err != nil {
// 		return nil, err
// 	}

// 	var wg sync.WaitGroup
// 	res := []interface{}{}
// 	wg.Add(len(zones))
// 	mux := &sync.Mutex{}

// 	for i := range zones {
// 		z := zones[i].(GcpProjectComputeServiceZone)
// 		zoneName, err := z.Name()
// 		if err != nil {
// 			return nil, err
// 		}

// 		go func(svc *compute.Service, projectId string, zone GcpProjectComputeServiceZone, zoneName string) {
// 			req := computeSvc.MachineTypes.List(projectId, zoneName)
// 			if err := req.Pages(ctx, func(page *compute.MachineTypeList) error {
// 				for _, machinetype := range page.Items {
// 					mqlMachineType, err := newMqlMachineType(g.MotorRuntime, machinetype, projectId, zone)
// 					if err != nil {
// 						return err
// 					} else {
// 						mux.Lock()
// 						res = append(res, mqlMachineType)
// 						mux.Unlock()
// 					}
// 				}
// 				return nil
// 			}); err != nil {
// 				log.Error().Err(err).Send()
// 			}
// 			wg.Done()
// 		}(computeSvc, projectId, z, zoneName)
// 	}
// 	wg.Wait()
// 	return res, nil
// }

func (g *mqlGcpProjectComputeServiceInstance) id() (string, error) {
	if err := g.Id.Error; err != nil {
		return "", err
	}

	if err := g.ProjectId.Error; err != nil {
		return "", err
	}

	return "gcp.project.computeService.instance/" + g.ProjectId.Data + "/" + g.Id.Data, nil
}

func (g *mqlGcpProjectComputeServiceInstance) machineType() (*mqlGcpProjectComputeServiceMachineType, error) {
	if err := g.ProjectId.Error; err != nil {
		return nil, err
	}
	projectId := g.ProjectId.Data

	if err := g.Zone.Error; err != nil {
		return nil, err
	}
	zone := g.Zone.Data

	if err := zone.Name.Error; err != nil {
		return nil, err
	}
	zoneName := zone.Name.Data

	values := strings.Split(g.machineTypeUrl, "/")
	machineTypeValue := values[len(values)-1]

	// TODO: we can save calls if we move it to the into method
	provider, err := gcpProvider(g.MqlRuntime.Connection)
	if err != nil {
		return nil, err
	}

	client, err := provider.Client(cloudresourcemanager.CloudPlatformReadOnlyScope, iam.CloudPlatformScope, compute.CloudPlatformScope)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	computeSvc, err := compute.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}

	machineType, err := computeSvc.MachineTypes.Get(projectId, zoneName, machineTypeValue).Do()
	if err != nil {
		return nil, err
	}

	return newMqlMachineType(g.MqlRuntime, machineType, projectId, zone)
}

func newMqlServiceAccount(runtime *plugin.Runtime, sa *compute.ServiceAccount) (interface{}, error) {
	return CreateResource(runtime, "gcp.project.computeService.serviceaccount", map[string]*llx.RawData{
		"email":  llx.StringData(sa.Email),
		"scopes": llx.ArrayData(convert.SliceAnyToInterface(sa.Scopes), types.String),
	})
}

type mqlGcpProjectComputeServiceAttachedDiskInternal struct {
	lock               sync.Mutex
	attachedDsikSource string
}

func newMqlAttachedDisk(id string, projectId string, runtime *plugin.Runtime, attachedDisk *compute.AttachedDisk) (interface{}, error) {
	guestOsFeatures := []string{}
	for i := range attachedDisk.GuestOsFeatures {
		entry := attachedDisk.GuestOsFeatures[i]
		guestOsFeatures = append(guestOsFeatures, entry.Type)
	}

	mqlAttachedDisk, err := CreateResource(runtime, "gcp.project.computeService.attachedDisk", map[string]*llx.RawData{
		"id":              llx.StringData(id),
		"projectId":       llx.StringData(projectId),
		"architecture":    llx.StringData(attachedDisk.Architecture),
		"autoDelete":      llx.BoolData(attachedDisk.AutoDelete),
		"boot":            llx.BoolData(attachedDisk.Boot),
		"deviceName":      llx.StringData(attachedDisk.DeviceName),
		"diskSizeGb":      llx.IntData(attachedDisk.DiskSizeGb),
		"forceAttach":     llx.BoolData(attachedDisk.ForceAttach),
		"guestOsFeatures": llx.ArrayData(convert.SliceAnyToInterface(guestOsFeatures), types.String),
		"index":           llx.IntData(attachedDisk.Index),
		"interface":       llx.StringData(attachedDisk.Interface),
		"licenses":        llx.ArrayData(convert.SliceAnyToInterface(attachedDisk.Licenses), types.String),
		"mode":            llx.StringData(attachedDisk.Mode),
		"type":            llx.StringData(attachedDisk.Type),
	})
	if err != nil {
		return nil, err
	}
	mqlAttachedDisk.(*mqlGcpProjectComputeServiceAttachedDisk).attachedDsikSource = attachedDisk.Source
	return mqlAttachedDisk, nil
}

func (g *mqlGcpProjectComputeServiceAttachedDisk) id() (string, error) {
	if err := g.Id.Error; err != nil {
		return "", err
	}

	if err := g.ProjectId.Error; err != nil {
		return "", err
	}

	return "gcp.project.computeService.attachedDisk/" + g.ProjectId.Data + "/" + g.Id.Data, nil
}

func (g *mqlGcpProjectComputeServiceAttachedDisk) source() (*mqlGcpProjectComputeServiceDisk, error) {
	// TODO: seems like this was never implemented. Need to convert the URL from the cache to an actual disk
	return nil, nil
}

func newMqlInstance(projectId string, zone *mqlGcpProjectComputeServiceZone, runtime *plugin.Runtime, instance *compute.Instance) (*mqlGcpProjectComputeServiceInstance, error) {
	metadata := map[string]string{}
	for m := range instance.Metadata.Items {
		item := instance.Metadata.Items[m]
		metadata[item.Key] = convert.ToString(item.Value)
	}

	mqlServiceAccounts := []interface{}{}
	for i := range instance.ServiceAccounts {
		sa := instance.ServiceAccounts[i]

		mqlServiceAccount, err := newMqlServiceAccount(runtime, sa)
		if err != nil {
			log.Error().Err(err).Send()
		} else {
			mqlServiceAccounts = append(mqlServiceAccounts, mqlServiceAccount)
		}
	}

	var physicalHost string
	if instance.ResourceStatus != nil {
		physicalHost = instance.ResourceStatus.PhysicalHost
	}

	var enableIntegrityMonitoring bool
	var enableSecureBoot bool
	var enableVtpm bool
	if instance.ShieldedInstanceConfig != nil {
		enableIntegrityMonitoring = instance.ShieldedInstanceConfig.EnableIntegrityMonitoring
		enableSecureBoot = instance.ShieldedInstanceConfig.EnableSecureBoot
		enableVtpm = instance.ShieldedInstanceConfig.EnableVtpm
	}

	var enableDisplay bool
	if instance.DisplayDevice != nil {
		enableDisplay = instance.DisplayDevice.EnableDisplay
	}

	guestAccelerators, err := convert.JsonToDictSlice(instance.GuestAccelerators)
	if err != nil {
		return nil, err
	}

	networkInterfaces, err := convert.JsonToDictSlice(instance.NetworkInterfaces)
	if err != nil {
		return nil, err
	}

	reservationAffinity, err := convert.JsonToDict(instance.ReservationAffinity)
	if err != nil {
		return nil, err
	}

	scheduling, err := convert.JsonToDict(instance.Scheduling)
	if err != nil {
		return nil, err
	}

	var totalEgressBandwidthTier string
	if instance.NetworkPerformanceConfig != nil {
		totalEgressBandwidthTier = instance.NetworkPerformanceConfig.TotalEgressBandwidthTier
	}

	instanceId := strconv.FormatUint(instance.Id, 10)
	attachedDisks := []interface{}{}
	for i := range instance.Disks {
		disk := instance.Disks[i]
		attachedDiskID := instanceId + "/" + strconv.FormatInt(disk.Index, 10)
		attachedDisk, err := newMqlAttachedDisk(attachedDiskID, projectId, runtime, disk)
		if err != nil {
			log.Error().Err(err).Send()
			continue
		}
		attachedDisks = append(attachedDisks, attachedDisk)
	}

	var mqlConfCompute map[string]interface{}
	if instance.ConfidentialInstanceConfig != nil {
		type mqlConfidentialInstanceConfig struct {
			Enabled bool `json:"enabled,omitempty"`
		}
		mqlConfCompute, err = convert.JsonToDict(
			mqlConfidentialInstanceConfig{Enabled: instance.ConfidentialInstanceConfig.EnableConfidentialCompute})
		if err != nil {
			return nil, err
		}
	}

	entry, err := CreateResource(runtime, "gcp.project.computeService.instance", map[string]*llx.RawData{
		"id":                         llx.StringData(instanceId),
		"projectId":                  llx.StringData(projectId),
		"name":                       llx.StringData(instance.Name),
		"description":                llx.StringData(instance.Description),
		"confidentialInstanceConfig": llx.DictData(mqlConfCompute),
		"canIpForward":               llx.BoolData(instance.CanIpForward),
		"cpuPlatform":                llx.StringData(instance.CpuPlatform),
		"created":                    llx.TimeDataPtr(parseTime(instance.CreationTimestamp)),
		"deletionProtection":         llx.BoolData(instance.DeletionProtection),
		"enableDisplay":              llx.BoolData(enableDisplay),
		"guestAccelerators":          llx.ArrayData(guestAccelerators, types.Dict),
		"fingerprint":                llx.StringData(instance.Fingerprint),
		"hostname":                   llx.StringData(instance.Hostname),
		"keyRevocationActionType":    llx.StringData(instance.KeyRevocationActionType),
		"labels":                     llx.MapData(convert.MapToInterfaceMap(instance.Labels), types.String),
		"lastStartTimestamp":         llx.TimeDataPtr(parseTime(instance.LastStartTimestamp)),
		"lastStopTimestamp":          llx.TimeDataPtr(parseTime(instance.LastStopTimestamp)),
		"lastSuspendedTimestamp":     llx.TimeDataPtr(parseTime(instance.LastSuspendedTimestamp)),
		"metadata":                   llx.MapData(convert.MapToInterfaceMap(metadata), types.String),
		"minCpuPlatform":             llx.StringData(instance.MinCpuPlatform),
		"networkInterfaces":          llx.ArrayData(networkInterfaces, types.Dict),
		"privateIpv6GoogleAccess":    llx.StringData(instance.PrivateIpv6GoogleAccess),
		"reservationAffinity":        llx.DictData(reservationAffinity),
		"resourcePolicies":           llx.ArrayData(convert.SliceAnyToInterface(instance.ResourcePolicies), types.String),
		"physicalHostResourceStatus": llx.StringData(physicalHost),
		"scheduling":                 llx.DictData(scheduling),
		"enableIntegrityMonitoring":  llx.BoolData(enableIntegrityMonitoring),
		"enableSecureBoot":           llx.BoolData(enableSecureBoot),
		"enableVtpm":                 llx.BoolData(enableVtpm),
		"startRestricted":            llx.BoolData(instance.StartRestricted),
		"status":                     llx.StringData(instance.Status),
		"statusMessage":              llx.StringData(instance.StatusMessage),
		"sourceMachineImage":         llx.StringData(instance.SourceMachineImage),
		"tags":                       llx.ArrayData(convert.SliceAnyToInterface(instance.Tags.Items), types.String),
		"totalEgressBandwidthTier":   llx.StringData(totalEgressBandwidthTier),
		"serviceAccounts":            llx.ArrayData(mqlServiceAccounts, types.Resource("gcp.project.computeService.serviceaccount")),
		"disks":                      llx.ArrayData(attachedDisks, types.Resource("gcp.project.computeService.attachedDisk")),
		"zone":                       llx.ResourceData(zone, "gcp.project.computeService.zone"),
	})
	if err != nil {
		return nil, err
	}
	return entry.(*mqlGcpProjectComputeServiceInstance), nil
}

type mqlGcpProjectComputeServiceInstanceInternal struct {
	lock           sync.Mutex
	machineTypeUrl string
}

func (g *mqlGcpProjectComputeService) instances() ([]interface{}, error) {
	if err := g.ProjectId.Error; err != nil {
		return nil, err
	}
	projectId := g.ProjectId.Data

	// get list of zones first since we need this for all entries
	zonesData := g.GetZones()
	if err := zonesData.Error; err != nil {
		return nil, err
	}

	provider, err := gcpProvider(g.MqlRuntime.Connection)
	if err != nil {
		return nil, err
	}

	client, err := provider.Client(cloudresourcemanager.CloudPlatformReadOnlyScope, iam.CloudPlatformScope, compute.CloudPlatformScope)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()

	computeSvc, err := compute.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}

	var wg sync.WaitGroup
	res := []interface{}{}
	wg.Add(len(zonesData.Data))
	mux := &sync.Mutex{}

	for i := range zonesData.Data {
		z := zonesData.Data[i].(*mqlGcpProjectComputeServiceZone)
		if err := z.Name.Error; err != nil {
			return nil, err
		}
		zoneName := z.Name.Data
		go func(svc *compute.Service, project string, zone *mqlGcpProjectComputeServiceZone, zoneName string) {
			req := computeSvc.Instances.List(projectId, zoneName)
			if err := req.Pages(ctx, func(page *compute.InstanceList) error {
				for _, instance := range page.Items {

					mqlInstance, err := newMqlInstance(projectId, zone, g.MqlRuntime, instance)
					if err != nil {
						return err
					} else {
						mqlInstance.machineTypeUrl = instance.MachineType
						mux.Lock()
						res = append(res, mqlInstance)
						mux.Unlock()
					}
				}
				return nil
			}); err != nil {
				log.Error().Err(err).Send()
			}
			wg.Done()
		}(computeSvc, projectId, z, zoneName)
	}

	wg.Wait()
	return res, nil
}

func (g *mqlGcpProjectComputeServiceServiceaccount) id() (string, error) {
	if err := g.Email.Error; err != nil {
		return "", nil
	}
	return "gcp.project.computeService.serviceaccount/" + g.Email.Data, nil
}

func (g *mqlGcpProjectComputeServiceDisk) id() (string, error) {
	if err := g.Id.Error; err != nil {
		return "", nil
	}

	return "gcloud.compute.disk/" + g.Id.Data, nil
}

// func (g *mqlGcpProjectComputeServiceDisk) GetZone() (interface{}, error) {
// 	// TODO: implement
// 	return nil, errors.New("not implemented")
// }

// func (g *mqlGcpProjectComputeService) GetDisks() ([]interface{}, error) {
// 	projectId, err := g.ProjectId()
// 	if err != nil {
// 		return nil, err
// 	}

// 	// get list of zones first since we need this for all entries
// 	zones, err := g.Zones()
// 	if err != nil {
// 		return nil, err
// 	}

// 	provider, err := gcpProvider(g.MotorRuntime.Motor.Provider)
// 	if err != nil {
// 		return nil, err
// 	}

// 	client, err := provider.Client(cloudresourcemanager.CloudPlatformReadOnlyScope, iam.CloudPlatformScope, compute.CloudPlatformScope)
// 	if err != nil {
// 		return nil, err
// 	}

// 	ctx := context.Background()
// 	computeSvc, err := compute.NewService(ctx, option.WithHTTPClient(client))
// 	if err != nil {
// 		return nil, err
// 	}

// 	var wg sync.WaitGroup
// 	res := []interface{}{}
// 	wg.Add(len(zones))
// 	mux := &sync.Mutex{}

// 	var result error
// 	for i := range zones {
// 		z := zones[i].(GcpProjectComputeServiceZone)
// 		zoneName, err := z.Name()
// 		if err != nil {
// 			return nil, err
// 		}

// 		go func(svc *compute.Service, project string, zone GcpProjectComputeServiceZone, zoneName string) {
// 			req := computeSvc.Disks.List(projectId, zoneName)
// 			if err := req.Pages(ctx, func(page *compute.DiskList) error {
// 				for _, disk := range page.Items {
// 					guestOsFeatures := []string{}
// 					for i := range disk.GuestOsFeatures {
// 						entry := disk.GuestOsFeatures[i]
// 						guestOsFeatures = append(guestOsFeatures, entry.Type)
// 					}

// 					var mqlDiskEnc map[string]interface{}
// 					if disk.DiskEncryptionKey != nil {
// 						mqlDiskEnc = map[string]interface{}{
// 							"kmsKeyName":           disk.DiskEncryptionKey.KmsKeyName,
// 							"kmsKeyServiceAccount": disk.DiskEncryptionKey.KmsKeyServiceAccount,
// 							"rawKey":               disk.DiskEncryptionKey.RawKey,
// 							"rsaEncryptedKey":      disk.DiskEncryptionKey.RsaEncryptedKey,
// 							"sha256":               disk.DiskEncryptionKey.Sha256,
// 						}
// 					}

// 					mqlDisk, err := g.MotorRuntime.CreateResource("gcp.project.computeService.disk",
// 						"id", strconv.FormatUint(disk.Id, 10),
// 						"name", disk.Name,
// 						"architecture", disk.Architecture,
// 						"description", disk.Description,
// 						"guestOsFeatures", core.StrSliceToInterface(guestOsFeatures),
// 						"labels", core.StrMapToInterface(disk.Labels),
// 						"lastAttachTimestamp", parseTime(disk.LastAttachTimestamp),
// 						"lastDetachTimestamp", parseTime(disk.LastDetachTimestamp),
// 						"locationHint", disk.LocationHint,
// 						"licenses", core.StrSliceToInterface(disk.Licenses),
// 						"physicalBlockSizeBytes", disk.PhysicalBlockSizeBytes,
// 						"provisionedIops", disk.ProvisionedIops,
// 						// TODO: link to resources
// 						//"region", disk.Region,
// 						//"replicaZones", core.StrSliceToInterface(disk.ReplicaZones),
// 						//"resourcePolicies", core.StrSliceToInterface(disk.ResourcePolicies),
// 						"sizeGb", disk.SizeGb,
// 						// TODO: link to resources
// 						//"sourceDiskId", disk.SourceDiskId,
// 						//"sourceImageId", disk.SourceImageId,
// 						//"sourceSnapshotId", disk.SourceSnapshotId,
// 						"status", disk.Status,
// 						"zone", zone,
// 						"created", parseTime(disk.CreationTimestamp),
// 						"diskEncryptionKey", mqlDiskEnc,
// 					)
// 					if err != nil {
// 						return err
// 					} else {
// 						mux.Lock()
// 						res = append(res, mqlDisk)
// 						mux.Unlock()
// 					}
// 				}
// 				return nil
// 			}); err != nil {
// 				log.Error().Err(err).Send()
// 			}
// 			wg.Done()
// 		}(computeSvc, projectId, z, zoneName)
// 	}

// 	wg.Wait()
// 	return res, result
// }

// func (g *mqlGcpProjectComputeServiceFirewall) id() (string, error) {
// 	id, err := g.Id()
// 	if err != nil {
// 		return "", nil
// 	}

// 	return "gcloud.compute.firewall/" + id, nil
// }

// func (g *mqlGcpProjectComputeServiceFirewall) GetNetwork() (interface{}, error) {
// 	// TODO: implement
// 	return nil, errors.New("not implemented")
// }

// func (g *mqlGcpProjectComputeServiceFirewall) init(args *resources.Args) (*resources.Args, GcpProjectComputeServiceFirewall, error) {
// 	if len(*args) > 2 {
// 		return args, nil, nil
// 	}

// 	// If no args are set, try reading them from the platform ID
// 	if len(*args) == 0 {
// 		if ids := getAssetIdentifier(g.MotorRuntime); ids != nil {
// 			(*args)["name"] = ids.name
// 			(*args)["projectId"] = ids.project
// 		}
// 	}

// 	obj, err := g.MotorRuntime.CreateResource("gcp.project.computeService", "projectId", (*args)["projectId"])
// 	if err != nil {
// 		return nil, nil, err
// 	}
// 	computeSvc := obj.(GcpProjectComputeService)
// 	firewalls, err := computeSvc.Firewalls()
// 	if err != nil {
// 		return nil, nil, err
// 	}

// 	for _, f := range firewalls {
// 		firewall := f.(GcpProjectComputeServiceFirewall)
// 		name, err := firewall.Name()
// 		if err != nil {
// 			return nil, nil, err
// 		}
// 		projectId, err := firewall.ProjectId()
// 		if err != nil {
// 			return nil, nil, err
// 		}

// 		if name == (*args)["name"] && projectId == (*args)["projectId"] {
// 			return args, firewall, nil
// 		}
// 	}
// 	return nil, nil, &resources.ResourceNotFound{}
// }

// func (g *mqlGcpProjectComputeService) GetFirewalls() ([]interface{}, error) {
// 	projectId, err := g.ProjectId()
// 	if err != nil {
// 		return nil, err
// 	}

// 	provider, err := gcpProvider(g.MotorRuntime.Motor.Provider)
// 	if err != nil {
// 		return nil, err
// 	}

// 	client, err := provider.Client(cloudresourcemanager.CloudPlatformReadOnlyScope, iam.CloudPlatformScope, compute.CloudPlatformScope)
// 	if err != nil {
// 		return nil, err
// 	}

// 	ctx := context.Background()

// 	computeSvc, err := compute.NewService(ctx, option.WithHTTPClient(client))
// 	if err != nil {
// 		return nil, err
// 	}

// 	type mqlFirewall struct {
// 		IpProtocol string   `json:"ipProtocol"`
// 		Ports      []string `json:"ports"`
// 	}

// 	res := []interface{}{}
// 	req := computeSvc.Firewalls.List(projectId)
// 	if err := req.Pages(ctx, func(page *compute.FirewallList) error {
// 		for _, firewall := range page.Items {
// 			allowed := make([]mqlFirewall, 0, len(firewall.Allowed))
// 			for _, a := range firewall.Allowed {
// 				allowed = append(allowed, mqlFirewall{IpProtocol: a.IPProtocol, Ports: a.Ports})
// 			}
// 			allowedDict, err := core.JsonToDictSlice(allowed)
// 			if err != nil {
// 				return err
// 			}

// 			denied := make([]mqlFirewall, 0, len(firewall.Denied))
// 			for _, d := range firewall.Denied {
// 				denied = append(denied, mqlFirewall{IpProtocol: d.IPProtocol, Ports: d.Ports})
// 			}
// 			deniedDict, err := core.JsonToDictSlice(denied)
// 			if err != nil {
// 				return err
// 			}

// 			mqlFirewall, err := g.MotorRuntime.CreateResource("gcp.project.computeService.firewall",
// 				"id", strconv.FormatUint(firewall.Id, 10),
// 				"projectId", projectId,
// 				"name", firewall.Name,
// 				"description", firewall.Description,
// 				"priority", firewall.Priority,
// 				"disabled", firewall.Disabled,
// 				"direction", firewall.Direction,
// 				"sourceRanges", core.StrSliceToInterface(firewall.SourceRanges),
// 				"sourceServiceAccounts", core.StrSliceToInterface(firewall.SourceServiceAccounts),
// 				"sourceTags", core.StrSliceToInterface(firewall.SourceTags),
// 				"destinationRanges", core.StrSliceToInterface(firewall.DestinationRanges),
// 				"targetServiceAccounts", core.StrSliceToInterface(firewall.TargetServiceAccounts),
// 				"created", parseTime(firewall.CreationTimestamp),
// 				"allowed", allowedDict,
// 				"denied", deniedDict,
// 			)
// 			if err != nil {
// 				return err
// 			} else {
// 				res = append(res, mqlFirewall)
// 			}
// 		}
// 		return nil
// 	}); err != nil {
// 		return nil, err
// 	}

// 	return res, nil
// }

// func (g *mqlGcpProjectComputeServiceSnapshot) id() (string, error) {
// 	id, err := g.Id()
// 	if err != nil {
// 		return "", nil
// 	}

// 	return "gcloud.compute.snapshot/" + id, nil
// }

// func (g *mqlGcpProjectComputeServiceSnapshot) GetSourceDisk() (interface{}, error) {
// 	// TODO: implement
// 	return nil, errors.New("not implemented")
// }

// func (g *mqlGcpProjectComputeService) GetSnapshots() ([]interface{}, error) {
// 	projectId, err := g.ProjectId()
// 	if err != nil {
// 		return nil, err
// 	}

// 	provider, err := gcpProvider(g.MotorRuntime.Motor.Provider)
// 	if err != nil {
// 		return nil, err
// 	}

// 	client, err := provider.Client(cloudresourcemanager.CloudPlatformReadOnlyScope, iam.CloudPlatformScope, compute.CloudPlatformScope)
// 	if err != nil {
// 		return nil, err
// 	}

// 	ctx := context.Background()

// 	computeSvc, err := compute.NewService(ctx, option.WithHTTPClient(client))
// 	if err != nil {
// 		return nil, err
// 	}

// 	res := []interface{}{}
// 	req := computeSvc.Snapshots.List(projectId)
// 	if err := req.Pages(ctx, func(page *compute.SnapshotList) error {
// 		for _, snapshot := range page.Items {
// 			mqlSnapshpt, err := g.MotorRuntime.CreateResource("gcp.project.computeService.snapshot",
// 				"id", strconv.FormatUint(snapshot.Id, 10),
// 				"name", snapshot.Name,
// 				"description", snapshot.Description,
// 				"architecture", snapshot.Architecture,
// 				"autoCreated", snapshot.AutoCreated,
// 				"chainName", snapshot.ChainName,
// 				"creationSizeBytes", snapshot.CreationSizeBytes,
// 				"diskSizeGb", snapshot.DiskSizeGb,
// 				"downloadBytes", snapshot.DownloadBytes,
// 				"storageBytes", snapshot.StorageBytes,
// 				"storageBytesStatus", snapshot.StorageBytesStatus,
// 				"snapshotType", snapshot.SnapshotType,
// 				"licenses", core.StrSliceToInterface(snapshot.Licenses),
// 				"labels", core.StrMapToInterface(snapshot.Labels),
// 				"status", snapshot.Status,
// 				"created", parseTime(snapshot.CreationTimestamp),
// 			)
// 			if err != nil {
// 				return err
// 			}

// 			res = append(res, mqlSnapshpt)
// 		}
// 		return nil
// 	}); err != nil {
// 		return nil, err
// 	}

// 	return res, nil
// }

// func (g *mqlGcpProjectComputeServiceImage) id() (string, error) {
// 	id, err := g.Id()
// 	if err != nil {
// 		return "", nil
// 	}

// 	return "gcloud.compute.image/" + id, nil
// }

// func (g *mqlGcpProjectComputeServiceImage) init(args *resources.Args) (*resources.Args, GcpProjectComputeServiceImage, error) {
// 	if len(*args) > 2 {
// 		return args, nil, nil
// 	}

// 	// If no args are set, try reading them from the platform ID
// 	if len(*args) == 0 {
// 		if ids := getAssetIdentifier(g.MotorRuntime); ids != nil {
// 			(*args)["name"] = ids.name
// 			(*args)["projectId"] = ids.project
// 		}
// 	}

// 	obj, err := g.MotorRuntime.CreateResource("gcp.project.computeService", "projectId", (*args)["projectId"])
// 	if err != nil {
// 		return nil, nil, err
// 	}
// 	computeSvc := obj.(GcpProjectComputeService)
// 	images, err := computeSvc.Images()
// 	if err != nil {
// 		return nil, nil, err
// 	}

// 	for _, i := range images {
// 		image := i.(GcpProjectComputeServiceImage)
// 		name, err := image.Name()
// 		if err != nil {
// 			return nil, nil, err
// 		}
// 		projectId, err := image.ProjectId()
// 		if err != nil {
// 			return nil, nil, err
// 		}

// 		if name == (*args)["name"] && projectId == (*args)["projectId"] {
// 			return args, image, nil
// 		}
// 	}
// 	return nil, nil, &resources.ResourceNotFound{}
// }

// func (g *mqlGcpProjectComputeServiceImage) GetSourceDisk() (interface{}, error) {
// 	// TODO: implement
// 	return nil, errors.New("not implemented")
// }

// func (g *mqlGcpProjectComputeService) GetImages() ([]interface{}, error) {
// 	projectId, err := g.ProjectId()
// 	if err != nil {
// 		return nil, err
// 	}

// 	provider, err := gcpProvider(g.MotorRuntime.Motor.Provider)
// 	if err != nil {
// 		return nil, err
// 	}

// 	client, err := provider.Client(cloudresourcemanager.CloudPlatformReadOnlyScope, iam.CloudPlatformScope, compute.CloudPlatformScope)
// 	if err != nil {
// 		return nil, err
// 	}

// 	ctx := context.Background()

// 	computeSvc, err := compute.NewService(ctx, option.WithHTTPClient(client))
// 	if err != nil {
// 		return nil, err
// 	}

// 	res := []interface{}{}
// 	req := computeSvc.Images.List(projectId)
// 	if err := req.Pages(ctx, func(page *compute.ImageList) error {
// 		for _, image := range page.Items {
// 			mqlImage, err := g.MotorRuntime.CreateResource("gcp.project.computeService.image",
// 				"id", strconv.FormatUint(image.Id, 10),
// 				"projectId", projectId,
// 				"name", image.Name,
// 				"description", image.Description,
// 				"architecture", image.Architecture,
// 				"archiveSizeBytes", image.ArchiveSizeBytes,
// 				"diskSizeGb", image.DiskSizeGb,
// 				"family", image.Family,
// 				"licenses", core.StrSliceToInterface(image.Licenses),
// 				"labels", core.StrMapToInterface(image.Labels),
// 				"status", image.Status,
// 				"created", parseTime(image.CreationTimestamp),
// 			)
// 			if err != nil {
// 				return err
// 			}
// 			res = append(res, mqlImage)
// 		}
// 		return nil
// 	}); err != nil {
// 		return nil, err
// 	}

// 	return res, nil
// }

func (g *mqlGcpProjectComputeServiceNetwork) id() (string, error) {
	if err := g.Id.Error; err != nil {
		return "", nil
	}

	return "gcloud.compute.network/" + g.Id.Data, nil
}

func (g *mqlGcpProjectComputeServiceNetwork) subnetworks() ([]interface{}, error) {
	if err := g.SubnetworkUrls.Error; err != nil {
		return nil, err
	}
	subnetUrls := g.SubnetworkUrls.Data
	type resourceId struct {
		Project string
		Region  string
		Name    string
	}
	subnets := make([]interface{}, 0, len(subnetUrls))
	for _, subnetUrl := range subnetUrls {
		// Format is https://www.googleapis.com/compute/v1/projects/project1regions/us-central1/subnetworks/subnet-1
		params := strings.TrimPrefix(subnetUrl.(string), "https://www.googleapis.com/compute/v1/")
		parts := strings.Split(params, "/")
		resId := resourceId{Project: parts[1], Region: parts[3], Name: parts[5]}

		subnet, err := CreateResource(g.MqlRuntime, "gcp.project.computeService.subnetwork", map[string]*llx.RawData{
			"name":      llx.StringData(resId.Name),
			"projectId": llx.StringData(resId.Project),
			"region":    llx.StringData(resId.Region),
		})
		if err != nil {
			return nil, err
		}
		subnets = append(subnets, subnet)
	}
	return subnets, nil
}

// func (g *mqlGcpProjectComputeServiceNetwork) init(args *resources.Args) (*resources.Args, GcpProjectComputeServiceNetwork, error) {
// 	if len(*args) > 2 {
// 		return args, nil, nil
// 	}

// 	// If no args are set, try reading them from the platform ID
// 	if len(*args) == 0 {
// 		if ids := getAssetIdentifier(g.MotorRuntime); ids != nil {
// 			(*args)["name"] = ids.name
// 			(*args)["projectId"] = ids.project
// 		}
// 	}

// 	obj, err := g.MotorRuntime.CreateResource("gcp.project.computeService", "projectId", (*args)["projectId"])
// 	if err != nil {
// 		return nil, nil, err
// 	}
// 	computeSvc := obj.(GcpProjectComputeService)
// 	networks, err := computeSvc.Networks()
// 	if err != nil {
// 		return nil, nil, err
// 	}

// 	for _, n := range networks {
// 		network := n.(GcpProjectComputeServiceNetwork)
// 		name, err := network.Name()
// 		if err != nil {
// 			return nil, nil, err
// 		}
// 		projectId, err := network.ProjectId()
// 		if err != nil {
// 			return nil, nil, err
// 		}

// 		if name == (*args)["name"] && projectId == (*args)["projectId"] {
// 			return args, network, nil
// 		}
// 	}
// 	return nil, nil, &resources.ResourceNotFound{}
// }

// func (g *mqlGcpProjectComputeService) GetNetworks() ([]interface{}, error) {
// 	projectId, err := g.ProjectId()
// 	if err != nil {
// 		return nil, err
// 	}

// 	provider, err := gcpProvider(g.MotorRuntime.Motor.Provider)
// 	if err != nil {
// 		return nil, err
// 	}

// 	client, err := provider.Client(cloudresourcemanager.CloudPlatformReadOnlyScope, iam.CloudPlatformScope, compute.CloudPlatformScope)
// 	if err != nil {
// 		return nil, err
// 	}

// 	ctx := context.Background()

// 	computeSvc, err := compute.NewService(ctx, option.WithHTTPClient(client))
// 	if err != nil {
// 		return nil, err
// 	}

// 	res := []interface{}{}
// 	req := computeSvc.Networks.List(projectId)
// 	if err := req.Pages(ctx, func(page *compute.NetworkList) error {
// 		for _, network := range page.Items {

// 			peerings, err := core.JsonToDictSlice(network.Peerings)
// 			if err != nil {
// 				return err
// 			}

// 			var routingMode string
// 			if network.RoutingConfig != nil {
// 				routingMode = network.RoutingConfig.RoutingMode
// 			}

// 			mqlNetwork, err := g.MotorRuntime.CreateResource("gcp.project.computeService.network",
// 				"id", strconv.FormatUint(network.Id, 10),
// 				"projectId", projectId,
// 				"name", network.Name,
// 				"description", network.Description,
// 				"autoCreateSubnetworks", network.AutoCreateSubnetworks,
// 				"enableUlaInternalIpv6", network.EnableUlaInternalIpv6,
// 				"gatewayIPv4", network.GatewayIPv4,
// 				"mtu", network.Mtu,
// 				"networkFirewallPolicyEnforcementOrder", network.NetworkFirewallPolicyEnforcementOrder,
// 				"created", parseTime(network.CreationTimestamp),
// 				"peerings", peerings,
// 				"routingMode", routingMode,
// 				"mode", networkMode(network),
// 				"subnetworkUrls", core.StrSliceToInterface(network.Subnetworks),
// 			)
// 			if err != nil {
// 				return err
// 			}
// 			res = append(res, mqlNetwork)
// 		}
// 		return nil
// 	}); err != nil {
// 		return nil, err
// 	}

// 	return res, nil
// }

func (g *mqlGcpProjectComputeServiceSubnetwork) id() (string, error) {
	if err := g.Id.Error; err != nil {
		return "", nil
	}

	return "gcloud.compute.subnetwork/" + g.Id.Data, nil
}

// func (g *mqlGcpProjectComputeServiceSubnetwork) init(args *resources.Args) (*resources.Args, GcpProjectComputeServiceSubnetwork, error) {
// 	if len(*args) > 3 {
// 		return args, nil, nil
// 	}

// 	// If no args are set, try reading them from the platform ID
// 	if len(*args) == 0 {
// 		if ids := getAssetIdentifier(g.MotorRuntime); ids != nil {
// 			(*args)["name"] = ids.name
// 			(*args)["region"] = ids.region
// 			(*args)["projectId"] = ids.project
// 		}
// 	}

// 	obj, err := g.MotorRuntime.CreateResource("gcp.project.computeService", "projectId", (*args)["projectId"])
// 	if err != nil {
// 		return nil, nil, err
// 	}
// 	computeSvc := obj.(GcpProjectComputeService)
// 	subnetworks, err := computeSvc.Subnetworks()
// 	if err != nil {
// 		return nil, nil, err
// 	}

// 	for _, n := range subnetworks {
// 		subnetwork := n.(GcpProjectComputeServiceSubnetwork)
// 		name, err := subnetwork.Name()
// 		if err != nil {
// 			return nil, nil, err
// 		}
// 		regionUrl, err := subnetwork.RegionUrl()
// 		if err != nil {
// 			return nil, nil, err
// 		}
// 		region := RegionNameFromRegionUrl(regionUrl)
// 		projectId, err := subnetwork.ProjectId()
// 		if err != nil {
// 			return nil, nil, err
// 		}

// 		if name == (*args)["name"] && projectId == (*args)["projectId"] && region == (*args)["region"] {
// 			return args, subnetwork, nil
// 		}
// 	}
// 	return nil, nil, &resources.ResourceNotFound{}
// }

func (g *mqlGcpProjectComputeServiceSubnetworkLogConfig) id() (string, error) {
	if err := g.Id.Error; err != nil {
		return "", nil
	}

	return "gcloud.compute.subnetwork.logConfig/" + g.Id.Data, nil
}

func (g *mqlGcpProjectComputeServiceSubnetwork) region() (*mqlGcpProjectComputeServiceRegion, error) {
	if err := g.ProjectId.Error; err != nil {
		return nil, err
	}
	projectId := g.ProjectId.Data

	if err := g.RegionUrl.Error; err != nil {
		return nil, err
	}
	regionUrl := g.RegionUrl.Data
	regionName := RegionNameFromRegionUrl(regionUrl)

	// Find regionName for projectId
	obj, err := CreateResource(g.MqlRuntime, "gcp.project.computeService", map[string]*llx.RawData{
		"projectId": llx.StringData(projectId),
	})
	if err != nil {
		return nil, err
	}
	gcpCompute := obj.(*mqlGcpProjectComputeService)
	regionsData := gcpCompute.GetRegions()
	if err := regionsData.Error; err != nil {
		return nil, err
	}

	for _, r := range regionsData.Data {
		region := r.(*mqlGcpProjectComputeServiceRegion)
		if err := region.Name.Error; err != nil {
			return nil, err
		}
		if region.Name.Data == regionName {
			return region, nil
		}
	}

	return nil, errors.New(fmt.Sprintf("region %s not found", regionName))
}

// func (g *mqlGcpProjectComputeServiceSubnetwork) GetNetwork() ([]interface{}, error) {
// 	// TODO: implement
// 	return nil, errors.New("not implemented")
// }

func newMqlRegion(runtime *plugin.Runtime, r *compute.Region) (interface{}, error) {
	deprecated, err := convert.JsonToDict(r.Deprecated)
	if err != nil {
		return nil, err
	}

	quotas := map[string]interface{}{}
	for i := range r.Quotas {
		q := r.Quotas[i]
		quotas[q.Metric] = q.Limit
	}

	return CreateResource(runtime, "gcp.project.computeService.region", map[string]*llx.RawData{
		"id":          llx.StringData(strconv.FormatInt(int64(r.Id), 10)),
		"name":        llx.StringData(r.Name),
		"description": llx.StringData(r.Description),
		"status":      llx.StringData(r.Status),
		"created":     llx.TimeDataPtr(parseTime(r.CreationTimestamp)),
		"quotas":      llx.DictData(quotas),
		"deprecated":  llx.DictData(deprecated),
	})
}

// func newMqlSubnetwork(projectId string, runtime *resources.Runtime, subnetwork *computepb.Subnetwork, region GcpProjectComputeServiceRegion) (interface{}, error) {
// 	subnetId := strconv.FormatUint(subnetwork.GetId(), 10)
// 	var mqlLogConfig resources.ResourceType
// 	var err error
// 	if subnetwork.LogConfig != nil {
// 		mqlLogConfig, err = runtime.CreateResource("gcp.project.computeService.subnetwork.logConfig",
// 			"id", fmt.Sprintf("%s/logConfig", subnetId),
// 			"aggregationInterval", subnetwork.LogConfig.GetAggregationInterval(),
// 			"enable", subnetwork.LogConfig.GetEnable(),
// 			"filterExpression", subnetwork.LogConfig.GetFilterExpr(),
// 			"flowSampling", float64(subnetwork.LogConfig.GetFlowSampling()),
// 			"metadata", subnetwork.LogConfig.GetMetadata(),
// 			"metadataFields", core.StrSliceToInterface(subnetwork.LogConfig.MetadataFields),
// 		)
// 		if err != nil {
// 			return nil, err
// 		}
// 	}

// 	args := []interface{}{
// 		"id", subnetId,
// 		"projectId", projectId,
// 		"name", subnetwork.GetName(),
// 		"description", subnetwork.GetDescription(),
// 		"enableFlowLogs", subnetwork.GetEnableFlowLogs(),
// 		"externalIpv6Prefix", subnetwork.GetExternalIpv6Prefix(),
// 		"fingerprint", subnetwork.GetFingerprint(),
// 		"gatewayAddress", subnetwork.GetGatewayAddress(),
// 		"internalIpv6Prefix", subnetwork.GetInternalIpv6Prefix(),
// 		"ipCidrRange", subnetwork.GetIpCidrRange(),
// 		"ipv6AccessType", subnetwork.GetIpv6AccessType(),
// 		"ipv6CidrRange", subnetwork.GetIpv6CidrRange(),
// 		"logConfig", mqlLogConfig,
// 		"privateIpGoogleAccess", subnetwork.GetPrivateIpGoogleAccess(),
// 		"privateIpv6GoogleAccess", subnetwork.GetPrivateIpv6GoogleAccess(),
// 		"purpose", subnetwork.GetPurpose(),
// 		"regionUrl", subnetwork.GetRegion(),
// 		"role", subnetwork.GetRole(),
// 		"stackType", subnetwork.GetStackType(),
// 		"state", subnetwork.GetState(),
// 		"created", parseTime(subnetwork.GetCreationTimestamp()),
// 	}
// 	if region != nil {
// 		args = append(args, "region", region)
// 	}
// 	return runtime.CreateResource("gcp.project.computeService.subnetwork", args...)
// }

// func (g *mqlGcpProjectComputeService) GetSubnetworks() ([]interface{}, error) {
// 	projectId, err := g.ProjectId()
// 	if err != nil {
// 		return nil, err
// 	}

// 	provider, err := gcpProvider(g.MotorRuntime.Motor.Provider)
// 	if err != nil {
// 		return nil, err
// 	}

// 	client, err := provider.Client(cloudresourcemanager.CloudPlatformReadOnlyScope, iam.CloudPlatformScope, compute.CloudPlatformScope)
// 	if err != nil {
// 		return nil, err
// 	}

// 	ctx := context.Background()

// 	subnetSvc, err := computev1.NewSubnetworksRESTClient(ctx, option.WithHTTPClient(client))
// 	if err != nil {
// 		return nil, err
// 	}

// 	res := []interface{}{}
// 	it := subnetSvc.AggregatedList(ctx, &computepb.AggregatedListSubnetworksRequest{Project: projectId})
// 	for {
// 		resp, err := it.Next()
// 		if err == iterator.Done {
// 			break
// 		}
// 		if err != nil {
// 			return nil, err
// 		}
// 		subnets := resp.Value.GetSubnetworks()
// 		for _, subnet := range subnets {
// 			mqlSubnetwork, err := newMqlSubnetwork(projectId, g.MotorRuntime, subnet, nil)
// 			if err != nil {
// 				return nil, err
// 			}
// 			res = append(res, mqlSubnetwork)
// 		}
// 	}
// 	return res, nil
// }

// func (g *mqlGcpProjectComputeServiceRouter) id() (string, error) {
// 	id, err := g.Id()
// 	if err != nil {
// 		return "", nil
// 	}

// 	return "gcloud.compute.router/" + id, nil
// }

// func (g *mqlGcpProjectComputeServiceRouter) GetNetwork() ([]interface{}, error) {
// 	// TODO: implement
// 	return nil, errors.New("not implemented")
// }

// func (g *mqlGcpProjectComputeServiceRouter) GetRegion() ([]interface{}, error) {
// 	// TODO: implement
// 	return nil, errors.New("not implemented")
// }

// func newMqlRouter(projectId string, region GcpProjectComputeServiceRegion, runtime *resources.Runtime, router *compute.Router) (interface{}, error) {
// 	bgp, err := core.JsonToDict(router.Bgp)
// 	if err != nil {
// 		return nil, err
// 	}

// 	bgpPeers, err := core.JsonToDictSlice(router.BgpPeers)
// 	if err != nil {
// 		return nil, err
// 	}

// 	nats, err := core.JsonToDictSlice(router.Nats)
// 	if err != nil {
// 		return nil, err
// 	}

// 	return runtime.CreateResource("gcp.project.computeService.router",
// 		"id", strconv.FormatUint(router.Id, 10),
// 		"name", router.Name,
// 		"description", router.Description,
// 		"bgp", bgp,
// 		"bgpPeers", bgpPeers,
// 		"encryptedInterconnectRouter", router.EncryptedInterconnectRouter,
// 		"nats", nats,
// 		"created", parseTime(router.CreationTimestamp),
// 	)
// }

// func (g *mqlGcpProjectComputeService) GetRouters() ([]interface{}, error) {
// 	projectId, err := g.ProjectId()
// 	if err != nil {
// 		return nil, err
// 	}

// 	regions, err := g.Regions()
// 	if err != nil {
// 		return nil, err
// 	}

// 	provider, err := gcpProvider(g.MotorRuntime.Motor.Provider)
// 	if err != nil {
// 		return nil, err
// 	}

// 	client, err := provider.Client(cloudresourcemanager.CloudPlatformReadOnlyScope, iam.CloudPlatformScope, compute.CloudPlatformScope)
// 	if err != nil {
// 		return nil, err
// 	}

// 	ctx := context.Background()

// 	computeSvc, err := compute.NewService(ctx, option.WithHTTPClient(client))
// 	if err != nil {
// 		return nil, err
// 	}

// 	var wg sync.WaitGroup
// 	res := []interface{}{}
// 	wg.Add(len(regions))
// 	mux := &sync.Mutex{}

// 	for i := range regions {
// 		r := regions[i].(GcpProjectComputeServiceRegion)
// 		regionName, err := r.Name()
// 		if err != nil {
// 			return nil, err
// 		}
// 		go func(svc *compute.Service, project string, region GcpProjectComputeServiceRegion, regionName string) {
// 			req := computeSvc.Routers.List(projectId, regionName)
// 			if err := req.Pages(ctx, func(page *compute.RouterList) error {
// 				for _, router := range page.Items {

// 					mqlRouter, err := newMqlRouter(projectId, region, g.MotorRuntime, router)
// 					if err != nil {
// 						return err
// 					} else {
// 						mux.Lock()
// 						res = append(res, mqlRouter)
// 						mux.Unlock()
// 					}
// 				}
// 				return nil
// 			}); err != nil {
// 				log.Error().Err(err).Send()
// 			}
// 			wg.Done()
// 		}(computeSvc, projectId, r, regionName)
// 	}

// 	wg.Wait()
// 	return res, nil
// }

// func (g *mqlGcpProjectComputeService) GetBackendServices() ([]interface{}, error) {
// 	projectId, err := g.ProjectId()
// 	if err != nil {
// 		return nil, err
// 	}

// 	provider, err := gcpProvider(g.MotorRuntime.Motor.Provider)
// 	if err != nil {
// 		return nil, err
// 	}

// 	client, err := provider.Client(cloudresourcemanager.CloudPlatformReadOnlyScope, iam.CloudPlatformScope, compute.CloudPlatformScope)
// 	if err != nil {
// 		return nil, err
// 	}

// 	ctx := context.Background()
// 	computeSvc, err := compute.NewService(ctx, option.WithHTTPClient(client))
// 	if err != nil {
// 		return nil, err
// 	}

// 	list, err := computeSvc.BackendServices.AggregatedList(projectId).Do()
// 	if err != nil {
// 		return nil, err
// 	}

// 	res := make([]interface{}, 0, len(list.Items))
// 	for _, sb := range list.Items {
// 		for _, b := range sb.BackendServices {
// 			backendServiceId := strconv.FormatUint(b.Id, 10)
// 			mqlBackends := make([]interface{}, 0, len(b.Backends))
// 			for i, backend := range b.Backends {
// 				mqlBackend, err := g.MotorRuntime.CreateResource("gcp.project.computeService.backendService.backend",
// 					"id", fmt.Sprintf("gcp.project.computeService.backendService.backend/%s/%d", backendServiceId, i),
// 					"balancingMode", backend.BalancingMode,
// 					"capacityScaler", backend.CapacityScaler,
// 					"description", backend.Description,
// 					"failover", backend.Failover,
// 					"groupUrl", backend.Group,
// 					"maxConnections", backend.MaxConnections,
// 					"maxConnectionsPerEndpoint", backend.MaxConnectionsPerEndpoint,
// 					"maxConnectionsPerInstance", backend.MaxConnectionsPerInstance,
// 					"maxRate", backend.MaxRate,
// 					"maxRatePerEndpoint", backend.MaxRatePerEndpoint,
// 					"maxRatePerInstance", backend.MaxRatePerInstance,
// 					"maxUtilization", backend.MaxUtilization,
// 				)
// 				if err != nil {
// 					return nil, err
// 				}
// 				mqlBackends = append(mqlBackends, mqlBackend)
// 			}

// 			var cdnPolicy interface{}
// 			if b.CdnPolicy != nil {
// 				bypassCacheOnRequestHeaders := make([]interface{}, 0, len(b.CdnPolicy.BypassCacheOnRequestHeaders))
// 				for _, h := range b.CdnPolicy.BypassCacheOnRequestHeaders {
// 					mqlH := map[string]interface{}{"headerName": h.HeaderName}
// 					bypassCacheOnRequestHeaders = append(bypassCacheOnRequestHeaders, mqlH)
// 				}

// 				var mqlCacheKeyPolicy interface{}
// 				if b.CdnPolicy.CacheKeyPolicy != nil {
// 					mqlCacheKeyPolicy = map[string]interface{}{
// 						"includeHost":          b.CdnPolicy.CacheKeyPolicy.IncludeHost,
// 						"includeHttpHeaders":   core.StrSliceToInterface(b.CdnPolicy.CacheKeyPolicy.IncludeHttpHeaders),
// 						"includeNamedCookies":  core.StrSliceToInterface(b.CdnPolicy.CacheKeyPolicy.IncludeNamedCookies),
// 						"includeProtocol":      b.CdnPolicy.CacheKeyPolicy.IncludeProtocol,
// 						"includeQueryString":   b.CdnPolicy.CacheKeyPolicy.IncludeQueryString,
// 						"queryStringBlacklist": core.StrSliceToInterface(b.CdnPolicy.CacheKeyPolicy.QueryStringBlacklist),
// 						"queryStringWhitelist": core.StrSliceToInterface(b.CdnPolicy.CacheKeyPolicy.QueryStringWhitelist),
// 					}
// 				}

// 				mqlNegativeCachingPolicy := make([]interface{}, 0, len(b.CdnPolicy.NegativeCachingPolicy))
// 				for _, p := range b.CdnPolicy.NegativeCachingPolicy {
// 					mqlP := map[string]interface{}{
// 						"code": p.Code,
// 						"ttl":  p.Ttl,
// 					}
// 					mqlNegativeCachingPolicy = append(mqlNegativeCachingPolicy, mqlP)
// 				}

// 				cdnPolicy, err = g.MotorRuntime.CreateResource("gcp.project.computeService.backendService.cdnPolicy",
// 					"id", fmt.Sprintf("gcp.project.computeService.backendService.cdnPolicy/%s", backendServiceId),
// 					"bypassCacheOnRequestHeaders", bypassCacheOnRequestHeaders,
// 					"cacheKeyPolicy", mqlCacheKeyPolicy,
// 					"cacheMode", b.CdnPolicy.CacheMode,
// 					"clientTtl", b.CdnPolicy.ClientTtl,
// 					"defaultTtl", b.CdnPolicy.DefaultTtl,
// 					"maxTtl", b.CdnPolicy.MaxTtl,
// 					"negativeCaching", b.CdnPolicy.NegativeCaching,
// 					"negativeCachingPolicy", mqlNegativeCachingPolicy,
// 					"requestCoalescing", b.CdnPolicy.RequestCoalescing,
// 					"serveWhileStale", b.CdnPolicy.ServeWhileStale,
// 					"signedUrlCacheMaxAgeSec", b.CdnPolicy.SignedUrlCacheMaxAgeSec,
// 					"signedUrlKeyNames", core.StrSliceToInterface(b.CdnPolicy.SignedUrlKeyNames),
// 				)
// 				if err != nil {
// 					return nil, err
// 				}
// 			}

// 			var mqlCircuitBreakers interface{}
// 			if b.CircuitBreakers != nil {
// 				mqlCircuitBreakers = map[string]interface{}{
// 					"maxConnections":           b.CircuitBreakers.MaxConnections,
// 					"maxPendingRequests":       b.CircuitBreakers.MaxPendingRequests,
// 					"maxRequests":              b.CircuitBreakers.MaxRequests,
// 					"maxRequestsPerConnection": b.CircuitBreakers.MaxRequestsPerConnection,
// 					"maxRetries":               b.CircuitBreakers.MaxRetries,
// 				}
// 			}

// 			var mqlConnectionDraining interface{}
// 			if b.ConnectionDraining != nil {
// 				mqlConnectionDraining = map[string]interface{}{
// 					"drainingTimeoutSec": b.ConnectionDraining.DrainingTimeoutSec,
// 				}
// 			}

// 			var mqlConnectionTrackingPolicy interface{}
// 			if b.ConnectionTrackingPolicy != nil {
// 				mqlConnectionTrackingPolicy = map[string]interface{}{
// 					"connectionPersistenceOnUnhealthyBackends": b.ConnectionTrackingPolicy.ConnectionPersistenceOnUnhealthyBackends,
// 					"enableStrongAffinity":                     b.ConnectionTrackingPolicy.EnableStrongAffinity,
// 					"idleTimeoutSec":                           b.ConnectionTrackingPolicy.IdleTimeoutSec,
// 					"trackingMode":                             b.ConnectionTrackingPolicy.TrackingMode,
// 				}
// 			}

// 			var mqlConsistentHash interface{}
// 			if b.ConsistentHash != nil {
// 				mqlConsistentHash = map[string]interface{}{
// 					"httpCookie": map[string]interface{}{
// 						"name": b.ConsistentHash.HttpCookie.Name,
// 						"path": b.ConsistentHash.HttpCookie.Path,
// 						"ttl":  core.MqlTime(llx.DurationToTime(b.ConsistentHash.HttpCookie.Ttl.Seconds)),
// 					},
// 					"httpHeaderName":  b.ConsistentHash.HttpHeaderName,
// 					"minimumRingSize": b.ConsistentHash.MinimumRingSize,
// 				}
// 			}

// 			var mqlFailoverPolicy interface{}
// 			if b.FailoverPolicy != nil {
// 				mqlFailoverPolicy = map[string]interface{}{
// 					"disableConnectionDrainOnFailover": b.FailoverPolicy.DisableConnectionDrainOnFailover,
// 					"dropTrafficIfUnhealthy":           b.FailoverPolicy.DropTrafficIfUnhealthy,
// 					"failoverRatio":                    b.FailoverPolicy.FailoverRatio,
// 				}
// 			}

// 			var mqlIap interface{}
// 			if b.Iap != nil {
// 				mqlIap = map[string]interface{}{
// 					"enabled":                  b.Iap.Enabled,
// 					"oauth2ClientId":           b.Iap.Oauth2ClientId,
// 					"oauth2ClientSecret":       b.Iap.Oauth2ClientSecret,
// 					"oauth2ClientSecretSha256": b.Iap.Oauth2ClientSecretSha256,
// 				}
// 			}

// 			mqlLocalityLbPolicy := make([]interface{}, 0, len(b.LocalityLbPolicies))
// 			for _, p := range b.LocalityLbPolicies {
// 				var mqlCustomPolicy interface{}
// 				if p.CustomPolicy != nil {
// 					mqlCustomPolicy = map[string]interface{}{
// 						"data": p.CustomPolicy.Data,
// 						"name": p.CustomPolicy.Name,
// 					}
// 				}

// 				var mqlPolicy interface{}
// 				if p.Policy != nil {
// 					mqlPolicy = map[string]interface{}{
// 						"name": p.Policy.Name,
// 					}
// 				}
// 				mqlLocalityLbPolicy = append(mqlLocalityLbPolicy, map[string]interface{}{
// 					"customPolicy": mqlCustomPolicy,
// 					"policy":       mqlPolicy,
// 				})
// 			}

// 			var mqlLogConfig interface{}
// 			if b.LogConfig != nil {
// 				mqlLogConfig = map[string]interface{}{
// 					"enable":     b.LogConfig.Enable,
// 					"sampleRate": b.LogConfig.SampleRate,
// 				}
// 			}

// 			var mqlSecuritySettings interface{}
// 			if b.SecuritySettings != nil {
// 				mqlSecuritySettings = map[string]interface{}{
// 					"clientTlsPolicy": b.SecuritySettings.ClientTlsPolicy,
// 					"subjectAltNames": core.StrSliceToInterface(b.SecuritySettings.SubjectAltNames),
// 				}
// 			}

// 			var maxStreamDuration interface{}
// 			if b.MaxStreamDuration != nil {
// 				maxStreamDuration = core.MqlTime(llx.DurationToTime(b.MaxStreamDuration.Seconds))
// 			}

// 			mqlB, err := g.MotorRuntime.CreateResource("gcp.project.computeService.backendService",
// 				"id", backendServiceId,
// 				"affinityCookieTtlSec", b.AffinityCookieTtlSec,
// 				"backends", mqlBackends,
// 				"cdnPolicy", cdnPolicy,
// 				"circuitBreakers", mqlCircuitBreakers,
// 				"compressionMode", b.CompressionMode,
// 				"connectionDraining", mqlConnectionDraining,
// 				"connectionTrackingPolicy", mqlConnectionTrackingPolicy,
// 				"consistentHash", mqlConsistentHash,
// 				"created", parseTime(b.CreationTimestamp),
// 				"customRequestHeaders", core.StrSliceToInterface(b.CustomRequestHeaders),
// 				"customResponseHeaders", core.StrSliceToInterface(b.CustomResponseHeaders),
// 				"description", b.Description,
// 				"edgeSecurityPolicy", b.EdgeSecurityPolicy,
// 				"enableCDN", b.EnableCDN,
// 				"failoverPolicy", mqlFailoverPolicy,
// 				"healthChecks", core.StrSliceToInterface(b.HealthChecks),
// 				"iap", mqlIap,
// 				"loadBalancingScheme", b.LoadBalancingScheme,
// 				"localityLbPolicies", mqlLocalityLbPolicy,
// 				"localityLbPolicy", b.LocalityLbPolicy,
// 				"logConfig", mqlLogConfig,
// 				"maxStreamDuration", maxStreamDuration,
// 				"name", b.Name,
// 				"networkUrl", b.Network,
// 				"portName", b.PortName,
// 				"protocol", b.Protocol,
// 				"regionUrl", b.Region,
// 				"securityPolicyUrl", b.SecurityPolicy,
// 				"securitySettings", mqlSecuritySettings,
// 				"serviceBindingUrls", core.StrSliceToInterface(b.ServiceBindings),
// 				"sessionAffinity", b.SessionAffinity,
// 				"timeoutSec", b.TimeoutSec,
// 			)
// 			if err != nil {
// 				return nil, err
// 			}
// 			res = append(res, mqlB)
// 		}
// 	}
// 	return res, nil
// }

// func (g *mqlGcpProjectComputeServiceBackendService) id() (string, error) {
// 	id, err := g.Id()
// 	if err != nil {
// 		return "", nil
// 	}
// 	return "gcp.project.computeService.backendService/" + id, nil
// }

// func (g *mqlGcpProjectComputeServiceBackendServiceBackend) id() (string, error) {
// 	return g.Id()
// }

// func (g *mqlGcpProjectComputeServiceBackendServiceCdnPolicy) id() (string, error) {
// 	return g.Id()
// }

// func networkMode(n *compute.Network) string {
// 	if n.IPv4Range != "" {
// 		return "legacy"
// 	} else if n.AutoCreateSubnetworks {
// 		return "auto"
// 	} else {
// 		return "custom"
// 	}
// }

// func (g *mqlGcpProjectComputeService) GetAddresses() ([]interface{}, error) {
// 	projectId, err := g.ProjectId()
// 	if err != nil {
// 		return nil, err
// 	}

// 	provider, err := gcpProvider(g.MotorRuntime.Motor.Provider)
// 	if err != nil {
// 		return nil, err
// 	}

// 	client, err := provider.Client(cloudresourcemanager.CloudPlatformReadOnlyScope, iam.CloudPlatformScope, compute.CloudPlatformScope)
// 	if err != nil {
// 		return nil, err
// 	}

// 	ctx := context.Background()
// 	computeSvc, err := compute.NewService(ctx, option.WithHTTPClient(client))
// 	if err != nil {
// 		return nil, err
// 	}

// 	list, err := computeSvc.Addresses.AggregatedList(projectId).Do()
// 	if err != nil {
// 		return nil, err
// 	}
// 	var mqlAddresses []interface{}
// 	for _, as := range list.Items {
// 		for _, a := range as.Addresses {
// 			mqlA, err := g.MotorRuntime.CreateResource("gcp.project.computeService.address",
// 				"id", fmt.Sprintf("%d", a.Id),
// 				"address", a.Address,
// 				"addressType", a.AddressType,
// 				"created", parseTime(a.CreationTimestamp),
// 				"description", a.Description,
// 				"ipVersion", a.IpVersion,
// 				"ipv6EndpointType", a.Ipv6EndpointType,
// 				"name", a.Name,
// 				"networkUrl", a.Network,
// 				"networkTier", a.NetworkTier,
// 				"prefixLength", a.PrefixLength,
// 				"purpose", a.Purpose,
// 				"regionUrl", a.Region,
// 				"status", a.Status,
// 				"subnetworkUrl", a.Subnetwork,
// 				"resourceUrls", core.StrSliceToInterface(a.Users),
// 			)
// 			if err != nil {
// 				return nil, err
// 			}
// 			mqlAddresses = append(mqlAddresses, mqlA)
// 		}
// 	}
// 	return mqlAddresses, nil
// }

// func (g *mqlGcpProjectComputeServiceAddress) GetNetwork() (interface{}, error) {
// 	networkUrl, err := g.NetworkUrl()
// 	if err != nil {
// 		return nil, err
// 	}
// 	return getNetworkByUrl(networkUrl, g.MotorRuntime)
// }

// func (g *mqlGcpProjectComputeServiceAddress) GetSubnetwork() (interface{}, error) {
// 	subnetUrl, err := g.SubnetworkUrl()
// 	if err != nil {
// 		return nil, err
// 	}
// 	return getSubnetworkByUrl(subnetUrl, g.MotorRuntime)
// }

// func (g *mqlGcpProjectComputeServiceAddress) id() (string, error) {
// 	return g.Id()
// }

// func (g *mqlGcpProjectComputeService) GetForwardingRules() ([]interface{}, error) {
// 	projectId, err := g.ProjectId()
// 	if err != nil {
// 		return nil, err
// 	}

// 	provider, err := gcpProvider(g.MotorRuntime.Motor.Provider)
// 	if err != nil {
// 		return nil, err
// 	}

// 	client, err := provider.Client(cloudresourcemanager.CloudPlatformReadOnlyScope, iam.CloudPlatformScope, compute.CloudPlatformScope)
// 	if err != nil {
// 		return nil, err
// 	}

// 	ctx := context.Background()
// 	fwrSvc, err := computev1.NewForwardingRulesRESTClient(ctx, option.WithHTTPClient(client))
// 	if err != nil {
// 		return nil, err
// 	}

// 	var fwRules []interface{}
// 	it := fwrSvc.AggregatedList(ctx, &computepb.AggregatedListForwardingRulesRequest{Project: projectId, IncludeAllScopes: ptr.Bool(true)})
// 	for {
// 		resp, err := it.Next()
// 		if err == iterator.Done {
// 			break
// 		}
// 		if err != nil {
// 			return nil, err
// 		}
// 		for _, fwr := range resp.Value.ForwardingRules {
// 			metadataFilters := make([]interface{}, 0, len(fwr.GetMetadataFilters()))
// 			for _, m := range fwr.GetMetadataFilters() {
// 				filterLabels := make([]interface{}, 0, len(m.GetFilterLabels()))
// 				for _, l := range m.GetFilterLabels() {
// 					filterLabels = append(filterLabels, map[string]interface{}{
// 						"name":  l.GetName(),
// 						"value": l.GetValue(),
// 					})
// 				}
// 				metadataFilters = append(metadataFilters, map[string]interface{}{
// 					"filterLabels":        filterLabels,
// 					"filterMatchCriteria": m.GetFilterMatchCriteria(),
// 				})
// 			}

// 			serviceDirRegs := make([]interface{}, 0, len(fwr.GetServiceDirectoryRegistrations()))
// 			for _, s := range fwr.GetServiceDirectoryRegistrations() {
// 				serviceDirRegs = append(serviceDirRegs, map[string]interface{}{
// 					"namespace":              s.GetNamespace(),
// 					"service":                s.GetService(),
// 					"serviceDirectoryRegion": s.GetServiceDirectoryRegion(),
// 				})
// 			}
// 			mqlFwr, err := g.MotorRuntime.CreateResource("gcp.project.computeService.forwardingRule",
// 				"id", fmt.Sprintf("%d", fwr.Id),
// 				"ipAddress", fwr.GetIPAddress(),
// 				"ipProtocol", fwr.GetIPProtocol(),
// 				"allPorts", fwr.GetAllPorts(),
// 				"allowGlobalAccess", fwr.GetAllowGlobalAccess(),
// 				"backendService", fwr.GetBackendService(),
// 				"created", parseTime(fwr.GetCreationTimestamp()),
// 				"description", fwr.GetDescription(),
// 				"ipVersion", fwr.GetIpVersion(),
// 				"isMirroringCollector", fwr.GetIsMirroringCollector(),
// 				"labels", core.StrMapToInterface(fwr.GetLabels()),
// 				"loadBalancingScheme", fwr.GetLoadBalancingScheme(),
// 				"metadataFilters", metadataFilters,
// 				"name", fwr.GetName(),
// 				"networkUrl", fwr.GetNetwork(),
// 				"networkTier", fwr.GetNetworkTier(),
// 				"noAutomateDnsZone", fwr.GetNoAutomateDnsZone(),
// 				"portRange", fwr.GetPortRange(),
// 				"ports", core.StrSliceToInterface(fwr.GetPorts()),
// 				"regionUrl", fwr.GetRegion(),
// 				"serviceDirectoryRegistrations", serviceDirRegs,
// 				"serviceLabel", fwr.GetServiceLabel(),
// 				"serviceName", fwr.GetServiceName(),
// 				"subnetworkUrl", fwr.GetSubnetwork(),
// 				"targetUrl", fwr.GetTarget(),
// 			)
// 			if err != nil {
// 				return nil, err
// 			}
// 			fwRules = append(fwRules, mqlFwr)
// 		}
// 	}
// 	return fwRules, nil
// }

// func (g *mqlGcpProjectComputeServiceForwardingRule) id() (string, error) {
// 	return g.Id()
// }

// func (g *mqlGcpProjectComputeServiceForwardingRule) GetNetwork() (interface{}, error) {
// 	networkUrl, err := g.NetworkUrl()
// 	if err != nil {
// 		return nil, err
// 	}
// 	return getNetworkByUrl(networkUrl, g.MotorRuntime)
// }

// func (g *mqlGcpProjectComputeServiceForwardingRule) GetSubnetwork() (interface{}, error) {
// 	subnetUrl, err := g.SubnetworkUrl()
// 	if err != nil {
// 		return nil, err
// 	}
// 	return getSubnetworkByUrl(subnetUrl, g.MotorRuntime)
// }
