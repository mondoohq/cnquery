package gcp

import (
	"context"
	"net/http"
	"strconv"
	"sync"

	"google.golang.org/api/option"

	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/motor/asset"
	"go.mondoo.io/mondoo/motor/platform"
	"go.mondoo.io/mondoo/motor/transports"
	"google.golang.org/api/cloudresourcemanager/v1"
	"google.golang.org/api/compute/v1"
)

func NewCompute(client *http.Client) *Compute {
	return &Compute{
		client: client,
	}
}

type Compute struct {
	client *http.Client
	// NOTE: is empty by default since we read the username from ssh config
	// this would force a specific user
	InstanceSSHUsername string

	Insecure bool
}

func (a *Compute) Name() string {
	return "GCP Compute Discover"
}

// TODO: try to auto-detect the current project, otherwise return an error
func (a *Compute) ListInstancesInProject(project string) ([]*asset.Asset, error) {
	svc, err := compute.NewService(context.Background(), option.WithHTTPClient(a.client))
	if err != nil {
		return nil, err
	}

	var wg sync.WaitGroup
	assets := []*asset.Asset{}

	log.Debug().Str("project", project).Msg("search for instances")
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
			zoneAssets, err := a.instancesPerZone(svc, project, zone)
			if err == nil {
				mux.Lock()
				assets = append(assets, zoneAssets...)
				mux.Unlock()
			}
			wg.Done()
		}(svc, project, zone)
	}

	wg.Wait()
	return assets, nil
}

func (a *Compute) ListInstances() ([]*asset.Asset, error) {
	ctx := context.Background()
	svc, err := compute.NewService(ctx, option.WithHTTPClient(a.client))
	if err != nil {
		return nil, err
	}

	resSrv, err := cloudresourcemanager.NewService(ctx)
	if err != nil {
		return nil, err
	}

	projectsResp, err := resSrv.Projects.List().Do()
	if err != nil {
		return nil, err
	}

	var wg sync.WaitGroup
	assets := []*asset.Asset{}
	for i := range projectsResp.Projects {
		project := projectsResp.Projects[i].ProjectId
		log.Debug().Str("project", project).Msg("search for instances")
		zones, err := svc.Zones.List(project).Do()
		if err != nil {
			log.Warn().Err(err).Str("project", project).Msg("could not fetch zones in project")
			continue
		}

		// add zones
		wg.Add(len(zones.Items))
		mux := &sync.Mutex{}
		for j := range zones.Items {
			zone := zones.Items[j].Name
			go func(svc *compute.Service, project string, zone string) {
				zoneAssets, err := a.instancesPerZone(svc, project, zone)
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

func (a *Compute) instancesPerZone(svc *compute.Service, project string, zone string) ([]*asset.Asset, error) {
	log.Debug().Str("project", project).Str("zone", zone).Msg("search gcp project for assets")
	il, err := svc.Instances.List(project, zone).Do()
	if err != nil {
		return nil, err
	}

	instances := make([]*asset.Asset, len(il.Items))
	for i := range il.Items {
		instance := il.Items[i]

		connections := []*transports.TransportConfig{}

		// TODO: we may want to filter windows instances, use guestOsFeatures to identify the system
		// "guestOsFeatures": [{
		//     "type": "WINDOWS"
		// }, {
		//     "type": "VIRTIO_SCSI_MULTIQUEUE"
		// }, {
		//     "type": "MULTI_IP_SUBNET"
		// }],
		//
		// data, _ := json.Marshal(instance)
		// fmt.Println(string(data))

		// add external ip networkInterfaces[].accessConfigs[].natIP
		// https://cloud.google.com/compute/docs/reference/rest/v1/instances/get
		for ni := range instance.NetworkInterfaces {
			iface := instance.NetworkInterfaces[ni]
			for ac := range iface.AccessConfigs {
				if len(iface.AccessConfigs[ac].NatIP) > 0 {
					log.Debug().Str("instance", instance.Name).Str("ip", iface.AccessConfigs[ac].NatIP).Msg("found public ip")
					connections = append(connections, &transports.TransportConfig{
						Backend:  transports.TransportBackend_CONNECTION_SSH,
						User:     a.InstanceSSHUsername,
						Host:     iface.AccessConfigs[ac].NatIP,
						Insecure: a.Insecure,
					})
				}
			}
		}

		asset := &asset.Asset{
			PlatformIds: []string{MondooGcpInstanceID(project, zone, instance)},
			Name:        instance.Name,
			Platform: &platform.Platform{
				Kind:    transports.Kind_KIND_VIRTUAL_MACHINE,
				Runtime: transports.RUNTIME_GCP_COMPUTE,
			},
			Connections: connections,
			State:       mapInstanceState(instance.Status),
			Labels:      make(map[string]string),
		}

		for key := range instance.Labels {
			asset.Labels[key] = instance.Labels[key]
		}

		// fetch gcp specific metadata
		asset.Labels["gcp.mondoo.app/project"] = project
		asset.Labels["mondoo.app/region"] = zone
		asset.Labels["mondoo.app/instance"] = strconv.FormatUint(uint64(instance.Id), 10)

		instances[i] = asset
	}

	return instances, nil
}

func MondooGcpInstanceID(project string, zone string, instance *compute.Instance) string {
	return "//platformid.api.mondoo.app/runtime/gcp/compute/v1/projects/" + project + "/zones/" + zone + "/instances/" + strconv.FormatUint(uint64(instance.Id), 10)
}

func mapInstanceState(state string) asset.State {
	switch state {
	case "RUNNING":
		return asset.State_STATE_RUNNING
	case "PROVISIONING":
		return asset.State_STATE_PENDING
	case "STAGING":
		return asset.State_STATE_PENDING
	case "STOPPED":
		return asset.State_STATE_STOPPED
	case "STOPPING":
		return asset.State_STATE_STOPPING
	case "SUSPENDED":
		return asset.State_STATE_STOPPED
	case "SUSPENDING":
		return asset.State_STATE_STOPPING
	case "TERMINATED":
		return asset.State_STATE_TERMINATED
	default:
		log.Warn().Str("state", state).Msg("unknown gcp instance state")
		return asset.State_STATE_UNKNOWN
	}
}
