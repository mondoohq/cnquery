// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/networkfirewall"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/mql/v13/providers/oci/connection"
	"go.mondoo.com/mql/v13/types"
)

func (o *mqlOciNetworkFirewall) id() (string, error) {
	return "oci.networkFirewall", nil
}

func (o *mqlOciNetworkFirewall) firewalls() ([]any, error) {
	conn := o.MqlRuntime.Connection.(*connection.OciConnection)

	ociResource, err := CreateResource(o.MqlRuntime, "oci", nil)
	if err != nil {
		return nil, err
	}
	oci := ociResource.(*mqlOci)
	list := oci.GetRegions()
	if list.Error != nil {
		return nil, list.Error
	}

	return ociRunRegionPool(o.getFirewalls(conn, list.Data))
}

func (o *mqlOciNetworkFirewall) getFirewalls(conn *connection.OciConnection, regions []any) []*jobpool.Job {
	ctx := context.Background()
	tasks := make([]*jobpool.Job, 0)
	for _, region := range regions {
		regionResource, ok := region.(*mqlOciRegion)
		if !ok {
			return jobErr(errors.New("invalid region type"))
		}
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("calling oci network firewall with region %s", regionResource.Id.Data)

			svc, err := conn.NetworkFirewallClient(regionResource.Id.Data)
			if err != nil {
				return nil, err
			}

			firewalls := []networkfirewall.NetworkFirewallSummary{}
			var page *string
			for {
				response, err := svc.ListNetworkFirewalls(ctx, networkfirewall.ListNetworkFirewallsRequest{
					CompartmentId: common.String(conn.TenantID()),
					Page:          page,
				})
				if err != nil {
					return nil, err
				}

				firewalls = append(firewalls, response.Items...)

				if response.OpcNextPage == nil {
					break
				}
				page = response.OpcNextPage
			}

			var res []any
			for i := range firewalls {
				fw := firewalls[i]

				var created *time.Time
				if fw.TimeCreated != nil {
					created = &fw.TimeCreated.Time
				}
				var timeUpdated *time.Time
				if fw.TimeUpdated != nil {
					timeUpdated = &fw.TimeUpdated.Time
				}

				mqlInstance, err := CreateResource(o.MqlRuntime, "oci.networkFirewall.firewall", map[string]*llx.RawData{
					"id":                 llx.StringDataPtr(fw.Id),
					"name":               llx.StringDataPtr(fw.DisplayName),
					"compartmentID":      llx.StringDataPtr(fw.CompartmentId),
					"ipv4Address":        llx.StringDataPtr(fw.Ipv4Address),
					"ipv6Address":        llx.StringDataPtr(fw.Ipv6Address),
					"shape":              llx.StringDataPtr(fw.Shape),
					"state":              llx.StringData(string(fw.LifecycleState)),
					"created":            llx.TimeDataPtr(created),
					"timeUpdated":        llx.TimeDataPtr(timeUpdated),
					"securityAttributes": llx.MapData(definedTagsToAny(fw.SecurityAttributes), types.Dict),
					"systemTags":         llx.MapData(definedTagsToAny(fw.SystemTags), types.Dict),
				})
				if err != nil {
					return nil, err
				}
				mqlFw := mqlInstance.(*mqlOciNetworkFirewallFirewall)
				mqlFw.cacheSubnetId = stringValue(fw.SubnetId)
				mqlFw.cachePolicyId = stringValue(fw.NetworkFirewallPolicyId)
				mqlFw.region = regionResource.Id.Data
				res = append(res, mqlFw)
			}

			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

type mqlOciNetworkFirewallFirewallInternal struct {
	cacheSubnetId string
	cachePolicyId string
	region        string
}

func (o *mqlOciNetworkFirewallFirewall) id() (string, error) {
	return "oci.networkFirewall.firewall/" + o.Id.Data, nil
}

func (o *mqlOciNetworkFirewallFirewall) healthStatus() (string, error) {
	conn := o.MqlRuntime.Connection.(*connection.OciConnection)
	svc, err := conn.NetworkFirewallClient(o.region)
	if err != nil {
		return "", err
	}
	resp, err := svc.GetNetworkFirewallHealthStatus(context.Background(), networkfirewall.GetNetworkFirewallHealthStatusRequest{
		NetworkFirewallId: common.String(o.Id.Data),
	})
	if err != nil {
		return "", err
	}
	return string(resp.Status), nil
}

func (o *mqlOciNetworkFirewallFirewall) subnet() (*mqlOciNetworkSubnet, error) {
	if o.cacheSubnetId == "" {
		o.Subnet.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	mqlSubnet, err := NewResource(o.MqlRuntime, "oci.network.subnet", map[string]*llx.RawData{
		"id": llx.StringData(o.cacheSubnetId),
	})
	if err != nil {
		return nil, err
	}
	return mqlSubnet.(*mqlOciNetworkSubnet), nil
}

func (o *mqlOciNetworkFirewallFirewall) policy() (*mqlOciNetworkFirewallPolicy, error) {
	if o.cachePolicyId == "" {
		o.Policy.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	mqlPolicy, err := NewResource(o.MqlRuntime, "oci.networkFirewall.policy", map[string]*llx.RawData{
		"id": llx.StringData(o.cachePolicyId),
	})
	if err != nil {
		return nil, err
	}
	return mqlPolicy.(*mqlOciNetworkFirewallPolicy), nil
}

func (o *mqlOciNetworkFirewall) policies() ([]any, error) {
	conn := o.MqlRuntime.Connection.(*connection.OciConnection)

	ociResource, err := CreateResource(o.MqlRuntime, "oci", nil)
	if err != nil {
		return nil, err
	}
	oci := ociResource.(*mqlOci)
	list := oci.GetRegions()
	if list.Error != nil {
		return nil, list.Error
	}

	return ociRunRegionPool(o.getPolicies(conn, list.Data))
}

func (o *mqlOciNetworkFirewall) getPolicies(conn *connection.OciConnection, regions []any) []*jobpool.Job {
	ctx := context.Background()
	tasks := make([]*jobpool.Job, 0)
	for _, region := range regions {
		regionResource, ok := region.(*mqlOciRegion)
		if !ok {
			return jobErr(errors.New("invalid region type"))
		}
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("calling oci network firewall policies with region %s", regionResource.Id.Data)

			svc, err := conn.NetworkFirewallClient(regionResource.Id.Data)
			if err != nil {
				return nil, err
			}

			policies := []networkfirewall.NetworkFirewallPolicySummary{}
			var page *string
			for {
				response, err := svc.ListNetworkFirewallPolicies(ctx, networkfirewall.ListNetworkFirewallPoliciesRequest{
					CompartmentId: common.String(conn.TenantID()),
					Page:          page,
				})
				if err != nil {
					return nil, err
				}

				policies = append(policies, response.Items...)

				if response.OpcNextPage == nil {
					break
				}
				page = response.OpcNextPage
			}

			var res []any
			for i := range policies {
				p := policies[i]

				var created *time.Time
				if p.TimeCreated != nil {
					created = &p.TimeCreated.Time
				}

				mqlInstance, err := CreateResource(o.MqlRuntime, "oci.networkFirewall.policy", map[string]*llx.RawData{
					"id":            llx.StringDataPtr(p.Id),
					"name":          llx.StringDataPtr(p.DisplayName),
					"compartmentID": llx.StringDataPtr(p.CompartmentId),
					"state":         llx.StringData(string(p.LifecycleState)),
					"created":       llx.TimeDataPtr(created),
					"systemTags":    llx.MapData(definedTagsToAny(p.SystemTags), types.Dict),
				})
				if err != nil {
					return nil, err
				}
				mqlInstance.(*mqlOciNetworkFirewallPolicy).region = regionResource.Id.Data
				res = append(res, mqlInstance)
			}

			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

type mqlOciNetworkFirewallPolicyInternal struct {
	region    string
	fetched   atomic.Bool
	detail    *networkfirewall.NetworkFirewallPolicy
	fetchLock sync.Mutex
}

func (o *mqlOciNetworkFirewallPolicy) fetchDetail() (*networkfirewall.NetworkFirewallPolicy, error) {
	if o.fetched.Load() {
		return o.detail, nil
	}
	o.fetchLock.Lock()
	defer o.fetchLock.Unlock()
	if o.fetched.Load() {
		return o.detail, nil
	}
	conn := o.MqlRuntime.Connection.(*connection.OciConnection)
	svc, err := conn.NetworkFirewallClient(o.region)
	if err != nil {
		return nil, err
	}
	resp, err := svc.GetNetworkFirewallPolicy(context.Background(), networkfirewall.GetNetworkFirewallPolicyRequest{
		NetworkFirewallPolicyId: common.String(o.Id.Data),
	})
	if err != nil {
		return nil, err
	}
	o.detail = &resp.NetworkFirewallPolicy
	o.fetched.Store(true)
	return o.detail, nil
}

func (o *mqlOciNetworkFirewallPolicy) description() (string, error) {
	detail, err := o.fetchDetail()
	if err != nil {
		return "", err
	}
	return stringValue(detail.Description), nil
}

func (o *mqlOciNetworkFirewallPolicy) attachedFirewallCount() (int64, error) {
	detail, err := o.fetchDetail()
	if err != nil {
		return 0, err
	}
	if detail.AttachedNetworkFirewallCount == nil {
		return 0, nil
	}
	return int64(*detail.AttachedNetworkFirewallCount), nil
}

func initOciNetworkFirewallPolicy(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}

	idVal := ociArgString(args, "id")
	if idVal == "" {
		return nil, nil, errors.New("id required to fetch oci.networkFirewall.policy")
	}

	obj, err := CreateResource(runtime, "oci.networkFirewall", nil)
	if err != nil {
		return nil, nil, err
	}
	nf := obj.(*mqlOciNetworkFirewall)

	rawPolicies := nf.GetPolicies()
	if rawPolicies.Error != nil {
		return nil, nil, rawPolicies.Error
	}

	for _, raw := range rawPolicies.Data {
		policy := raw.(*mqlOciNetworkFirewallPolicy)
		if policy.Id.Data == idVal {
			return args, policy, nil
		}
	}

	return nil, nil, errors.New("oci.networkFirewall.policy not found: " + idVal)
}

func (o *mqlOciNetworkFirewallPolicy) id() (string, error) {
	return "oci.networkFirewall.policy/" + o.Id.Data, nil
}

func (o *mqlOciNetworkFirewallPolicy) decryptionProfiles() ([]any, error) {
	conn := o.MqlRuntime.Connection.(*connection.OciConnection)
	svc, err := conn.NetworkFirewallClient(o.region)
	if err != nil {
		return nil, err
	}
	ctx := context.Background()

	// The list returns summaries only (name/type); the blocking booleans
	// require a per-profile Get.
	summaries := []networkfirewall.DecryptionProfileSummary{}
	var page *string
	for {
		resp, err := svc.ListDecryptionProfiles(ctx, networkfirewall.ListDecryptionProfilesRequest{
			NetworkFirewallPolicyId: common.String(o.Id.Data),
			Page:                    page,
		})
		if err != nil {
			return nil, err
		}
		summaries = append(summaries, resp.Items...)
		if resp.OpcNextPage == nil {
			break
		}
		page = resp.OpcNextPage
	}

	res := make([]any, 0, len(summaries))
	for i := range summaries {
		name := stringValue(summaries[i].Name)

		getResp, err := svc.GetDecryptionProfile(ctx, networkfirewall.GetDecryptionProfileRequest{
			NetworkFirewallPolicyId: common.String(o.Id.Data),
			DecryptionProfileName:   common.String(name),
		})
		if err != nil {
			return nil, err
		}

		fields := decryptionProfileFields(getResp.DecryptionProfile, summaries[i])
		fields["__id"] = llx.StringData(o.Id.Data + "/decryptionProfile/" + name)

		mqlProfile, err := CreateResource(o.MqlRuntime, "oci.networkFirewall.policy.decryptionProfile", fields)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlProfile)
	}
	return res, nil
}

// decryptionProfileFields maps an OCI decryption profile to its MQL fields.
// Forward-proxy profiles carry all ten blocking controls; inbound-inspection
// profiles carry only the three shared ones, so the certificate-validation
// controls stay null (llx.BoolDataPtr(nil)) for them. The caller adds the
// synthetic __id.
func decryptionProfileFields(dp networkfirewall.DecryptionProfile, summary networkfirewall.DecryptionProfileSummary) map[string]*llx.RawData {
	fields := map[string]*llx.RawData{
		"name": llx.StringData(stringValue(summary.Name)),
		// Forward-proxy-only fields are null on inbound profiles; the
		// forward-proxy case below overwrites them.
		"isExpiredCertificateBlocked":        llx.BoolDataPtr(nil),
		"isUntrustedIssuerBlocked":           llx.BoolDataPtr(nil),
		"isRevocationStatusTimeoutBlocked":   llx.BoolDataPtr(nil),
		"isUnknownRevocationStatusBlocked":   llx.BoolDataPtr(nil),
		"areCertificateExtensionsRestricted": llx.BoolDataPtr(nil),
		"isAutoIncludeAltName":               llx.BoolDataPtr(nil),
	}

	switch p := dp.(type) {
	case networkfirewall.SslForwardProxyProfile:
		fields["type"] = llx.StringData(string(networkfirewall.InspectionTypeSslForwardProxy))
		fields["description"] = llx.StringDataPtr(p.Description)
		fields["isUnsupportedVersionBlocked"] = llx.BoolDataPtr(p.IsUnsupportedVersionBlocked)
		fields["isUnsupportedCipherBlocked"] = llx.BoolDataPtr(p.IsUnsupportedCipherBlocked)
		fields["isOutOfCapacityBlocked"] = llx.BoolDataPtr(p.IsOutOfCapacityBlocked)
		fields["isExpiredCertificateBlocked"] = llx.BoolDataPtr(p.IsExpiredCertificateBlocked)
		fields["isUntrustedIssuerBlocked"] = llx.BoolDataPtr(p.IsUntrustedIssuerBlocked)
		fields["isRevocationStatusTimeoutBlocked"] = llx.BoolDataPtr(p.IsRevocationStatusTimeoutBlocked)
		fields["isUnknownRevocationStatusBlocked"] = llx.BoolDataPtr(p.IsUnknownRevocationStatusBlocked)
		fields["areCertificateExtensionsRestricted"] = llx.BoolDataPtr(p.AreCertificateExtensionsRestricted)
		fields["isAutoIncludeAltName"] = llx.BoolDataPtr(p.IsAutoIncludeAltName)
	case networkfirewall.SslInboundInspectionProfile:
		fields["type"] = llx.StringData(string(networkfirewall.InspectionTypeSslInboundInspection))
		fields["description"] = llx.StringDataPtr(p.Description)
		fields["isUnsupportedVersionBlocked"] = llx.BoolDataPtr(p.IsUnsupportedVersionBlocked)
		fields["isUnsupportedCipherBlocked"] = llx.BoolDataPtr(p.IsUnsupportedCipherBlocked)
		fields["isOutOfCapacityBlocked"] = llx.BoolDataPtr(p.IsOutOfCapacityBlocked)
	default:
		// Unknown profile type (an inspection type newer than the pinned SDK):
		// surface the summary type and report the controls as not blocking.
		// Null would make `isUnsupportedVersionBlocked &&
		// isUnsupportedCipherBlocked` evaluate to true, so a profile we could
		// not decode would pass a TLS-hygiene check it was never measured for.
		fields["type"] = llx.StringData(string(summary.Type))
		fields["description"] = llx.StringDataPtr(summary.Description)
		fields["isUnsupportedVersionBlocked"] = llx.BoolData(false)
		fields["isUnsupportedCipherBlocked"] = llx.BoolData(false)
		fields["isOutOfCapacityBlocked"] = llx.BoolData(false)
	}
	return fields
}
