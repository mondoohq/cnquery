// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package explorer

import (
	"context"
	"errors"
	"sort"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v10/checksums"
	"go.mondoo.com/cnquery/v10/utils/multierr"
	"go.mondoo.com/cnquery/v10/utils/sortx"
)

func (p *QueryPack) InvalidateAllChecksums() {
	p.LocalContentChecksum = ""
	p.LocalExecutionChecksum = ""
}

// RefreshMRN computes a MRN from the UID or validates the existing MRN.
// Both of these need to fit the ownerMRN. It also removes the UID.
func (p *QueryPack) RefreshMRN(ownerMRN string) error {
	nu, err := RefreshMRN(ownerMRN, p.Mrn, MRN_RESOURCE_QUERYPACK, p.Uid)
	if err != nil {
		log.Error().Err(err).Str("owner", ownerMRN).Str("uid", p.Uid).Msg("failed to refresh mrn")
		return multierr.Wrap(err, "failed to refresh mrn for query "+p.Name)
	}

	p.Mrn = nu
	p.Uid = ""
	return nil
}

func (p *QueryPack) UpdateChecksums() error {
	p.LocalContentChecksum = ""
	p.LocalExecutionChecksum = ""

	// Note: this relies on the fact that the bundle was compiled before

	executionChecksum := checksums.New
	contentChecksum := checksums.New

	contentChecksum = contentChecksum.Add(p.Mrn).Add(p.Name).Add(p.Version).Add(p.OwnerMrn)
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
	keys := sortx.Keys(p.Tags)
	for _, k := range keys {
		contentChecksum = contentChecksum.Add(k).Add(p.Tags[k])
	}

	c, e := p.Filters.Checksum()
	contentChecksum = contentChecksum.AddUint(uint64(c))
	executionChecksum = executionChecksum.AddUint(uint64(e))

	c, e = ChecksumQueries(p.Queries)
	contentChecksum = contentChecksum.AddUint(uint64(c))
	executionChecksum = executionChecksum.AddUint(uint64(e))

	// Groups
	for i := range p.Groups {
		group := p.Groups[i]

		contentChecksum = contentChecksum.
			Add(group.Title).
			AddUint(uint64(group.Created)).
			AddUint(uint64(group.Modified))

		c, e := group.Filters.Checksum()
		contentChecksum = contentChecksum.AddUint(uint64(c))
		executionChecksum = executionChecksum.AddUint(uint64(e))

		c, e = ChecksumQueries(p.Queries)
		contentChecksum = contentChecksum.AddUint(uint64(c))
		executionChecksum = executionChecksum.AddUint(uint64(e))
	}

	contentChecksum = contentChecksum.AddUint(uint64(executionChecksum))

	p.LocalContentChecksum = contentChecksum.String()
	p.LocalExecutionChecksum = executionChecksum.String()

	return nil
}

// Computes the checksums for a list of queries, which is sorted and then
// split into a content and execution checksum. These queries must have been
// previously compiled and ready, otherwise the checksums cannot be computed.
func ChecksumQueries(queries []*Mquery) (checksums.Fast, checksums.Fast) {
	content := checksums.New
	execution := checksums.New

	if len(queries) == 0 {
		return content, execution
	}

	queryIDs := make([]string, len(queries))
	queryMap := make(map[string]*Mquery, len(queries))
	for i := range queries {
		query := queries[i]
		queryIDs[i] = query.Mrn
		queryMap[query.Mrn] = query
	}
	sort.Strings(queryIDs)

	for _, queryID := range queryIDs {
		query := queryMap[queryID]

		// we add this sanity check since we expose the method, but can't ensure
		// that users have compiled everything beforehand
		if query.Checksum == "" || query.CodeId == "" {
			panic("internal error processing filter checksums: query is compiled")
		}

		// we use the checksum for doc, tag and ref changes
		content = content.Add(query.Checksum)
		execution = execution.Add(query.CodeId)
	}

	content = content.AddUint(uint64(execution))

	return content, execution
}

// ComputeFilters into mql
func (p *QueryPack) ComputeFilters(ctx context.Context, ownerMRN string) ([]*Mquery, error) {
	numFilters := 0
	if p.Filters != nil {
		numFilters += len(p.Filters.Items)
	}
	for i := range p.Groups {
		if p.Groups[i].Filters == nil {
			return nil, errors.New("cannot compute filters for a querypack, unless it was compiled first")
		}
		numFilters += len(p.Groups[i].Filters.Items)
	}

	res := make([]*Mquery, numFilters)
	idx := 0
	if p.Filters != nil {
		for _, v := range p.Filters.Items {
			res[idx] = v
			idx++
		}
	}
	for i := range p.Groups {
		for _, v := range p.Groups[i].Filters.Items {
			res[idx] = v
			idx++
		}
	}

	sort.Slice(res, func(i, j int) bool {
		return res[i].CodeId < res[j].CodeId
	})

	return res, nil
}
