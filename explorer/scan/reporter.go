// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package scan

import (
	"go.mondoo.com/cnquery/v10/explorer"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v10/utils/multierr"
)

type Reporter interface {
	AddReport(asset *inventory.Asset, results *AssetReport)
	AddScanError(asset *inventory.Asset, err error)
}

type AssetReport struct {
	Mrn      string
	Bundle   *explorer.Bundle
	Report   *explorer.Report
	Resolved *explorer.ResolvedPack
}

type AggregateReporter struct {
	assets       map[string]*explorer.Asset
	assetReports map[string]*explorer.Report
	assetErrors  map[string]error
	bundle       *explorer.Bundle
	resolved     map[string]*explorer.ResolvedPack
}

func NewAggregateReporter(assetList []*inventory.Asset) *AggregateReporter {
	assets := make(map[string]*explorer.Asset, len(assetList))
	for i := range assetList {
		cur := assetList[i]
		assets[cur.Mrn] = &explorer.Asset{
			Mrn:  cur.Mrn,
			Name: cur.Name,
		}
	}

	return &AggregateReporter{
		assets:       assets,
		assetReports: map[string]*explorer.Report{},
		assetErrors:  map[string]error{},
		resolved:     map[string]*explorer.ResolvedPack{},
	}
}

func (r *AggregateReporter) AddReport(asset *inventory.Asset, results *AssetReport) {
	r.assetReports[asset.Mrn] = results.Report
	r.resolved[asset.Mrn] = results.Resolved
	r.bundle = results.Bundle
}

func (r *AggregateReporter) AddScanError(asset *inventory.Asset, err error) {
	r.assetErrors[asset.Mrn] = err
}

func (r *AggregateReporter) Reports() *explorer.ReportCollection {
	errors := make(map[string]*explorer.ErrorStatus, len(r.assetErrors))
	for k, v := range r.assetErrors {
		errors[k] = explorer.NewErrorStatus(v)
	}

	return &explorer.ReportCollection{
		Assets:   r.assets,
		Reports:  r.assetReports,
		Errors:   errors,
		Bundle:   r.bundle,
		Resolved: r.resolved,
	}
}

func (r *AggregateReporter) Error() error {
	var err multierr.Errors
	for _, curError := range r.assetErrors {
		err.Add(curError)
	}
	return err.Deduplicate()
}
