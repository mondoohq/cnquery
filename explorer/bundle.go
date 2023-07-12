package explorer

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/segmentio/ksuid"
	"go.mondoo.com/cnquery/checksums"
	"go.mondoo.com/cnquery/mrn"
	"sigs.k8s.io/yaml"
)

const (
	MRN_RESOURCE_QUERY     = "queries"
	MRN_RESOURCE_QUERYPACK = "querypacks"
	MRN_RESOURCE_ASSET     = "assets"
)

// BundleMap is a Bundle with easier access to its data
type BundleMap struct {
	OwnerMrn string                `json:"owner_mrn,omitempty"`
	Packs    map[string]*QueryPack `json:"packs,omitempty"`
	Queries  map[string]*Mquery    `json:"queries,omitempty"`
	Props    map[string]*Mquery    `json:"props,omitempty"`
}

// NewBundleMap creates a new empty initialized map
// dataLake (optional) connects an additional data layer which may provide queries/packs
func NewBundleMap(ownerMrn string) *BundleMap {
	return &BundleMap{
		OwnerMrn: ownerMrn,
		Packs:    make(map[string]*QueryPack),
		Queries:  make(map[string]*Mquery),
		Props:    make(map[string]*Mquery),
	}
}

// BundleFromPaths loads a single bundle file or a bundle that
// was split into multiple files into a single Bundle struct
func BundleFromPaths(paths ...string) (*Bundle, error) {
	// load all the source files
	resolvedFilenames, err := walkBundleFiles(paths)
	if err != nil {
		log.Error().Err(err).Msg("could not resolve bundle files")
		return nil, err
	}

	// aggregate all files into a single bundle
	aggregatedBundle, err := aggregateFilesToBundle(resolvedFilenames)
	if err != nil {
		log.Error().Err(err).Msg("could merge bundle files")
		return nil, err
	}
	return aggregatedBundle, nil
}

// walkBundleFiles iterates over all provided filenames and
// checks if the name is a file or a directory. If the filename
// is a directory, it walks the directory recursively
func walkBundleFiles(filenames []string) ([]string, error) {
	// resolve file names
	resolvedFilenames := []string{}
	for i := range filenames {
		filename := filenames[i]
		fi, err := os.Stat(filename)
		if err != nil {
			return nil, errors.Wrap(err, "could not load bundle file: "+filename)
		}

		if fi.IsDir() {
			filepath.WalkDir(filename, func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					return err
				}
				// we ignore nested directories
				if d.IsDir() {
					return nil
				}

				// only consider .yaml|.yml files
				if strings.HasSuffix(d.Name(), ".yaml") || strings.HasSuffix(d.Name(), ".yml") {
					resolvedFilenames = append(resolvedFilenames, path)
				}

				return nil
			})
		} else {
			resolvedFilenames = append(resolvedFilenames, filename)
		}
	}

	return resolvedFilenames, nil
}

// aggregateFilesToBundle iterates over all provided files and loads its content.
// It assumes that all provided files are checked upfront and are not a directory
func aggregateFilesToBundle(paths []string) (*Bundle, error) {
	// iterate over all files, load them and merge them
	mergedBundle := &Bundle{}

	for i := range paths {
		path := paths[i]
		bundle, err := bundleFromSingleFile(path)
		if err != nil {
			return nil, errors.Wrap(err, "could not load file: "+path)
		}
		combineBundles(mergedBundle, bundle)
	}

	return mergedBundle, nil
}

// Combine two bundles, even if they aren't compiled yet.
// Uses the existing owner MRN if it is set, otherwise the other is used.
func combineBundles(into *Bundle, other *Bundle) {
	if into.OwnerMrn == "" {
		into.OwnerMrn = other.OwnerMrn
	}

	into.Packs = append(into.Packs, other.Packs...)
	into.Queries = append(into.Queries, other.Queries...)
}

// bundleFromSingleFile loads a bundle from a single file
func bundleFromSingleFile(path string) (*Bundle, error) {
	bundleData, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return BundleFromYAML(bundleData)
}

// BundleFromYAML create a bundle from yaml contents
func BundleFromYAML(data []byte) (*Bundle, error) {
	var res Bundle
	err := yaml.Unmarshal(data, &res)
	res.EnsureUIDs()
	return &res, err
}

// ToYAML returns the bundle as yaml
func (p *Bundle) ToYAML() ([]byte, error) {
	return yaml.Marshal(p)
}

func (p *Bundle) SourceHash() (string, error) {
	raw, err := p.ToYAML()
	if err != nil {
		return "", err
	}
	c := checksums.New
	c = c.Add(string(raw))
	return c.String(), nil
}

// ToMap turns the Bundle into a BundleMap
func (p *Bundle) ToMap() *BundleMap {
	res := NewBundleMap(p.OwnerMrn)

	for i := range p.Queries {
		q := p.Queries[i]
		res.Queries[q.Mrn] = q
	}

	for i := range p.Packs {
		c := p.Packs[i]
		res.Packs[c.Mrn] = c
	}

	return res
}

// Add another bundle into this. No duplicate packs, queries, or
// properties are allowed and will lead to an error. Both bundles must have
// MRNs for everything. OwnerMRNs must be identical as well.
func (p *Bundle) AddBundle(other *Bundle) error {
	if p.OwnerMrn == "" {
		p.OwnerMrn = other.OwnerMrn
	} else if p.OwnerMrn != other.OwnerMrn {
		return errors.New("when combining bundles the owner MRNs must be identical")
	}

	for i := range other.Packs {
		c := other.Packs[i]
		if c.Mrn == "" {
			return errors.New("source bundle that is added has missing query pack MRNs")
		}

		for j := range p.Packs {
			if p.Packs[j].Mrn == c.Mrn {
				return errors.New("cannot combine query packs, duplicate query packs: " + c.Mrn)
			}
		}

		p.Packs = append(p.Packs, c)
	}

	return nil
}

// Compile a bundle
// Does a few things:
// 1. turns it into a map for easier access
// 2. compile all queries and validates them
// 3. validation of all contents
// 4. generate MRNs for all packs, queries, and updates referencing local fields
// 5. snapshot all queries into the packs
// 6. make queries public that are only embedded
func (p *Bundle) Compile(ctx context.Context) (*BundleMap, error) {
	ownerMrn := p.OwnerMrn
	if ownerMrn == "" {
		// this only happens for local bundles where queries have no mrn yet
		ownerMrn = "//local.cnquery.io/run/local-execution"
	}

	cache := &bundleCache{
		ownerMrn:    ownerMrn,
		bundle:      p,
		uid2mrn:     map[string]string{},
		lookupProp:  map[string]PropertyRef{},
		lookupQuery: map[string]*Mquery{},
	}

	if err := cache.compileQueries(p.Queries, nil); err != nil {
		return nil, err
	}

	// index packs + update MRNs and checksums, link properties via MRNs
	for i := range p.Packs {
		pack := p.Packs[i]

		// !this is very important to prevent user overrides! vv
		pack.InvalidateAllChecksums()
		pack.ComputedFilters = &Filters{
			Items: map[string]*Mquery{},
		}

		err := pack.RefreshMRN(ownerMrn)
		if err != nil {
			return nil, errors.New("failed to refresh query pack " + pack.Mrn + ": " + err.Error())
		}

		if err = pack.Filters.Compile(ownerMrn); err != nil {
			return nil, errors.Wrap(err, "failed to compile querypack filters")
		}
		pack.ComputedFilters.AddFilters(pack.Filters)

		if err := cache.compileQueries(pack.Queries, pack.ComputedFilters); err != nil {
			return nil, err
		}

		for i := range pack.Groups {
			group := pack.Groups[i]

			// When filters are initially added they haven't been compiled
			if err = group.Filters.Compile(ownerMrn); err != nil {
				return nil, errors.Wrap(err, "failed to compile querypack filters")
			}
			// we must have filters set per group, they are required for selection
			if group.Filters == nil {
				group.Filters = NewFilters()
			}

			if err := cache.compileQueries(group.Queries, group.Filters); err != nil {
				return nil, err
			}

			pack.ComputedFilters.AddFilters(group.Filters)
		}
	}

	return p.ToMap(), cache.error()
}

type bundleCache struct {
	ownerMrn    string
	lookupQuery map[string]*Mquery
	lookupProp  map[string]PropertyRef
	uid2mrn     map[string]string
	bundle      *Bundle
	errors      []error
}

type PropertyRef struct {
	*Property
	Name string
}

func (c *bundleCache) hasErrors() bool {
	return len(c.errors) != 0
}

func (c *bundleCache) error() error {
	if len(c.errors) == 0 {
		return nil
	}

	var msg strings.Builder
	for i := range c.errors {
		msg.WriteString(c.errors[i].Error())
		msg.WriteString("\n")
	}
	return errors.New(msg.String())
}

func (c *bundleCache) compileQueries(queries []*Mquery, filters *Filters) error {
	for i := range queries {
		c.precompileQuery(queries[i], filters == nil)
	}

	for i := range queries {
		c.processQueryFilters(queries[i], filters)
	}

	// After the first pass we may have errors. We try to collect as many errors
	// as we can before returning, so more problems can be fixed at once.
	// We have to return at this point, because these errors will prevent us from
	// compiling the queries.
	if c.hasErrors() {
		return c.error()
	}

	for i := range queries {
		c.compileQuery(queries[i])
	}

	// The second pass on errors is done after we have compiled as much as possible.
	// Since shared queries may be used in other places, any errors here will prevent
	// us from compiling further.
	return c.error()
}

// precompileQuery indexes the query, turns UIDs into MRNs, compiles properties
// and filters, and pre-processes variants. Also makes sure the query isn't nil.
func (c *bundleCache) precompileQuery(query *Mquery, isGlobal bool) {
	if query == nil {
		c.errors = append(c.errors, errors.New("received null query"))
		return
	}

	// remove leading and trailing whitespace of docs, refs and tags
	query.Sanitize()

	// ensure the correct mrn is set
	uid := query.Uid
	if err := query.RefreshMRN(c.ownerMrn); err != nil {
		c.errors = append(c.errors, errors.New("failed to refresh MRN for query "+query.Uid))
		return
	}
	if uid != "" {
		c.uid2mrn[uid] = query.Mrn
	}

	if isGlobal {
		c.lookupQuery[query.Mrn] = query
	} else if existing, ok := c.lookupQuery[query.Mrn]; ok {
		query.AddBase(existing)
	} else {
		// Any other query that is in a pack, that does not exist globally,
		// we share out to be available in the bundle.
		c.bundle.Queries = append(c.bundle.Queries, query)
		c.lookupQuery[query.Mrn] = query
	}

	// ensure MRNs for properties
	for i := range query.Props {
		if err := c.compileProp(query.Props[i]); err != nil {
			c.errors = append(c.errors, errors.New("failed to compile properties for query "+query.Mrn))
			return
		}
	}

	// ensure MRNs for variants
	for i := range query.Variants {
		variant := query.Variants[i]
		uid := variant.Uid
		if err := variant.RefreshMRN(c.ownerMrn); err != nil {
			c.errors = append(c.errors, errors.New("failed to refresh MRN for variant in query "+query.Uid))
			return
		}
		if uid != "" {
			c.uid2mrn[uid] = variant.Mrn
		}
	}
}

// prepareQuery turns UIDs into MRNs and indexes the queries, compiles properties
// and filters, and pre-processes variants. Also makes sure the query isn't nil.
func (c *bundleCache) processQueryFilters(query *Mquery, filters *Filters) {
	query = c.lookupQuery[query.Mrn]

	// filters have no dependencies, so we can compile them early
	if err := query.Filters.Compile(c.ownerMrn); err != nil {
		c.errors = append(c.errors, errors.New("failed to compile filters for query "+query.Mrn))
		return
	}

	// filters will need to be aggregated into the pack's filters
	if filters != nil {
		if err := filters.AddQueryFilters(query, c.lookupQuery); err != nil {
			c.errors = append(c.errors, errors.New("failed to register filters for query "+query.Mrn))
			return
		}
	}
}

// Note: you only want to run this, after you are sure that all connected
// dependencies have been processed. Properties must be compiled. Connected
// queries may not be ready yet, but we have to have precompiled them.
func (c *bundleCache) compileQuery(query *Mquery) {
	_, err := query.RefreshChecksumAndType(c.lookupQuery, c.lookupProp)
	if err != nil {
		c.errors = append(c.errors, errors.Wrap(err, "failed to validate query '"+query.Mrn+"'"))
	}
}

func (c *bundleCache) compileProp(prop *Property) error {
	var name string

	if prop.Mrn == "" {
		uid := prop.Uid
		if err := prop.RefreshMRN(c.ownerMrn); err != nil {
			return err
		}
		if uid != "" {
			c.uid2mrn[uid] = prop.Mrn
		}

		// TODO: uid's can be namespaced, extract the name
		name = uid
	} else {
		m, err := mrn.NewMRN(prop.Mrn)
		if err != nil {
			return errors.Wrap(err, "failed to compile prop, invalid mrn: "+prop.Mrn)
		}

		name = m.Basename()
	}

	if _, err := prop.RefreshChecksumAndType(); err != nil {
		return err
	}

	c.lookupProp[prop.Mrn] = PropertyRef{
		Property: prop,
		Name:     name,
	}

	return nil
}

// FilterQueryPacks only keeps the given UIDs or MRNs and removes every other one.
// If a given query pack has a MRN set (but no UID) it will try to get the UID from the MRN
// and also filter by that criteria.
// If the list of IDs is empty this function doesn't do anything.
// If all packs in the bundles were filtered out, return true.
func (p *Bundle) FilterQueryPacks(IDs []string) bool {
	if len(IDs) == 0 {
		return false
	}

	if p == nil {
		return true
	}

	valid := make(map[string]struct{}, len(IDs))
	for i := range IDs {
		valid[IDs[i]] = struct{}{}
	}

	var res []*QueryPack
	for i := range p.Packs {
		cur := p.Packs[i]

		if cur.Mrn != "" {
			if _, ok := valid[cur.Mrn]; ok {
				res = append(res, cur)
				continue
			}

			uid, _ := mrn.GetResource(cur.Mrn, MRN_RESOURCE_QUERYPACK)
			if _, ok := valid[uid]; ok {
				res = append(res, cur)
			}

			// if we have a MRN we do not check the UID
			continue
		}

		if _, ok := valid[cur.Uid]; ok {
			res = append(res, cur)
		}
	}

	p.Packs = res

	return len(res) == 0
}

// Makes sure every query in the bundle and every query pack has a UID set,
// IF the MRN is empty. Otherwise MRNs suffice.
func (p *Bundle) EnsureUIDs() {
	for i := range p.Packs {
		pack := p.Packs[i]
		if pack.Mrn == "" && pack.Uid == "" {
			pack.Uid = ksuid.New().String()
		}

		for j := range pack.Queries {
			query := pack.Queries[j]
			if query.Mrn == "" && query.Uid == "" {
				query.Uid = ksuid.New().String()
			}
		}
	}
}

// Filters retrieves the aggregated filters for all querypacks in this bundle.
func (p *Bundle) Filters() []*Mquery {
	uniq := map[string]*Mquery{}
	for i := range p.Packs {
		// TODO: Currently we don't process the difference between local pack filters
		// and their group filters correctly. These need aggregation.

		pack := p.Packs[i]
		if pack.ComputedFilters != nil {
			for k, v := range pack.ComputedFilters.Items {
				uniq[k] = v
			}
		}
	}

	res := make([]*Mquery, len(uniq))
	i := 0
	for _, v := range uniq {
		res[i] = v
		i++
	}

	return res
}
