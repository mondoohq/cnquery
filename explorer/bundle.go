// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package explorer

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v11"
	"go.mondoo.com/cnquery/v11/checksums"
	"go.mondoo.com/cnquery/v11/mqlc"
	"go.mondoo.com/cnquery/v11/mrn"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/resources"
	"go.mondoo.com/cnquery/v11/utils/multierr"
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
			return nil, multierr.Wrap(err, "could not load bundle file: "+filename)
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
			return nil, multierr.Wrap(err, "could not load file: "+path)
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

// Compile a bundle. See CompileExt for a full description.
func (p *Bundle) Compile(ctx context.Context, schema resources.ResourcesSchema) (*BundleMap, error) {
	return p.CompileExt(ctx, BundleCompileConf{
		CompilerConfig: mqlc.NewConfig(schema, cnquery.DefaultFeatures),
	})
}

type BundleCompileConf struct {
	mqlc.CompilerConfig
	RemoveFailing bool
}

// Compile a bundle
// Does a few things:
// 1. turns it into a map for easier access
// 2. compile all queries and validates them
// 3. validation of all contents
// 4. generate MRNs for all packs, queries, and updates referencing local fields
// 5. snapshot all queries into the packs
// 6. make queries public that are only embedded
func (bundle *Bundle) CompileExt(ctx context.Context, conf BundleCompileConf) (*BundleMap, error) {
	ownerMrn := bundle.OwnerMrn
	if ownerMrn == "" {
		// this only happens for local bundles where queries have no mrn yet
		ownerMrn = "//local.cnquery.io/run/local-execution"
	}

	cache := &bundleCache{
		ownerMrn:      ownerMrn,
		bundle:        bundle,
		uid2mrn:       map[string]string{},
		removeQueries: map[string]struct{}{},
		lookupProp:    map[string]PropertyRef{},
		lookupQuery:   map[string]*Mquery{},
		conf:          conf,
	}

	if err := cache.compileQueries(bundle.Queries, nil); err != nil {
		return nil, err
	}

	// index packs + update MRNs and checksums, link properties via MRNs
	for i := range bundle.Packs {
		pack := bundle.Packs[i]

		// !this is very important to prevent user overrides! vv
		pack.InvalidateAllChecksums()
		pack.ComputedFilters = &Filters{
			Items: map[string]*Mquery{},
		}

		err := pack.RefreshMRN(ownerMrn)
		if err != nil {
			return nil, err
		}

		if err = pack.Filters.Compile(ownerMrn, conf.CompilerConfig); err != nil {
			return nil, multierr.Wrap(err, "failed to compile querypack filters")
		}
		pack.ComputedFilters.AddFilters(pack.Filters)

		if err := cache.compileQueries(pack.Queries, pack); err != nil {
			return nil, err
		}

		for i := range pack.Groups {
			group := pack.Groups[i]

			// When filters are initially added they haven't been compiled
			if err = group.Filters.Compile(ownerMrn, conf.CompilerConfig); err != nil {
				return nil, multierr.Wrap(err, "failed to compile querypack filters")
			}
			pack.ComputedFilters.AddFilters(group.Filters)

			if err := cache.compileQueries(group.Queries, pack); err != nil {
				return nil, err
			}
		}
	}

	// Removing any failing queries happens at the very end, when everything is
	// set to go. We do this to the original bundle, because the intent is to
	// clean it up with this option.
	cache.removeFailing(bundle)

	return bundle.ToMap(), cache.error()
}

type bundleCache struct {
	ownerMrn      string
	lookupQuery   map[string]*Mquery
	lookupProp    map[string]PropertyRef
	uid2mrn       map[string]string
	removeQueries map[string]struct{}
	bundle        *Bundle
	errors        []error
	conf          BundleCompileConf
}

type PropertyRef struct {
	*Property
	Name string
}

func (c *bundleCache) removeFailing(res *Bundle) {
	if !c.conf.RemoveFailing {
		return
	}

	res.Queries = FilterQueryMRNs(c.removeQueries, res.Queries)

	for i := range res.Packs {
		pack := res.Packs[i]
		pack.Queries = FilterQueryMRNs(c.removeQueries, pack.Queries)

		groups := []*QueryGroup{}
		for j := range pack.Groups {
			group := pack.Groups[j]
			group.Queries = FilterQueryMRNs(c.removeQueries, group.Queries)
			if len(group.Queries) != 0 {
				groups = append(groups, group)
			}
		}

		pack.Groups = groups
	}
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
		if i != 0 {
			msg.WriteString("\n")
		}
		msg.WriteString(c.errors[i].Error())
	}
	return errors.New(msg.String())
}

func (c *bundleCache) compileQueries(queries []*Mquery, pack *QueryPack) error {
	for i := range queries {
		c.precompileQuery(queries[i], pack)
	}

	// Topologically sort the queries so that variant queries are compiled after the
	// actual query they include.
	topoSortedQueries, err := topologicalSortQueries(queries)
	if err != nil {
		return err
	}
	for i := range topoSortedQueries {
		// filters will need to be aggregated into the pack's filters
		if pack != nil {
			if err := pack.ComputedFilters.AddQueryFilters(topoSortedQueries[i], c.lookupQuery); err != nil {
				c.errors = append(c.errors, err)
				c.errors = append(c.errors, errors.New("failed to register filters for query "+topoSortedQueries[i].Mrn))
			}
		}
	}

	// After the first pass we may have errors. We try to collect as many errors
	// as we can before returning, so more problems can be fixed at once.
	// We have to return at this point, because these errors will prevent us from
	// compiling the queries.
	if c.hasErrors() {
		return c.error()
	}

	for i := range topoSortedQueries {
		c.compileQuery(topoSortedQueries[i])
	}

	// The second pass on errors is done after we have compiled as much as possible.
	// Since shared queries may be used in other places, any errors here will prevent
	// us from compiling further.
	return c.error()
}

// precompileQuery indexes the query, turns UIDs into MRNs, compiles properties
// and filters, and pre-processes variants. Also makes sure the query isn't nil.
func (c *bundleCache) precompileQuery(query *Mquery, pack *QueryPack) {
	if query == nil {
		c.errors = append(c.errors, errors.New("received null query"))
		return
	}

	// remove leading and trailing whitespace of docs, refs and tags
	query.Sanitize()

	// ensure the correct mrn is set
	uid := query.Uid
	if err := query.RefreshMRN(c.ownerMrn); err != nil {
		c.errors = append(c.errors, err)
		return
	}
	if uid != "" {
		c.uid2mrn[uid] = query.Mrn
	}

	// the pack is only nil if we are dealing with shared queries
	if pack == nil {
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

	// filters have no dependencies, so we can compile them early
	if err := query.Filters.Compile(c.ownerMrn, c.conf.CompilerConfig); err != nil {
		c.errors = append(c.errors, errors.New("failed to compile filters for query "+query.Mrn))
		return
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

// Note: you only want to run this, after you are sure that all connected
// dependencies have been processed. Properties must be compiled. Connected
// queries may not be ready yet, but we have to have precompiled them.
func (c *bundleCache) compileQuery(query *Mquery) {
	_, err := query.RefreshChecksumAndType(c.lookupQuery, c.lookupProp, c.conf.CompilerConfig)
	if err != nil {
		if c.conf.RemoveFailing {
			c.removeQueries[query.Mrn] = struct{}{}
		} else {
			c.errors = append(c.errors, multierr.Wrap(err, "failed to validate query '"+query.Mrn+"'"))
		}
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
			return multierr.Wrap(err, "failed to compile prop, invalid mrn: "+prop.Mrn)
		}

		name = m.Basename()
	}

	if _, err := prop.RefreshChecksumAndType(c.conf.CompilerConfig); err != nil {
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

func topologicalSortQueries(queries []*Mquery) ([]*Mquery, error) {
	// Gather all top-level queries with variants
	queriesMap := map[string]*Mquery{}
	for _, q := range queries {
		if q == nil {
			continue
		}
		if q.Mrn == "" {
			// This should never happen. This function is called after all
			// queries have their MRNs set.
			panic("BUG: expected query MRN to be set for topological sort")
		}
		queriesMap[q.Mrn] = q
	}

	// Topologically sort the queries
	sorted := &Mqueries{}
	visited := map[string]struct{}{}
	for _, q := range queriesMap {
		err := topologicalSortQueriesDFS(q.Mrn, queriesMap, visited, sorted)
		if err != nil {
			return nil, err
		}
	}

	return sorted.Items, nil
}

func topologicalSortQueriesDFS(queryMrn string, queriesMap map[string]*Mquery, visited map[string]struct{}, sorted *Mqueries) error {
	if _, ok := visited[queryMrn]; ok {
		return nil
	}
	visited[queryMrn] = struct{}{}
	q := queriesMap[queryMrn]
	if q == nil {
		return nil
	}
	for _, variant := range q.Variants {
		if variant.Mrn == "" {
			// This should never happen. This function is called after all
			// queries have their MRNs set.
			panic("BUG: expected variant MRN to be set for topological sort")
		}
		err := topologicalSortQueriesDFS(variant.Mrn, queriesMap, visited, sorted)
		if err != nil {
			return err
		}
	}
	sorted.Items = append(sorted.Items, q)
	return nil
}
