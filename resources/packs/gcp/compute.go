package gcp

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/resources/packs/core"
	"google.golang.org/api/cloudresourcemanager/v3"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/iam/v1"
	"google.golang.org/api/option"
)

func (g *mqlGcpProjectComputeService) init(args *resources.Args) (*resources.Args, GcpProjectComputeService, error) {
	if len(*args) > 2 {
		return args, nil, nil
	}

	provider, err := gcpProvider(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, nil, err
	}

	projectId := provider.ResourceID()
	(*args)["projectId"] = projectId

	return args, nil, nil
}

func (g *mqlGcpProject) GetCompute() (interface{}, error) {
	projectId, err := g.Id()
	if err != nil {
		return nil, err
	}

	return g.MotorRuntime.CreateResource("gcp.project.computeService",
		"projectId", projectId,
	)
}

func (g *mqlGcpProjectComputeService) id() (string, error) {
	projectId, err := g.ProjectId()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/gcp.project.computeService", projectId), nil
}

func (g *mqlGcpProjectComputeServiceRegion) id() (string, error) {
	id, err := g.Name()
	if err != nil {
		return "", err
	}
	return "gcp.project.computeService.region/" + id, nil
}

func (g *mqlGcpProjectComputeServiceRegion) init(args *resources.Args) (*resources.Args, GcpProjectComputeServiceRegion, error) {
	if len(*args) > 2 {
		return args, nil, nil
	}

	provider, err := gcpProvider(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, nil, err
	}

	projectId := provider.ResourceID()
	(*args)["projectId"] = projectId

	return args, nil, nil
}

func (g *mqlGcpProjectComputeService) GetRegions() ([]interface{}, error) {
	projectId, err := g.ProjectId()
	if err != nil {
		return nil, err
	}

	provider, err := gcpProvider(g.MotorRuntime.Motor.Provider)
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
		mqlRegion, err := newMqlRegion(g.MotorRuntime, r)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlRegion)
	}

	return res, nil
}

func (g *mqlGcpProjectComputeServiceZone) id() (string, error) {
	id, err := g.Id()
	if err != nil {
		return "", err
	}
	return "gcp.project.computeService.zone/" + id, nil
}

func (g *mqlGcpProjectComputeServiceZone) GetRegion() (interface{}, error) {
	// TODO: implement
	return nil, errors.New("not implemented")
}

func (g *mqlGcpProjectComputeService) GetZones() ([]interface{}, error) {
	projectId, err := g.ProjectId()
	if err != nil {
		return nil, err
	}

	provider, err := gcpProvider(g.MotorRuntime.Motor.Provider)
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
			mqlZone, err := g.MotorRuntime.CreateResource("gcp.project.computeService.zone",
				"id", strconv.FormatInt(int64(zone.Id), 10),
				"name", zone.Name,
				"description", zone.Description,
				"status", zone.Status,
				"created", parseTime(zone.CreationTimestamp),
			)
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
	id, err := g.Id()
	if err != nil {
		return "", err
	}

	projectId, err := g.ProjectId()
	if err != nil {
		return "", err
	}

	return "gcp.project.computeService.machineType/" + projectId + "/" + id, nil
}

func (g *mqlGcpProjectComputeServiceMachineType) GetZone() (interface{}, error) {
	// NOTE: this should never be called since we add the zone during construction of the resource
	return nil, errors.New("not implemented")
}

func newMqlMachineType(runtime *resources.Runtime, entry *compute.MachineType, projectId string, zone GcpProjectComputeServiceZone) (interface{}, error) {
	return runtime.CreateResource("gcp.project.computeService.machineType",
		"id", strconv.FormatInt(int64(entry.Id), 10),
		"projectId", projectId,
		"name", entry.Name,
		"description", entry.Description,
		"guestCpus", entry.GuestCpus,
		"isSharedCpu", entry.IsSharedCpu,
		"maximumPersistentDisks", entry.MaximumPersistentDisks,
		"maximumPersistentDisksSizeGb", entry.MaximumPersistentDisksSizeGb,
		"memoryMb", entry.MemoryMb,
		"created", parseTime(entry.CreationTimestamp),
		"zone", zone,
	)
}

func (g *mqlGcpProjectComputeService) GetMachineTypes() ([]interface{}, error) {
	projectId, err := g.ProjectId()
	if err != nil {
		return nil, err
	}

	// get list of zones first since we need this for all entries
	zones, err := g.Zones()
	if err != nil {
		return nil, err
	}

	provider, err := gcpProvider(g.MotorRuntime.Motor.Provider)
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
	wg.Add(len(zones))
	mux := &sync.Mutex{}

	for i := range zones {
		z := zones[i].(GcpProjectComputeServiceZone)
		zoneName, err := z.Name()
		if err != nil {
			return nil, err
		}

		go func(svc *compute.Service, projectId string, zone GcpProjectComputeServiceZone, zoneName string) {
			req := computeSvc.MachineTypes.List(projectId, zoneName)
			if err := req.Pages(ctx, func(page *compute.MachineTypeList) error {
				for _, machinetype := range page.Items {
					mqlMachineType, err := newMqlMachineType(g.MotorRuntime, machinetype, projectId, zone)
					if err != nil {
						return err
					} else {
						mux.Lock()
						res = append(res, mqlMachineType)
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

func (g *mqlGcpProjectComputeServiceInstance) id() (string, error) {
	id, err := g.Id()
	if err != nil {
		return "", err
	}

	projectId, err := g.ProjectId()
	if err != nil {
		return "", err
	}

	return "gcp.project.computeService.instance/" + projectId + "/" + id, nil
}

func (g *mqlGcpProjectComputeServiceInstance) GetMachineType() (interface{}, error) {
	projectId, err := g.ProjectId()
	if err != nil {
		return "", err
	}

	zone, err := g.Zone()
	if err != nil {
		return "", err
	}

	zoneName, err := zone.Name()
	if err != nil {
		return "", err
	}

	entry, ok := g.MqlResource().Cache.Load("_machineType")
	if !ok {
		return nil, errors.New("could not fine a ")
	}
	machineTypeUrl := entry.Data.(string)
	values := strings.Split(machineTypeUrl, "/")
	machineTypeValue := values[len(values)-1]

	// TODO: we can save calls if we move it to the into method
	provider, err := gcpProvider(g.MotorRuntime.Motor.Provider)
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

	return newMqlMachineType(g.MotorRuntime, machineType, projectId, zone)
}

func newMqlServiceAccount(runtime *resources.Runtime, sa *compute.ServiceAccount) (interface{}, error) {
	return runtime.CreateResource("gcp.project.computeService.serviceaccount",
		"email", sa.Email,
		"scopes", core.StrSliceToInterface(sa.Scopes),
	)
}

func newMqlAttachedDisk(id string, projectId string, runtime *resources.Runtime, attachedDisk *compute.AttachedDisk) (interface{}, error) {
	guestOsFeatures := []string{}
	for i := range attachedDisk.GuestOsFeatures {
		entry := attachedDisk.GuestOsFeatures[i]
		guestOsFeatures = append(guestOsFeatures, entry.Type)
	}

	mqlAttachedDisk, err := runtime.CreateResource("gcp.project.computeService.attachedDisk",
		"id", id,
		"projectId", projectId,
		"architecture", attachedDisk.Architecture,
		"autoDelete", attachedDisk.AutoDelete,
		"boot", attachedDisk.Boot,
		"deviceName", attachedDisk.DeviceName,
		"diskSizeGb", attachedDisk.DiskSizeGb,
		"forceAttach", attachedDisk.ForceAttach,
		"guestOsFeatures", core.StrSliceToInterface(guestOsFeatures),
		"index", attachedDisk.Index,
		"interface", attachedDisk.Interface,
		"licenses", core.StrSliceToInterface(attachedDisk.Licenses),
		"mode", attachedDisk.Mode,
		"type", attachedDisk.Type,
	)
	if err != nil {
		return nil, err
	}
	mqlAttachedDisk.MqlResource().Cache.Store("_attachedDiskSource", &resources.CacheEntry{Data: attachedDisk.Source})
	return mqlAttachedDisk, nil
}

func (g *mqlGcpProjectComputeServiceAttachedDisk) id() (string, error) {
	id, err := g.Id()
	if err != nil {
		return "", err
	}

	projectId, err := g.ProjectId()
	if err != nil {
		return "", err
	}

	return "gcp.project.computeService.attachedDisk/" + projectId + "/" + id, nil
}

func (g *mqlGcpProjectComputeServiceAttachedDisk) GetSource() (interface{}, error) {
	entry, ok := g.MqlResource().Cache.Load("_attachedDiskSource")
	if !ok {
		return nil, errors.New("could not find persistent disk")
	}

	sourceDisk := entry.Data.(string)
	log.Info().Msg(sourceDisk)

	return nil, nil
}

func newMqlInstance(projectId string, zone GcpProjectComputeServiceZone, runtime *resources.Runtime, instance *compute.Instance) (GcpProjectComputeServiceInstance, error) {
	metadata := map[string]string{}
	for m := range instance.Metadata.Items {
		item := instance.Metadata.Items[m]
		metadata[item.Key] = core.ToString(item.Value)
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

	guestAccelerators, err := core.JsonToDictSlice(instance.GuestAccelerators)
	if err != nil {
		return nil, err
	}

	networkInterfaces, err := core.JsonToDictSlice(instance.NetworkInterfaces)
	if err != nil {
		return nil, err
	}

	reservationAffinity, err := core.JsonToDict(instance.ReservationAffinity)
	if err != nil {
		return nil, err
	}

	scheduling, err := core.JsonToDict(instance.Scheduling)
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
		mqlConfCompute, err = core.JsonToDict(
			mqlConfidentialInstanceConfig{Enabled: instance.ConfidentialInstanceConfig.EnableConfidentialCompute})
		if err != nil {
			return nil, err
		}
	}

	entry, err := runtime.CreateResource("gcp.project.computeService.instance",
		"id", instanceId,
		"projectId", projectId,
		"name", instance.Name,
		"cpuPlatform", instance.CpuPlatform,
		"description", instance.Description,
		"confidentialInstanceConfig", mqlConfCompute,
		"canIpForward", instance.CanIpForward,
		"cpuPlatform", instance.CpuPlatform,
		"created", parseTime(instance.CreationTimestamp),
		"deletionProtection", instance.DeletionProtection,
		"enableDisplay", enableDisplay,
		"guestAccelerators", guestAccelerators,
		"fingerprint", instance.Fingerprint,
		"hostname", instance.Hostname,
		"keyRevocationActionType", instance.KeyRevocationActionType,
		"labels", core.StrMapToInterface(instance.Labels),
		"lastStartTimestamp", parseTime(instance.LastStartTimestamp),
		"lastStopTimestamp", parseTime(instance.LastStopTimestamp),
		"lastSuspendedTimestamp", parseTime(instance.LastSuspendedTimestamp),
		"metadata", core.StrMapToInterface(metadata),
		"minCpuPlatform", instance.MinCpuPlatform,
		"networkInterfaces", networkInterfaces,
		"privateIpv6GoogleAccess", instance.PrivateIpv6GoogleAccess,
		"reservationAffinity", reservationAffinity,
		"resourcePolicies", core.StrSliceToInterface(instance.ResourcePolicies),
		"physicalHostResourceStatus", physicalHost,
		"scheduling", scheduling,
		"enableIntegrityMonitoring", enableIntegrityMonitoring,
		"enableSecureBoot", enableSecureBoot,
		"enableVtpm", enableVtpm,
		"startRestricted", instance.StartRestricted,
		"status", instance.Status,
		"statusMessage", instance.StatusMessage,
		"sourceMachineImage", instance.SourceMachineImage,
		"tags", core.StrSliceToInterface(instance.Tags.Items),
		"totalEgressBandwidthTier", totalEgressBandwidthTier,
		"serviceAccounts", mqlServiceAccounts,
		"disks", attachedDisks,
		"zone", zone,
	)
	if err != nil {
		return nil, err
	}
	return entry.(GcpProjectComputeServiceInstance), nil
}

func (g *mqlGcpProjectComputeService) GetInstances() ([]interface{}, error) {
	projectId, err := g.ProjectId()
	if err != nil {
		return nil, err
	}

	// get list of zones first since we need this for all entries
	zones, err := g.Zones()
	if err != nil {
		return nil, err
	}

	provider, err := gcpProvider(g.MotorRuntime.Motor.Provider)
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
	wg.Add(len(zones))
	mux := &sync.Mutex{}

	for i := range zones {
		z := zones[i].(GcpProjectComputeServiceZone)
		zoneName, err := z.Name()
		if err != nil {
			return nil, err
		}
		go func(svc *compute.Service, project string, zone GcpProjectComputeServiceZone, zoneName string) {
			req := computeSvc.Instances.List(projectId, zoneName)
			if err := req.Pages(ctx, func(page *compute.InstanceList) error {
				for _, instance := range page.Items {

					mqlInstance, err := newMqlInstance(projectId, zone, g.MotorRuntime, instance)
					if err != nil {
						return err
					} else {
						mqlInstance.MqlResource().Cache.Store("_machineType", &resources.CacheEntry{Data: instance.MachineType})
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
	email, err := g.Email()
	if err != nil {
		return "", nil
	}
	return "gcp.project.computeService.serviceaccount/" + email, nil
}

func (g *mqlGcpProjectComputeServiceDisk) id() (string, error) {
	id, err := g.Id()
	if err != nil {
		return "", nil
	}

	return "gcloud.compute.disk/" + id, nil
}

func (g *mqlGcpProjectComputeServiceDisk) GetZone() (interface{}, error) {
	// TODO: implement
	return nil, errors.New("not implemented")
}

func (g *mqlGcpProjectComputeService) GetDisks() ([]interface{}, error) {
	projectId, err := g.ProjectId()
	if err != nil {
		return nil, err
	}

	// get list of zones first since we need this for all entries
	zones, err := g.Zones()
	if err != nil {
		return nil, err
	}

	provider, err := gcpProvider(g.MotorRuntime.Motor.Provider)
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
	wg.Add(len(zones))
	mux := &sync.Mutex{}

	var result error
	for i := range zones {
		z := zones[i].(GcpProjectComputeServiceZone)
		zoneName, err := z.Name()
		if err != nil {
			return nil, err
		}

		go func(svc *compute.Service, project string, zone GcpProjectComputeServiceZone, zoneName string) {
			req := computeSvc.Disks.List(projectId, zoneName)
			if err := req.Pages(ctx, func(page *compute.DiskList) error {
				for _, disk := range page.Items {
					guestOsFeatures := []string{}
					for i := range disk.GuestOsFeatures {
						entry := disk.GuestOsFeatures[i]
						guestOsFeatures = append(guestOsFeatures, entry.Type)
					}

					var mqlDiskEnc map[string]interface{}
					if disk.DiskEncryptionKey != nil {
						mqlDiskEnc = map[string]interface{}{
							"kmsKeyName":           disk.DiskEncryptionKey.KmsKeyName,
							"kmsKeyServiceAccount": disk.DiskEncryptionKey.KmsKeyServiceAccount,
							"rawKey":               disk.DiskEncryptionKey.RawKey,
							"rsaEncryptedKey":      disk.DiskEncryptionKey.RsaEncryptedKey,
							"sha256":               disk.DiskEncryptionKey.Sha256,
						}
					}

					mqlDisk, err := g.MotorRuntime.CreateResource("gcp.project.computeService.disk",
						"id", strconv.FormatUint(disk.Id, 10),
						"name", disk.Name,
						"architecture", disk.Architecture,
						"description", disk.Description,
						"guestOsFeatures", core.StrSliceToInterface(guestOsFeatures),
						"labels", core.StrMapToInterface(disk.Labels),
						"lastAttachTimestamp", parseTime(disk.LastAttachTimestamp),
						"lastDetachTimestamp", parseTime(disk.LastDetachTimestamp),
						"locationHint", disk.LocationHint,
						"licenses", core.StrSliceToInterface(disk.Licenses),
						"physicalBlockSizeBytes", disk.PhysicalBlockSizeBytes,
						"provisionedIops", disk.ProvisionedIops,
						// TODO: link to resources
						//"region", disk.Region,
						//"replicaZones", core.StrSliceToInterface(disk.ReplicaZones),
						//"resourcePolicies", core.StrSliceToInterface(disk.ResourcePolicies),
						"sizeGb", disk.SizeGb,
						// TODO: link to resources
						//"sourceDiskId", disk.SourceDiskId,
						//"sourceImageId", disk.SourceImageId,
						//"sourceSnapshotId", disk.SourceSnapshotId,
						"status", disk.Status,
						"zone", zone,
						"created", parseTime(disk.CreationTimestamp),
						"diskEncryptionKey", mqlDiskEnc,
					)
					if err != nil {
						return err
					} else {
						mux.Lock()
						res = append(res, mqlDisk)
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
	return res, result
}

func (g *mqlGcpProjectComputeServiceFirewall) id() (string, error) {
	id, err := g.Id()
	if err != nil {
		return "", nil
	}

	return "gcloud.compute.firewall/" + id, nil
}

func (g *mqlGcpProjectComputeServiceFirewall) GetNetwork() (interface{}, error) {
	// TODO: implement
	return nil, errors.New("not implemented")
}

func (g *mqlGcpProjectComputeServiceFirewall) init(args *resources.Args) (*resources.Args, GcpProjectComputeServiceFirewall, error) {
	if len(*args) > 2 {
		return args, nil, nil
	}

	if ids := getAssetIdentifier(g.MotorRuntime); ids != nil {
		(*args)["name"] = ids.name
		(*args)["projectId"] = ids.project
	}

	obj, err := g.MotorRuntime.CreateResource("gcp.project.computeService", "projectId", (*args)["projectId"])
	if err != nil {
		return nil, nil, err
	}
	computeSvc := obj.(GcpProjectComputeService)
	firewalls, err := computeSvc.Firewalls()
	if err != nil {
		return nil, nil, err
	}

	for _, f := range firewalls {
		firewall := f.(GcpProjectComputeServiceFirewall)
		name, err := firewall.Name()
		if err != nil {
			return nil, nil, err
		}
		projectId, err := firewall.ProjectId()
		if err != nil {
			return nil, nil, err
		}

		if name == (*args)["name"] && projectId == (*args)["projectId"] {
			return args, firewall, nil
		}
	}
	return nil, nil, &resources.ResourceNotFound{}
}

func (g *mqlGcpProjectComputeService) GetFirewalls() ([]interface{}, error) {
	projectId, err := g.ProjectId()
	if err != nil {
		return nil, err
	}

	provider, err := gcpProvider(g.MotorRuntime.Motor.Provider)
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

	type mqlFirewall struct {
		IpProtocol string   `json:"ipProtocol"`
		Ports      []string `json:"ports"`
	}

	res := []interface{}{}
	req := computeSvc.Firewalls.List(projectId)
	if err := req.Pages(ctx, func(page *compute.FirewallList) error {
		for _, firewall := range page.Items {
			allowed := make([]mqlFirewall, 0, len(firewall.Allowed))
			for _, a := range firewall.Allowed {
				allowed = append(allowed, mqlFirewall{IpProtocol: a.IPProtocol, Ports: a.Ports})
			}
			allowedDict, err := core.JsonToDictSlice(allowed)
			if err != nil {
				return err
			}

			denied := make([]mqlFirewall, 0, len(firewall.Denied))
			for _, d := range firewall.Denied {
				denied = append(denied, mqlFirewall{IpProtocol: d.IPProtocol, Ports: d.Ports})
			}
			deniedDict, err := core.JsonToDictSlice(denied)
			if err != nil {
				return err
			}

			mqlFirewall, err := g.MotorRuntime.CreateResource("gcp.project.computeService.firewall",
				"id", strconv.FormatUint(firewall.Id, 10),
				"projectId", projectId,
				"name", firewall.Name,
				"description", firewall.Description,
				"priority", firewall.Priority,
				"disabled", firewall.Disabled,
				"direction", firewall.Direction,
				"sourceRanges", core.StrSliceToInterface(firewall.SourceRanges),
				"sourceServiceAccounts", core.StrSliceToInterface(firewall.SourceServiceAccounts),
				"sourceTags", core.StrSliceToInterface(firewall.SourceTags),
				"destinationRanges", core.StrSliceToInterface(firewall.DestinationRanges),
				"targetServiceAccounts", core.StrSliceToInterface(firewall.TargetServiceAccounts),
				"created", parseTime(firewall.CreationTimestamp),
				"allowed", allowedDict,
				"denied", deniedDict,
			)
			if err != nil {
				return err
			} else {
				res = append(res, mqlFirewall)
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return res, nil
}

func (g *mqlGcpProjectComputeServiceSnapshot) id() (string, error) {
	id, err := g.Id()
	if err != nil {
		return "", nil
	}

	return "gcloud.compute.snapshot/" + id, nil
}

func (g *mqlGcpProjectComputeServiceSnapshot) GetSourceDisk() (interface{}, error) {
	// TODO: implement
	return nil, errors.New("not implemented")
}

func (g *mqlGcpProjectComputeService) GetSnapshots() ([]interface{}, error) {
	projectId, err := g.ProjectId()
	if err != nil {
		return nil, err
	}

	provider, err := gcpProvider(g.MotorRuntime.Motor.Provider)
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

	res := []interface{}{}
	req := computeSvc.Snapshots.List(projectId)
	if err := req.Pages(ctx, func(page *compute.SnapshotList) error {
		for _, snapshot := range page.Items {
			mqlSnapshpt, err := g.MotorRuntime.CreateResource("gcp.project.computeService.snapshot",
				"id", strconv.FormatUint(snapshot.Id, 10),
				"name", snapshot.Name,
				"description", snapshot.Description,
				"architecture", snapshot.Architecture,
				"autoCreated", snapshot.AutoCreated,
				"chainName", snapshot.ChainName,
				"creationSizeBytes", snapshot.CreationSizeBytes,
				"diskSizeGb", snapshot.DiskSizeGb,
				"downloadBytes", snapshot.DownloadBytes,
				"storageBytes", snapshot.StorageBytes,
				"storageBytesStatus", snapshot.StorageBytesStatus,
				"snapshotType", snapshot.SnapshotType,
				"licenses", core.StrSliceToInterface(snapshot.Licenses),
				"labels", core.StrMapToInterface(snapshot.Labels),
				"status", snapshot.Status,
				"created", parseTime(snapshot.CreationTimestamp),
			)
			if err != nil {
				return err
			}

			res = append(res, mqlSnapshpt)
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return res, nil
}

func (g *mqlGcpProjectComputeServiceImage) id() (string, error) {
	id, err := g.Id()
	if err != nil {
		return "", nil
	}

	return "gcloud.compute.image/" + id, nil
}

func (g *mqlGcpProjectComputeServiceImage) init(args *resources.Args) (*resources.Args, GcpProjectComputeServiceImage, error) {
	if len(*args) > 2 {
		return args, nil, nil
	}

	if ids := getAssetIdentifier(g.MotorRuntime); ids != nil {
		(*args)["name"] = ids.name
		(*args)["projectId"] = ids.project
	}

	obj, err := g.MotorRuntime.CreateResource("gcp.project.computeService", "projectId", (*args)["projectId"])
	if err != nil {
		return nil, nil, err
	}
	computeSvc := obj.(GcpProjectComputeService)
	images, err := computeSvc.Images()
	if err != nil {
		return nil, nil, err
	}

	for _, i := range images {
		image := i.(GcpProjectComputeServiceImage)
		name, err := image.Name()
		if err != nil {
			return nil, nil, err
		}
		projectId, err := image.ProjectId()
		if err != nil {
			return nil, nil, err
		}

		if name == (*args)["name"] && projectId == (*args)["projectId"] {
			return args, image, nil
		}
	}
	return nil, nil, &resources.ResourceNotFound{}
}

func (g *mqlGcpProjectComputeServiceImage) GetSourceDisk() (interface{}, error) {
	// TODO: implement
	return nil, errors.New("not implemented")
}

func (g *mqlGcpProjectComputeService) GetImages() ([]interface{}, error) {
	projectId, err := g.ProjectId()
	if err != nil {
		return nil, err
	}

	provider, err := gcpProvider(g.MotorRuntime.Motor.Provider)
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

	res := []interface{}{}
	req := computeSvc.Images.List(projectId)
	if err := req.Pages(ctx, func(page *compute.ImageList) error {
		for _, image := range page.Items {
			mqlImage, err := g.MotorRuntime.CreateResource("gcp.project.computeService.image",
				"id", strconv.FormatUint(image.Id, 10),
				"projectId", projectId,
				"name", image.Name,
				"description", image.Description,
				"architecture", image.Architecture,
				"archiveSizeBytes", image.ArchiveSizeBytes,
				"diskSizeGb", image.DiskSizeGb,
				"family", image.Family,
				"licenses", core.StrSliceToInterface(image.Licenses),
				"labels", core.StrMapToInterface(image.Labels),
				"status", image.Status,
				"created", parseTime(image.CreationTimestamp),
			)
			if err != nil {
				return err
			}
			res = append(res, mqlImage)
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return res, nil
}

func (g *mqlGcpProjectComputeServiceNetwork) id() (string, error) {
	id, err := g.Id()
	if err != nil {
		return "", nil
	}

	return "gcloud.compute.network/" + id, nil
}

func (g *mqlGcpProjectComputeServiceNetwork) GetSubnetworks() ([]interface{}, error) {
	provider, err := gcpProvider(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	client, err := provider.Client(cloudresourcemanager.CloudPlatformReadOnlyScope, iam.CloudPlatformScope, compute.CloudPlatformScope)
	if err != nil {
		return nil, err
	}

	subnetUrls, err := g.SubnetworkUrls()
	if err != nil {
		return nil, err
	}
	type resourceId struct {
		Project string
		Region  string
		Name    string
	}
	ids := make([]resourceId, 0, len(subnetUrls))
	for _, subnetUrl := range subnetUrls {
		// Format is https://www.googleapis.com/compute/v1/projects/mondoo-edge/regions/us-central1/subnetworks/mondoo-gke-cluster-2-subnet
		params := strings.TrimPrefix(subnetUrl.(string), "https://www.googleapis.com/compute/v1/")
		parts := strings.Split(params, "/")
		ids = append(ids, resourceId{Project: parts[1], Region: parts[3], Name: parts[5]})
	}

	ctx := context.Background()
	computeSvc, err := compute.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}

	var wg sync.WaitGroup
	res := []interface{}{}
	wg.Add(len(ids))
	mux := &sync.Mutex{}
	for _, id := range ids {
		go func(id resourceId) {
			defer wg.Done()
			subnet, err := computeSvc.Subnetworks.Get(id.Project, id.Region, id.Name).Do()
			if err != nil {
				log.Error().Err(err).Send()
			}
			mqlSubnet, err := newMqlSubnetwork(id.Project, g.MotorRuntime, subnet, nil)
			if err != nil {
				log.Error().Err(err).Send()
			}
			mux.Lock()
			res = append(res, mqlSubnet)
			mux.Unlock()
		}(id)
	}
	wg.Wait()
	return res, nil
}

func (g *mqlGcpProjectComputeServiceNetwork) init(args *resources.Args) (*resources.Args, GcpProjectComputeServiceNetwork, error) {
	if len(*args) > 2 {
		return args, nil, nil
	}

	if ids := getAssetIdentifier(g.MotorRuntime); ids != nil {
		(*args)["name"] = ids.name
		(*args)["projectId"] = ids.project
	}

	obj, err := g.MotorRuntime.CreateResource("gcp.project.computeService", "projectId", (*args)["projectId"])
	if err != nil {
		return nil, nil, err
	}
	computeSvc := obj.(GcpProjectComputeService)
	networks, err := computeSvc.Networks()
	if err != nil {
		return nil, nil, err
	}

	for _, n := range networks {
		network := n.(GcpProjectComputeServiceNetwork)
		name, err := network.Name()
		if err != nil {
			return nil, nil, err
		}
		projectId, err := network.ProjectId()
		if err != nil {
			return nil, nil, err
		}

		if name == (*args)["name"] && projectId == (*args)["projectId"] {
			return args, network, nil
		}
	}
	return nil, nil, &resources.ResourceNotFound{}
}

func (g *mqlGcpProjectComputeService) GetNetworks() ([]interface{}, error) {
	projectId, err := g.ProjectId()
	if err != nil {
		return nil, err
	}

	provider, err := gcpProvider(g.MotorRuntime.Motor.Provider)
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

	res := []interface{}{}
	req := computeSvc.Networks.List(projectId)
	if err := req.Pages(ctx, func(page *compute.NetworkList) error {
		for _, network := range page.Items {

			peerings, err := core.JsonToDictSlice(network.Peerings)
			if err != nil {
				return err
			}

			var routingMode string
			if network.RoutingConfig != nil {
				routingMode = network.RoutingConfig.RoutingMode
			}

			mqlNetwork, err := g.MotorRuntime.CreateResource("gcp.project.computeService.network",
				"id", strconv.FormatUint(network.Id, 10),
				"projectId", projectId,
				"name", network.Name,
				"description", network.Description,
				"autoCreateSubnetworks", network.AutoCreateSubnetworks,
				"enableUlaInternalIpv6", network.EnableUlaInternalIpv6,
				"gatewayIPv4", network.GatewayIPv4,
				"mtu", network.Mtu,
				"networkFirewallPolicyEnforcementOrder", network.NetworkFirewallPolicyEnforcementOrder,
				"created", parseTime(network.CreationTimestamp),
				"peerings", peerings,
				"routingMode", routingMode,
				"mode", networkMode(network),
				"subnetworkUrls", core.StrSliceToInterface(network.Subnetworks),
			)
			if err != nil {
				return err
			}
			res = append(res, mqlNetwork)
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return res, nil
}

func (g *mqlGcpProjectComputeServiceSubnetwork) id() (string, error) {
	id, err := g.Id()
	if err != nil {
		return "", nil
	}

	return "gcloud.compute.subnetwork/" + id, nil
}

func (g *mqlGcpProjectComputeServiceSubnetworkLogConfig) id() (string, error) {
	id, err := g.Id()
	if err != nil {
		return "", nil
	}

	return "gcloud.compute.subnetwork.logConfig/" + id, nil
}

func (g *mqlGcpProjectComputeServiceSubnetwork) GetRegion() (interface{}, error) {
	projectId, err := g.ProjectId()
	if err != nil {
		return nil, err
	}

	regionUrl, err := g.RegionUrl()
	if err != nil {
		return nil, err
	}

	regionUrlSegments := strings.Split(regionUrl, "/")
	regionName := regionUrlSegments[len(regionUrlSegments)-1]

	// Find regionName for projectId
	obj, err := g.MotorRuntime.CreateResource("gcp.project.computeService", "projectId", projectId)
	if err != nil {
		return nil, err
	}
	gcpCompute := obj.(GcpProjectComputeService)
	regions, err := gcpCompute.Regions()
	if err != nil {
		return nil, err
	}

	for _, r := range regions {
		region := r.(GcpProjectComputeServiceRegion)
		name, err := region.Name()
		if err != nil {
			return nil, err
		}
		if name == regionName {
			return region, nil
		}
	}

	return nil, errors.New(fmt.Sprintf("region %s not found", regionName))
}

func (g *mqlGcpProjectComputeServiceSubnetwork) GetNetwork() ([]interface{}, error) {
	// TODO: implement
	return nil, errors.New("not implemented")
}

func newMqlRegion(runtime *resources.Runtime, r *compute.Region) (interface{}, error) {
	deprecated, err := core.JsonToDict(r.Deprecated)
	if err != nil {
		return nil, err
	}

	quotas := map[string]interface{}{}
	for i := range r.Quotas {
		q := r.Quotas[i]
		quotas[q.Metric] = q.Limit
	}

	return runtime.CreateResource("gcp.project.computeService.region",
		"id", strconv.FormatInt(int64(r.Id), 10),
		"name", r.Name,
		"description", r.Description,
		"status", r.Status,
		"created", parseTime(r.CreationTimestamp),
		"quotas", quotas,
		"deprecated", deprecated,
	)
}

func newMqlSubnetwork(projectId string, runtime *resources.Runtime, subnetwork *compute.Subnetwork, region GcpProjectComputeServiceRegion) (interface{}, error) {
	subnetId := strconv.FormatUint(subnetwork.Id, 10)
	var mqlLogConfig resources.ResourceType
	var err error
	if subnetwork.LogConfig != nil {
		mqlLogConfig, err = runtime.CreateResource("gcp.project.computeService.subnetwork.logConfig",
			"id", fmt.Sprintf("%s/logConfig", subnetId),
			"aggregationInterval", subnetwork.LogConfig.AggregationInterval,
			"enable", subnetwork.LogConfig.Enable,
			"filterExpression", subnetwork.LogConfig.FilterExpr,
			"flowSampling", subnetwork.LogConfig.FlowSampling,
			"metadata", subnetwork.LogConfig.Metadata,
			"metadataFields", core.StrSliceToInterface(subnetwork.LogConfig.MetadataFields),
		)
		if err != nil {
			return nil, err
		}
	}

	args := []interface{}{
		"id", subnetId,
		"projectId", projectId,
		"name", subnetwork.Name,
		"description", subnetwork.Description,
		"enableFlowLogs", subnetwork.EnableFlowLogs,
		"externalIpv6Prefix", subnetwork.ExternalIpv6Prefix,
		"fingerprint", subnetwork.Fingerprint,
		"gatewayAddress", subnetwork.GatewayAddress,
		"internalIpv6Prefix", subnetwork.InternalIpv6Prefix,
		"ipCidrRange", subnetwork.IpCidrRange,
		"ipv6AccessType", subnetwork.Ipv6AccessType,
		"ipv6CidrRange", subnetwork.Ipv6CidrRange,
		"logConfig", mqlLogConfig,
		"privateIpGoogleAccess", subnetwork.PrivateIpGoogleAccess,
		"privateIpv6GoogleAccess", subnetwork.PrivateIpv6GoogleAccess,
		"purpose", subnetwork.Purpose,
		"regionUrl", subnetwork.Region,
		"role", subnetwork.Role,
		"stackType", subnetwork.StackType,
		"state", subnetwork.State,
		"created", parseTime(subnetwork.CreationTimestamp),
	}
	if region != nil {
		args = append(args, "region", region)
	}
	return runtime.CreateResource("gcp.project.computeService.subnetwork", args...)
}

func (g *mqlGcpProjectComputeService) GetSubnetworks() ([]interface{}, error) {
	projectId, err := g.ProjectId()
	if err != nil {
		return nil, err
	}

	regions, err := g.Regions()
	if err != nil {
		return nil, err
	}

	provider, err := gcpProvider(g.MotorRuntime.Motor.Provider)
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
	wg.Add(len(regions))
	mux := &sync.Mutex{}

	for i := range regions {
		r := regions[i].(GcpProjectComputeServiceRegion)
		regionName, err := r.Name()
		if err != nil {
			return nil, err
		}
		go func(svc *compute.Service, project string, region GcpProjectComputeServiceRegion, regionName string) {
			req := computeSvc.Subnetworks.List(projectId, regionName)
			if err := req.Pages(ctx, func(page *compute.SubnetworkList) error {
				for _, subnetwork := range page.Items {

					mqlSubnetwork, err := newMqlSubnetwork(projectId, g.MotorRuntime, subnetwork, region)
					if err != nil {
						return err
					} else {
						// mqlInstance.MqlResource().Cache.Store("_machineType", &resources.CacheEntry{Data: instance.MachineType})
						mux.Lock()
						res = append(res, mqlSubnetwork)
						mux.Unlock()
					}
				}
				return nil
			}); err != nil {
				log.Error().Err(err).Send()
			}
			wg.Done()
		}(computeSvc, projectId, r, regionName)
	}

	wg.Wait()
	return res, nil
}

func (g *mqlGcpProjectComputeServiceRouter) id() (string, error) {
	id, err := g.Id()
	if err != nil {
		return "", nil
	}

	return "gcloud.compute.router/" + id, nil
}

func (g *mqlGcpProjectComputeServiceRouter) GetNetwork() ([]interface{}, error) {
	// TODO: implement
	return nil, errors.New("not implemented")
}

func (g *mqlGcpProjectComputeServiceRouter) GetRegion() ([]interface{}, error) {
	// TODO: implement
	return nil, errors.New("not implemented")
}

func newMqlRouter(projectId string, region GcpProjectComputeServiceRegion, runtime *resources.Runtime, router *compute.Router) (interface{}, error) {
	bgp, err := core.JsonToDict(router.Bgp)
	if err != nil {
		return nil, err
	}

	bgpPeers, err := core.JsonToDictSlice(router.BgpPeers)
	if err != nil {
		return nil, err
	}

	nats, err := core.JsonToDictSlice(router.Nats)
	if err != nil {
		return nil, err
	}

	return runtime.CreateResource("gcp.project.computeService.router",
		"id", strconv.FormatUint(router.Id, 10),
		"name", router.Name,
		"description", router.Description,
		"bgp", bgp,
		"bgpPeers", bgpPeers,
		"encryptedInterconnectRouter", router.EncryptedInterconnectRouter,
		"nats", nats,
		"created", parseTime(router.CreationTimestamp),
	)
}

func (g *mqlGcpProjectComputeService) GetRouters() ([]interface{}, error) {
	projectId, err := g.ProjectId()
	if err != nil {
		return nil, err
	}

	regions, err := g.Regions()
	if err != nil {
		return nil, err
	}

	provider, err := gcpProvider(g.MotorRuntime.Motor.Provider)
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
	wg.Add(len(regions))
	mux := &sync.Mutex{}

	for i := range regions {
		r := regions[i].(GcpProjectComputeServiceRegion)
		regionName, err := r.Name()
		if err != nil {
			return nil, err
		}
		go func(svc *compute.Service, project string, region GcpProjectComputeServiceRegion, regionName string) {
			req := computeSvc.Routers.List(projectId, regionName)
			if err := req.Pages(ctx, func(page *compute.RouterList) error {
				for _, router := range page.Items {

					mqlRouter, err := newMqlRouter(projectId, region, g.MotorRuntime, router)
					if err != nil {
						return err
					} else {
						mux.Lock()
						res = append(res, mqlRouter)
						mux.Unlock()
					}
				}
				return nil
			}); err != nil {
				log.Error().Err(err).Send()
			}
			wg.Done()
		}(computeSvc, projectId, r, regionName)
	}

	wg.Wait()
	return res, nil
}

func (g *mqlGcpProjectComputeService) GetBackendServices() ([]interface{}, error) {
	projectId, err := g.ProjectId()
	if err != nil {
		return nil, err
	}

	provider, err := gcpProvider(g.MotorRuntime.Motor.Provider)
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

	list, err := computeSvc.BackendServices.AggregatedList(projectId).Do()
	if err != nil {
		return nil, err
	}

	res := make([]interface{}, 0, len(list.Items))
	for _, sb := range list.Items {
		for _, b := range sb.BackendServices {
			backendServiceId := strconv.FormatUint(b.Id, 10)
			mqlBackends := make([]interface{}, 0, len(b.Backends))
			for i, backend := range b.Backends {
				mqlBackend, err := g.MotorRuntime.CreateResource("gcp.project.computeService.backendService.backend",
					"id", fmt.Sprintf("gcp.project.computeService.backendService.backend/%s/%d", backendServiceId, i),
					"balancingMode", backend.BalancingMode,
					"capacityScaler", backend.CapacityScaler,
					"description", backend.Description,
					"failover", backend.Failover,
					"groupUrl", backend.Group,
					"maxConnections", backend.MaxConnections,
					"maxConnectionsPerEndpoint", backend.MaxConnectionsPerEndpoint,
					"maxConnectionsPerInstance", backend.MaxConnectionsPerInstance,
					"maxRate", backend.MaxRate,
					"maxRatePerEndpoint", backend.MaxRatePerEndpoint,
					"maxRatePerInstance", backend.MaxRatePerInstance,
					"maxUtilization", backend.MaxUtilization,
				)
				if err != nil {
					return nil, err
				}
				mqlBackends = append(mqlBackends, mqlBackend)
			}

			var cdnPolicy interface{}
			if b.CdnPolicy != nil {
				bypassCacheOnRequestHeaders := make([]interface{}, 0, len(b.CdnPolicy.BypassCacheOnRequestHeaders))
				for _, h := range b.CdnPolicy.BypassCacheOnRequestHeaders {
					mqlH := map[string]interface{}{"headerName": h.HeaderName}
					bypassCacheOnRequestHeaders = append(bypassCacheOnRequestHeaders, mqlH)
				}

				var mqlCacheKeyPolicy interface{}
				if b.CdnPolicy.CacheKeyPolicy != nil {
					mqlCacheKeyPolicy = map[string]interface{}{
						"includeHost":          b.CdnPolicy.CacheKeyPolicy.IncludeHost,
						"includeHttpHeaders":   core.StrSliceToInterface(b.CdnPolicy.CacheKeyPolicy.IncludeHttpHeaders),
						"includeNamedCookies":  core.StrSliceToInterface(b.CdnPolicy.CacheKeyPolicy.IncludeNamedCookies),
						"includeProtocol":      b.CdnPolicy.CacheKeyPolicy.IncludeProtocol,
						"includeQueryString":   b.CdnPolicy.CacheKeyPolicy.IncludeQueryString,
						"queryStringBlacklist": core.StrSliceToInterface(b.CdnPolicy.CacheKeyPolicy.QueryStringBlacklist),
						"queryStringWhitelist": core.StrSliceToInterface(b.CdnPolicy.CacheKeyPolicy.QueryStringWhitelist),
					}
				}

				mqlNegativeCachingPolicy := make([]interface{}, 0, len(b.CdnPolicy.NegativeCachingPolicy))
				for _, p := range b.CdnPolicy.NegativeCachingPolicy {
					mqlP := map[string]interface{}{
						"code": p.Code,
						"ttl":  p.Ttl,
					}
					mqlNegativeCachingPolicy = append(mqlNegativeCachingPolicy, mqlP)
				}

				cdnPolicy, err = g.MotorRuntime.CreateResource("gcp.project.computeService.backendService.cdnPolicy",
					"id", fmt.Sprintf("gcp.project.computeService.backendService.cdnPolicy/%s", backendServiceId),
					"bypassCacheOnRequestHeaders", bypassCacheOnRequestHeaders,
					"cacheKeyPolicy", mqlCacheKeyPolicy,
					"cacheMode", b.CdnPolicy.CacheMode,
					"clientTtl", b.CdnPolicy.ClientTtl,
					"defaultTtl", b.CdnPolicy.DefaultTtl,
					"maxTtl", b.CdnPolicy.MaxTtl,
					"negativeCaching", b.CdnPolicy.NegativeCaching,
					"negativeCachingPolicy", mqlNegativeCachingPolicy,
					"requestCoalescing", b.CdnPolicy.RequestCoalescing,
					"serveWhileStale", b.CdnPolicy.ServeWhileStale,
					"signedUrlCacheMaxAgeSec", b.CdnPolicy.SignedUrlCacheMaxAgeSec,
					"signedUrlKeyNames", core.StrSliceToInterface(b.CdnPolicy.SignedUrlKeyNames),
				)
				if err != nil {
					return nil, err
				}
			}

			var mqlCircuitBreakers interface{}
			if b.CircuitBreakers != nil {
				mqlCircuitBreakers = map[string]interface{}{
					"maxConnections":           b.CircuitBreakers.MaxConnections,
					"maxPendingRequests":       b.CircuitBreakers.MaxPendingRequests,
					"maxRequests":              b.CircuitBreakers.MaxRequests,
					"maxRequestsPerConnection": b.CircuitBreakers.MaxRequestsPerConnection,
					"maxRetries":               b.CircuitBreakers.MaxRetries,
				}
			}

			var mqlConnectionDraining interface{}
			if b.ConnectionDraining != nil {
				mqlConnectionDraining = map[string]interface{}{
					"drainingTimeoutSec": b.ConnectionDraining.DrainingTimeoutSec,
				}
			}

			var mqlConnectionTrackingPolicy interface{}
			if b.ConnectionTrackingPolicy != nil {
				mqlConnectionTrackingPolicy = map[string]interface{}{
					"connectionPersistenceOnUnhealthyBackends": b.ConnectionTrackingPolicy.ConnectionPersistenceOnUnhealthyBackends,
					"enableStrongAffinity":                     b.ConnectionTrackingPolicy.EnableStrongAffinity,
					"idleTimeoutSec":                           b.ConnectionTrackingPolicy.IdleTimeoutSec,
					"trackingMode":                             b.ConnectionTrackingPolicy.TrackingMode,
				}
			}

			var mqlConsistentHash interface{}
			if b.ConsistentHash != nil {
				mqlConsistentHash = map[string]interface{}{
					"httpCookie": map[string]interface{}{
						"name": b.ConsistentHash.HttpCookie.Name,
						"path": b.ConsistentHash.HttpCookie.Path,
						"ttl":  core.MqlTime(llx.DurationToTime(b.ConsistentHash.HttpCookie.Ttl.Seconds)),
					},
					"httpHeaderName":  b.ConsistentHash.HttpHeaderName,
					"minimumRingSize": b.ConsistentHash.MinimumRingSize,
				}
			}

			var mqlFailoverPolicy interface{}
			if b.FailoverPolicy != nil {
				mqlFailoverPolicy = map[string]interface{}{
					"disableConnectionDrainOnFailover": b.FailoverPolicy.DisableConnectionDrainOnFailover,
					"dropTrafficIfUnhealthy":           b.FailoverPolicy.DropTrafficIfUnhealthy,
					"failoverRatio":                    b.FailoverPolicy.FailoverRatio,
				}
			}

			var mqlIap interface{}
			if b.Iap != nil {
				mqlIap = map[string]interface{}{
					"enabled":                  b.Iap.Enabled,
					"oauth2ClientId":           b.Iap.Oauth2ClientId,
					"oauth2ClientSecret":       b.Iap.Oauth2ClientSecret,
					"oauth2ClientSecretSha256": b.Iap.Oauth2ClientSecretSha256,
				}
			}

			mqlLocalityLbPolicy := make([]interface{}, 0, len(b.LocalityLbPolicies))
			for _, p := range b.LocalityLbPolicies {
				var mqlCustomPolicy interface{}
				if p.CustomPolicy != nil {
					mqlCustomPolicy = map[string]interface{}{
						"data": p.CustomPolicy.Data,
						"name": p.CustomPolicy.Name,
					}
				}

				var mqlPolicy interface{}
				if p.Policy != nil {
					mqlPolicy = map[string]interface{}{
						"name": p.Policy.Name,
					}
				}
				mqlLocalityLbPolicy = append(mqlLocalityLbPolicy, map[string]interface{}{
					"customPolicy": mqlCustomPolicy,
					"policy":       mqlPolicy,
				})
			}

			var mqlLogConfig interface{}
			if b.LogConfig != nil {
				mqlLogConfig = map[string]interface{}{
					"enable":     b.LogConfig.Enable,
					"sampleRate": b.LogConfig.SampleRate,
				}
			}

			var mqlSecuritySettings interface{}
			if b.SecuritySettings != nil {
				mqlSecuritySettings = map[string]interface{}{
					"clientTlsPolicy": b.SecuritySettings.ClientTlsPolicy,
					"subjectAltNames": core.StrSliceToInterface(b.SecuritySettings.SubjectAltNames),
				}
			}

			var maxStreamDuration interface{}
			if b.MaxStreamDuration != nil {
				maxStreamDuration = core.MqlTime(llx.DurationToTime(b.MaxStreamDuration.Seconds))
			}

			mqlB, err := g.MotorRuntime.CreateResource("gcp.project.computeService.backendService",
				"id", backendServiceId,
				"affinityCookieTtlSec", b.AffinityCookieTtlSec,
				"backends", mqlBackends,
				"cdnPolicy", cdnPolicy,
				"circuitBreakers", mqlCircuitBreakers,
				"compressionMode", b.CompressionMode,
				"connectionDraining", mqlConnectionDraining,
				"connectionTrackingPolicy", mqlConnectionTrackingPolicy,
				"consistentHash", mqlConsistentHash,
				"created", parseTime(b.CreationTimestamp),
				"customRequestHeaders", core.StrSliceToInterface(b.CustomRequestHeaders),
				"customResponseHeaders", core.StrSliceToInterface(b.CustomResponseHeaders),
				"description", b.Description,
				"edgeSecurityPolicy", b.EdgeSecurityPolicy,
				"enableCDN", b.EnableCDN,
				"failoverPolicy", mqlFailoverPolicy,
				"healthChecks", core.StrSliceToInterface(b.HealthChecks),
				"iap", mqlIap,
				"loadBalancingScheme", b.LoadBalancingScheme,
				"localityLbPolicies", mqlLocalityLbPolicy,
				"localityLbPolicy", b.LocalityLbPolicy,
				"logConfig", mqlLogConfig,
				"maxStreamDuration", maxStreamDuration,
				"name", b.Name,
				"networkUrl", b.Network,
				"portName", b.PortName,
				"protocol", b.Protocol,
				"regionUrl", b.Region,
				"securityPolicyUrl", b.SecurityPolicy,
				"securitySettings", mqlSecuritySettings,
				"serviceBindingUrls", core.StrSliceToInterface(b.ServiceBindings),
				"sessionAffinity", b.SessionAffinity,
				"timeoutSec", b.TimeoutSec,
			)
			if err != nil {
				return nil, err
			}
			res = append(res, mqlB)
		}
	}
	return res, nil
}

func (g *mqlGcpProjectComputeServiceBackendService) id() (string, error) {
	id, err := g.Id()
	if err != nil {
		return "", nil
	}
	return "gcp.project.computeService.backendService/" + id, nil
}

func (g *mqlGcpProjectComputeServiceBackendServiceBackend) id() (string, error) {
	return g.Id()
}

func (g *mqlGcpProjectComputeServiceBackendServiceCdnPolicy) id() (string, error) {
	return g.Id()
}

func networkMode(n *compute.Network) string {
	if n.IPv4Range != "" {
		return "legacy"
	} else if n.AutoCreateSubnetworks {
		return "auto"
	} else {
		return "custom"
	}
}
