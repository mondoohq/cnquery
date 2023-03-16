package explorer

import (
	"encoding/json"
	"errors"
	"sort"
	"strconv"

	"go.mondoo.com/cnquery/checksums"
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

	// FIXME: DEPRECATED, remove in v9.0 vv
	// This old style of specifying filters is going to be removed, we
	// have an alternative with list and keys
	var arr []string
	err = json.Unmarshal(data, &arr)
	if err == nil {
		s.Items = map[string]*Mquery{}
		for i := range arr {
			s.Items[strconv.Itoa(i)] = &Mquery{Mql: arr[i]}
		}
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

func (s *Filters) Compile(ownerMRN string) error {
	if s == nil || len(s.Items) == 0 {
		return nil
	}

	res := make(map[string]*Mquery, len(s.Items))
	for _, query := range s.Items {
		query.RefreshAsFilter(ownerMRN)

		if _, ok := res[query.CodeId]; ok {
			continue
		}

		res[query.CodeId] = query
	}

	s.Items = res
	return nil
}

// RegisterChild takes all filters in a given child and adds them to the parent.
// Note: The parent must be non-empty and non-nil, or this method will panic.
func (s *Filters) RegisterChild(child *Filters) {
	if child == nil {
		return
	}

	for k, v := range child.Items {
		s.Items[k] = v
	}
}

// RegisterQuery attempt to take a query (or nil) and register all its filters.
// This includes any variants that the query might have as well. It will also
// try to look up the base query, if requested.
func (s *Filters) RegisterQuery(query *Mquery, lookupQueries map[string]*Mquery) error {
	if query == nil {
		return nil
	}

	s.RegisterChild(query.Filters)

	for i := range query.Variants {
		mrn := query.Variants[i].Mrn
		if variant, ok := lookupQueries[mrn]; ok {
			s.RegisterQuery(variant, lookupQueries)
		} else {
			return errors.New("cannot find query variant " + mrn)
		}
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
