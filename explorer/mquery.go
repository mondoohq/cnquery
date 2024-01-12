// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package explorer

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"strings"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v10/checksums"
	llx "go.mondoo.com/cnquery/v10/llx"
	"go.mondoo.com/cnquery/v10/mqlc"
	"go.mondoo.com/cnquery/v10/mrn"
	"go.mondoo.com/cnquery/v10/types"
	"go.mondoo.com/cnquery/v10/utils/multierr"
	"go.mondoo.com/cnquery/v10/utils/sortx"
	"google.golang.org/protobuf/proto"
)

// Compile a given query and return the bundle. Both v1 and v2 versions are compiled.
// Both versions will be given the same code id.
func (m *Mquery) Compile(props map[string]*llx.Primitive, conf mqlc.CompilerConfig) (*llx.CodeBundle, error) {
	if m.Mql == "" {
		if m.Query == "" {
			return nil, errors.New("query is not implemented '" + m.Mrn + "'")
		}
		m.Mql = m.Query
		m.Query = ""
	}

	v2Code, err := mqlc.Compile(m.Mql, props, conf)
	if err != nil {
		return nil, err
	}

	return v2Code, nil
}

func RefreshMRN(ownerMRN string, existingMRN string, resource string, uid string) (string, error) {
	// NOTE: asset bundles may not have an owner set, therefore we skip if the query already has an mrn
	if existingMRN != "" {
		if !mrn.IsValid(existingMRN) {
			return "", errors.New("invalid MRN: " + existingMRN)
		}
		return existingMRN, nil
	}

	if ownerMRN == "" {
		return "", errors.New("cannot refresh MRN if the owner MRN is empty")
	}

	if uid == "" {
		return "", errors.New("cannot refresh MRN with an empty UID")
	}

	mrn, err := mrn.NewChildMRN(ownerMRN, resource, uid)
	if err != nil {
		return "", err
	}

	return mrn.String(), nil
}

// RefreshMRN computes a MRN from the UID or validates the existing MRN.
// Both of these need to fit the ownerMRN. It also removes the UID.
func (m *Mquery) RefreshMRN(ownerMRN string) error {
	nu, err := RefreshMRN(ownerMRN, m.Mrn, MRN_RESOURCE_QUERY, m.Uid)
	if err != nil {
		log.Error().Err(err).Str("owner", ownerMRN).Str("uid", m.Uid).Msg("failed to refresh mrn")
		return multierr.Wrap(err, "failed to refresh mrn for query "+m.Title)
	}

	m.Mrn = nu
	m.Uid = ""

	for i := range m.Props {
		if err := m.Props[i].RefreshMRN(ownerMRN); err != nil {
			return err
		}
	}

	return nil
}

// RefreshMRN computes a MRN from the UID or validates the existing MRN.
// Both of these need to fit the ownerMRN. It also removes the UID.
func (m *ObjectRef) RefreshMRN(ownerMRN string) error {
	nu, err := RefreshMRN(ownerMRN, m.Mrn, MRN_RESOURCE_QUERY, m.Uid)
	if err != nil {
		log.Error().Err(err).Str("owner", ownerMRN).Str("uid", m.Uid).Msg("failed to refresh mrn")
		return multierr.Wrap(err, "failed to refresh mrn for query reference "+m.Uid)
	}

	m.Mrn = nu
	m.Uid = ""
	return nil
}

// RefreshChecksum of a query without re-compiling anything. Properties cannot
// be nil. Make sure everything has been compiled beforehand.
//
// Note: this will use whatever type and codeID we have in the query and
// just compute a checksum from the rest.
//
// queries is an optional lookup that is necessary for composed queries,
// since their internal checksum is not stored in this query.
func (m *Mquery) RefreshChecksum(
	ctx context.Context,
	conf mqlc.CompilerConfig,
	getQuery func(ctx context.Context, mrn string) (*Mquery, error),
) error {
	c := checksums.New.
		Add(m.Mql).
		Add(m.CodeId).
		Add(m.Mrn).
		Add(m.Context).
		Add(m.Type).
		Add(m.Title).Add("v2").
		AddUint(m.Impact.Checksum())

	for i := range m.Props {
		prop := m.Props[i]
		if _, err := prop.RefreshChecksumAndType(conf); err != nil {
			return err
		}
		if prop.Checksum == "" {
			return errors.New("referenced property '" + prop.Mrn + "' checksum is empty")
		}
		c = c.Add(prop.Checksum)
	}

	for i := range m.Variants {
		ref := m.Variants[i]
		if q, err := getQuery(context.Background(), ref.Mrn); err == nil {
			if err := q.RefreshChecksum(ctx, conf, getQuery); err != nil {
				return err
			}
			if q.Checksum == "" {
				return errors.New("referenced query '" + ref.Mrn + "'checksum is empty")
			}
			c = c.Add(q.Checksum)
		} else {
			return errors.New("cannot find dependent composed query '" + ref.Mrn + "'")
		}
	}

	// TODO: filters don't support properties yet
	if m.Filters != nil {
		keys := sortx.Keys(m.Filters.Items)
		for _, k := range keys {
			query := m.Filters.Items[k]
			if query.Checksum == "" {
				// FIXME: we don't want this here, it should not be tied to the query
				log.Warn().
					Str("mql", m.Mql).
					Str("filter", query.Mql).
					Msg("refresh checksum on filter of query , which should have been pre-compiled")
				_, err := query.RefreshAsFilter(m.Mrn, conf)
				if err != nil {
					return multierr.Wrap(err, "cannot refresh checksum for query, failed to compile")
				}
				if query.Checksum == "" {
					return errors.New("cannot refresh checksum for query, its filters were not compiled")
				}
			}
			c = c.Add(query.Checksum)
		}
	}

	if m.Docs != nil {
		c = c.
			Add(m.Docs.Desc).
			Add(m.Docs.Audit)

		if m.Docs.Remediation != nil {
			for i := range m.Docs.Remediation.Items {
				doc := m.Docs.Remediation.Items[i]
				c = c.Add(doc.Id).Add(doc.Desc)
			}
		}

		for i := range m.Docs.Refs {
			c = c.
				Add(m.Docs.Refs[i].Title).
				Add(m.Docs.Refs[i].Url)
		}
	}

	keys := sortx.Keys(m.Tags)
	for _, k := range keys {
		c = c.
			Add(k).
			Add(m.Tags[k])
	}

	m.Checksum = c.String()
	return nil
}

// RefreshChecksumAndType by compiling the query and updating the Checksum field
func (m *Mquery) RefreshChecksumAndType(queries map[string]*Mquery, props map[string]PropertyRef, conf mqlc.CompilerConfig) (*llx.CodeBundle, error) {
	return m.refreshChecksumAndType(queries, props, conf)
}

type QueryMap map[string]*Mquery

func (m QueryMap) GetQuery(ctx context.Context, mrn string) (*Mquery, error) {
	if m == nil {
		return nil, errors.New("query not found: " + mrn)
	}

	res, ok := m[mrn]
	if !ok {
		return nil, errors.New("query not found: " + mrn)
	}
	return res, nil
}

func (m *Mquery) refreshChecksumAndType(queries map[string]*Mquery, props map[string]PropertyRef, conf mqlc.CompilerConfig) (*llx.CodeBundle, error) {
	localProps := map[string]*llx.Primitive{}
	for i := range m.Props {
		prop := m.Props[i]

		if prop.Mrn == "" {
			return nil, errors.New("missing MRN (or UID) for property in query " + m.Mrn)
		}

		v, ok := props[prop.Mrn]
		if !ok {
			return nil, errors.New("cannot find property " + prop.Mrn + " in query " + m.Mrn)
		}

		localProps[v.Name] = &llx.Primitive{
			Type: v.Property.Type,
		}

		prop.Checksum = v.Checksum
		prop.CodeId = v.CodeId
		prop.Type = v.Type
	}

	// If this is a variant, we won't compile anything, since there is no MQL snippets
	if len(m.Variants) != 0 {
		if m.Mql != "" {
			log.Warn().Str("msn", m.Mrn).Msg("a composed query is trying to define an mql snippet, which will be ignored")
		}
		return nil, m.RefreshChecksum(context.Background(), conf, QueryMap(queries).GetQuery)
	}

	bundle, err := m.Compile(localProps, conf)
	if err != nil {
		return bundle, multierr.Wrap(err, "failed to compile query '"+m.Mql+"'")
	}

	if bundle.GetCodeV2().GetId() == "" {
		return bundle, errors.New("failed to compile query: received empty result values")
	}

	// We think its ok to always use the new code id
	m.CodeId = bundle.CodeV2.Id

	// the compile step also dedents the code
	m.Mql = bundle.Source

	// TODO: record multiple entrypoints and types
	// TODO(jaym): is it possible that the 2 could produce different types
	if entrypoints := bundle.CodeV2.Entrypoints(); len(entrypoints) == 1 {
		ep := entrypoints[0]
		chunk := bundle.CodeV2.Chunk(ep)
		typ := chunk.Type()
		m.Type = string(typ)
	} else {
		m.Type = string(types.Any)
	}

	return bundle, m.RefreshChecksum(context.Background(), conf, QueryMap(queries).GetQuery)
}

// RefreshAsFilter filters treats this query as an asset filter and sets its Mrn, Title, and Checksum
func (m *Mquery) RefreshAsFilter(mrn string, conf mqlc.CompilerConfig) (*llx.CodeBundle, error) {
	bundle, err := m.refreshChecksumAndType(nil, nil, conf)
	if err != nil {
		return bundle, err
	}
	if bundle == nil {
		return nil, errors.New("filters require MQL snippets (no compiled code generated)")
	}

	checksumInvalidated := false
	if mrn != "" {
		m.Mrn = mrn + "/filter/" + m.CodeId
		checksumInvalidated = true
	}

	if checksumInvalidated {
		if err := m.RefreshChecksum(context.Background(), conf, nil); err != nil {
			return nil, err
		}
	}

	return bundle, nil
}

// Sanitize ensure the content is in good shape and removes leading and trailing whitespace
func (m *Mquery) Sanitize() {
	if m == nil {
		return
	}

	if m.Docs != nil {
		m.Docs.Desc = strings.TrimSpace(m.Docs.Desc)
		m.Docs.Audit = strings.TrimSpace(m.Docs.Audit)

		if m.Docs.Remediation != nil {
			for i := range m.Docs.Remediation.Items {
				doc := m.Docs.Remediation.Items[i]
				doc.Desc = strings.TrimSpace(doc.Desc)
			}
		}

		for i := range m.Docs.Refs {
			r := m.Docs.Refs[i]
			r.Title = strings.TrimSpace(r.Title)
			r.Url = strings.TrimSpace(r.Url)
		}
	}

	if m.Tags != nil {
		sanitizedTags := map[string]string{}
		for k, v := range m.Tags {
			sk := strings.TrimSpace(k)
			sv := strings.TrimSpace(v)
			sanitizedTags[sk] = sv
		}
		m.Tags = sanitizedTags
	}
}

// Merge a given query with a base query and create a new query object as a
// result of it. Anything that is not set in the query, is pulled from the base.
func (m *Mquery) Merge(base *Mquery) *Mquery {
	// TODO: lots of potential to speed things up here
	res := proto.Clone(m).(*Mquery)
	res.AddBase(base)
	return res
}

// AddBase adds a base query into the query object. Anything that is not set
// in the query, is pulled from the base.
func (m *Mquery) AddBase(base *Mquery) {
	if m.Mql == "" {
		// MQL, type and codeID go hand in hand, so make sure to always pull them
		// fully when doing this.
		m.Mql = base.Mql
		m.CodeId = base.CodeId
		m.Type = base.Type
	}
	if m.Type == "" {
		m.Type = base.Type
	}
	if m.Context == "" {
		m.Context = base.Context
	}
	if m.Title == "" {
		m.Title = base.Title
	}
	if m.Docs == nil {
		m.Docs = base.Docs
	} else if base.Docs != nil {
		if m.Docs.Desc == "" {
			m.Docs.Desc = base.Docs.Desc
		}
		if m.Docs.Audit == "" {
			m.Docs.Audit = base.Docs.Audit
		}
		if m.Docs.Remediation == nil {
			m.Docs.Remediation = base.Docs.Remediation
		}
		if m.Docs.Refs == nil {
			m.Docs.Refs = base.Docs.Refs
		}
	}
	if m.Desc == "" {
		m.Desc = base.Desc
	}
	if m.Impact == nil {
		m.Impact = base.Impact
	} else {
		m.Impact.AddBase(base.Impact)
	}
	if m.Tags == nil {
		m.Tags = base.Tags
	}
	if m.Filters == nil {
		m.Filters = base.Filters
	}
	if m.Props == nil {
		m.Props = base.Props
	}
	if m.Variants == nil {
		m.Variants = base.Variants
	}
}

// FilterQueryMRNs removes all queries from the given list, whose MRN matches
// the given list of filters. Special handling for variants:
// - remove all variants that match the MRNs
// - if a composed query ends up empty, it is added to the list of filterMrns and removed
func FilterQueryMRNs(filterMrns map[string]struct{}, queries []*Mquery) []*Mquery {
	if len(filterMrns) == 0 {
		return queries
	}

	var res []*Mquery
	for i := range queries {
		cur := queries[i]
		if _, ok := filterMrns[cur.Mrn]; ok {
			continue
		}

		if len(cur.Variants) != 0 {
			var variants []*ObjectRef
			for j := range cur.Variants {
				cvar := cur.Variants[j]
				if _, ok := filterMrns[cvar.Mrn]; ok {
					continue
				}
				variants = append(variants, cvar)
			}

			// a composed query, which has no more child queries, can safely be removed
			if len(variants) == 0 {
				filterMrns[cur.Mrn] = struct{}{}
				continue
			}

			cur.Variants = variants
		}

		res = append(res, cur)
	}

	return res
}

func (r *Remediation) UnmarshalJSON(data []byte) error {
	var res string
	if err := json.Unmarshal(data, &res); err == nil {
		r.Items = []*TypedDoc{{Id: "default", Desc: res}}
		return nil
	}

	if err := json.Unmarshal(data, &r.Items); err == nil {
		return nil
	}

	// prevent recursive calls into UnmarshalJSON with a placeholder type
	type tmp Remediation
	return json.Unmarshal(data, (*tmp)(r))
}

func (r *Remediation) MarshalJSON() ([]byte, error) {
	if r == nil {
		return []byte{}, nil
	}
	return json.Marshal(r.Items)
}

func ChecksumFilters(queries []*Mquery, conf mqlc.CompilerConfig) (string, error) {
	for i := range queries {
		if _, err := queries[i].refreshChecksumAndType(nil, nil, conf); err != nil {
			return "", multierr.Wrap(err, "failed to compile query")
		}
	}

	sort.Slice(queries, func(i, j int) bool {
		return queries[i].CodeId < queries[j].CodeId
	})

	afc := checksums.New
	for i := range queries {
		afc = afc.Add(queries[i].CodeId)
	}

	return afc.String(), nil
}
