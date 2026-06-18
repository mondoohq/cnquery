// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"time"

	"github.com/stackitcloud/stackit-sdk-go/services/sfs"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
)

// The SDK exposes a distinct struct for the single-resource GET responses
// (e.g. GetResourcePoolResponseResourcePool) versus the LIST element
// (ResourcePool), even though they carry identical getters and nested types.
// These interfaces let one mapper serve both code paths.

type sfsResourcePoolData interface {
	GetId() string
	GetName() string
	GetState() string
	GetPerformanceClass() sfs.ResourcePoolPerformanceClass
	GetSpace() sfs.ResourcePoolSpace
	GetSnapshotPolicy() *sfs.NullableResourcePoolSnapshotPolicy
	GetAvailabilityZone() string
	GetMountPath() string
	GetCountShares() int64
	GetIpAcl() []string
	GetSnapshotsAreVisible() bool
	GetLabels() map[string]string
	GetCreatedAtOk() (time.Time, bool)
}

type sfsExportPolicyData interface {
	GetId() string
	GetName() string
	GetSharesUsingExportPolicy() int64
	GetLabels() map[string]string
	GetCreatedAtOk() (time.Time, bool)
}

// derefFloat returns the value behind a *float64, or 0 when nil. SFS reports
// the space-usage gauges as optional floats; an absent value reads as 0.
func derefFloat(p *float64) float64 {
	if p == nil {
		return 0
	}
	return *p
}

// ------------------------- SFS namespace -------------------------

func (r *mqlStackitSfs) resourcePools() ([]any, error) {
	c := conn(r.MqlRuntime)
	client, err := c.Sfs()
	if err != nil {
		return nil, err
	}
	resp, err := client.ListResourcePoolsExecute(bgctx(), c.ProjectID(), c.Region())
	if err != nil {
		if isAccessDenied(err) {
			return []any{}, nil
		}
		return nil, err
	}
	pools, _ := resp.GetResourcePoolsOk()
	out := make([]any, 0, len(pools))
	for i := range pools {
		res, err := CreateResource(r.MqlRuntime, "stackit.sfs.resourcePool", sfsResourcePoolArgs(c.Region(), &pools[i]))
		if err != nil {
			return nil, err
		}
		out = append(out, res)
	}
	return out, nil
}

func (r *mqlStackitSfs) exportPolicies() ([]any, error) {
	c := conn(r.MqlRuntime)
	client, err := c.Sfs()
	if err != nil {
		return nil, err
	}
	resp, err := client.ListShareExportPoliciesExecute(bgctx(), c.ProjectID(), c.Region())
	if err != nil {
		if isAccessDenied(err) {
			return []any{}, nil
		}
		return nil, err
	}
	policies, _ := resp.GetShareExportPoliciesOk()
	out := make([]any, 0, len(policies))
	for i := range policies {
		res, err := newSfsExportPolicy(r.MqlRuntime, c.Region(), &policies[i])
		if err != nil {
			return nil, err
		}
		out = append(out, res)
	}
	return out, nil
}

func (r *mqlStackitSfs) lockId() (string, error) {
	c := conn(r.MqlRuntime)
	client, err := c.Sfs()
	if err != nil {
		return "", err
	}
	resp, err := client.GetLockExecute(bgctx(), c.Region(), c.ProjectID())
	if err != nil {
		if isAccessDenied(err) || isNotFound(err) {
			return "", nil
		}
		return "", err
	}
	return resp.GetLockId(), nil
}

// ------------------------- SFS resource pool -------------------------

func sfsResourcePoolArgs(region string, rp sfsResourcePoolData) map[string]*llx.RawData {
	pc := rp.GetPerformanceClass()
	sp := rp.GetSpace()
	snapPolicyID, snapPolicyName := "", ""
	if np := rp.GetSnapshotPolicy(); np != nil && np.IsSet() {
		if v := np.Get(); v != nil {
			snapPolicyID = v.GetId()
			snapPolicyName = v.GetName()
		}
	}
	return map[string]*llx.RawData{
		"id":                         llx.StringData(rp.GetId()),
		"name":                       llx.StringData(rp.GetName()),
		"state":                      llx.StringData(rp.GetState()),
		"region":                     llx.StringData(region),
		"performanceClass":           llx.StringData(pc.GetName()),
		"performanceClassPeakIops":   llx.IntData(pc.GetPeakIops()),
		"performanceClassThroughput": llx.IntData(pc.GetThroughput()),
		"availabilityZone":           llx.StringData(rp.GetAvailabilityZone()),
		"mountPath":                  llx.StringData(rp.GetMountPath()),
		"countShares":                llx.IntData(rp.GetCountShares()),
		"sizeGigabytes":              llx.IntData(sp.GetSizeGigabytes()),
		"usedGigabytes":              llx.FloatData(derefFloat(sp.GetUsedGigabytes())),
		"availableGigabytes":         llx.FloatData(derefFloat(sp.GetAvailableGigabytes())),
		"usedBySnapshotsGigabytes":   llx.FloatData(derefFloat(sp.GetUsedBySnapshotsGigabytes())),
		"ipAcl":                      strSliceData(rp.GetIpAcl()),
		"snapshotsAreVisible":        llx.BoolData(rp.GetSnapshotsAreVisible()),
		"snapshotPolicyId":           llx.StringData(snapPolicyID),
		"snapshotPolicyName":         llx.StringData(snapPolicyName),
		"labels":                     labelData(rp.GetLabels()),
		"createdAt":                  llx.TimeDataPtr(timeOrNil(rp.GetCreatedAtOk())),
	}
}

func (r *mqlStackitSfsResourcePool) id() (string, error) {
	return "stackit.sfs.resourcePool/" + r.Id.Data, nil
}

func (r *mqlStackitSfsResourcePool) shares() ([]any, error) {
	c := conn(r.MqlRuntime)
	client, err := c.Sfs()
	if err != nil {
		return nil, err
	}
	resp, err := client.ListSharesExecute(bgctx(), c.ProjectID(), r.Region.Data, r.Id.Data)
	if err != nil {
		if isAccessDenied(err) {
			return []any{}, nil
		}
		return nil, err
	}
	shares, _ := resp.GetSharesOk()
	out := make([]any, 0, len(shares))
	for i := range shares {
		sh := shares[i]
		exportPolicyID := ""
		if np := sh.GetExportPolicy(); np != nil && np.IsSet() {
			if v := np.Get(); v != nil {
				exportPolicyID = v.GetId()
			}
		}
		res, err := CreateResource(r.MqlRuntime, "stackit.sfs.share", map[string]*llx.RawData{
			"id":                      llx.StringData(sh.GetId()),
			"name":                    llx.StringData(sh.GetName()),
			"state":                   llx.StringData(sh.GetState()),
			"mountPath":               llx.StringData(sh.GetMountPath()),
			"spaceHardLimitGigabytes": llx.IntData(sh.GetSpaceHardLimitGigabytes()),
			"labels":                  labelData(sh.GetLabels()),
			"createdAt":               llx.TimeDataPtr(timeOrNil(sh.GetCreatedAtOk())),
		})
		if err != nil {
			return nil, err
		}
		share := res.(*mqlStackitSfsShare)
		share.cacheExportPolicyID = exportPolicyID
		share.cacheRegion = r.Region.Data
		out = append(out, share)
	}
	return out, nil
}

func (r *mqlStackitSfsResourcePool) snapshots() ([]any, error) {
	c := conn(r.MqlRuntime)
	client, err := c.Sfs()
	if err != nil {
		return nil, err
	}
	resp, err := client.ListResourcePoolSnapshotsExecute(bgctx(), c.ProjectID(), r.Region.Data, r.Id.Data)
	if err != nil {
		if isAccessDenied(err) {
			return []any{}, nil
		}
		return nil, err
	}
	snaps, _ := resp.GetResourcePoolSnapshotsOk()
	out := make([]any, 0, len(snaps))
	for i := range snaps {
		snap := snaps[i]
		res, err := CreateResource(r.MqlRuntime, "stackit.sfs.snapshot", map[string]*llx.RawData{
			"__id":                 llx.StringData("stackit.sfs.snapshot/" + r.Id.Data + "/" + snap.GetSnapshotName()),
			"name":                 llx.StringData(snap.GetSnapshotName()),
			"sizeGigabytes":        llx.IntData(snap.GetSizeGigabytes()),
			"logicalSizeGigabytes": llx.IntData(snap.GetLogicalSizeGigabytes()),
			"comment":              llx.StringData(ptrStr(snap.GetComment())),
			"snaplockExpiryTime":   llx.TimeDataPtr(snap.GetSnaplockExpiryTime()),
			"createdAt":            llx.TimeDataPtr(timeOrNil(snap.GetCreatedAtOk())),
		})
		if err != nil {
			return nil, err
		}
		out = append(out, res)
	}
	return out, nil
}

func initStackitSfsResourcePool(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	id, ok := idArg(args, "id")
	if !ok {
		return args, nil, nil
	}
	c := conn(runtime)
	client, err := c.Sfs()
	if err != nil {
		return nil, nil, err
	}
	resp, err := client.GetResourcePoolExecute(bgctx(), c.ProjectID(), c.Region(), id)
	if err != nil {
		return nil, nil, err
	}
	pool, ok := resp.GetResourcePoolOk()
	if !ok {
		return args, nil, nil
	}
	res, err := CreateResource(runtime, "stackit.sfs.resourcePool", sfsResourcePoolArgs(c.Region(), &pool))
	if err != nil {
		return nil, nil, err
	}
	return nil, res, nil
}

// ------------------------- SFS share -------------------------

type mqlStackitSfsShareInternal struct {
	cacheExportPolicyID string
	cacheRegion         string
}

func (r *mqlStackitSfsShare) id() (string, error) {
	return "stackit.sfs.share/" + r.Id.Data, nil
}

func (r *mqlStackitSfsShare) exportPolicy() (*mqlStackitSfsExportPolicy, error) {
	if r.cacheExportPolicyID == "" {
		r.ExportPolicy.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	res, err := NewResource(r.MqlRuntime, "stackit.sfs.exportPolicy", map[string]*llx.RawData{
		"id": llx.StringData(r.cacheExportPolicyID),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlStackitSfsExportPolicy), nil
}

// ------------------------- SFS share export policy -------------------------

type mqlStackitSfsExportPolicyInternal struct {
	cacheRegion string
}

func newSfsExportPolicy(runtime *plugin.Runtime, region string, p sfsExportPolicyData) (*mqlStackitSfsExportPolicy, error) {
	res, err := CreateResource(runtime, "stackit.sfs.exportPolicy", map[string]*llx.RawData{
		"id":                      llx.StringData(p.GetId()),
		"name":                    llx.StringData(p.GetName()),
		"sharesUsingExportPolicy": llx.IntData(p.GetSharesUsingExportPolicy()),
		"labels":                  labelData(p.GetLabels()),
		"createdAt":               llx.TimeDataPtr(timeOrNil(p.GetCreatedAtOk())),
	})
	if err != nil {
		return nil, err
	}
	ep := res.(*mqlStackitSfsExportPolicy)
	ep.cacheRegion = region
	return ep, nil
}

func (r *mqlStackitSfsExportPolicy) id() (string, error) {
	return "stackit.sfs.exportPolicy/" + r.Id.Data, nil
}

func (r *mqlStackitSfsExportPolicy) rules() ([]any, error) {
	c := conn(r.MqlRuntime)
	client, err := c.Sfs()
	if err != nil {
		return nil, err
	}
	region := r.cacheRegion
	if region == "" {
		region = c.Region()
	}
	resp, err := client.GetShareExportPolicyExecute(bgctx(), c.ProjectID(), region, r.Id.Data)
	if err != nil {
		if isAccessDenied(err) {
			return []any{}, nil
		}
		return nil, err
	}
	pol, ok := resp.GetShareExportPolicyOk()
	if !ok {
		return []any{}, nil
	}
	rules, _ := pol.GetRulesOk()
	out := make([]any, 0, len(rules))
	for i := range rules {
		rule := rules[i]
		res, err := CreateResource(r.MqlRuntime, "stackit.sfs.exportPolicy.rule", map[string]*llx.RawData{
			"id":          llx.StringData(rule.GetId()),
			"order":       llx.IntData(rule.GetOrder()),
			"ipAcl":       strSliceData(rule.GetIpAcl()),
			"description": llx.StringData(ptrStr(rule.GetDescription())),
			"createdAt":   llx.TimeDataPtr(timeOrNil(rule.GetCreatedAtOk())),
		})
		if err != nil {
			return nil, err
		}
		out = append(out, res)
	}
	return out, nil
}

func (r *mqlStackitSfsExportPolicyRule) id() (string, error) {
	return "stackit.sfs.exportPolicy.rule/" + r.Id.Data, nil
}

func initStackitSfsExportPolicy(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	id, ok := idArg(args, "id")
	if !ok {
		return args, nil, nil
	}
	c := conn(runtime)
	client, err := c.Sfs()
	if err != nil {
		return nil, nil, err
	}
	resp, err := client.GetShareExportPolicyExecute(bgctx(), c.ProjectID(), c.Region(), id)
	if err != nil {
		return nil, nil, err
	}
	pol, ok := resp.GetShareExportPolicyOk()
	if !ok {
		return args, nil, nil
	}
	res, err := newSfsExportPolicy(runtime, c.Region(), &pol)
	if err != nil {
		return nil, nil, err
	}
	return nil, res, nil
}
