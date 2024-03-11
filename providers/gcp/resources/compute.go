// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/aws/smithy-go/ptr"

	computev1 "cloud.google.com/go/compute/apiv1"
	"cloud.google.com/go/compute/apiv1/computepb"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v10/llx"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v10/providers/gcp/connection"
	"go.mondoo.com/cnquery/v10/types"
	"google.golang.org/api/cloudresourcemanager/v3"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/iam/v1"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

func initGcpProjectComputeService(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}

	conn := runtime.Connection.(*connection.GcpConnection)

	projectId := conn.ResourceID()
	args["projectId"] = llx.StringData(projectId)

	return args, nil, nil
}

func (g *mqlGcpProject) compute() (*mqlGcpProjectComputeService, error) {
	if g.Id.Error != nil {
		return nil, g.Id.Error
	}
	projectId := g.Id.Data

	res, err := CreateResource(g.MqlRuntime, "gcp.project.computeService", map[string]*llx.RawData{
		"projectId": llx.StringData(projectId),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlGcpProjectComputeService), nil
}

func (g *mqlGcpProjectComputeService) id() (string, error) {
	if g.ProjectId.Error != nil {
		return "", g.ProjectId.Error
	}
	projectId := g.ProjectId.Data
	return fmt.Sprintf("%s/gcp.project.computeService", projectId), nil
}

func (g *mqlGcpProjectComputeServiceRegion) id() (string, error) {
	if g.Name.Error != nil {
		return "", g.Name.Error
	}
	id := g.Name.Data
	return "gcp.project.computeService.region/" + id, nil
}

func initGcpProjectComputeServiceRegion(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}

	conn := runtime.Connection.(*connection.GcpConnection)

	projectId := conn.ResourceID()
	args["projectId"] = llx.StringData(projectId)

	return args, nil, nil
}

func (g *mqlGcpProjectComputeService) regions() ([]interface{}, error) {
	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)

	client, err := conn.Client(cloudresourcemanager.CloudPlatformReadOnlyScope, iam.CloudPlatformScope, compute.CloudPlatformScope)
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
	if g.Id.Error != nil {
		return "", g.Id.Error
	}
	id := g.Id.Data
	return "gcp.project.computeService.zone/" + id, nil
}

func (g *mqlGcpProjectComputeServiceZone) region() (interface{}, error) {
	// TODO: implement
	return nil, errors.New("not implemented")
}

func (g *mqlGcpProjectComputeService) zones() ([]interface{}, error) {
	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)

	client, err := conn.Client(cloudresourcemanager.CloudPlatformReadOnlyScope)
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
	if g.Id.Error != nil {
		return "", g.Id.Error
	}
	id := g.Id.Data

	if g.ProjectId.Error != nil {
		return "", g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	return "gcp.project.computeService.machineType/" + projectId + "/" + id, nil
}

func (g *mqlGcpProjectComputeServiceMachineType) zone() (interface{}, error) {
	// NOTE: this should never be called since we add the zone during construction of the resource
	return nil, errors.New("not implemented")
}

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

func (g *mqlGcpProjectComputeService) machineTypes() ([]interface{}, error) {
	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	// get list of zones first since we need this for all entries
	zones := g.GetZones()
	if zones.Error != nil {
		return nil, zones.Error
	}

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)

	client, err := conn.Client(cloudresourcemanager.CloudPlatformReadOnlyScope, iam.CloudPlatformScope, compute.CloudPlatformScope)
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
	wg.Add(len(zones.Data))
	mux := &sync.Mutex{}

	for i := range zones.Data {
		z := zones.Data[i].(*mqlGcpProjectComputeServiceZone)
		zoneName := z.GetName()
		if zoneName.Error != nil {
			return nil, zoneName.Error
		}

		go func(svc *compute.Service, projectId string, zone *mqlGcpProjectComputeServiceZone, zoneName string) {
			req := computeSvc.MachineTypes.List(projectId, zoneName)
			if err := req.Pages(ctx, func(page *compute.MachineTypeList) error {
				for _, machinetype := range page.Items {
					mqlMachineType, err := newMqlMachineType(g.MqlRuntime, machinetype, projectId, zone)
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
		}(computeSvc, projectId, z, zoneName.Data)
	}
	wg.Wait()
	return res, nil
}

func initGcpProjectComputeServiceInstance(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 3 {
		return args, nil, nil
	}

	// If no args are set, try reading them from the platform ID
	if len(args) == 0 {
		if ids := getAssetIdentifier(runtime); ids != nil {
			args["name"] = llx.StringData(ids.name)
			args["region"] = llx.StringData(ids.region)
			args["projectId"] = llx.StringData(ids.project)
		} else {
			return nil, nil, errors.New("no asset identifier found")
		}
	}

	obj, err := CreateResource(runtime, "gcp.project.computeService", map[string]*llx.RawData{
		"projectId": args["projectId"],
	})
	if err != nil {
		return nil, nil, err
	}
	computeSvc := obj.(*mqlGcpProjectComputeService)
	instances := computeSvc.GetInstances()
	if instances.Error != nil {
		return nil, nil, instances.Error
	}

	for _, inst := range instances.Data {
		instance := inst.(*mqlGcpProjectComputeServiceInstance)
		name := instance.GetName()
		if name.Error != nil {
			return nil, nil, name.Error
		}
		projectId := instance.GetProjectId()
		if projectId.Error != nil {
			return nil, nil, projectId.Error
		}
		instanceZone := instance.GetZone()
		if instanceZone.Error != nil {
			return nil, nil, instanceZone.Error
		}

		if instanceZone.Data.Name.Data == args["region"].Value && name.Data == args["name"].Value && projectId.Data == args["projectId"].Value {
			return args, instance, nil
		}
	}
	return nil, nil, nil
}

func (g *mqlGcpProjectComputeServiceInstance) id() (string, error) {
	if g.Id.Error != nil {
		return "", g.Id.Error
	}
	id := g.Id.Data

	if g.ProjectId.Error != nil {
		return "", g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	return "gcp.project.computeService.instance/" + projectId + "/" + id, nil
}

func (g *mqlGcpProjectComputeServiceInstance) machineType() (*mqlGcpProjectComputeServiceMachineType, error) {
	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	zone := g.GetZone()
	if zone.Error != nil {
		return nil, zone.Error
	}

	zoneName := zone.Data.GetName()
	if zoneName.Error != nil {
		return nil, zoneName.Error
	}

	machineTypeUrl := g.instanceMachineType
	values := strings.Split(machineTypeUrl, "/")
	machineTypeValue := values[len(values)-1]

	// TODO: we can save calls if we move it to the into method
	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)

	client, err := conn.Client(cloudresourcemanager.CloudPlatformReadOnlyScope, iam.CloudPlatformScope, compute.CloudPlatformScope)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	computeSvc, err := compute.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}

	machineType, err := computeSvc.MachineTypes.Get(projectId, zoneName.Data, machineTypeValue).Do()
	if err != nil {
		return nil, err
	}

	return newMqlMachineType(g.MqlRuntime, machineType, projectId, zone.Data)
}

func newMqlServiceAccount(runtime *plugin.Runtime, sa *compute.ServiceAccount) (interface{}, error) {
	return CreateResource(runtime, "gcp.project.computeService.serviceaccount", map[string]*llx.RawData{
		"email":  llx.StringData(sa.Email),
		"scopes": llx.ArrayData(convert.SliceAnyToInterface(sa.Scopes), types.String),
	})
}

type mqlGcpProjectComputeServiceAttachedDiskInternal struct {
	attachedDiskSource string
}

func newMqlAttachedDisk(id string, projectId string, runtime *plugin.Runtime, attachedDisk *compute.AttachedDisk) (*mqlGcpProjectComputeServiceAttachedDisk, error) {
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
	r := mqlAttachedDisk.(*mqlGcpProjectComputeServiceAttachedDisk)
	r.attachedDiskSource = attachedDisk.Source
	return r, nil
}

func (g *mqlGcpProjectComputeServiceAttachedDisk) id() (string, error) {
	if g.Id.Error != nil {
		return "", g.Id.Error
	}
	id := g.Id.Data

	if g.ProjectId.Error != nil {
		return "", g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	return "gcp.project.computeService.attachedDisk/" + projectId + "/" + id, nil
}

func (g *mqlGcpProjectComputeServiceAttachedDisk) source() (*mqlGcpProjectComputeServiceDisk, error) {
	// g.attachedDiskSource
	// TODO search for reference resource
	return nil, nil
}

type mqlGcpProjectComputeServiceInstanceInternal struct {
	instanceMachineType string
}

func newMqlComputeServiceInstance(projectId string, zone *mqlGcpProjectComputeServiceZone, runtime *plugin.Runtime, instance *compute.Instance) (*mqlGcpProjectComputeServiceInstance, error) {
	metadata := map[string]string{}
	for m := range instance.Metadata.Items {
		item := instance.Metadata.Items[m]

		value := ""
		if item.Value != nil {
			value = *item.Value
		}
		metadata[item.Key] = value
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
	mqlR := entry.(*mqlGcpProjectComputeServiceInstance)
	mqlR.instanceMachineType = instance.MachineType
	return mqlR, nil
}

func (g *mqlGcpProjectComputeService) instances() ([]interface{}, error) {
	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	// get list of zones first since we need this for all entries

	zones := g.GetZones()
	if zones.Error != nil {
		return nil, zones.Error
	}

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)

	client, err := conn.Client(cloudresourcemanager.CloudPlatformReadOnlyScope, iam.CloudPlatformScope, compute.CloudPlatformScope)
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
	wg.Add(len(zones.Data))
	mux := &sync.Mutex{}

	for i := range zones.Data {
		z := zones.Data[i].(*mqlGcpProjectComputeServiceZone)
		zoneName := z.GetName()
		if zoneName.Error != nil {
			return nil, zoneName.Error
		}
		go func(svc *compute.Service, project string, zone *mqlGcpProjectComputeServiceZone, zoneName string) {
			req := computeSvc.Instances.List(projectId, zoneName)
			if err := req.Pages(ctx, func(page *compute.InstanceList) error {
				for _, instance := range page.Items {

					mqlInstance, err := newMqlComputeServiceInstance(projectId, zone, g.MqlRuntime, instance)
					if err != nil {
						return err
					} else {
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
		}(computeSvc, projectId, z, zoneName.Data)
	}

	wg.Wait()
	return res, nil
}

func (g *mqlGcpProjectComputeServiceServiceaccount) id() (string, error) {
	if g.Email.Error != nil {
		return "", g.Email.Error
	}
	email := g.Email.Data
	return "gcp.project.computeService.serviceaccount/" + email, nil
}

func (g *mqlGcpProjectComputeServiceDisk) id() (string, error) {
	if g.Id.Error != nil {
		return "", g.Id.Error
	}
	id := g.Id.Data
	return "gcloud.compute.disk/" + id, nil
}

func (g *mqlGcpProjectComputeServiceDisk) zone() (interface{}, error) {
	// TODO: implement
	return nil, errors.New("not implemented")
}

func (g *mqlGcpProjectComputeService) disks() ([]interface{}, error) {
	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	// get list of zones first since we need this for all entries
	zones := g.GetZones()
	if zones.Error != nil {
		return nil, zones.Error
	}

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)

	client, err := conn.Client(cloudresourcemanager.CloudPlatformReadOnlyScope, iam.CloudPlatformScope, compute.CloudPlatformScope)
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
	wg.Add(len(zones.Data))
	mux := &sync.Mutex{}

	var result error
	for i := range zones.Data {
		z := zones.Data[i].(*mqlGcpProjectComputeServiceZone)
		zoneName := z.GetName()
		if zoneName.Error != nil {
			return nil, zoneName.Error
		}

		go func(svc *compute.Service, project string, zone *mqlGcpProjectComputeServiceZone, zoneName string) {
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

					mqlDisk, err := CreateResource(g.MqlRuntime, "gcp.project.computeService.disk", map[string]*llx.RawData{
						"id":                     llx.StringData(strconv.FormatUint(disk.Id, 10)),
						"name":                   llx.StringData(disk.Name),
						"architecture":           llx.StringData(disk.Architecture),
						"description":            llx.StringData(disk.Description),
						"guestOsFeatures":        llx.ArrayData(convert.SliceAnyToInterface(guestOsFeatures), types.String),
						"labels":                 llx.MapData(convert.MapToInterfaceMap(disk.Labels), types.String),
						"lastAttachTimestamp":    llx.TimeDataPtr(parseTime(disk.LastAttachTimestamp)),
						"lastDetachTimestamp":    llx.TimeDataPtr(parseTime(disk.LastDetachTimestamp)),
						"locationHint":           llx.StringData(disk.LocationHint),
						"licenses":               llx.ArrayData(convert.SliceAnyToInterface(disk.Licenses), types.String),
						"physicalBlockSizeBytes": llx.IntData(disk.PhysicalBlockSizeBytes),
						"provisionedIops":        llx.IntData(disk.ProvisionedIops),
						// TODO: link to resources
						//"region": llx.StringData(disk.Region),
						//"replicaZones": llx.StringData(convert.SliceAnyToInterface(disk.ReplicaZones)),
						//"resourcePolicies": llx.StringData(convert.SliceAnyToInterface(disk.ResourcePolicies)),
						"sizeGb": llx.IntData(disk.SizeGb),
						// TODO: link to resources
						//"sourceDiskId": llx.StringData(disk.SourceDiskId),
						//"sourceImageId": llx.StringData(disk.SourceImageId),
						//"sourceSnapshotId": llx.StringData(disk.SourceSnapshotId),
						"status":            llx.StringData(disk.Status),
						"zone":              llx.ResourceData(zone, "gcp.project.computeService.zone"),
						"created":           llx.TimeDataPtr(parseTime(disk.CreationTimestamp)),
						"diskEncryptionKey": llx.DictData(mqlDiskEnc),
					})
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
		}(computeSvc, projectId, z, zoneName.Data)
	}

	wg.Wait()
	return res, result
}

func (g *mqlGcpProjectComputeServiceFirewall) id() (string, error) {
	if g.Id.Error != nil {
		return "", g.Id.Error
	}
	id := g.Id.Data
	return "gcloud.compute.firewall/" + id, nil
}

func (g *mqlGcpProjectComputeServiceFirewall) network() (interface{}, error) {
	// TODO: implement
	return nil, errors.New("not implemented")
}

func initGcpProjectComputeServiceFirewall(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}

	// If no args are set, try reading them from the platform ID
	if len(args) == 0 {
		if ids := getAssetIdentifier(runtime); ids != nil {
			args["name"] = llx.StringData(ids.name)
			args["projectId"] = llx.StringData(ids.project)
		} else {
			return nil, nil, errors.New("no asset identifier found")
		}
	}

	obj, err := CreateResource(runtime, "gcp.project.computeService", map[string]*llx.RawData{
		"projectId": llx.StringData(args["projectId"].Value.(string)),
	})
	if err != nil {
		return nil, nil, err
	}
	computeSvc := obj.(*mqlGcpProjectComputeService)
	firewalls := computeSvc.GetFirewalls()
	if firewalls.Error != nil {
		return nil, nil, firewalls.Error
	}

	for _, f := range firewalls.Data {
		firewall := f.(*mqlGcpProjectComputeServiceFirewall)
		name := firewall.GetName()
		if name.Error != nil {
			return nil, nil, name.Error
		}
		projectId := firewall.GetProjectId()
		if projectId.Error != nil {
			return nil, nil, projectId.Error
		}

		if name.Data == args["name"].Value && projectId.Data == args["projectId"].Value {
			return args, firewall, nil
		}
	}
	return nil, nil, nil
}

func (g *mqlGcpProjectComputeService) firewalls() ([]interface{}, error) {
	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)

	client, err := conn.Client(cloudresourcemanager.CloudPlatformReadOnlyScope, iam.CloudPlatformScope, compute.CloudPlatformScope)
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
			allowedDict, err := convert.JsonToDictSlice(allowed)
			if err != nil {
				return err
			}

			denied := make([]mqlFirewall, 0, len(firewall.Denied))
			for _, d := range firewall.Denied {
				denied = append(denied, mqlFirewall{IpProtocol: d.IPProtocol, Ports: d.Ports})
			}
			deniedDict, err := convert.JsonToDictSlice(denied)
			if err != nil {
				return err
			}

			mqlFirewall, err := CreateResource(g.MqlRuntime, "gcp.project.computeService.firewall", map[string]*llx.RawData{
				"id":                    llx.StringData(strconv.FormatUint(firewall.Id, 10)),
				"projectId":             llx.StringData(projectId),
				"name":                  llx.StringData(firewall.Name),
				"description":           llx.StringData(firewall.Description),
				"priority":              llx.IntData(firewall.Priority),
				"disabled":              llx.BoolData(firewall.Disabled),
				"direction":             llx.StringData(firewall.Direction),
				"sourceRanges":          llx.ArrayData(convert.SliceAnyToInterface(firewall.SourceRanges), types.String),
				"sourceServiceAccounts": llx.ArrayData(convert.SliceAnyToInterface(firewall.SourceServiceAccounts), types.String),
				"sourceTags":            llx.ArrayData(convert.SliceAnyToInterface(firewall.SourceTags), types.String),
				"destinationRanges":     llx.ArrayData(convert.SliceAnyToInterface(firewall.DestinationRanges), types.String),
				"targetServiceAccounts": llx.ArrayData(convert.SliceAnyToInterface(firewall.TargetServiceAccounts), types.String),
				"created":               llx.TimeDataPtr(parseTime(firewall.CreationTimestamp)),
				"allowed":               llx.ArrayData(allowedDict, types.Dict),
				"denied":                llx.ArrayData(deniedDict, types.Dict),
			})
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
	if g.Id.Error != nil {
		return "", g.Id.Error
	}
	id := g.Id.Data
	return "gcloud.compute.snapshot/" + id, nil
}

func (g *mqlGcpProjectComputeServiceSnapshot) sourceDisk() (interface{}, error) {
	// TODO: implement
	return nil, errors.New("not implemented")
}

func (g *mqlGcpProjectComputeService) snapshots() ([]interface{}, error) {
	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)

	client, err := conn.Client(cloudresourcemanager.CloudPlatformReadOnlyScope, iam.CloudPlatformScope, compute.CloudPlatformScope)
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
			mqlSnapshpt, err := CreateResource(g.MqlRuntime, "gcp.project.computeService.snapshot", map[string]*llx.RawData{
				"id":                 llx.StringData(strconv.FormatUint(snapshot.Id, 10)),
				"name":               llx.StringData(snapshot.Name),
				"description":        llx.StringData(snapshot.Description),
				"architecture":       llx.StringData(snapshot.Architecture),
				"autoCreated":        llx.BoolData(snapshot.AutoCreated),
				"chainName":          llx.StringData(snapshot.ChainName),
				"creationSizeBytes":  llx.IntData(snapshot.CreationSizeBytes),
				"diskSizeGb":         llx.IntData(snapshot.DiskSizeGb),
				"downloadBytes":      llx.IntData(snapshot.DownloadBytes),
				"storageBytes":       llx.IntData(snapshot.StorageBytes),
				"storageBytesStatus": llx.StringData(snapshot.StorageBytesStatus),
				"snapshotType":       llx.StringData(snapshot.SnapshotType),
				"licenses":           llx.ArrayData(convert.SliceAnyToInterface(snapshot.Licenses), types.String),
				"labels":             llx.MapData(convert.MapToInterfaceMap(snapshot.Labels), types.String),
				"status":             llx.StringData(snapshot.Status),
				"created":            llx.TimeDataPtr(parseTime(snapshot.CreationTimestamp)),
			})
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
	if g.Id.Error != nil {
		return "", g.Id.Error
	}
	id := g.Id.Data
	return "gcloud.compute.image/" + id, nil
}

func initGcpProjectComputeServiceImage(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}

	// If no args are set, try reading them from the platform ID
	if len(args) == 0 {
		if ids := getAssetIdentifier(runtime); ids != nil {
			args["name"] = llx.StringData(ids.name)
			args["projectId"] = llx.StringData(ids.project)
		} else {
			return nil, nil, errors.New("no asset identifier found")
		}
	}

	obj, err := CreateResource(runtime, "gcp.project.computeService", map[string]*llx.RawData{
		"projectId": llx.StringData(args["projectId"].Value.(string)),
	})
	if err != nil {
		return nil, nil, err
	}
	computeSvc := obj.(*mqlGcpProjectComputeService)
	images := computeSvc.GetImages()
	if images.Error != nil {
		return nil, nil, images.Error
	}

	for _, i := range images.Data {
		image := i.(*mqlGcpProjectComputeServiceImage)
		if image.Name.Error != nil {
			return nil, nil, image.Name.Error
		}
		name := image.Name.Data
		projectId := image.GetProjectId()
		if projectId.Error != nil {
			return nil, nil, projectId.Error
		}

		if name == args["name"].Value && projectId.Data == args["projectId"].Value {
			return args, image, nil
		}
	}
	return nil, nil, nil
}

func (g *mqlGcpProjectComputeServiceImage) sourceDisk() (interface{}, error) {
	// TODO: implement
	return nil, errors.New("not implemented")
}

func (g *mqlGcpProjectComputeService) images() ([]interface{}, error) {
	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)

	client, err := conn.Client(cloudresourcemanager.CloudPlatformReadOnlyScope, iam.CloudPlatformScope, compute.CloudPlatformScope)
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
			mqlImage, err := CreateResource(g.MqlRuntime, "gcp.project.computeService.image", map[string]*llx.RawData{
				"id":               llx.StringData(strconv.FormatUint(image.Id, 10)),
				"projectId":        llx.StringData(projectId),
				"name":             llx.StringData(image.Name),
				"description":      llx.StringData(image.Description),
				"architecture":     llx.StringData(image.Architecture),
				"archiveSizeBytes": llx.IntData(image.ArchiveSizeBytes),
				"diskSizeGb":       llx.IntData(image.DiskSizeGb),
				"family":           llx.StringData(image.Family),
				"licenses":         llx.ArrayData(convert.SliceAnyToInterface(image.Licenses), types.String),
				"labels":           llx.MapData(convert.MapToInterfaceMap(image.Labels), types.String),
				"status":           llx.StringData(image.Status),
				"created":          llx.TimeDataPtr(parseTime(image.CreationTimestamp)),
			})
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
	if g.Id.Error != nil {
		return "", g.Id.Error
	}
	id := g.Id.Data
	return "gcloud.compute.network/" + id, nil
}

func (g *mqlGcpProjectComputeServiceNetwork) subnetworks() ([]interface{}, error) {
	if g.SubnetworkUrls.Error != nil {
		return nil, g.SubnetworkUrls.Error
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

func initGcpProjectComputeServiceNetwork(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}

	// If no args are set, try reading them from the platform ID
	if len(args) == 0 {
		if ids := getAssetIdentifier(runtime); ids != nil {
			args["name"] = llx.StringData(ids.name)
			args["projectId"] = llx.StringData(ids.project)
		} else {
			return nil, nil, errors.New("no asset identifier found")
		}
	}

	obj, err := CreateResource(runtime, "gcp.project.computeService", map[string]*llx.RawData{
		"projectId": args["projectId"],
	})
	if err != nil {
		return nil, nil, err
	}
	computeSvc := obj.(*mqlGcpProjectComputeService)
	networks := computeSvc.GetNetworks()
	if networks.Error != nil {
		return nil, nil, networks.Error
	}

	for _, n := range networks.Data {
		network := n.(*mqlGcpProjectComputeServiceNetwork)
		name := network.GetName()
		if name.Error != nil {
			return nil, nil, name.Error
		}
		projectId := network.GetProjectId()
		if projectId.Error != nil {
			return nil, nil, projectId.Error
		}

		if name.Data == args["name"].Value && projectId.Data == args["projectId"].Value {
			return args, network, nil
		}
	}
	return nil, nil, nil
}

func (g *mqlGcpProjectComputeService) networks() ([]interface{}, error) {
	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)

	client, err := conn.Client(cloudresourcemanager.CloudPlatformReadOnlyScope, iam.CloudPlatformScope, compute.CloudPlatformScope)
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

			peerings, err := convert.JsonToDictSlice(network.Peerings)
			if err != nil {
				return err
			}

			var routingMode string
			if network.RoutingConfig != nil {
				routingMode = network.RoutingConfig.RoutingMode
			}

			mqlNetwork, err := CreateResource(g.MqlRuntime, "gcp.project.computeService.network", map[string]*llx.RawData{
				"id":                                    llx.StringData(strconv.FormatUint(network.Id, 10)),
				"projectId":                             llx.StringData(projectId),
				"name":                                  llx.StringData(network.Name),
				"description":                           llx.StringData(network.Description),
				"autoCreateSubnetworks":                 llx.BoolData(network.AutoCreateSubnetworks),
				"enableUlaInternalIpv6":                 llx.BoolData(network.EnableUlaInternalIpv6),
				"gatewayIPv4":                           llx.StringData(network.GatewayIPv4),
				"mtu":                                   llx.IntData(network.Mtu),
				"networkFirewallPolicyEnforcementOrder": llx.StringData(network.NetworkFirewallPolicyEnforcementOrder),
				"created":                               llx.TimeDataPtr(parseTime(network.CreationTimestamp)),
				"peerings":                              llx.ArrayData(peerings, types.Dict),
				"routingMode":                           llx.StringData(routingMode),
				"mode":                                  llx.StringData(networkMode(network)),
				"subnetworkUrls":                        llx.ArrayData(convert.SliceAnyToInterface(network.Subnetworks), types.Resource("gcp.project.computeService.subnetwork")),
			})
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
	if g.Id.Error != nil {
		return "", g.Id.Error
	}
	id := g.Id.Data
	return "gcloud.compute.subnetwork/" + id, nil
}

func initGcpProjectComputeServiceSubnetwork(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 3 {
		return args, nil, nil
	}

	// If no args are set, try reading them from the platform ID
	if len(args) == 0 {
		if ids := getAssetIdentifier(runtime); ids != nil {
			args["name"] = llx.StringData(ids.name)
			args["region"] = llx.StringData(ids.region)
			args["projectId"] = llx.StringData(ids.project)
		} else {
			return nil, nil, errors.New("no asset identifier found")
		}
	}

	obj, err := CreateResource(runtime, "gcp.project.computeService", map[string]*llx.RawData{
		"projectId": llx.StringData(args["projectId"].Value.(string)),
	})
	if err != nil {
		return nil, nil, err
	}
	computeSvc := obj.(*mqlGcpProjectComputeService)
	subnetworks := computeSvc.GetSubnetworks()
	if subnetworks.Error != nil {
		return nil, nil, subnetworks.Error
	}

	for _, n := range subnetworks.Data {
		subnetwork := n.(*mqlGcpProjectComputeServiceSubnetwork)
		name := subnetwork.GetName()
		if name.Error != nil {
			return nil, nil, name.Error
		}
		regionUrl := subnetwork.GetRegionUrl()
		if regionUrl.Error != nil {
			return nil, nil, regionUrl.Error
		}
		region := RegionNameFromRegionUrl(regionUrl.Data)
		projectId := subnetwork.GetProjectId()
		if projectId.Error != nil {
			return nil, nil, projectId.Error
		}

		if name.Data == args["name"].Value && projectId.Data == args["projectId"].Value && region == args["region"].Value {
			return args, subnetwork, nil
		}
	}
	return nil, nil, nil
}

func (g *mqlGcpProjectComputeServiceSubnetworkLogConfig) id() (string, error) {
	if g.Id.Error != nil {
		return "", g.Id.Error
	}
	id := g.Id.Data
	return "gcloud.compute.subnetwork.logConfig/" + id, nil
}

func (g *mqlGcpProjectComputeServiceSubnetwork) region() (*mqlGcpProjectComputeServiceRegion, error) {
	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	regionUrl := g.GetRegionUrl()
	if regionUrl.Error != nil {
		return nil, regionUrl.Error
	}

	regionName := RegionNameFromRegionUrl(regionUrl.Data)

	// Find regionName for projectId
	obj, err := CreateResource(g.MqlRuntime, "gcp.project.computeService", map[string]*llx.RawData{
		"projectId": llx.StringData(projectId),
	})
	if err != nil {
		return nil, err
	}
	gcpCompute := obj.(*mqlGcpProjectComputeService)
	regions := gcpCompute.GetRegions()
	if regions.Error != nil {
		return nil, regions.Error
	}

	for _, r := range regions.Data {
		region := r.(*mqlGcpProjectComputeServiceRegion)
		name := region.GetName()
		if name.Error != nil {
			return nil, name.Error
		}
		if name.Data == regionName {
			return region, nil
		}
	}

	return nil, errors.New(fmt.Sprintf("region %s not found", regionName))
}

func (g *mqlGcpProjectComputeServiceSubnetwork) network() ([]interface{}, error) {
	// TODO: implement
	return nil, errors.New("not implemented")
}

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
		"quotas":      llx.MapData(quotas, types.Float),
		"deprecated":  llx.DictData(deprecated),
	})
}

func newMqlSubnetwork(projectId string, runtime *plugin.Runtime, subnetwork *computepb.Subnetwork, region *mqlGcpProjectComputeServiceRegion) (interface{}, error) {
	subnetId := strconv.FormatUint(subnetwork.GetId(), 10)
	var mqlLogConfig plugin.Resource
	var err error
	if subnetwork.LogConfig != nil {
		mqlLogConfig, err = CreateResource(runtime, "gcp.project.computeService.subnetwork.logConfig", map[string]*llx.RawData{
			"id":                  llx.StringData(fmt.Sprintf("%s/logConfig", subnetId)),
			"aggregationInterval": llx.StringData(subnetwork.LogConfig.GetAggregationInterval()),
			"enable":              llx.BoolData(subnetwork.LogConfig.GetEnable()),
			"filterExpression":    llx.StringData(subnetwork.LogConfig.GetFilterExpr()),
			"flowSampling":        llx.FloatData(float64(subnetwork.LogConfig.GetFlowSampling())),
			"metadata":            llx.StringData(subnetwork.LogConfig.GetMetadata()),
			"metadataFields":      llx.ArrayData(convert.SliceAnyToInterface(subnetwork.LogConfig.MetadataFields), types.String),
		})
		if err != nil {
			return nil, err
		}
	}

	args := map[string]*llx.RawData{
		"id":                      llx.StringData(subnetId),
		"projectId":               llx.StringData(projectId),
		"name":                    llx.StringData(subnetwork.GetName()),
		"description":             llx.StringData(subnetwork.GetDescription()),
		"enableFlowLogs":          llx.BoolData(subnetwork.GetEnableFlowLogs()),
		"externalIpv6Prefix":      llx.StringData(subnetwork.GetExternalIpv6Prefix()),
		"fingerprint":             llx.StringData(subnetwork.GetFingerprint()),
		"gatewayAddress":          llx.StringData(subnetwork.GetGatewayAddress()),
		"internalIpv6Prefix":      llx.StringData(subnetwork.GetInternalIpv6Prefix()),
		"ipCidrRange":             llx.StringData(subnetwork.GetIpCidrRange()),
		"ipv6AccessType":          llx.StringData(subnetwork.GetIpv6AccessType()),
		"ipv6CidrRange":           llx.StringData(subnetwork.GetIpv6CidrRange()),
		"logConfig":               llx.ResourceData(mqlLogConfig, "gcp.project.computeService.subnetwork.logConfig"),
		"privateIpGoogleAccess":   llx.BoolData(subnetwork.GetPrivateIpGoogleAccess()),
		"privateIpv6GoogleAccess": llx.StringData(subnetwork.GetPrivateIpv6GoogleAccess()),
		"purpose":                 llx.StringData(subnetwork.GetPurpose()),
		"regionUrl":               llx.StringData(subnetwork.GetRegion()),
		"role":                    llx.StringData(subnetwork.GetRole()),
		"stackType":               llx.StringData(subnetwork.GetStackType()),
		"state":                   llx.StringData(subnetwork.GetState()),
		"created":                 llx.TimeDataPtr(parseTime(subnetwork.GetCreationTimestamp())),
	}
	if region != nil {
		args["region"] = llx.ResourceData(region, "gcp.project.computeService.region")
	}
	return CreateResource(runtime, "gcp.project.computeService.subnetwork", args)
}

func (g *mqlGcpProjectComputeService) subnetworks() ([]interface{}, error) {
	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)

	client, err := conn.Client(cloudresourcemanager.CloudPlatformReadOnlyScope, iam.CloudPlatformScope, compute.CloudPlatformScope)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()

	subnetSvc, err := computev1.NewSubnetworksRESTClient(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}

	res := []interface{}{}
	it := subnetSvc.AggregatedList(ctx, &computepb.AggregatedListSubnetworksRequest{Project: projectId})
	for {
		resp, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		subnets := resp.Value.GetSubnetworks()
		for _, subnet := range subnets {
			mqlSubnetwork, err := newMqlSubnetwork(projectId, g.MqlRuntime, subnet, nil)
			if err != nil {
				return nil, err
			}
			res = append(res, mqlSubnetwork)
		}
	}
	return res, nil
}

func (g *mqlGcpProjectComputeServiceRouter) id() (string, error) {
	if g.Id.Error != nil {
		return "", g.Id.Error
	}
	id := g.Id.Data
	return "gcloud.compute.router/" + id, nil
}

func (g *mqlGcpProjectComputeServiceRouter) network() ([]interface{}, error) {
	// TODO: implement
	return nil, errors.New("not implemented")
}

func (g *mqlGcpProjectComputeServiceRouter) region() ([]interface{}, error) {
	// TODO: implement
	return nil, errors.New("not implemented")
}

func newMqlRouter(projectId string, region *mqlGcpProjectComputeServiceRegion, runtime *plugin.Runtime, router *compute.Router) (interface{}, error) {
	bgp, err := convert.JsonToDict(router.Bgp)
	if err != nil {
		return nil, err
	}

	bgpPeers, err := convert.JsonToDictSlice(router.BgpPeers)
	if err != nil {
		return nil, err
	}

	nats, err := convert.JsonToDictSlice(router.Nats)
	if err != nil {
		return nil, err
	}

	return CreateResource(runtime, "gcp.project.computeService.router", map[string]*llx.RawData{
		"id":                          llx.StringData(strconv.FormatUint(router.Id, 10)),
		"name":                        llx.StringData(router.Name),
		"description":                 llx.StringData(router.Description),
		"bgp":                         llx.DictData(bgp),
		"bgpPeers":                    llx.ArrayData(bgpPeers, types.Dict),
		"encryptedInterconnectRouter": llx.BoolData(router.EncryptedInterconnectRouter),
		"nats":                        llx.ArrayData(nats, types.Dict),
		"created":                     llx.TimeDataPtr(parseTime(router.CreationTimestamp)),
	})
}

func (g *mqlGcpProjectComputeService) routers() ([]interface{}, error) {
	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	regions := g.GetRegions()
	if regions.Error != nil {
		return nil, regions.Error
	}

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)

	client, err := conn.Client(cloudresourcemanager.CloudPlatformReadOnlyScope, iam.CloudPlatformScope, compute.CloudPlatformScope)
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
	wg.Add(len(regions.Data))
	mux := &sync.Mutex{}

	for i := range regions.Data {
		r := regions.Data[i].(*mqlGcpProjectComputeServiceRegion)
		regionName := r.GetName()
		if err != nil {
			return nil, err
		}
		go func(svc *compute.Service, project string, region *mqlGcpProjectComputeServiceRegion, regionName string) {
			req := computeSvc.Routers.List(projectId, regionName)
			if err := req.Pages(ctx, func(page *compute.RouterList) error {
				for _, router := range page.Items {

					mqlRouter, err := newMqlRouter(projectId, region, g.MqlRuntime, router)
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
		}(computeSvc, projectId, r, regionName.Data)
	}

	wg.Wait()
	return res, nil
}

func (g *mqlGcpProjectComputeService) backendServices() ([]interface{}, error) {
	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)

	client, err := conn.Client(cloudresourcemanager.CloudPlatformReadOnlyScope, iam.CloudPlatformScope, compute.CloudPlatformScope)
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
				mqlBackend, err := CreateResource(g.MqlRuntime, "gcp.project.computeService.backendService.backend", map[string]*llx.RawData{
					"id":                        llx.StringData(fmt.Sprintf("gcp.project.computeService.backendService.backend/%s/%d", backendServiceId, i)),
					"balancingMode":             llx.StringData(backend.BalancingMode),
					"capacityScaler":            llx.FloatData(backend.CapacityScaler),
					"description":               llx.StringData(backend.Description),
					"failover":                  llx.BoolData(backend.Failover),
					"groupUrl":                  llx.StringData(backend.Group),
					"maxConnections":            llx.IntData(backend.MaxConnections),
					"maxConnectionsPerEndpoint": llx.IntData(backend.MaxConnectionsPerEndpoint),
					"maxConnectionsPerInstance": llx.IntData(backend.MaxConnectionsPerInstance),
					"maxRate":                   llx.IntData(backend.MaxRate),
					"maxRatePerEndpoint":        llx.FloatData(backend.MaxRatePerEndpoint),
					"maxRatePerInstance":        llx.FloatData(backend.MaxRatePerInstance),
					"maxUtilization":            llx.FloatData(backend.MaxUtilization),
				})
				if err != nil {
					return nil, err
				}
				mqlBackends = append(mqlBackends, mqlBackend)
			}

			var cdnPolicy plugin.Resource
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
						"includeHttpHeaders":   convert.SliceAnyToInterface(b.CdnPolicy.CacheKeyPolicy.IncludeHttpHeaders),
						"includeNamedCookies":  convert.SliceAnyToInterface(b.CdnPolicy.CacheKeyPolicy.IncludeNamedCookies),
						"includeProtocol":      b.CdnPolicy.CacheKeyPolicy.IncludeProtocol,
						"includeQueryString":   b.CdnPolicy.CacheKeyPolicy.IncludeQueryString,
						"queryStringBlacklist": convert.SliceAnyToInterface(b.CdnPolicy.CacheKeyPolicy.QueryStringBlacklist),
						"queryStringWhitelist": convert.SliceAnyToInterface(b.CdnPolicy.CacheKeyPolicy.QueryStringWhitelist),
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

				cdnPolicy, err = CreateResource(g.MqlRuntime, "gcp.project.computeService.backendService.cdnPolicy", map[string]*llx.RawData{
					"id":                          llx.StringData(fmt.Sprintf("gcp.project.computeService.backendService.cdnPolicy/%s", backendServiceId)),
					"bypassCacheOnRequestHeaders": llx.ArrayData(bypassCacheOnRequestHeaders, types.Dict),
					"cacheKeyPolicy":              llx.DictData(mqlCacheKeyPolicy),
					"cacheMode":                   llx.StringData(b.CdnPolicy.CacheMode),
					"clientTtl":                   llx.IntData(b.CdnPolicy.ClientTtl),
					"defaultTtl":                  llx.IntData(b.CdnPolicy.DefaultTtl),
					"maxTtl":                      llx.IntData(b.CdnPolicy.MaxTtl),
					"negativeCaching":             llx.BoolData(b.CdnPolicy.NegativeCaching),
					"negativeCachingPolicy":       llx.ArrayData(mqlNegativeCachingPolicy, types.Dict),
					"requestCoalescing":           llx.BoolData(b.CdnPolicy.RequestCoalescing),
					"serveWhileStale":             llx.IntData(b.CdnPolicy.ServeWhileStale),
					"signedUrlCacheMaxAgeSec":     llx.IntData(b.CdnPolicy.SignedUrlCacheMaxAgeSec),
					"signedUrlKeyNames":           llx.ArrayData(convert.SliceAnyToInterface(b.CdnPolicy.SignedUrlKeyNames), types.String),
				})
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
						"ttl":  llx.TimeData(llx.DurationToTime(b.ConsistentHash.HttpCookie.Ttl.Seconds)),
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
					"subjectAltNames": convert.SliceAnyToInterface(b.SecuritySettings.SubjectAltNames),
				}
			}

			var maxStreamDuration *time.Time
			if b.MaxStreamDuration != nil {
				v := llx.DurationToTime(b.MaxStreamDuration.Seconds)
				maxStreamDuration = &v
			}

			mqlB, err := CreateResource(g.MqlRuntime, "gcp.project.computeService.backendService", map[string]*llx.RawData{
				"id":                       llx.StringData(backendServiceId),
				"affinityCookieTtlSec":     llx.IntData(b.AffinityCookieTtlSec),
				"backends":                 llx.ArrayData(mqlBackends, types.Resource("gcp.project.computeService.backendService.backend")),
				"cdnPolicy":                llx.ResourceData(cdnPolicy, " gcp.project.computeService.backendService.cdnPolicy"),
				"circuitBreakers":          llx.DictData(mqlCircuitBreakers),
				"compressionMode":          llx.StringData(b.CompressionMode),
				"connectionDraining":       llx.DictData(mqlConnectionDraining),
				"connectionTrackingPolicy": llx.DictData(mqlConnectionTrackingPolicy),
				"consistentHash":           llx.DictData(mqlConsistentHash),
				"created":                  llx.TimeDataPtr(parseTime(b.CreationTimestamp)),
				"customRequestHeaders":     llx.ArrayData(convert.SliceAnyToInterface(b.CustomRequestHeaders), types.String),
				"customResponseHeaders":    llx.ArrayData(convert.SliceAnyToInterface(b.CustomResponseHeaders), types.String),
				"description":              llx.StringData(b.Description),
				"edgeSecurityPolicy":       llx.StringData(b.EdgeSecurityPolicy),
				"enableCDN":                llx.BoolData(b.EnableCDN),
				"failoverPolicy":           llx.DictData(mqlFailoverPolicy),
				"healthChecks":             llx.ArrayData(convert.SliceAnyToInterface(b.HealthChecks), types.String),
				"iap":                      llx.DictData(mqlIap),
				"loadBalancingScheme":      llx.StringData(b.LoadBalancingScheme),
				"localityLbPolicies":       llx.ArrayData(mqlLocalityLbPolicy, types.Dict),
				"localityLbPolicy":         llx.StringData(b.LocalityLbPolicy),
				"logConfig":                llx.DictData(mqlLogConfig),
				"maxStreamDuration":        llx.TimeDataPtr(maxStreamDuration),
				"name":                     llx.StringData(b.Name),
				"networkUrl":               llx.StringData(b.Network),
				"portName":                 llx.StringData(b.PortName),
				"protocol":                 llx.StringData(b.Protocol),
				"regionUrl":                llx.StringData(b.Region),
				"securityPolicyUrl":        llx.StringData(b.SecurityPolicy),
				"securitySettings":         llx.DictData(mqlSecuritySettings),
				"serviceBindingUrls":       llx.ArrayData(convert.SliceAnyToInterface(b.ServiceBindings), types.String),
				"sessionAffinity":          llx.StringData(b.SessionAffinity),
				"timeoutSec":               llx.IntData(b.TimeoutSec),
			})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlB)
		}
	}
	return res, nil
}

func (g *mqlGcpProjectComputeServiceBackendService) id() (string, error) {
	if g.Id.Error != nil {
		return "", g.Id.Error
	}
	id := g.Id.Data
	return "gcp.project.computeService.backendService/" + id, nil
}

func (g *mqlGcpProjectComputeServiceBackendServiceBackend) id() (string, error) {
	return g.Id.Data, g.Id.Error
}

func (g *mqlGcpProjectComputeServiceBackendServiceCdnPolicy) id() (string, error) {
	return g.Id.Data, g.Id.Error
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

func (g *mqlGcpProjectComputeService) addresses() ([]interface{}, error) {
	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)

	client, err := conn.Client(cloudresourcemanager.CloudPlatformReadOnlyScope, iam.CloudPlatformScope, compute.CloudPlatformScope)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	computeSvc, err := compute.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}

	list, err := computeSvc.Addresses.AggregatedList(projectId).Do()
	if err != nil {
		return nil, err
	}
	var mqlAddresses []interface{}
	for _, as := range list.Items {
		for _, a := range as.Addresses {
			mqlA, err := CreateResource(g.MqlRuntime, "gcp.project.computeService.address", map[string]*llx.RawData{
				"id":               llx.StringData(fmt.Sprintf("%d", a.Id)),
				"address":          llx.StringData(a.Address),
				"addressType":      llx.StringData(a.AddressType),
				"created":          llx.TimeDataPtr(parseTime(a.CreationTimestamp)),
				"description":      llx.StringData(a.Description),
				"ipVersion":        llx.StringData(a.IpVersion),
				"ipv6EndpointType": llx.StringData(a.Ipv6EndpointType),
				"name":             llx.StringData(a.Name),
				"networkUrl":       llx.StringData(a.Network),
				"networkTier":      llx.StringData(a.NetworkTier),
				"prefixLength":     llx.IntData(a.PrefixLength),
				"purpose":          llx.StringData(a.Purpose),
				"regionUrl":        llx.StringData(a.Region),
				"status":           llx.StringData(a.Status),
				"subnetworkUrl":    llx.StringData(a.Subnetwork),
				"resourceUrls":     llx.ArrayData(convert.SliceAnyToInterface(a.Users), types.String),
			})
			if err != nil {
				return nil, err
			}
			mqlAddresses = append(mqlAddresses, mqlA)
		}
	}
	return mqlAddresses, nil
}

func (g *mqlGcpProjectComputeServiceAddress) network() (*mqlGcpProjectComputeServiceNetwork, error) {
	if g.NetworkUrl.Error != nil {
		return nil, g.NetworkUrl.Error
	}
	networkUrl := g.NetworkUrl.Data
	return getNetworkByUrl(networkUrl, g.MqlRuntime)
}

func (g *mqlGcpProjectComputeServiceAddress) subnetwork() (*mqlGcpProjectComputeServiceSubnetwork, error) {
	if g.SubnetworkUrl.Error != nil {
		return nil, g.SubnetworkUrl.Error
	}
	subnetUrl := g.SubnetworkUrl.Data
	return getSubnetworkByUrl(subnetUrl, g.MqlRuntime)
}

func (g *mqlGcpProjectComputeServiceAddress) id() (string, error) {
	return g.Id.Data, g.Id.Error
}

func (g *mqlGcpProjectComputeService) forwardingRules() ([]interface{}, error) {
	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)

	client, err := conn.Client(cloudresourcemanager.CloudPlatformReadOnlyScope, iam.CloudPlatformScope, compute.CloudPlatformScope)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	fwrSvc, err := computev1.NewForwardingRulesRESTClient(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}

	var fwRules []interface{}
	it := fwrSvc.AggregatedList(ctx, &computepb.AggregatedListForwardingRulesRequest{Project: projectId, IncludeAllScopes: ptr.Bool(true)})
	for {
		resp, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		for _, fwr := range resp.Value.ForwardingRules {
			metadataFilters := make([]interface{}, 0, len(fwr.GetMetadataFilters()))
			for _, m := range fwr.GetMetadataFilters() {
				filterLabels := make([]interface{}, 0, len(m.GetFilterLabels()))
				for _, l := range m.GetFilterLabels() {
					filterLabels = append(filterLabels, map[string]interface{}{
						"name":  l.GetName(),
						"value": l.GetValue(),
					})
				}
				metadataFilters = append(metadataFilters, map[string]interface{}{
					"filterLabels":        filterLabels,
					"filterMatchCriteria": m.GetFilterMatchCriteria(),
				})
			}

			serviceDirRegs := make([]interface{}, 0, len(fwr.GetServiceDirectoryRegistrations()))
			for _, s := range fwr.GetServiceDirectoryRegistrations() {
				serviceDirRegs = append(serviceDirRegs, map[string]interface{}{
					"namespace":              s.GetNamespace(),
					"service":                s.GetService(),
					"serviceDirectoryRegion": s.GetServiceDirectoryRegion(),
				})
			}
			mqlFwr, err := CreateResource(g.MqlRuntime, "gcp.project.computeService.forwardingRule", map[string]*llx.RawData{
				"id":                            llx.StringData(fmt.Sprintf("%d", fwr.Id)),
				"ipAddress":                     llx.StringData(fwr.GetIPAddress()),
				"ipProtocol":                    llx.StringData(fwr.GetIPProtocol()),
				"allPorts":                      llx.BoolData(fwr.GetAllPorts()),
				"allowGlobalAccess":             llx.BoolData(fwr.GetAllowGlobalAccess()),
				"backendService":                llx.StringData(fwr.GetBackendService()),
				"created":                       llx.TimeDataPtr(parseTime(fwr.GetCreationTimestamp())),
				"description":                   llx.StringData(fwr.GetDescription()),
				"ipVersion":                     llx.StringData(fwr.GetIpVersion()),
				"isMirroringCollector":          llx.BoolData(fwr.GetIsMirroringCollector()),
				"labels":                        llx.MapData(convert.MapToInterfaceMap(fwr.GetLabels()), types.String),
				"loadBalancingScheme":           llx.StringData(fwr.GetLoadBalancingScheme()),
				"metadataFilters":               llx.ArrayData(metadataFilters, types.Dict),
				"name":                          llx.StringData(fwr.GetName()),
				"networkUrl":                    llx.StringData(fwr.GetNetwork()),
				"networkTier":                   llx.StringData(fwr.GetNetworkTier()),
				"noAutomateDnsZone":             llx.BoolData(fwr.GetNoAutomateDnsZone()),
				"portRange":                     llx.StringData(fwr.GetPortRange()),
				"ports":                         llx.ArrayData(convert.SliceAnyToInterface(fwr.GetPorts()), types.String),
				"regionUrl":                     llx.StringData(fwr.GetRegion()),
				"serviceDirectoryRegistrations": llx.ArrayData(serviceDirRegs, types.Dict),
				"serviceLabel":                  llx.StringData(fwr.GetServiceLabel()),
				"serviceName":                   llx.StringData(fwr.GetServiceName()),
				"subnetworkUrl":                 llx.StringData(fwr.GetSubnetwork()),
				"targetUrl":                     llx.StringData(fwr.GetTarget()),
			})
			if err != nil {
				return nil, err
			}
			fwRules = append(fwRules, mqlFwr)
		}
	}
	return fwRules, nil
}

func (g *mqlGcpProjectComputeServiceForwardingRule) id() (string, error) {
	return g.Id.Data, g.Id.Error
}

func (g *mqlGcpProjectComputeServiceForwardingRule) network() (*mqlGcpProjectComputeServiceNetwork, error) {
	if g.NetworkUrl.Error != nil {
		return nil, g.NetworkUrl.Error
	}
	networkUrl := g.NetworkUrl.Data
	return getNetworkByUrl(networkUrl, g.MqlRuntime)
}

func (g *mqlGcpProjectComputeServiceForwardingRule) subnetwork() (*mqlGcpProjectComputeServiceSubnetwork, error) {
	if g.SubnetworkUrl.Error != nil {
		return nil, g.SubnetworkUrl.Error
	}
	subnetUrl := g.SubnetworkUrl.Data
	return getSubnetworkByUrl(subnetUrl, g.MqlRuntime)
}
