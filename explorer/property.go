package explorer

import (
	"sort"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery"
	"go.mondoo.com/cnquery/checksums"
	llx "go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/mqlc"
	"go.mondoo.com/cnquery/resources/packs/all/info"
	"go.mondoo.com/cnquery/types"
)

// RefreshMRN computes a MRN from the UID or validates the existing MRN.
// Both of these need to fit the ownerMRN. It also removes the UID.
func (p *Property) RefreshMRN(ownerMRN string) error {
	nu, err := RefreshMRN(ownerMRN, p.Mrn, MRN_RESOURCE_QUERY, p.Uid)
	if err != nil {
		log.Error().Err(err).Str("owner", ownerMRN).Str("uid", p.Uid).Msg("failed to refresh mrn")
		return errors.Wrap(err, "failed to refresh mrn for query "+p.Title)
	}

	p.Mrn = nu
	p.Uid = ""
	return nil
}

// Compile a given property and return the bundle.
func (p *Property) Compile(props map[string]*llx.Primitive) (*llx.CodeBundle, error) {
	schema := info.Registry.Schema()
	return mqlc.Compile(p.Mql, props, mqlc.NewConfig(schema, cnquery.DefaultFeatures))
}

// RefreshChecksumAndType by compiling the query and updating the Checksum field
func (p *Property) RefreshChecksumAndType(lookup map[string]queryRef) (*llx.CodeBundle, error) {
	return p.refreshChecksumAndType(lookup)
}

func (p *Property) refreshChecksumAndType(lookup map[string]queryRef) (*llx.CodeBundle, error) {
	bundle, err := p.Compile(nil)
	if err != nil {
		return bundle, errors.New("failed to compile query '" + p.Mql + "': " + err.Error())
	}

	if bundle.GetCodeV2().GetId() == "" {
		return bundle, errors.New("failed to compile query: received empty result values")
	}

	// We think its ok to always use the new code id
	p.CodeId = bundle.CodeV2.Id

	// the compile step also dedents the code
	p.Mql = bundle.Source

	// TODO: record multiple entrypoints and types
	// TODO(jaym): is it possible that the 2 could produce different types
	if entrypoints := bundle.CodeV2.Entrypoints(); len(entrypoints) == 1 {
		ep := entrypoints[0]
		chunk := bundle.CodeV2.Chunk(ep)
		typ := chunk.Type()
		p.Type = string(typ)
	} else {
		p.Type = string(types.Any)
	}

	c := checksums.New.
		Add(p.Mql).
		Add(p.CodeId).
		Add(p.Mrn).
		Add(p.Type).
		Add(p.Title).Add("v2")

	for i := range p.Props {
		// we checked this above, so it has to exist
		prop := p.Props[i]
		v := lookup[prop.Mrn]

		c = c.Add(v.query.Checksum)
		if v.query.Mql != "" {
			c = c.Add(v.query.Mql)
		}
	}

	if p.Docs != nil {
		c = c.
			Add(p.Docs.Desc).
			Add(p.Docs.Audit).
			Add(p.Docs.Remediation)

		for i := range p.Docs.Refs {
			c = c.
				Add(p.Docs.Refs[i].Title).
				Add(p.Docs.Refs[i].Url)
		}
	}

	keys := make([]string, len(p.Tags))
	i := 0
	for k := range p.Tags {
		keys[i] = k
		i++
	}
	sort.Strings(keys)
	for _, k := range keys {
		c = c.
			Add(k).
			Add(p.Tags[k])
	}

	p.Checksum = c.String()

	return bundle, nil
}
