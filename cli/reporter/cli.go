// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package reporter

import (
	"fmt"
	"io"
	"sort"
	"strconv"

	"github.com/muesli/termenv"
	"go.mondoo.com/cnquery/v10/explorer"
	"go.mondoo.com/cnquery/v10/llx"
	"go.mondoo.com/cnquery/v10/mrn"
	"go.mondoo.com/cnquery/v10/utils/stringx"
)

type cliReporter struct {
	*Reporter
	isCompact bool
	isSummary bool
	out       io.Writer
	data      *explorer.ReportCollection

	// vv the items below will be automatically filled
	bundle *explorer.BundleMap
}

func (r *cliReporter) print() error {
	// catch case where the scan was not successful and no bundle was fetched from server
	if r.data == nil {
		return nil
	}

	if r.data.Bundle != nil {
		r.bundle = r.data.Bundle.ToMap()
	} else {
		r.bundle = nil
	}

	if !r.isSummary && r.bundle != nil {
		r.printQueryData()
	}

	r.printSummary()
	return nil
}

func (r *cliReporter) printSummary() {
	r.out.Write([]byte(r.Printer.H1("Summary (" + strconv.Itoa(len(r.data.Assets)) + " assets)")))

	for mrn, asset := range r.data.Assets {
		r.printAssetSummary(mrn, asset)
	}
}

func (r *cliReporter) printAssetSummary(assetMrn string, asset *explorer.Asset) {
	target := asset.Name
	if target == "" {
		target = assetMrn
	}

	r.out.Write([]byte(termenv.String(fmt.Sprintf("Target:     %s\n", target)).Foreground(r.Colors.Primary).String()))
	report, ok := r.data.Reports[assetMrn]
	if !ok {
		// If scanning the asset has failed, there will be no report, we should first look if there's an error for that target.
		if errStatus, ok := r.data.Errors[assetMrn]; ok {
			switch errStatus.ErrorCode().Category() {
			case explorer.ErrorCategoryInformational:
				r.out.Write([]byte(errStatus.Message + "\n\n"))
			case explorer.ErrorCategoryWarning:
				r.out.Write([]byte(r.Printer.Warn(errStatus.Message) + "\n\n"))
			case explorer.ErrorCategoryError:
				r.out.Write([]byte(r.Printer.Error(errStatus.Message) + "\n\n"))
			}
		} else {
			r.out.Write([]byte(fmt.Sprintf(
				`✕ Could not find asset %s`,
				target,
			)))
			r.out.Write([]byte("\n\n"))
		}
		return
	}

	r.out.Write([]byte(fmt.Sprintf("Datapoints: %d\n", len(report.Data))))
	r.out.Write([]byte("\n"))
}

func (r *cliReporter) printQueryData() {
	r.out.Write([]byte(r.Printer.H1("Data (" + strconv.Itoa(len(r.data.Assets)) + " assets)")))

	if len(r.data.Assets) == 0 {
		r.out.Write([]byte("No assets to report on."))
		return
	}

	queriesMap := map[string]*explorer.Mquery{}
	for mrn, q := range r.bundle.Queries {
		queriesMap[mrn] = q
	}
	for _, p := range r.bundle.Packs {
		for i := range p.Queries {
			query := p.Queries[i]
			queriesMap[query.Mrn] = query
		}

		for i := range p.Groups {
			group := p.Groups[i]
			for j := range group.Queries {
				query := group.Queries[j]
				queriesMap[query.Mrn] = query
			}
		}
	}

	queries := make([]*explorer.Mquery, len(queriesMap))
	i := 0
	for _, v := range queriesMap {
		queries[i] = v
		i++
	}
	sort.Slice(queries, func(i, j int) bool {
		a := queries[i].Title
		b := queries[j].Title
		if a == "" {
			a = queries[i].Query
		}
		if b == "" {
			b = queries[j].Query
		}
		return a < b
	})

	for k := range r.data.Assets {
		cur := r.data.Assets[k]

		r.out.Write([]byte(r.Printer.H2("Asset: " + cur.Name)))

		// check if the asset has an error
		errStatus, ok := r.data.Errors[k]
		if ok {
			switch errStatus.ErrorCode().Category() {
			case explorer.ErrorCategoryInformational:
				r.out.Write([]byte(errStatus.Message + "\n\n"))
			case explorer.ErrorCategoryWarning:
				r.out.Write([]byte(r.Printer.Warn(errStatus.Message) + "\n\n"))
			case explorer.ErrorCategoryError:
				r.out.Write([]byte(r.Printer.Error(errStatus.Message) + "\n\n"))
			}
			continue
		}

		// print the query data
		r.printAssetQueries(k, queries)
	}
}

func (r *cliReporter) printAssetQueries(assetMrn string, queries []*explorer.Mquery) {
	report, ok := r.data.Reports[assetMrn]
	if !ok {
		// nothing to do, we get an error message in the summary code
		return
	}

	resolved, ok := r.data.Resolved[assetMrn]
	if !ok {
		// TODO: we can compute these on the fly too
		return
	}

	results := report.RawResults()

	for i := range queries {
		query := queries[i]
		equery, ok := resolved.ExecutionJob.Queries[query.CodeId]
		if !ok {
			continue
		}

		subRes := map[string]*llx.RawResult{}
		sums := equery.Code.EntrypointChecksums()
		for j := range sums {
			sum := sums[j]
			subRes[sum] = results[sum]
		}
		sums = equery.Code.DatapointChecksums()
		for j := range sums {
			sum := sums[j]
			subRes[sum] = results[sum]
		}

		result := r.Reporter.Printer.Results(equery.Code, subRes)
		if result == "" {
			return
		}
		if r.isCompact {
			result = stringx.MaxLines(10, result)
		}

		title := query.Title
		if title == "" {
			title, _ = mrn.GetResource(query.Mrn, "queries")
		}
		if title != "" {
			title += ":\n"
		}

		r.out.Write([]byte(title))
		r.out.Write([]byte(result))
		r.out.Write([]byte("\n\n"))
	}
}
