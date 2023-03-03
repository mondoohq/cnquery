package explorer

import (
	"encoding/json"
	"sort"
	"strings"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery"
	"go.mondoo.com/cnquery/checksums"
	llx "go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/mqlc"
	"go.mondoo.com/cnquery/mrn"
	"go.mondoo.com/cnquery/resources/packs/all/info"
	"go.mondoo.com/cnquery/sortx"
	"go.mondoo.com/cnquery/types"
	"google.golang.org/protobuf/proto"
)

// Compile a given query and return the bundle. Both v1 and v2 versions are compiled.
// Both versions will be given the same code id.
func (m *Mquery) Compile(props map[string]*llx.Primitive) (*llx.CodeBundle, error) {
	if m.Mql == "" {
		if m.Query == "" {
			return nil, errors.New("query is not implemented '" + m.Mrn + "'")
		}
		m.Mql = m.Query
		m.Query = ""
	}

	schema := info.Registry.Schema()

	v2Code, err := mqlc.Compile(m.Mql, props, mqlc.NewConfig(schema, cnquery.DefaultFeatures))
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
		return errors.Wrap(err, "failed to refresh mrn for query "+m.Title)
	}

	m.Mrn = nu
	m.Uid = ""
	return nil
}

// RefreshChecksum of a query without re-compiling anything. Note: this will
// use whatever type and codeID we have in the query and just compute a checksum
// from the rest.
func (m *Mquery) RefreshChecksum() error {
	c := checksums.New.
		Add(m.Mql).
		Add(m.CodeId).
		Add(m.Mrn).
		Add(m.Context).
		Add(m.Type).
		Add(m.Title).Add("v2")

	for i := range m.Props {
		// we checked this above, so it has to exist
		prop := m.Props[i]
		c = c.Add(prop.Checksum)
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
				query.RefreshAsFilter(m.Mrn)
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
func (m *Mquery) RefreshChecksumAndType(lookup map[string]PropertyRef) (*llx.CodeBundle, error) {
	return m.refreshChecksumAndType(lookup)
}

func (m *Mquery) refreshChecksumAndType(lookup map[string]PropertyRef) (*llx.CodeBundle, error) {
	localProps := map[string]*llx.Primitive{}
	for i := range m.Props {
		prop := m.Props[i]

		if prop.Mrn == "" {
			return nil, errors.New("missing MRN (or UID) for property in query " + m.Mrn)
		}

		v, ok := lookup[prop.Mrn]
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

	bundle, err := m.Compile(localProps)
	if err != nil {
		return bundle, errors.New("failed to compile query '" + m.Mql + "': " + err.Error())
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

	return bundle, m.RefreshChecksum()
}

// RefreshAsFilter filters treats this query as an asset filter and sets its Mrn, Title, and Checksum
func (m *Mquery) RefreshAsFilter(mrn string) (*llx.CodeBundle, error) {
	bundle, err := m.refreshChecksumAndType(nil)
	if err != nil {
		return bundle, err
	}

	if mrn != "" {
		m.Mrn = mrn + "/filter/" + m.CodeId
	}

	if m.Title == "" {
		m.Title = m.Query
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
	if m.Compose == nil {
		m.Compose = base.Compose
	}
}

func (v *Impact) Merge(base *Impact) {
	if base == nil {
		return
	}

	if v.Scoring == Impact_SCORING_UNSPECIFIED {
		v.Scoring = base.Scoring
	}
	if v.Value == nil {
		v.Value = base.Value
	}
	if v.Weight < 1 {
		v.Weight = base.Weight
	}
}

func (v *Impact) UnmarshalJSON(data []byte) error {
	var res int32
	if err := json.Unmarshal(data, &res); err == nil {
		v.Value = &ImpactValue{Value: res}
		return nil
	}

	type tmp Impact
	return json.Unmarshal(data, (*tmp)(v))
}

func (v *ImpactValue) MarshalJSON() ([]byte, error) {
	if v == nil {
		return []byte{}, nil
	}
	return json.Marshal(v.Value)
}

func (v *ImpactValue) UnmarshalJSON(data []byte) error {
	var res int32
	if err := json.Unmarshal(data, &res); err == nil {
		v.Value = res
	} else {
		vInternal := &struct {
			Value int32 `json:"value"`
		}{}
		if err := json.Unmarshal(data, &vInternal); err != nil {
			return err
		}
		v.Value = vInternal.Value
	}

	if v.Value < 0 || v.Value > 100 {
		return errors.New("impact must be between 0 and 100")
	}

	return nil
}

func (r *Remediation) UnmarshalJSON(data []byte) error {
	var res string
	if err := json.Unmarshal(data, &res); err == nil {
		r.Items = []*TypedDoc{{Id: "default", Desc: res}}
		return nil
	}

	// prevent recursive calls into UnmarshalJSON with a placeholder type
	type tmp Remediation
	return json.Unmarshal(data, (*tmp)(r))
}

func ChecksumFilters(queries []*Mquery) (string, error) {
	for i := range queries {
		if _, err := queries[i].refreshChecksumAndType(nil); err != nil {
			return "", errors.New("failed to compile query: " + err.Error())
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
