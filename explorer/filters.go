// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package explorer

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v11/checksums"
	"go.mondoo.com/cnquery/v11/mqlc"
	"go.mondoo.com/cnquery/v11/utils/multierr"
	"go.mondoo.com/cnquery/v11/utils/sortx"
)

// NewFilters creates a Filters object from a simple list of MQL snippets
func NewFilters(queries ...string) *Filters {
	res := &Filters{
		Items: map[string]*Mquery{},
	}

	for i := range queries {
		res.Items[strconv.Itoa(i)] = &Mquery{Mql: queries[i]}
	}

	return res
}

// Computes the checksum for the filters and adds it to the aggregate
// execution and content checksums. Filters must have been previously compiled!
// We need it to be ready for checksums and don't want to do the compile
// step here because it's not the primary function.
func (filters *Filters) Checksum() (checksums.Fast, checksums.Fast) {
	content := checksums.New
	execution := checksums.New

	if filters == nil {
		return content, execution
	}

	keys := make([]string, len(filters.Items))
	i := 0
	for k := range filters.Items {
		// we add this sanity check since we expose the method, but can't ensure
		// that users have compiled everything beforehand
		if len(k) < 2 {
			panic("internal error processing filter checksums: queries are not compiled")
		}

		keys[i] = k
		i++
	}
	sort.Strings(keys)

	for i := range keys {
		filter := filters.Items[keys[i]]
		content = content.Add(filter.Title).Add(filter.Desc)

		// we add this sanity check since we expose the method, but can't ensure
		// that users have compiled everything beforehand
		if filter.Checksum == "" || filter.CodeId == "" {
			panic("internal error processing filter checksums: query is compiled")
		}

		content = content.Add(filter.Checksum)
		execution = execution.Add(filter.CodeId)
	}

	content = content.AddUint(uint64(execution))

	return content, execution
}

func (s *Filters) UnmarshalJSON(data []byte) error {
	var str string
	err := json.Unmarshal(data, &str)
	if err == nil {
		s.Items = map[string]*Mquery{}
		s.Items[""] = &Mquery{
			Mql: str,
		}
		return nil
	}

	// FIXME: DEPRECATED, remove in v12.0 vv
	// This old style of specifying filters is going to be removed, we
	// have an alternative with list and keys
	var arr []string
	err = json.Unmarshal(data, &arr)
	if err == nil {
		s.Items = map[string]*Mquery{}
		for i := range arr {
			s.Items[strconv.Itoa(i)] = &Mquery{Mql: arr[i]}
		}
		log.Warn().Msg("Found an old use of filters (as a list of strings). This will be removed in the next major version. Please migrate to:\n- mql: filter 1\n- mql: filter 2")
		return nil
	}
	// ^^

	var list []*Mquery
	err = json.Unmarshal(data, &list)
	if err == nil {
		s.Items = map[string]*Mquery{}
		for i := range list {
			s.Items[strconv.Itoa(i)] = list[i]
		}
		return nil
	}

	// prevent recursive calls into UnmarshalJSON with a placeholder type
	type tmp Filters
	return json.Unmarshal(data, (*tmp)(s))
}

func (s *Filters) Compile(ownerMRN string, conf mqlc.CompilerConfig) error {
	if s == nil || len(s.Items) == 0 {
		return nil
	}

	res := make(map[string]*Mquery, len(s.Items))
	for _, query := range s.Items {
		_, err := query.RefreshAsFilter(ownerMRN, conf)
		if err != nil {
			return err
		}

		if _, ok := res[query.CodeId]; ok {
			continue
		}

		res[query.CodeId] = query
	}

	s.Items = res
	return nil
}

func (s *Filters) ComputeChecksum(checksum checksums.Fast, queryMrn string, conf mqlc.CompilerConfig) (checksums.Fast, error) {
	if s == nil {
		return checksum, nil
	}

	// TODO: filters don't support properties yet
	keys := sortx.Keys(s.Items)
	for _, k := range keys {
		query := s.Items[k]
		if query.Checksum == "" {
			// this is a fallback safeguard
			log.Warn().
				Str("filter", query.Mql).
				Msg("refresh checksum on filters, which should have been pre-compiled")
			_, err := query.RefreshAsFilter(queryMrn, conf)
			if err != nil {
				return checksum, multierr.Wrap(err, "cannot refresh checksum for query, failed to compile")
			}
			if query.Checksum == "" {
				return checksum, errors.New("cannot refresh checksum for query, its filters were not compiled")
			}
		}
		checksum = checksum.Add(query.Checksum)
	}
	return checksum, nil
}

// AddFilters takes all given filters (or nil) and adds them to the parent.
// Note: The parent must be non-empty and non-nil, or this method will panic.
func (s *Filters) AddFilters(child *Filters) {
	if child == nil {
		return
	}

	for k, v := range child.Items {
		s.Items[k] = v
	}
}

var ErrQueryNotFound = errors.New("query not found")

// AddQueryFilters attempt to take a query (or nil) and register all its filters.
// This includes any variants that the query might have as well.
func (s *Filters) AddQueryFilters(query *Mquery, lookupQueries map[string]*Mquery) error {
	if query == nil {
		return nil
	}

	return s.AddQueryFiltersFn(context.Background(), query, func(_ context.Context, mrn string) (*Mquery, error) {
		q, ok := lookupQueries[mrn]
		if !ok {
			return nil, ErrQueryNotFound
		}
		return q, nil
	})
}

// AddQueryFiltersFn attempt to take a query (or nil) and register all its filters.
// This includes any variants that the query might have as well.
func (s *Filters) AddQueryFiltersFn(ctx context.Context, query *Mquery, lookupQuery func(ctx context.Context, mrn string) (*Mquery, error)) error {
	if query == nil {
		return nil
	}

	s.AddFilters(query.Filters)

	for i := range query.Variants {
		mrn := query.Variants[i].Mrn
		variant, err := lookupQuery(ctx, mrn)
		if err != nil {
			return multierr.Wrap(err, "cannot find query variant "+mrn)
		}
		s.AddQueryFiltersFn(ctx, variant, lookupQuery)
	}
	return nil
}

// Checks if the given queries (via CodeIDs) are supported by this set of
// asset filters. Asset filters that are not defined return true.
// If any of the filters is supported, the set returns true.
func (s *Filters) Supports(supported map[string]struct{}) bool {
	if s == nil || len(s.Items) == 0 {
		return true
	}

	for k := range s.Items {
		if _, ok := supported[k]; ok {
			return true
		}
	}

	return false
}

func (s *Filters) Summarize() string {
	if s == nil || len(s.Items) == 0 {
		return ""
	}

	filters := make([]string, len(s.Items))
	i := 0
	for _, filter := range s.Items {
		if filter.Title != "" {
			filters[i] = filter.Title
		} else {
			filters[i] = filter.Mql
		}
		i++
	}

	sort.Strings(filters)
	return strings.Join(filters, ", ")
}
