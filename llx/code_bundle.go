package llx

func (x *CodeBundle) IsV2() bool {
	return x.CodeV2 != nil
}

func (x *CodeBundle) FilterResults(results map[string]*RawResult) map[string]*RawResult {
	filteredResults := map[string]*RawResult{}

	if x.IsV2() {
		for i := range x.CodeV2.Checksums {
			checksum := x.CodeV2.Checksums[i]

			res := results[checksum]
			if res != nil {
				filteredResults[checksum] = res
			}
		}
	} else {
		for i := range x.DeprecatedV5Code.Checksums {
			checksum := x.DeprecatedV5Code.Checksums[i]

			res := results[checksum]
			if res != nil {
				filteredResults[checksum] = res
			}
		}
	}

	return filteredResults
}

func Results2Assessment(bundle *CodeBundle, results map[string]*RawResult, useV2Code bool) *Assessment {
	if useV2Code {
		return Results2AssessmentLookupV2(bundle, func(s string) (*RawResult, bool) {
			r := results[s]
			return r, r != nil
		})
	}
	return Results2AssessmentLookupV1(bundle, func(s string) (*RawResult, bool) {
		r := results[s]
		return r, r != nil
	})
}

func ReturnValuesV2(bundle *CodeBundle, f func(s string) (*RawResult, bool)) []*RawResult {
	return bundle.CodeV2.returnValues(bundle, f)
}

// Results2Assessment converts a list of raw results into an assessment for the query
func Results2AssessmentV2(bundle *CodeBundle, results map[string]*RawResult) *Assessment {
	return Results2AssessmentLookupV2(bundle, func(s string) (*RawResult, bool) {
		r := results[s]
		return r, r != nil
	})
}

// Results2AssessmentLookup creates an assessment for a bundle using a lookup hook to get all results
func Results2AssessmentLookupV2(bundle *CodeBundle, f func(s string) (*RawResult, bool)) *Assessment {
	code := bundle.CodeV2

	res := Assessment{
		Success:  true,
		Checksum: code.Id,
	}
	res.Success = true

	entrypoints := code.Entrypoints()
	for i := range entrypoints {
		ep := entrypoints[i]
		cur := code.entrypoint2assessment(bundle, ep, f)
		if cur == nil {
			continue
		}

		res.Results = append(res.Results, cur)
		if !cur.Success {
			res.Success = false
		}

		// We don't want to lose errors
		if cur.IsAssertion || cur.Error != "" {
			res.IsAssertion = true
		}
	}

	if !res.IsAssertion {
		return nil
	}

	return &res
}

// CodepointChecksums returns the entrypoint- and datapoint-checksums
func (x *CodeBundle) CodepointChecksums(useV2Code bool) []string {
	return append(
		x.EntrypointChecksums(useV2Code),
		x.DatapointChecksums(useV2Code)...)
}

// EntrypointChecksums returns the checksums for all entrypoints
func (x *CodeBundle) EntrypointChecksums(useV2Code bool) []string {
	var checksums []string
	if useV2Code {
		// TODO (jaym): double check with dom this is the way to get entrypoints
		checksums = make([]string, len(x.CodeV2.Blocks[0].Entrypoints))
		for i, ref := range x.CodeV2.Blocks[0].Entrypoints {
			checksums[i] = x.CodeV2.Checksums[ref]
		}
	} else {
		checksums = make([]string, len(x.DeprecatedV5Code.Entrypoints))
		for i, ref := range x.DeprecatedV5Code.Entrypoints {
			checksums[i] = x.DeprecatedV5Code.Checksums[ref]
		}
	}
	return checksums
}

// DatapointChecksums returns the checksums for all datapoints
func (x *CodeBundle) DatapointChecksums(useV2Code bool) []string {
	var checksums []string
	if useV2Code {
		// TODO (jaym): double check with dom this is the way to get entrypoints
		checksums = make([]string, len(x.CodeV2.Blocks[0].Datapoints))
		for i, ref := range x.CodeV2.Blocks[0].Datapoints {
			checksums[i] = x.CodeV2.Checksums[ref]
		}
	} else {
		checksums = make([]string, len(x.DeprecatedV5Code.Datapoints))
		for i, ref := range x.DeprecatedV5Code.Datapoints {
			checksums[i] = x.DeprecatedV5Code.Checksums[ref]
		}
	}
	return checksums
}
