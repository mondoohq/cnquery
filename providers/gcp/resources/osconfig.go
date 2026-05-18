// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/gcp/connection"
	"go.mondoo.com/mql/v13/types"
	"golang.org/x/sync/errgroup"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/option"
	osconfig "google.golang.org/api/osconfig/v1"
)

type mqlGcpProjectOsConfigServiceInternal struct {
	serviceEnabled bool
}

func (g *mqlGcpProject) osConfig() (*mqlGcpProjectOsConfigService, error) {
	if g.Id.Error != nil {
		return nil, g.Id.Error
	}
	projectId := g.Id.Data

	// Check service enablement before creating the resource: a transient error
	// here must not leave a resource cached with serviceEnabled = false, which
	// would make every child accessor silently return nil on later calls.
	serviceEnabled, err := g.isServiceEnabled(service_osconfig)
	if err != nil {
		return nil, err
	}

	res, err := CreateResource(g.MqlRuntime, "gcp.project.osConfigService", map[string]*llx.RawData{
		"projectId": llx.StringData(projectId),
	})
	if err != nil {
		return nil, err
	}

	svc := res.(*mqlGcpProjectOsConfigService)
	svc.serviceEnabled = serviceEnabled
	if !serviceEnabled {
		log.Debug().Str("service", service_osconfig).Msg("gcp service is not enabled, skipping")
	}

	return svc, nil
}

func initGcpProjectOsConfigService(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 0 {
		return args, nil, nil
	}

	conn, ok := runtime.Connection.(*connection.GcpConnection)
	if !ok {
		return nil, nil, errors.New("invalid connection provided, it is not a GCP connection")
	}

	args["projectId"] = llx.StringData(conn.ResourceID())
	return args, nil, nil
}

func (g *mqlGcpProjectOsConfigService) id() (string, error) {
	if g.ProjectId.Error != nil {
		return "", g.ProjectId.Error
	}
	return fmt.Sprintf("%s/gcp.project.osConfigService", g.ProjectId.Data), nil
}

func (g *mqlGcpProjectOsConfigServicePatchDeployment) id() (string, error) {
	return g.Name.Data, g.Name.Error
}

func (g *mqlGcpProjectOsConfigServiceOsPolicyAssignment) id() (string, error) {
	return g.Name.Data, g.Name.Error
}

// parseSecondsDuration converts a GCP duration string (e.g. "3600s") to whole seconds.
func parseSecondsDuration(d string) int64 {
	if d == "" {
		return 0
	}
	secs, err := strconv.ParseFloat(strings.TrimSuffix(d, "s"), 64)
	if err != nil {
		return 0
	}
	return int64(secs)
}

func (g *mqlGcpProjectOsConfigService) patchDeployments() ([]any, error) {
	if !g.serviceEnabled {
		return nil, nil
	}
	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	conn, ok := g.MqlRuntime.Connection.(*connection.GcpConnection)
	if !ok {
		return nil, errors.New("invalid connection provided, it is not a GCP connection")
	}
	client, err := conn.Client(osconfig.CloudPlatformScope)
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	osConfigSvc, err := osconfig.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}
	res := []any{}
	call := osConfigSvc.Projects.PatchDeployments.List("projects/" + projectId)
	if err := call.Pages(ctx, func(page *osconfig.ListPatchDeploymentsResponse) error {
		for _, pd := range page.PatchDeployments {
			instanceFilter, err := convert.JsonToDict(pd.InstanceFilter)
			if err != nil {
				return err
			}
			patchConfig, err := convert.JsonToDict(pd.PatchConfig)
			if err != nil {
				return err
			}
			oneTimeSchedule, err := convert.JsonToDict(pd.OneTimeSchedule)
			if err != nil {
				return err
			}
			recurringSchedule, err := convert.JsonToDict(pd.RecurringSchedule)
			if err != nil {
				return err
			}
			rollout, err := convert.JsonToDict(pd.Rollout)
			if err != nil {
				return err
			}

			mqlPd, err := CreateResource(g.MqlRuntime, "gcp.project.osConfigService.patchDeployment", map[string]*llx.RawData{
				"name":              llx.StringData(pd.Name),
				"description":       llx.StringData(pd.Description),
				"instanceFilter":    llx.DictData(instanceFilter),
				"patchConfig":       llx.DictData(patchConfig),
				"duration":          llx.IntData(parseSecondsDuration(pd.Duration)),
				"oneTimeSchedule":   llx.DictData(oneTimeSchedule),
				"recurringSchedule": llx.DictData(recurringSchedule),
				"rollout":           llx.DictData(rollout),
				"state":             llx.StringData(pd.State),
				"created":           llx.TimeDataPtr(parseTime(pd.CreateTime)),
				"updated":           llx.TimeDataPtr(parseTime(pd.UpdateTime)),
				"lastExecuteTime":   llx.TimeDataPtr(parseTime(pd.LastExecuteTime)),
			})
			if err != nil {
				return err
			}
			res = append(res, mqlPd)
		}
		return nil
	}); err != nil {
		if isHTTPSkippable(err) {
			log.Warn().Err(err).Msg("could not list OS patch deployments")
			return nil, nil
		}
		return nil, err
	}
	return res, nil
}

func (g *mqlGcpProjectOsConfigService) osPolicyAssignments() ([]any, error) {
	if !g.serviceEnabled {
		return nil, nil
	}
	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	conn, ok := g.MqlRuntime.Connection.(*connection.GcpConnection)
	if !ok {
		return nil, errors.New("invalid connection provided, it is not a GCP connection")
	}
	client, err := conn.Client(osconfig.CloudPlatformScope)
	if err != nil {
		return nil, err
	}
	ctx := context.Background()

	// OS policy assignments are zonal, so we have to enumerate every zone.
	computeSvc, err := compute.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}
	var zones []string
	if err := computeSvc.Zones.List(projectId).Pages(ctx, func(page *compute.ZoneList) error {
		for _, zone := range page.Items {
			zones = append(zones, zone.Name)
		}
		return nil
	}); err != nil {
		if isHTTPSkippable(err) {
			log.Warn().Err(err).Msg("could not list zones for OS policy assignments")
			return nil, nil
		}
		return nil, err
	}

	osConfigSvc, err := osconfig.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}

	// Fan out across zones, but cap concurrency so a project with many zones
	// doesn't trigger GCP API rate-limiting. A non-skippable error from any
	// zone fails the whole call so the caller never sees partial results.
	var (
		res []any
		mux sync.Mutex
	)
	grp, grpCtx := errgroup.WithContext(ctx)
	grp.SetLimit(10)
	for i := range zones {
		zone := zones[i]
		grp.Go(func() error {
			parent := fmt.Sprintf("projects/%s/locations/%s", projectId, zone)
			err := osConfigSvc.Projects.Locations.OsPolicyAssignments.List(parent).Pages(grpCtx, func(page *osconfig.ListOSPolicyAssignmentsResponse) error {
				for _, a := range page.OsPolicyAssignments {
					mqlAssignment, err := newMqlOsPolicyAssignment(g.MqlRuntime, a)
					if err != nil {
						return err
					}
					mux.Lock()
					res = append(res, mqlAssignment)
					mux.Unlock()
				}
				return nil
			})
			if err != nil {
				if isHTTPSkippable(err) {
					log.Warn().Err(err).Str("zone", zone).Msg("could not list OS policy assignments")
					return nil
				}
				return err
			}
			return nil
		})
	}
	if err := grp.Wait(); err != nil {
		return nil, err
	}
	return res, nil
}

func newMqlOsPolicyAssignment(runtime *plugin.Runtime, a *osconfig.OSPolicyAssignment) (plugin.Resource, error) {
	osPolicies, err := convert.JsonToDictSlice(a.OsPolicies)
	if err != nil {
		return nil, err
	}
	instanceFilter, err := convert.JsonToDict(a.InstanceFilter)
	if err != nil {
		return nil, err
	}
	rollout, err := convert.JsonToDict(a.Rollout)
	if err != nil {
		return nil, err
	}
	return CreateResource(runtime, "gcp.project.osConfigService.osPolicyAssignment", map[string]*llx.RawData{
		"name":               llx.StringData(a.Name),
		"description":        llx.StringData(a.Description),
		"osPolicies":         llx.ArrayData(osPolicies, types.Dict),
		"instanceFilter":     llx.DictData(instanceFilter),
		"rollout":            llx.DictData(rollout),
		"revisionId":         llx.StringData(a.RevisionId),
		"revisionCreateTime": llx.TimeDataPtr(parseTime(a.RevisionCreateTime)),
		"rolloutState":       llx.StringData(a.RolloutState),
		"baseline":           llx.BoolData(a.Baseline),
		"deleted":            llx.BoolData(a.Deleted),
		"reconciling":        llx.BoolData(a.Reconciling),
	})
}

func (g *mqlGcpProjectComputeServiceInstanceOsInventory) id() (string, error) {
	return g.Name.Data, g.Name.Error
}

func (g *mqlGcpProjectComputeServiceInstanceVulnerabilityReport) id() (string, error) {
	return g.Name.Data, g.Name.Error
}

// instanceOsConfigName builds the VM Manager resource path for an instance:
// projects/{project}/locations/{zone}/instances/{instanceId}.
func (g *mqlGcpProjectComputeServiceInstance) instanceOsConfigName() (string, error) {
	if g.ProjectId.Error != nil {
		return "", g.ProjectId.Error
	}
	if g.Id.Error != nil {
		return "", g.Id.Error
	}
	zone := g.GetZone()
	if zone.Error != nil {
		return "", zone.Error
	}
	zoneName := zone.Data.GetName()
	if zoneName.Error != nil {
		return "", zoneName.Error
	}
	return fmt.Sprintf("projects/%s/locations/%s/instances/%s", g.ProjectId.Data, zoneName.Data, g.Id.Data), nil
}

func (g *mqlGcpProjectComputeServiceInstance) inventory() (*mqlGcpProjectComputeServiceInstanceOsInventory, error) {
	parent, err := g.instanceOsConfigName()
	if err != nil {
		return nil, err
	}

	conn, ok := g.MqlRuntime.Connection.(*connection.GcpConnection)
	if !ok {
		return nil, errors.New("invalid connection provided, it is not a GCP connection")
	}
	client, err := conn.Client(osconfig.CloudPlatformScope)
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	osConfigSvc, err := osconfig.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}

	inv, err := osConfigSvc.Projects.Locations.Instances.Inventories.Get(parent + "/inventory").View("FULL").Do()
	if err != nil {
		if isHTTPSkippable(err) {
			g.Inventory.State = plugin.StateIsNull | plugin.StateIsSet
			return nil, nil
		}
		return nil, err
	}

	osInfo, err := convert.JsonToDict(inv.OsInfo)
	if err != nil {
		return nil, err
	}
	items := make([]any, 0, len(inv.Items))
	for _, item := range inv.Items {
		dict, err := convert.JsonToDict(item)
		if err != nil {
			return nil, err
		}
		items = append(items, dict)
	}

	res, err := CreateResource(g.MqlRuntime, "gcp.project.computeService.instance.osInventory", map[string]*llx.RawData{
		"name":       llx.StringData(inv.Name),
		"osInfo":     llx.DictData(osInfo),
		"items":      llx.ArrayData(items, types.Dict),
		"updateTime": llx.TimeDataPtr(parseTime(inv.UpdateTime)),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlGcpProjectComputeServiceInstanceOsInventory), nil
}

func (g *mqlGcpProjectComputeServiceInstance) vulnerabilityReport() (*mqlGcpProjectComputeServiceInstanceVulnerabilityReport, error) {
	parent, err := g.instanceOsConfigName()
	if err != nil {
		return nil, err
	}

	conn, ok := g.MqlRuntime.Connection.(*connection.GcpConnection)
	if !ok {
		return nil, errors.New("invalid connection provided, it is not a GCP connection")
	}
	client, err := conn.Client(osconfig.CloudPlatformScope)
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	osConfigSvc, err := osconfig.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}

	report, err := osConfigSvc.Projects.Locations.Instances.VulnerabilityReports.Get(parent + "/vulnerabilityReport").Do()
	if err != nil {
		if isHTTPSkippable(err) {
			g.VulnerabilityReport.State = plugin.StateIsNull | plugin.StateIsSet
			return nil, nil
		}
		return nil, err
	}

	vulnerabilities, err := convert.JsonToDictSlice(report.Vulnerabilities)
	if err != nil {
		return nil, err
	}

	res, err := CreateResource(g.MqlRuntime, "gcp.project.computeService.instance.vulnerabilityReport", map[string]*llx.RawData{
		"name":                         llx.StringData(report.Name),
		"vulnerabilities":              llx.ArrayData(vulnerabilities, types.Dict),
		"highestUpgradableCveSeverity": llx.StringData(report.HighestUpgradableCveSeverity),
		"updateTime":                   llx.TimeDataPtr(parseTime(report.UpdateTime)),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlGcpProjectComputeServiceInstanceVulnerabilityReport), nil
}
