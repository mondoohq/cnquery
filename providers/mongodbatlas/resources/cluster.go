// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/types"
)

func (r *mqlMongodbatlas) clusters() ([]any, error) {
	pid, err := projectID(r.MqlRuntime)
	if err != nil {
		return nil, err
	}
	client := atlasClient(r.MqlRuntime)
	ctx := context.Background()

	out := []any{}
	for page := 1; ; page++ {
		resp, _, err := client.ClustersApi.ListClusters(ctx, pid).ItemsPerPage(pageSize).PageNum(page).Execute()
		if err != nil {
			return nil, err
		}
		results := resp.GetResults()
		for i := range results {
			c := results[i]

			tlsMin := ""
			tlsMode := ""
			if adv, ok := c.GetAdvancedConfigurationOk(); ok {
				tlsMin = adv.GetMinimumEnabledTlsProtocol()
				tlsMode = adv.GetTlsCipherConfigMode()
			}

			regionConfigs := []any{}
			for _, spec := range c.GetReplicationSpecs() {
				for _, rc := range spec.GetRegionConfigs() {
					entry := map[string]any{
						"providerName": rc.GetProviderName(),
						"regionName":   rc.GetRegionName(),
						"priority":     int64(rc.GetPriority()),
					}
					if es, ok := rc.GetElectableSpecsOk(); ok {
						entry["instanceSize"] = es.GetInstanceSize()
						entry["nodeCount"] = int64(es.GetNodeCount())
						entry["diskSizeGB"] = es.GetDiskSizeGB()
					}
					regionConfigs = append(regionConfigs, entry)
				}
			}

			res, err := CreateResource(r.MqlRuntime, "mongodbatlas.cluster", map[string]*llx.RawData{
				"__id":                         llx.StringData("mongodbatlas.cluster/" + pid + "/" + c.GetName()),
				"id":                           llx.StringData(c.GetId()),
				"name":                         llx.StringData(c.GetName()),
				"mongoDBMajorVersion":          llx.StringData(c.GetMongoDBMajorVersion()),
				"mongoDBVersion":               llx.StringData(c.GetMongoDBVersion()),
				"clusterType":                  llx.StringData(c.GetClusterType()),
				"stateName":                    llx.StringData(c.GetStateName()),
				"backupEnabled":                llx.BoolData(c.GetBackupEnabled()),
				"pitEnabled":                   llx.BoolData(c.GetPitEnabled()),
				"encryptionAtRestProvider":     llx.StringData(c.GetEncryptionAtRestProvider()),
				"minimumEnabledTlsProtocol":    llx.StringData(tlsMin),
				"tlsCipherConfigMode":          llx.StringData(tlsMode),
				"redactClientLogData":          llx.BoolData(c.GetRedactClientLogData()),
				"terminationProtectionEnabled": llx.BoolData(c.GetTerminationProtectionEnabled()),
				"paused":                       llx.BoolData(c.GetPaused()),
				"versionReleaseSystem":         llx.StringData(c.GetVersionReleaseSystem()),
				"rootCertType":                 llx.StringData(c.GetRootCertType()),
				"createDate":                   llx.TimeDataPtr(timePtr(c.GetCreateDate())),
				"regionConfigs":                llx.ArrayData(regionConfigs, types.Dict),
			})
			if err != nil {
				return nil, err
			}
			out = append(out, res)
		}
		if len(results) < pageSize {
			break
		}
	}
	return out, nil
}
