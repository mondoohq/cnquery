package explorer

import (
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
func (p *Property) RefreshChecksumAndType() (*llx.CodeBundle, error) {
	return p.refreshChecksumAndType()
}

func (p *Property) refreshChecksumAndType() (*llx.CodeBundle, error) {
	bundle, err := p.Compile(nil)
	if err != nil {
		return bundle, errors.New("failed to compile property '" + p.Mql + "': " + err.Error())
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
		Add(p.Context).
		Add(p.Title).Add("v2").
		Add(p.Desc)

	for i := range p.For {
		f := p.For[i]
		c = c.Add(f.Mrn)
	}

	p.Checksum = c.String()

	return bundle, nil
}

func (p *Property) Merge(base *Property) {
	if p.Mql == "" {
		p.Mql = base.Mql
	}
	if p.Type == "" {
		p.Type = base.Type
	}
	if p.Context == "" {
		p.Context = base.Context
	}
	if p.Title == "" {
		p.Title = base.Title
	}
	if p.Desc == "" {
		p.Desc = base.Desc
	}
	if len(p.For) == 0 {
		p.For = base.For
	}
}
