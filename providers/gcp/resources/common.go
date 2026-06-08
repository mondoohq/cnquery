// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/gcp/connection"

	"google.golang.org/api/googleapi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// certExpired reports whether an expiry timestamp is in the past. A nil
// timestamp (for example a not-yet-provisioned managed certificate) reports
// false.
func certExpired(expireTime plugin.TValue[*time.Time]) (bool, error) {
	if expireTime.Error != nil {
		return false, expireTime.Error
	}
	if expireTime.Data == nil {
		return false, nil
	}
	return expireTime.Data.Before(time.Now()), nil
}

// certDaysUntilExpiry returns the whole number of days until an expiry
// timestamp, rounded down toward expiry: a value of 0 means the certificate
// expires within the next 24 hours, and the result goes negative once the
// certificate has expired. A nil timestamp returns 0.
func certDaysUntilExpiry(expireTime plugin.TValue[*time.Time]) (int64, error) {
	if expireTime.Error != nil {
		return 0, expireTime.Error
	}
	if expireTime.Data == nil {
		return 0, nil
	}
	const day = 24 * time.Hour
	remaining := time.Until(*expireTime.Data)
	days := remaining / day
	// Integer division truncates toward zero; floor negative durations so an
	// already-expired certificate consistently reports a negative day count.
	if remaining < 0 && remaining%day != 0 {
		days--
	}
	return int64(days), nil
}

// projectFromResourceName extracts the project id from a GCP resource name of
// the form "projects/{project}/...". Returns "" when no project segment is
// present.
func projectFromResourceName(name string) string {
	parts := strings.Split(name, "/")
	for i := 0; i+1 < len(parts); i++ {
		if parts[i] == "projects" {
			return parts[i+1]
		}
	}
	return ""
}

// resolveServiceAccountRef resolves a service account reference to the typed
// gcp.project.iamService.serviceAccount resource. The raw value may be a bare
// email or a full "projects/{project}/serviceAccounts/{email}" path; for a bare
// email the fallbackProjectId is used. Returns nil when the reference is empty
// or cannot be resolved to a project + email.
func resolveServiceAccountRef(runtime *plugin.Runtime, raw, fallbackProjectId string) (*mqlGcpProjectIamServiceServiceAccount, error) {
	if raw == "" {
		return nil, nil
	}
	projectId, email := fallbackProjectId, raw
	if idx := strings.Index(raw, "/serviceAccounts/"); idx != -1 {
		email = raw[idx+len("/serviceAccounts/"):]
		if p := projectFromResourceName(raw); p != "" {
			projectId = p
		}
	}
	if projectId == "" || email == "" {
		return nil, nil
	}
	res, err := NewResource(runtime, "gcp.project.iamService.serviceAccount", map[string]*llx.RawData{
		"projectId": llx.StringData(projectId),
		"email":     llx.StringData(email),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlGcpProjectIamServiceServiceAccount), nil
}

// protoToDict converts a protobuf message to a map[string]any suitable for use as a dict field.
// Returns nil for nil input, including typed nil interface values.
func protoToDict(msg proto.Message) (map[string]any, error) {
	if msg == nil {
		return nil, nil
	}
	if v := reflect.ValueOf(msg); v.Kind() == reflect.Ptr && v.IsNil() {
		return nil, nil
	}
	data, err := protojson.Marshal(msg)
	if err != nil {
		return nil, err
	}
	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// isHTTPSkippable returns true for REST API errors that indicate the API
// is not enabled, the caller lacks permission, or the resource is not found.
func isHTTPSkippable(err error) bool {
	gerr, ok := err.(*googleapi.Error)
	if !ok {
		return false
	}
	if gerr.Code == 403 || gerr.Code == 404 {
		return true
	}
	return strings.Contains(gerr.Message, "not enabled") || strings.Contains(gerr.Message, "has not been used")
}

// isGRPCSkippable returns true for gRPC errors that indicate the API
// is not enabled, the caller lacks permission, or the resource is not found.
func isGRPCSkippable(err error) bool {
	if s, ok := status.FromError(err); ok {
		switch s.Code() {
		case codes.PermissionDenied, codes.Unimplemented, codes.NotFound:
			return true
		}
	}
	return false
}

func (g *mqlGcpRetryConfig) id() (string, error) {
	return g.Id.Data, g.Id.Error
}

func newRetryConfigResource(runtime *plugin.Runtime, parentId string, maxAttempts int64, minBackoff, maxBackoff string, maxDoublings int64, maxRetryDuration string) (*mqlGcpRetryConfig, error) {
	res, err := CreateResource(runtime, "gcp.retryConfig", map[string]*llx.RawData{
		"id":               llx.StringData(parentId + "/retryConfig"),
		"maxAttempts":      llx.IntData(maxAttempts),
		"minBackoff":       llx.StringData(minBackoff),
		"maxBackoff":       llx.StringData(maxBackoff),
		"maxDoublings":     llx.IntData(maxDoublings),
		"maxRetryDuration": llx.StringData(maxRetryDuration),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlGcpRetryConfig), nil
}

func RegionNameFromRegionUrl(regionUrl string) string {
	regionUrlSegments := strings.Split(regionUrl, "/")
	return regionUrlSegments[len(regionUrlSegments)-1]
}

// zoneNamesFromUrls extracts the zone name (last path segment) from a list of
// full zone URLs (e.g. ".../zones/us-central1-a" -> "us-central1-a").
func zoneNamesFromUrls(urls []string) []any {
	res := make([]any, 0, len(urls))
	for _, u := range urls {
		segments := strings.Split(u, "/")
		res = append(res, segments[len(segments)-1])
	}
	return res
}

func timestampAsTimePtr(t *timestamppb.Timestamp) *time.Time {
	if t == nil {
		return nil
	}
	tm := t.AsTime()
	return &tm
}

func boolValueToPtr(b *wrapperspb.BoolValue) *bool {
	if b == nil {
		return nil
	}
	v := b.GetValue()
	return &v
}

// parseResourceName returns the name of a resource from either a full path or just the name.
func parseResourceName(fullPath string) string {
	segments := strings.Split(fullPath, "/")
	return segments[len(segments)-1]
}

// parseProjectFromPath extracts the project ID from a GCP resource path.
// The path format is: projects/{project}/...
// Returns "" if no project segment is found.
func parseProjectFromPath(fullPath string) string {
	segments := strings.Split(fullPath, "/")
	for i, s := range segments {
		if s == "projects" && i+1 < len(segments) {
			return segments[i+1]
		}
	}
	return ""
}

// parseLocationFromPath extracts the location/region from a GCP resource path.
// The path format is: projects/{project}/locations/{location}/...
// Returns "global" if no location segment is found.
func parseLocationFromPath(fullPath string) string {
	segments := strings.Split(fullPath, "/")
	for i, s := range segments {
		if s == "locations" && i+1 < len(segments) {
			return segments[i+1]
		}
	}
	return "global"
}

type assetIdentifier struct {
	name    string
	region  string
	project string
}

func getAssetIdentifier(runtime *plugin.Runtime) *assetIdentifier {
	conn, ok := runtime.Connection.(*connection.GcpConnection)
	if !ok {
		return nil
	}
	id := conn.Asset().PlatformIds[0]

	if strings.HasPrefix(id, "//platformid.api.mondoo.app/runtime/gcp/") {
		// "//platformid.api.mondoo.app/runtime/gcp/{o.service}/v1/projects/{project}/regions/{region}/{objectType}/{name}"
		segments := strings.Split(id, "/")
		if len(segments) < 12 {
			return nil
		}
		name := segments[len(segments)-1]
		region := segments[10]
		project := segments[8]
		return &assetIdentifier{name: name, region: region, project: project}
	}

	return nil
}

type resourceId struct {
	Project string
	Region  string
	Name    string
}

func getDiskByUrl(diskUrl string, runtime *plugin.Runtime) (*mqlGcpProjectComputeServiceDisk, error) {
	if diskUrl == "" {
		return nil, nil
	}

	// URL format: https://www.googleapis.com/compute/v1/projects/{project}/zones/{zone}/disks/{disk}
	//          or https://compute.googleapis.com/compute/v1/projects/{project}/zones/{zone}/disks/{disk}
	// Also handles regional disks: .../projects/{project}/regions/{region}/disks/{disk}
	params := diskUrl
	switch {
	case strings.HasPrefix(params, "https://www.googleapis.com/compute/v1/"):
		params = strings.TrimPrefix(params, "https://www.googleapis.com/compute/v1/")
	case strings.HasPrefix(params, "https://compute.googleapis.com/compute/v1/"):
		params = strings.TrimPrefix(params, "https://compute.googleapis.com/compute/v1/")
	default:
		return nil, errors.New("unrecognized source disk URL prefix: " + diskUrl)
	}
	parts := strings.Split(params, "/")
	// Expect at least: projects/{project}/{zones|regions}/{loc}/disks/{disk}
	if len(parts) < 6 {
		return nil, errors.New("invalid source disk URL: " + diskUrl)
	}

	res, err := NewResource(runtime, "gcp.project.computeService.disk", map[string]*llx.RawData{
		"name":      llx.StringData(parts[len(parts)-1]),
		"projectId": llx.StringData(parts[1]),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlGcpProjectComputeServiceDisk), nil
}

// getNetworkByUrl resolves a typed network resource from either of the two
// reference formats GCP APIs hand back:
//
//   - Self-link URL:  https://www.googleapis.com/compute/v1/projects/{project}/global/networks/{name}
//     (also https://compute.googleapis.com/compute/v1/...)
//     Used by Compute, Memorystore PSC connections, etc.
//
//   - Bare resource name:  projects/{project}/global/networks/{name}
//     Used by Datastream's VpcPeeringConfig.Vpc, Memcache's AuthorizedNetwork, etc.
//
// After the prefix strip, both shapes collapse to the same path layout and
// can be split into 5 segments: ["projects", project, "global", "networks", name].
// Anything that doesn't fit that shape (empty, malformed) is rejected.
func getNetworkByUrl(networkUrl string, runtime *plugin.Runtime) (*mqlGcpProjectComputeServiceNetwork, error) {
	// A reference to a network is not mandatory for this resource
	if networkUrl == "" {
		return nil, nil
	}

	params := strings.TrimPrefix(networkUrl, "https://www.googleapis.com/compute/v1/")
	params = strings.TrimPrefix(params, "https://compute.googleapis.com/compute/v1/")
	parts := strings.Split(params, "/")
	if len(parts) < 5 || parts[0] != "projects" || parts[3] != "networks" {
		return nil, fmt.Errorf("unrecognized network reference: %q", networkUrl)
	}
	project, name := parts[1], parts[4]

	// Use NewResource so initGcpProjectComputeServiceNetwork runs and
	// populates every field (scalars like autoCreateSubnetworks would
	// otherwise surface as "no type information" when accessed).
	res, err := NewResource(runtime, "gcp.project.computeService.network", map[string]*llx.RawData{
		"name":      llx.StringData(name),
		"projectId": llx.StringData(project),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlGcpProjectComputeServiceNetwork), nil
}

func getSubnetworkByUrl(subnetUrl string, runtime *plugin.Runtime) (*mqlGcpProjectComputeServiceSubnetwork, error) {
	// A reference to a subnetwork is not mandatory for this resource
	if subnetUrl == "" {
		return nil, nil
	}

	// Format is https://www.googleapis.com/compute/v1/projects/project1/regions/us-central1/subnetworks/subnet-1
	// or https://compute.googleapis.com/compute/v1/projects/project1/regions/us-central1/subnetworks/subnet-1
	params := strings.TrimPrefix(subnetUrl, "https://www.googleapis.com/compute/v1/")
	params = strings.TrimPrefix(params, "https://compute.googleapis.com/compute/v1/")
	parts := strings.Split(params, "/")
	resId := resourceId{Project: parts[1], Region: parts[3], Name: parts[5]}
	// regionUrl is the full URL up to and including the region segment
	regionUrl := "https://www.googleapis.com/compute/v1/projects/" + resId.Project + "/regions/" + resId.Region

	// Use NewResource so initGcpProjectComputeServiceSubnetwork runs and
	// populates every field instead of leaving the resource partially set.
	res, err := NewResource(runtime, "gcp.project.computeService.subnetwork", map[string]*llx.RawData{
		"name":      llx.StringData(resId.Name),
		"projectId": llx.StringData(resId.Project),
		"regionUrl": llx.StringData(regionUrl),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlGcpProjectComputeServiceSubnetwork), nil
}

func getDiskIdByUrl(diskUrl string) (*resourceId, error) {
	// A reference to a subnetwork is not mandatory for this resource
	if diskUrl == "" {
		return nil, errors.New("diskUrl is empty")
	}

	// Format is https://www.googleapis.com/compute/v1/projects/project1/regions/us-central1/disks/disk-1
	// or https://compute.googleapis.com/compute/v1/projects/project1/regions/us-central1/disks/disk-1
	params := strings.TrimPrefix(diskUrl, "https://www.googleapis.com/compute/v1/")
	params = strings.TrimPrefix(params, "https://compute.googleapis.com/compute/v1/")
	parts := strings.Split(params, "/")
	return &resourceId{Project: parts[1], Region: parts[3], Name: parts[5]}, nil
}

// parseProjectFromComputeUrl extracts the project id from a Compute self-link.
// Returns "" when the URL doesn't match either compute self-link prefix.
func parseProjectFromComputeUrl(url string) string {
	params := strings.TrimPrefix(url, "https://www.googleapis.com/compute/v1/")
	params = strings.TrimPrefix(params, "https://compute.googleapis.com/compute/v1/")
	parts := strings.Split(params, "/")
	if len(parts) < 2 || parts[0] != "projects" {
		return ""
	}
	return parts[1]
}

// getRegionByUrl resolves the typed region resource matching a region URL.
// Returns (nil, nil) for empty URLs (global resources). The init function on
// `gcp.project.computeService.region` does the actual fetch — either via
// `Regions.Get(projectId, name)` when the region isn't already in the
// runtime cache, or by returning the cached entry if `regions()` was
// previously listed on the same project.
func getRegionByUrl(regionUrl string, runtime *plugin.Runtime) (*mqlGcpProjectComputeServiceRegion, error) {
	if regionUrl == "" {
		return nil, nil
	}
	regionName := RegionNameFromRegionUrl(regionUrl)
	if regionName == "" {
		return nil, fmt.Errorf("could not extract region name from %q", regionUrl)
	}
	projectId := parseProjectFromComputeUrl(regionUrl)
	if projectId == "" {
		return nil, fmt.Errorf("could not extract project id from region url %q", regionUrl)
	}
	res, err := NewResource(runtime, "gcp.project.computeService.region", map[string]*llx.RawData{
		"name":      llx.StringData(regionName),
		"projectId": llx.StringData(projectId),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlGcpProjectComputeServiceRegion), nil
}

// getZoneByUrl resolves the typed zone resource matching a zone URL.
// Returns (nil, nil) for empty URLs (regional resources). Resolution goes
// through the zone resource's init function (single `Zones.Get` call, or
// runtime cache hit if `zones()` was previously listed).
func getZoneByUrl(zoneUrl string, runtime *plugin.Runtime) (*mqlGcpProjectComputeServiceZone, error) {
	if zoneUrl == "" {
		return nil, nil
	}
	segments := strings.Split(zoneUrl, "/")
	zoneName := segments[len(segments)-1]
	if zoneName == "" {
		return nil, fmt.Errorf("could not extract zone name from %q", zoneUrl)
	}
	projectId := parseProjectFromComputeUrl(zoneUrl)
	if projectId == "" {
		return nil, fmt.Errorf("could not extract project id from zone url %q", zoneUrl)
	}
	res, err := NewResource(runtime, "gcp.project.computeService.zone", map[string]*llx.RawData{
		"name":      llx.StringData(zoneName),
		"projectId": llx.StringData(projectId),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlGcpProjectComputeServiceZone), nil
}
