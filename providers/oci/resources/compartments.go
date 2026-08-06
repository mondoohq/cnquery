// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/mql/v13/providers/oci/connection"
)

// ociCompartmentPoolConcurrency bounds how many (region, compartment) list
// calls run at once. Compartment fan-out multiplies the request count by the
// compartment tree size, so this is deliberately higher than the plain
// per-region pool while still staying well inside OCI's per-tenancy rate
// limits.
const ociCompartmentPoolConcurrency = 10

// ociCompartmentLister lists resources of a single type inside one compartment
// in one region. Implementations return the already-constructed MQL resources.
type ociCompartmentLister func(ctx context.Context, region string, compartmentID string) ([]any, error)

// ociRunCompartmentRegionPool fans a lister out over every (region, compartment)
// pair in the tenancy and joins the results.
//
// Most OCI list APIs are scoped to a single compartment and, unlike Cloud Guard
// or Logging, offer no `compartmentIdInSubtree` flag. Listing only the tenancy
// root - which is what passing conn.TenantID() as the compartment does - then
// reports nothing at all for any tenancy that organizes its resources into
// child compartments, which is the normal way to run OCI. An empty result that
// looks authoritative is the worst outcome for an inventory tool, so the
// compartment tree is walked explicitly instead.
func ociRunCompartmentRegionPool(conn *connection.OciConnection, regions []any, list ociCompartmentLister) ([]any, error) {
	ctx := context.Background()

	compartments, err := conn.GetCompartments(ctx)
	if err != nil {
		return nil, err
	}

	jobs := make([]*jobpool.Job, 0, len(regions)*len(compartments))
	for _, region := range regions {
		regionResource, ok := region.(*mqlOciRegion)
		if !ok {
			return nil, errors.New("invalid region type")
		}
		regionID := regionResource.Id.Data

		for i := range compartments {
			compartmentID := stringValue(compartments[i].Id)
			if compartmentID == "" {
				continue
			}

			jobs = append(jobs, jobpool.NewJob(func() (jobpool.JobResult, error) {
				items, err := list(ctx, regionID, compartmentID)
				if err != nil {
					return nil, err
				}
				return jobpool.JobResult(items), nil
			}))
		}
	}

	return ociCollectCompartmentJobs(jobs)
}

// ociCollectCompartmentJobs joins the results of a compartment fan-out.
//
// Error handling has to distinguish three cases that all surface as an error
// from a single job:
//
//   - The service has no endpoint in this region. Expected; skipped, matching
//     ociRunRegionPool.
//   - The caller cannot read this one compartment. Also expected: OCI policies
//     are routinely written per-compartment, so a scanning principal that can
//     read ten of fifty compartments is correctly configured, not broken. The
//     compartment is skipped and counted.
//   - Anything else (throttling, 5xx, malformed request) is a real fault and is
//     returned, because silently dropping it would under-report resources.
//
// The one case that must not be swallowed is *every* compartment refusing
// access. That is an under-scoped token, and reporting it as an empty tenancy
// would turn a broken credential into a clean scan.
func ociCollectCompartmentJobs(jobs []*jobpool.Job) ([]any, error) {
	if len(jobs) == 0 {
		return []any{}, nil
	}

	poolOfJobs := jobpool.CreatePool(jobs, ociCompartmentPoolConcurrency)
	poolOfJobs.Run()

	res := []any{}
	var (
		hardErr   error
		deniedErr error
		denied    int
		succeeded int
	)
	for i := range poolOfJobs.Jobs {
		job := poolOfJobs.Jobs[i]
		if job.Err != nil {
			if ociRegionServiceUnavailable(job.Err) {
				log.Debug().Err(job.Err).Msg("skipping oci region where the service is unavailable")
				continue
			}
			if ociCompartmentInaccessible(job.Err) {
				denied++
				if deniedErr == nil {
					deniedErr = job.Err
				}
				continue
			}
			hardErr = errors.Join(hardErr, job.Err)
			continue
		}

		succeeded++
		items, ok := job.Result.([]any)
		if !ok {
			continue
		}
		res = append(res, items...)
	}

	if hardErr != nil {
		return nil, hardErr
	}

	if succeeded == 0 && denied > 0 {
		return nil, deniedErr
	}

	if denied > 0 {
		// Visible rather than silent: the result is a partial view of the
		// tenancy and the operator should know how partial.
		log.Warn().
			Int("compartments_denied", denied).
			Int("compartments_read", succeeded).
			Msg("oci: listed a partial view of the tenancy, some compartments were not readable")
	}

	return res, nil
}

// ociCompartmentInaccessible reports whether an error means "you cannot see
// into this particular compartment" rather than a tenancy-wide fault.
//
// OCI collapses "does not exist" and "you are not authorized" into the same
// NotAuthorizedOrNotFound response precisely so that a caller cannot probe for
// the existence of resources it has no access to, so 403 and 404 are both
// treated as an inaccessible compartment here. This is safe only because the
// caller checks that at least one compartment did succeed.
func ociCompartmentInaccessible(err error) bool {
	svcErr, ok := common.IsServiceError(err)
	if !ok {
		return false
	}
	switch svcErr.GetHTTPStatusCode() {
	case 403, 404:
		return true
	default:
		return false
	}
}
