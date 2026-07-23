// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"net"
	"strings"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/rs/zerolog/log"
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

// ociRegionServiceUnavailable reports whether the error means the service has
// no endpoint in that region, either because it is not deployed there or the
// tenancy is not entitled to it. Such a region is an expected absence and is
// skipped so a tenancy-wide query still returns what does exist.
//
// It deliberately does not treat an authorization or throttling failure as an
// absence: those are real problems, and reporting a short list as authoritative
// is worse than reporting the error. See ociRunRegionPool, which is the main
// consumer.
func ociRegionServiceUnavailable(err error) bool {
	if svcErr, ok := common.IsServiceError(err); ok {
		// Only a 404 can mean "this service has no endpoint in this region".
		// 401 is a credential problem and 403 (NotAuthorizedOrNotFound) is the
		// standard IAM-policy gap, which OCI returns in *every* region -
		// swallowing either turns an under-scoped token into an authoritative
		// "this tenancy has no resources", which is worse than an error.
		if svcErr.GetHTTPStatusCode() == 404 {
			return true
		}
		// A service error carries a real API response, so the transport-level
		// signatures below cannot apply to it.
		return false
	}
	// Regions where the service is not deployed have no regional endpoint, so
	// the DNS lookup for the host fails with "no such host".
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return true
	}
	// Some regions publish a wildcard DNS record for a service that is not
	// actually deployed there, so the host resolves but the TCP connection
	// times out. Treat connection timeouts (and the deadline they surface as)
	// the same as an absent endpoint.
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}
	// The OCI SDK wraps transport errors in a type that does not implement
	// Unwrap, so errors.As/Is above can miss a timeout. Fall back to matching
	// the message for the unreachable-endpoint signatures.
	msg := strings.ToLower(err.Error())
	for _, s := range []string{
		"timeout", "timed out", "deadline exceeded",
		"no such host", "connection refused", "no route to host",
	} {
		if strings.Contains(msg, s) {
			return true
		}
	}
	return false
}

// ociRunRegionPool runs a set of per-region jobs and returns the union of the
// ones that succeeded, skipping only the regions where the service genuinely
// has no endpoint.
//
// The distinction matters in both directions. Failing the whole collection
// when any region errors turned one unsubscribed or undeployed region into a
// tenancy-wide failure. But skipping *every* error is just as wrong the other
// way: a 403 from an IAM gap, a 429 throttle or a 500 would silently
// under-report resources, and an authoritative-looking short list is worse
// than an error in an inventory tool.
//
// So ociRegionServiceUnavailable decides. An absent endpoint is an expected
// condition and is skipped; anything else is a real problem and is reported,
// joined across regions so a broken token names every region it affected.
func ociRunRegionPool(jobs []*jobpool.Job) ([]any, error) {
	poolOfJobs := jobpool.CreatePool(jobs, 5)
	poolOfJobs.Run()

	res := []any{}
	var hardErr error
	for i := range poolOfJobs.Jobs {
		job := poolOfJobs.Jobs[i]
		if job.Err != nil {
			if ociRegionServiceUnavailable(job.Err) {
				log.Debug().Err(job.Err).Msg("skipping oci region where the service is unavailable")
				continue
			}
			hardErr = errors.Join(hardErr, job.Err)
			continue
		}
		items, ok := job.Result.([]any)
		if !ok {
			continue
		}
		res = append(res, items...)
	}

	if hardErr != nil {
		return nil, hardErr
	}
	return res, nil
}
