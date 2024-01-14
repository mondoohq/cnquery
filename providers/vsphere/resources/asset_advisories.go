// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/resources"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/upstream/mvd"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v10/providers/vsphere/connection"
)

// fetches the vulnerability report and returns the full report
func (p *mqlAsset) vulnerabilityReport() (interface{}, error) {
	runtime := p.MqlRuntime

	mcc := runtime.Upstream
	if mcc == nil || mcc.ApiEndpoint == "" {
		return nil, resources.MissingUpstreamError{}
	}

	// get new mvd client
	scannerClient, err := mvd.NewAdvisoryScannerClient(mcc.ApiEndpoint, mcc.HttpClient, mcc.Plugins...)
	if err != nil {
		return nil, err
	}

	conn := runtime.Connection.(*connection.VsphereConnection)
	apiPackages := []*mvd.Package{}

	scanjob := &mvd.AnalyseAssetRequest{
		Platform: mvd.NewMvdPlatform(conn.Asset().Platform),
		Packages: apiPackages,
	}

	report, err := scannerClient.AnalyseAsset(context.Background(), scanjob)
	if err != nil {
		return nil, err
	}

	return convert.JsonToDict(report)
}

// FIXME: DEPRECATED, update in v10.0 vv
func (p *mqlPlatform) vulnerabilityReport() (interface{}, error) {
	runtime := p.MqlRuntime

	mcc := runtime.Upstream
	if mcc == nil || mcc.ApiEndpoint == "" {
		return nil, resources.MissingUpstreamError{}
	}

	// get new mvd client
	scannerClient, err := mvd.NewAdvisoryScannerClient(mcc.ApiEndpoint, mcc.HttpClient, mcc.Plugins...)
	if err != nil {
		return nil, err
	}

	conn := runtime.Connection.(*connection.VsphereConnection)
	apiPackages := []*mvd.Package{}

	scanjob := &mvd.AnalyseAssetRequest{
		Platform: mvd.NewMvdPlatform(conn.Asset().Platform),
		Packages: apiPackages,
	}

	report, err := scannerClient.AnalyseAsset(context.Background(), scanjob)
	if err != nil {
		return nil, err
	}

	return convert.JsonToDict(report)
}

// ^^
