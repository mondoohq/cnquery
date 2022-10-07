package explorer

import llx "go.mondoo.com/cnquery/llx"

func (r *Report) RawResults() map[string]*llx.RawResult {
	results := map[string]*llx.RawResult{}

	// covert all proto results to raw results
	for k := range r.Data {
		result := r.Data[k]
		results[k] = result.RawResultV2()
	}

	return results
}
