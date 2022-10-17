package reporter

import (
	"fmt"
	"io"
	"sort"
	"strconv"

	"github.com/muesli/termenv"
	"go.mondoo.com/cnquery/explorer"
	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/stringx"
)

type defaultReporter struct {
	*Reporter
	isCompact bool
	isSummary bool
	out       io.Writer
	data      *explorer.ReportCollection

	// vv the items below will be automatically filled
	bundle *explorer.BundleMap
}

func (r *defaultReporter) print() error {
	// catch case where the scan was not successful and no bundle was fetched from server
	if r.data == nil || r.data.Bundle == nil {
		return nil
	}

	r.bundle = r.data.Bundle.ToMap()

	if !r.isSummary {
		r.printQueries()
	}

	r.printSummary()
	return nil
}

func (r *defaultReporter) printSummary() {
	r.out.Write([]byte("Summary (" + strconv.Itoa(len(r.data.Assets)) + " assets)\n"))
	r.out.Write([]byte("========================\n"))

	for mrn, asset := range r.data.Assets {
		r.printAssetSummary(mrn, asset)
	}
}

func (r *defaultReporter) printAssetSummary(assetMrn string, asset *explorer.Asset) {
	target := asset.Name
	if target == "" {
		target = assetMrn
	}

	report, ok := r.data.Reports[assetMrn]
	if !ok {
		// If scanning the asset has failed, there will be no report, we should first look if there's an error for that target.
		if err, ok := r.data.Errors[assetMrn]; ok {
			r.out.Write([]byte(termenv.String(fmt.Sprintf(
				`✕ Error for asset %s: %s`,
				target, err,
			)).Foreground(r.Colors.Error).String()))
		} else {
			r.out.Write([]byte(fmt.Sprintf(
				`✕ Could not find asset %s`,
				target,
			)))
		}
		return
	}

	r.out.Write([]byte(termenv.String(fmt.Sprintf("Target:     %s\n", target)).Foreground(r.Colors.Primary).String()))
	r.out.Write([]byte(fmt.Sprintf("Datapoints: %d\n", len(report.Data))))
	r.out.Write([]byte("\n"))
}

func (r *defaultReporter) printQueries() {
	if len(r.data.Assets) == 0 {
		r.out.Write([]byte("No assets to report on."))
		return
	}

	queriesMap := r.bundle.Queries
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
		r.out.Write([]byte("Asset: " + cur.Name + "\n"))
		r.out.Write([]byte("========================\n\n"))

		r.printAssetQueries(k, queries)

		r.out.Write([]byte{'\n'})
	}
}

func (r *defaultReporter) printAssetQueries(assetMrn string, queries []*explorer.Mquery) {
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

		// TODO: only in long version + needs styling
		// r.out.Write([]byte(query.Query))
		// r.out.Write([]byte{'\n'})

		r.out.Write([]byte(result))
		r.out.Write([]byte{'\n'})
	}
	r.out.Write([]byte("\n"))
}
