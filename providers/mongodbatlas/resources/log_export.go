// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"net/http"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
)

func (r *mqlMongodbatlas) pushBasedLogExport() (*mqlMongodbatlasPushBasedLogConfig, error) {
	pid, err := projectID(r.MqlRuntime)
	if err != nil {
		return nil, err
	}
	cfg, httpResp, err := atlasClient(r.MqlRuntime).PushBasedLogExportApi.GetPushBasedLogConfiguration(context.Background(), pid).Execute()
	if err != nil {
		// Push-based log export is an optional feature that is not configured on
		// every project; degrade to null rather than failing the scan when it is
		// unavailable or the credential cannot read it.
		if isAccessDenied(httpResp) || (httpResp != nil && httpResp.StatusCode == http.StatusNotFound) {
			r.PushBasedLogExport.State = plugin.StateIsSet | plugin.StateIsNull
			return nil, nil
		}
		return nil, err
	}

	res, err := CreateResource(r.MqlRuntime, "mongodbatlas.pushBasedLogConfig", map[string]*llx.RawData{
		"__id":       llx.StringData("mongodbatlas.pushBasedLogConfig/" + pid),
		"bucketName": llx.StringData(cfg.GetBucketName()),
		"iamRoleId":  llx.StringData(cfg.GetIamRoleId()),
		"prefixPath": llx.StringData(cfg.GetPrefixPath()),
		"state":      llx.StringData(cfg.GetState()),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlMongodbatlasPushBasedLogConfig), nil
}
