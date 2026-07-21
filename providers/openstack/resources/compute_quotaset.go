// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/quotasets"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
)

func (o *mqlOpenstack) computeQuotaSet() (*mqlOpenstackComputeQuotaSet, error) {
	c := conn(o.MqlRuntime)
	client, err := c.ComputeClient()
	if err != nil {
		if serviceMissing(err) {
			o.ComputeQuotaSet.State = plugin.StateIsSet | plugin.StateIsNull
			return nil, nil
		}
		return nil, err
	}
	projectId := c.ProjectID()
	q, err := quotasets.Get(ctx(), client, projectId).Extract()
	if err != nil {
		if translateGetError(err) == nil {
			o.ComputeQuotaSet.State = plugin.StateIsSet | plugin.StateIsNull
			return nil, nil
		}
		return nil, err
	}
	res, err := CreateResource(o.MqlRuntime, "openstack.compute.quotaSet", map[string]*llx.RawData{
		"__id":                     llx.StringData("openstack.compute.quotaSet/" + projectId),
		"projectId":                llx.StringData(projectId),
		"instances":                llx.IntData(int64(q.Instances)),
		"cores":                    llx.IntData(int64(q.Cores)),
		"ram":                      llx.IntData(int64(q.RAM)),
		"keyPairs":                 llx.IntData(int64(q.KeyPairs)),
		"metadataItems":            llx.IntData(int64(q.MetadataItems)),
		"serverGroups":             llx.IntData(int64(q.ServerGroups)),
		"serverGroupMembers":       llx.IntData(int64(q.ServerGroupMembers)),
		"securityGroups":           llx.IntData(int64(q.SecurityGroups)),
		"securityGroupRules":       llx.IntData(int64(q.SecurityGroupRules)),
		"fixedIps":                 llx.IntData(int64(q.FixedIPs)),
		"floatingIps":              llx.IntData(int64(q.FloatingIPs)),
		"injectedFiles":            llx.IntData(int64(q.InjectedFiles)),
		"injectedFileContentBytes": llx.IntData(int64(q.InjectedFileContentBytes)),
		"injectedFilePathBytes":    llx.IntData(int64(q.InjectedFilePathBytes)),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlOpenstackComputeQuotaSet), nil
}

func (r *mqlOpenstackComputeQuotaSet) project() (*mqlOpenstackProject, error) {
	return resolveProject(r.MqlRuntime, r.ProjectId.Data, &r.Project)
}
