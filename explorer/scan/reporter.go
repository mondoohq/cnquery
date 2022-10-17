package scan

import (
	"github.com/hashicorp/go-multierror"
	"go.mondoo.com/cnquery/explorer"
	"go.mondoo.com/cnquery/motor/asset"
)

type Reporter interface {
	AddReport(asset *asset.Asset, results *AssetReport)
	AddScanError(asset *asset.Asset, err error)
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

func NewAggregateReporter(assetList []*asset.Asset) *AggregateReporter {
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

func (r *AggregateReporter) AddReport(asset *asset.Asset, results *AssetReport) {
	r.assetReports[asset.Mrn] = results.Report
	r.resolved[asset.Mrn] = results.Resolved
	r.bundle = results.Bundle
}

func (r *AggregateReporter) AddScanError(asset *asset.Asset, err error) {
	r.assetErrors[asset.Mrn] = err
}

func (r *AggregateReporter) Reports() *explorer.ReportCollection {
	errors := make(map[string]string, len(r.assetErrors))
	for k, v := range r.assetErrors {
		errors[k] = v.Error()
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
	var err error

	for _, curError := range r.assetErrors {
		err = multierror.Append(err, curError)
	}
	return err
}
