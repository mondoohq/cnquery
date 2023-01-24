package gcp

import (
	"strings"
	"time"

	"go.mondoo.com/cnquery/resources"
	"google.golang.org/protobuf/types/known/timestamppb"
)

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
			name = segments[len(segments)-1]
			region = segments[10]
			project = segments[8]
			break
		}
	}
	return &assetIdentifier{name: name, region: region, project: project}
}
