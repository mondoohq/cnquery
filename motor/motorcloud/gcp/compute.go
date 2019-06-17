package gcp

import (
	"strconv"
	"sync"

	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/nexus/assets"
	"google.golang.org/api/cloudresourcemanager/v1"
	"google.golang.org/api/compute/v1"
)

func (a *GcpCompute) List() ([]*assets.Asset, error) {
	client, err := gcpClient(compute.ComputeScope, compute.CloudPlatformScope)
	svc, err := compute.New(client)
	if err != nil {
		return nil, err
	}

	resSrv, err := cloudresourcemanager.New(client)
	if err != nil {
		return nil, err
	}

	projectsResp, err := resSrv.Projects.List().Do()
	if err != nil {
		return nil, err
	}

	var wg sync.WaitGroup
	assets := []*assets.Asset{}
	for i := range projectsResp.Projects {
		project := projectsResp.Projects[i].ProjectId
		zones, err := svc.Zones.List(project).Do()
		if err != nil {
			return nil, err
		}

		// add zones
		wg.Add(len(zones.Items))
		mux := &sync.Mutex{}
		for j := range zones.Items {
			zone := zones.Items[j].Name
			go func(svc *compute.Service, project string, zone string) {
				zoneAssets, err := instancesPerZone(svc, project, zone)
				if err == nil {
					mux.Lock()
					assets = append(assets, zoneAssets...)
					mux.Unlock()
				}
				wg.Done()
			}(svc, project, zone)
		}
	}

	wg.Wait()
	return assets, nil
}

func instancesPerZone(svc *compute.Service, project string, zone string) ([]*assets.Asset, error) {
	log.Debug().Str("project", project).Str("zone", zone).Msg("search gcp project for assets")
	il, err := svc.Instances.List(project, zone).Do()
	if err != nil {
		return nil, err
	}

	instances := make([]*assets.Asset, len(il.Items))
	for i := range il.Items {
		instance := il.Items[i]

		connections := []*assets.Connection{}

		// add ssh and ssm run command if the node is part of ssm
		// connections = append(connections, &assets.Connection{
		// 	Backend: assets.ConnectionBackend_SSH,
		// 	Host:    instance.Name,
		// })

		asset := &assets.Asset{
			ReferenceID: MondooGcpInstanceID(project, zone, instance),
			Name:        instance.Name,
			Platform: &assets.Platform{
				Kind:    assets.Kind_KIND_VIRTUAL_MACHINE,
				Runtime: "gcp compute",
			},
			Connections: connections,
			State:       mapInstanceState(instance.Status),
			Labels:      make(map[string]string),
		}

		for key := range instance.Labels {
			asset.Labels[key] = instance.Labels[key]
		}

		// fetch gcp specific metadata
		asset.Labels["mondoo.app/region"] = zone
		asset.Labels["mondoo.app/instance"] = strconv.FormatUint(uint64(instance.Id), 10)

		instances[i] = asset
	}

	return instances, nil
}

func MondooGcpInstanceID(project string, zone string, instance *compute.Instance) string {
	return "gcp://compute/v1/projects/" + project + "/zones/" + zone + "/instances/" + strconv.FormatUint(uint64(instance.Id), 10)
}

func mapInstanceState(state string) assets.State {
	switch state {
	case "RUNNING":
		return assets.State_STATE_RUNNING
	case "PROVISIONING":
		return assets.State_STATE_PENDING
	case "STAGING":
		return assets.State_STATE_PENDING
	case "STOPPED":
		return assets.State_STATE_STOPPED
	case "STOPPING":
		return assets.State_STATE_STOPPING
	case "SUSPENDED":
		return assets.State_STATE_STOPPED
	case "SUSPENDING":
		return assets.State_STATE_STOPPING
	case "TERMINATED":
		return assets.State_STATE_TERMINATED
	default:
		log.Warn().Str("state", state).Msg("unknown gcp instance state")
		return assets.State_STATE_UNKNOWN
	}
}
