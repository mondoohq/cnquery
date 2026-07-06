// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"strings"

	"github.com/oracle/oci-go-sdk/v65/common"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/jobpool"
)

func stringValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func boolValue(s *bool) bool {
	if s == nil {
		return false
	}
	return *s
}

func int64Value(i *int64) int64 {
	if i == nil {
		return 0
	}
	return *i
}

func intValue(i *int) int64 {
	if i == nil {
		return 0
	}
	return int64(*i)
}

// isOcid returns true if the string looks like a valid OCI resource identifier.
// OCI uses placeholder values like "ORACLE_MANAGED_KEY" for system-managed
// resources; those should not be resolved via init lookups.
func isOcid(s string) bool {
	return strings.HasPrefix(s, "ocid1.")
}

// ociRegionFromOCID extracts the region from an OCI resource OCID. OCIDs have
// the shape ocid1.<resourceType>.<realm>.<region>.<uniqueID>, so the region is
// the fourth dot-separated segment (e.g. "us-sanjose-1"). It is empty for
// global resources (ocid1.user.oc1..aaaa). Returns "" when the OCID is
// malformed or carries no region; callers should fall back to a known region.
func ociRegionFromOCID(ocid string) string {
	parts := strings.Split(ocid, ".")
	if len(parts) < 5 {
		return ""
	}
	return parts[3]
}

// ociResourceTypeFromOCID extracts the resource-type segment from an OCI OCID.
// OCIDs have the shape ocid1.<resourceType>.<realm>.<region>.<uniqueID>, so the
// type is the second dot-separated segment (e.g. "internetgateway", "drg",
// "natgateway"). Returns "" when the OCID is malformed.
func ociResourceTypeFromOCID(ocid string) string {
	parts := strings.Split(ocid, ".")
	if len(parts) < 2 {
		return ""
	}
	return parts[1]
}

// ociRouteTargetType maps a route rule's target OCID to the kind of network
// entity it forwards traffic to. Returns the uppercased raw OCID resource type
// for entity kinds without a dedicated route accessor, or "" for a malformed
// OCID.
func ociRouteTargetType(ocid string) string {
	switch ociResourceTypeFromOCID(ocid) {
	case "":
		return ""
	case "internetgateway":
		return "INTERNET_GATEWAY"
	case "natgateway":
		return "NAT_GATEWAY"
	case "servicegateway":
		return "SERVICE_GATEWAY"
	case "drg":
		return "DRG"
	case "localpeeringgateway":
		return "LOCAL_PEERING_GATEWAY"
	case "privateip":
		return "PRIVATE_IP"
	default:
		return strings.ToUpper(ociResourceTypeFromOCID(ocid))
	}
}

func jobErr(err error) []*jobpool.Job {
	return []*jobpool.Job{{Err: err}}
}

// sdkTimeData wraps an OCI SDKTime as RawData, returning NilData for nil.
func sdkTimeData(t *common.SDKTime) *llx.RawData {
	if t == nil {
		return llx.NilData
	}
	return llx.TimeData(t.Time)
}

// stringsToAny converts an OCI-typed []string to []any for llx.ArrayData.
func stringsToAny(in []string) []any {
	out := make([]any, len(in))
	for i, s := range in {
		out[i] = s
	}
	return out
}

// strMapToAny converts an OCI freeform-tags-style map[string]string to
// map[string]any so it can be passed to llx.MapData.
func strMapToAny(in map[string]string) map[string]any {
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

// definedTagsToAny converts the OCI defined-tags shape (namespace -> key -> value)
// to map[string]any (namespace -> map[string]any), preserving string values and
// passing through anything non-string unchanged.
func definedTagsToAny(in map[string]map[string]interface{}) map[string]any {
	out := make(map[string]any, len(in))
	for ns, kv := range in {
		nsOut := make(map[string]any, len(kv))
		for k, v := range kv {
			if s, ok := v.(string); ok {
				nsOut[k] = s
				continue
			}
			nsOut[k] = v
		}
		out[ns] = nsOut
	}
	return out
}
