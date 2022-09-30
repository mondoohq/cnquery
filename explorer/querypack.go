package explorer

import (
	"context"
	"sort"

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
	nu, err := RefreshMRN(ownerMRN, p.Mrn, "policies", p.Uid)
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

// ComputeAssetFilters into mql
func (p *QueryPack) ComputeAssetFilters(ctx context.Context) ([]*Mquery, error) {
	res := make([]*Mquery, len(p.Filters))
	for i := range p.Filters {
		code := p.Filters[i]
		res[i] = &Mquery{
			Query: code,
		}
	}

	return res, nil
}
