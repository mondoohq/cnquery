package gcp

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"sync"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/resources/packs/core"
	"google.golang.org/api/cloudresourcemanager/v3"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/iam/v1"
	"google.golang.org/api/option"
)

func (g *mqlGcpCompute) init(args *resources.Args) (*resources.Args, GcpCompute, error) {
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

func (g *mqlGcpCompute) id() (string, error) {
	id, err := g.ProjectId()
	if err != nil {
		return "", err
	}
	return "gcp.compute/" + id, nil
}

func (g *mqlGcpComputeRegion) id() (string, error) {
	id, err := g.Name()
	if err != nil {
		return "", err
	}
	return "gcp.compute.region/" + id, nil
}

func (g *mqlGcpComputeRegion) init(args *resources.Args) (*resources.Args, GcpComputeRegion, error) {
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

func (g *mqlGcpCompute) GetRegions() ([]interface{}, error) {
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
	req := computeSvc.Regions.List(projectId)
	if err := req.Pages(ctx, func(page *compute.RegionList) error {
		for i := range page.Items {
			r := page.Items[i]

			deprecated, err := core.JsonToDict(r.Deprecated)
			if err != nil {
				return err
			}

			quotas := map[string]interface{}{}
			for i := range r.Quotas {
				q := r.Quotas[i]
				quotas[q.Metric] = q.Limit
			}

			mqlRegion, err := g.MotorRuntime.CreateResource("gcp.compute.region",
				"id", strconv.FormatInt(int64(r.Id), 10),
				"name", r.Name,
				"description", r.Description,
				"status", r.Status,
				"created", parseTime(r.CreationTimestamp),
				"quotas", quotas,
				"deprecated", deprecated,
			)
			if err != nil {
				return err
			}
			res = append(res, mqlRegion)
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return res, nil
}

func (g *mqlGcpComputeZone) id() (string, error) {
	id, err := g.Id()
	if err != nil {
		return "", err
	}
	return "gcp.compute.zone/" + id, nil
}

func (g *mqlGcpComputeZone) GetRegion() (interface{}, error) {
	// TODO: implement
	return nil, errors.New("not implemented")
}

func (g *mqlGcpCompute) GetZones() ([]interface{}, error) {
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
			mqlZone, err := g.MotorRuntime.CreateResource("gcp.compute.zone",
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

func (g *mqlGcpComputeMachineType) id() (string, error) {
	id, err := g.Id()
	if err != nil {
		return "", err
	}

	projectId, err := g.ProjectId()
	if err != nil {
		return "", err
	}

	return "gcp.compute.machineType/" + projectId + "/" + id, nil
}

func (g *mqlGcpComputeMachineType) GetZone() (interface{}, error) {
	// NOTE: this should never be called since we add the zone during construction of the resource
	return nil, errors.New("not implemented")
}

func newMqlMachineType(runtime *resources.Runtime, entry *compute.MachineType, projectId string, zone GcpComputeZone) (interface{}, error) {
	return runtime.CreateResource("gcp.compute.machineType",
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

func (g *mqlGcpCompute) GetMachineTypes() ([]interface{}, error) {
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
		z := zones[i].(GcpComputeZone)
		zoneName, err := z.Name()
		if err != nil {
			return nil, err
		}

		go func(svc *compute.Service, projectId string, zone GcpComputeZone, zoneName string) {
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

func (g *mqlGcpComputeInstance) id() (string, error) {
	id, err := g.Id()
	if err != nil {
		return "", err
	}

	projectId, err := g.ProjectId()
	if err != nil {
		return "", err
	}

	return "gcp.compute.instance/" + projectId + "/" + id, nil
}

func (g *mqlGcpComputeInstance) GetMachineType() (interface{}, error) {
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
	return runtime.CreateResource("gcp.compute.serviceaccount",
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

	mqlAttachedDisk, err := runtime.CreateResource("gcp.compute.attachedDisk",
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

func (g *mqlGcpComputeAttachedDisk) id() (string, error) {
	id, err := g.Id()
	if err != nil {
		return "", err
	}

	projectId, err := g.ProjectId()
	if err != nil {
		return "", err
	}

	return "gcp.compute.attachedDisk/" + projectId + "/" + id, nil
}

func (g *mqlGcpComputeAttachedDisk) GetSource() (interface{}, error) {
	entry, ok := g.MqlResource().Cache.Load("_attachedDiskSource")
	if !ok {
		return nil, errors.New("could not find persistent disk")
	}

	sourceDisk := entry.Data.(string)
	log.Info().Msg(sourceDisk)

	return nil, nil
}

func newMqlInstance(projectId string, zone GcpComputeZone, runtime *resources.Runtime, instance *compute.Instance) (GcpComputeInstance, error) {
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

	entry, err := runtime.CreateResource("gcp.compute.instance",
		"id", instanceId,
		"projectId", projectId,
		"name", instance.Name,
		"cpuPlatform", instance.CpuPlatform,
		"description", instance.Description,
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
	return entry.(GcpComputeInstance), nil
}

func (g *mqlGcpCompute) GetInstances() ([]interface{}, error) {
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
		z := zones[i].(GcpComputeZone)
		zoneName, err := z.Name()
		if err != nil {
			return nil, err
		}
		go func(svc *compute.Service, project string, zone GcpComputeZone, zoneName string) {
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

func (g *mqlGcpComputeServiceaccount) id() (string, error) {
	email, err := g.Email()
	if err != nil {
		return "", nil
	}
	return "gcp.compute.serviceaccount/" + email, nil
}

func (g *mqlGcpComputeDisk) id() (string, error) {
	id, err := g.Id()
	if err != nil {
		return "", nil
	}

	return "gcloud.compute.disk/" + id, nil
}

func (g *mqlGcpComputeDisk) GetZone() (interface{}, error) {
	// TODO: implement
	return nil, errors.New("not implemented")
}

func (g *mqlGcpCompute) GetDisks() ([]interface{}, error) {
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
		z := zones[i].(GcpComputeZone)
		zoneName, err := z.Name()
		if err != nil {
			return nil, err
		}

		go func(svc *compute.Service, project string, zone GcpComputeZone, zoneName string) {
			req := computeSvc.Disks.List(projectId, zoneName)
			if err := req.Pages(ctx, func(page *compute.DiskList) error {
				for _, disk := range page.Items {
					guestOsFeatures := []string{}
					for i := range disk.GuestOsFeatures {
						entry := disk.GuestOsFeatures[i]
						guestOsFeatures = append(guestOsFeatures, entry.Type)
					}

					mqlDisk, err := g.MotorRuntime.CreateResource("gcp.compute.disk",
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

func (g *mqlGcpComputeFirewall) id() (string, error) {
	id, err := g.Id()
	if err != nil {
		return "", nil
	}

	return "gcloud.compute.firewall/" + id, nil
}

func (g *mqlGcpComputeFirewall) GetNetwork() (interface{}, error) {
	// TODO: implement
	return nil, errors.New("not implemented")
}

func (g *mqlGcpCompute) GetFirewalls() ([]interface{}, error) {
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

			mqlFirewall, err := g.MotorRuntime.CreateResource("gcp.compute.firewall",
				"id", strconv.FormatUint(firewall.Id, 10),
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

func (g *mqlGcpComputeSnapshot) id() (string, error) {
	id, err := g.Id()
	if err != nil {
		return "", nil
	}

	return "gcloud.compute.snapshot/" + id, nil
}

func (g *mqlGcpComputeSnapshot) GetSourceDisk() (interface{}, error) {
	// TODO: implement
	return nil, errors.New("not implemented")
}

func (g *mqlGcpCompute) GetSnapshots() ([]interface{}, error) {
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
			mqlSnapshpt, err := g.MotorRuntime.CreateResource("gcp.compute.snapshot",
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

func (g *mqlGcpComputeImage) id() (string, error) {
	id, err := g.Id()
	if err != nil {
		return "", nil
	}

	return "gcloud.compute.image/" + id, nil
}

func (g *mqlGcpComputeImage) GetSourceDisk() (interface{}, error) {
	// TODO: implement
	return nil, errors.New("not implemented")
}

func (g *mqlGcpCompute) GetImages() ([]interface{}, error) {
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
			mqlImage, err := g.MotorRuntime.CreateResource("gcp.compute.image",
				"id", strconv.FormatUint(image.Id, 10),
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

func (g *mqlGcpComputeNetwork) id() (string, error) {
	id, err := g.Id()
	if err != nil {
		return "", nil
	}

	return "gcloud.compute.network/" + id, nil
}

func (g *mqlGcpComputeNetwork) GetSubnetworks() ([]interface{}, error) {
	// TODO: implement
	return nil, errors.New("not implemented")
}

func (g *mqlGcpCompute) GetNetworks() ([]interface{}, error) {
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

			mqlNetwork, err := g.MotorRuntime.CreateResource("gcp.compute.network",
				"id", strconv.FormatUint(network.Id, 10),
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

func (g *mqlGcpComputeSubnetwork) id() (string, error) {
	id, err := g.Id()
	if err != nil {
		return "", nil
	}

	return "gcloud.compute.subnetwork/" + id, nil
}

func (g *mqlGcpComputeSubnetwork) GetNetwork() ([]interface{}, error) {
	// TODO: implement
	return nil, errors.New("not implemented")
}

func newMqlSubnetwork(projectId string, region GcpComputeRegion, runtime *resources.Runtime, subnetwork *compute.Subnetwork) (interface{}, error) {
	return runtime.CreateResource("gcp.compute.subnetwork",
		"id", strconv.FormatUint(subnetwork.Id, 10),
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
		"privateIpGoogleAccess", subnetwork.PrivateIpGoogleAccess,
		"privateIpv6GoogleAccess", subnetwork.PrivateIpv6GoogleAccess,
		"purpose", subnetwork.Purpose,
		"region", region,
		"role", subnetwork.Role,
		"stackType", subnetwork.StackType,
		"state", subnetwork.State,
		"created", parseTime(subnetwork.CreationTimestamp),
	)
}

func (g *mqlGcpCompute) GetSubnetworks() ([]interface{}, error) {
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
		r := regions[i].(GcpComputeRegion)
		regionName, err := r.Name()
		if err != nil {
			return nil, err
		}
		go func(svc *compute.Service, project string, region GcpComputeRegion, regionName string) {
			req := computeSvc.Subnetworks.List(projectId, regionName)
			if err := req.Pages(ctx, func(page *compute.SubnetworkList) error {
				for _, subnetwork := range page.Items {

					mqlSubnetwork, err := newMqlSubnetwork(projectId, region, g.MotorRuntime, subnetwork)
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

func (g *mqlGcpComputeRouter) id() (string, error) {
	id, err := g.Id()
	if err != nil {
		return "", nil
	}

	return "gcloud.compute.router/" + id, nil
}

func (g *mqlGcpComputeRouter) GetNetwork() ([]interface{}, error) {
	// TODO: implement
	return nil, errors.New("not implemented")
}

func (g *mqlGcpComputeRouter) GetRegion() ([]interface{}, error) {
	// TODO: implement
	return nil, errors.New("not implemented")
}

func newMqlRouter(projectId string, region GcpComputeRegion, runtime *resources.Runtime, router *compute.Router) (interface{}, error) {
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

	return runtime.CreateResource("gcp.compute.router",
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

func (g *mqlGcpCompute) GetRouters() ([]interface{}, error) {
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
		r := regions[i].(GcpComputeRegion)
		regionName, err := r.Name()
		if err != nil {
			return nil, err
		}
		go func(svc *compute.Service, project string, region GcpComputeRegion, regionName string) {
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
