package gcp

import (
	"strings"
	"time"

	"go.mondoo.com/cnquery/resources"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func RegionNameFromRegionUrl(regionUrl string) string {
	regionUrlSegments := strings.Split(regionUrl, "/")
	return regionUrlSegments[len(regionUrlSegments)-1]
}

func timestampAsTimePtr(t *timestamppb.Timestamp) *time.Time {
	if t == nil {
		return nil
	}
	tm := t.AsTime()
	return &tm
}

// parseResourceName returns the name of a resource from either a full path or just the name.
func parseResourceName(fullPath string) string {
	segments := strings.Split(fullPath, "/")
	return segments[len(segments)-1]
}

type assetIdentifier struct {
	name    string
	region  string
	project string
}

func getAssetIdentifier(runtime *resources.Runtime) *assetIdentifier {
	a := runtime.Motor.GetAsset()
	if a == nil {
		return nil
	}
	var name, region, project string
	for _, id := range a.PlatformIds {
		if strings.HasPrefix(id, "//platformid.api.mondoo.app/runtime/gcp/") {
			// "//platformid.api.mondoo.app/runtime/gcp/{o.service}/v1/projects/{project}/regions/{region}/{objectType}/{name}"
			segments := strings.Split(id, "/")
			if len(segments) < 12 {
				return nil
			}
			name = segments[len(segments)-1]
			region = segments[10]
			project = segments[8]
			break
		}
	}
	return &assetIdentifier{name: name, region: region, project: project}
}

type resourceId struct {
	Project string
	Region  string
	Name    string
}

func getNetworkByUrl(networkUrl string, runtime *resources.Runtime) (interface{}, error) {
	// A reference to a network is not mandatory for this resource
	if networkUrl == "" {
		return nil, nil
	}

	// Format is https://www.googleapis.com/compute/v1/projects/project1/global/networks/net-1
	params := strings.TrimPrefix(networkUrl, "https://www.googleapis.com/compute/v1/")
	parts := strings.Split(params, "/")
	resId := resourceId{Project: parts[1], Region: parts[2], Name: parts[4]}

	return runtime.CreateResource("gcp.project.computeService.network",
		"name", resId.Name,
		"projectId", resId.Project,
	)
}

func getSubnetworkByUrl(subnetUrl string, runtime *resources.Runtime) (interface{}, error) {
	// A reference to a subnetwork is not mandatory for this resource
	if subnetUrl == "" {
		return nil, nil
	}

	// Format is https://www.googleapis.com/compute/v1/projects/project1/regions/us-central1/subnetworks/subnet-1
	params := strings.TrimPrefix(subnetUrl, "https://www.googleapis.com/compute/v1/")
	parts := strings.Split(params, "/")
	resId := resourceId{Project: parts[1], Region: parts[3], Name: parts[5]}

	return runtime.CreateResource("gcp.project.computeService.subnetwork",
		"name", resId.Name,
		"projectId", resId.Project,
		"region", resId.Region,
	)
}
