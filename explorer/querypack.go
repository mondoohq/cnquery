package explorer

import (
	"context"
	"encoding/json"
	"sort"
	"strconv"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/checksums"
)

func (p *QueryPack) InvalidateAllChecksums() {
	p.LocalContentChecksum = ""
	p.LocalExecutionChecksum = ""
}

// RefreshMRN computes a MRN from the UID or validates the existing MRN.
// Both of these need to fit the ownerMRN. It also removes the UID.
func (p *QueryPack) RefreshMRN(ownerMRN string) error {
	nu, err := RefreshMRN(ownerMRN, p.Mrn, "querypack", p.Uid)
	if err != nil {
		log.Error().Err(err).Str("owner", ownerMRN).Str("uid", p.Uid).Msg("failed to refresh mrn")
		return errors.Wrap(err, "failed to refresh mrn for query "+p.Name)
	}

	p.Mrn = nu
	p.Uid = ""
	return nil
}

func (p *QueryPack) UpdateChecksums() error {
	p.LocalContentChecksum = ""
	p.LocalExecutionChecksum = ""

	// Note: this relies on the fact that the bundle was compiled before

	var i int
	executionChecksum := checksums.New
	contentChecksum := checksums.New

	contentChecksum = contentChecksum.Add(p.Mrn).Add(p.Name).Add(p.Version).Add(p.OwnerMrn)
	if p.IsPublic {
		contentChecksum = contentChecksum.AddUint(1)
	} else {
		contentChecksum = contentChecksum.AddUint(0)
	}
	for i := range p.Authors {
		author := p.Authors[i]
		contentChecksum = contentChecksum.Add(author.Email).Add(author.Name)
	}
	contentChecksum = contentChecksum.AddUint(uint64(p.Created)).AddUint(uint64(p.Modified))

	if p.Docs != nil {
		contentChecksum = contentChecksum.Add(p.Docs.Desc)
	}

	executionChecksum = executionChecksum.Add(p.Mrn)

	// tags
	arr := make([]string, len(p.Tags))
	i = 0
	for k := range p.Tags {
		arr[i] = k
		i++
	}
	sort.Strings(arr)
	for _, k := range arr {
		contentChecksum = contentChecksum.Add(k).Add(p.Tags[k])
	}

	// QUERIES (must be sorted)
	queryIDs := make([]string, len(p.Queries))
	queries := make(map[string]*Mquery, len(p.Queries))
	for i := range p.Queries {
		query := p.Queries[i]
		queryIDs[i] = query.Mrn
		queries[query.Mrn] = query
	}
	sort.Strings(queryIDs)
	for _, queryID := range queryIDs {
		q, ok := queries[queryID]
		if !ok {
			return errors.New("cannot find query " + queryID)
		}

		// we use the checksum for doc, tag and ref changes
		contentChecksum = contentChecksum.Add(q.Checksum)
		executionChecksum = executionChecksum.Add(q.CodeId)
	}

	return nil
}

// ComputeFilters into mql
func (p *QueryPack) ComputeFilters(ctx context.Context, ownerMRN string) ([]*Mquery, error) {
	if p.Filters == nil {
		return nil, nil
	}

	if err := p.Filters.compile(ownerMRN); err != nil {
		return nil, err
	}

	for i := range p.Queries {
		query := p.Queries[i]
		if query.Filter == nil {
			continue
		}

		if err := query.Filter.compile(ownerMRN); err != nil {
			return nil, err
		}

		for k, v := range query.Filter.Items {
			p.Filters.Items[k] = v
		}
	}

	res := make([]*Mquery, len(p.Filters.Items))
	idx := 0
	for _, v := range p.Filters.Items {
		res[idx] = v
		idx++
	}

	return res, nil
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

	return json.Unmarshal(data, &s.Items)
}

func (s *Filters) compile(ownerMRN string) error {
	if s == nil || len(s.Items) == 0 {
		return nil
	}

	res := make(map[string]*Mquery, len(s.Items))
	for _, query := range s.Items {
		bundle, err := query.Compile(nil)
		if err != nil {
			return errors.Wrap(err, "failed to compile asset filter")
		}

		query.Mrn = ownerMRN + "/assetfilter/" + bundle.CodeV2.Id
		query.CodeId = bundle.CodeV2.Id

		if _, ok := res[query.CodeId]; ok {
			continue
		}

		res[query.CodeId] = query
	}

	s.Items = res
	return nil
}
