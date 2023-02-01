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
	MRN_RESOURCE_QUERYPACK = "querypack"
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

		bundle.EnsureUIDs()

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

type PropertyRef struct {
	*Property
	Name string
}

func (p *Bundle) compileProp(prop *Property, ownerMrn string, lookupProp map[string]PropertyRef, uid2mrn map[string]string) error {
	var name string

	if prop.Mrn == "" {
		uid := prop.Uid
		if err := prop.RefreshMRN(ownerMrn); err != nil {
			return err
		}
		if uid != "" {
			uid2mrn[uid] = prop.Mrn
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

	lookupProp[prop.Mrn] = PropertyRef{
		Property: prop,
		Name:     name,
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

	var warnings []error
	var err error

	uid2mrn := map[string]string{}

	// Index properties
	lookupProp := map[string]PropertyRef{}
	lookupQuery := map[string]*Mquery{}

	// Index queries + update MRNs and checksums
	for i := range p.Queries {
		query := p.Queries[i]
		if query == nil {
			return nil, errors.New("received null query")
		}

		// remove leading and trailing whitespace of docs, refs and tags
		query.Sanitize()

		// ensure the correct mrn is set
		uid := query.Uid
		if err = query.RefreshMRN(ownerMrn); err != nil {
			return nil, err
		}
		if uid != "" {
			uid2mrn[uid] = query.Mrn
		}
		lookupQuery[query.Mrn] = query

		// ensure MRNs for properties
		for i := range query.Props {
			if err = p.compileProp(query.Props[i], ownerMrn, lookupProp, uid2mrn); err != nil {
				return nil, err
			}
		}

		// ensure MRNs for compositions
		for i := range query.Compose {
			comp := query.Compose[i]
			uid := comp.Uid
			if err = comp.RefreshMRN(ownerMrn); err != nil {
				return nil, err
			}
			if uid != "" {
				uid2mrn[uid] = comp.Mrn
			}
		}

		// recalculate the checksums
		_, err := query.RefreshChecksumAndType(lookupProp)
		if err != nil {
			log.Error().Err(err).Msg("could not compile the query")
			warnings = append(warnings, errors.Wrap(err, "failed to validate query '"+query.Mrn+"'"))
		}
	}

	// Index packs + update MRNs and checksums, link properties via MRNs
	for i := range p.Packs {
		querypack := p.Packs[i]
		if querypack.Filters == nil {
			querypack.Filters = &Filters{
				Items: map[string]*Mquery{},
			}
		}

		// !this is very important to prevent user overrides! vv
		querypack.InvalidateAllChecksums()

		err := querypack.RefreshMRN(ownerMrn)
		if err != nil {
			return nil, errors.New("failed to refresh query pack " + querypack.Mrn + ": " + err.Error())
		}

		for i := range querypack.Queries {
			query := querypack.Queries[i]

			// remove leading and trailing whitespace of docs, refs and tags
			query.Sanitize()

			// ensure the correct mrn is set
			if err = query.RefreshMRN(ownerMrn); err != nil {
				return nil, err
			}

			existing, ok := lookupQuery[query.Mrn]
			if ok {
				query.Merge(existing)
				query.RefreshChecksumAndType(lookupProp)
				continue
			}

			// recalculate the checksums
			_, err := query.RefreshChecksumAndType(lookupProp)
			if err != nil {
				log.Error().Err(err).Msg("could not compile the query")
				warnings = append(warnings, errors.Wrap(err, "failed to validate query '"+query.Mrn+"'"))
			}

			lookupQuery[query.Mrn] = query

			// we may have embed-only queries, that we externalize and make available
			p.Queries = append(p.Queries, query)
		}
	}

	res := p.ToMap()

	if len(warnings) != 0 {
		var msg strings.Builder
		for i := range warnings {
			msg.WriteString(warnings[i].Error())
			msg.WriteString("\n")
		}
		return res, errors.New(msg.String())
	}

	return res, nil
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

func (p *Bundle) Filters() []*Mquery {
	uniq := map[string]*Mquery{}
	for i := range p.Packs {
		pack := p.Packs[i]
		if pack.Filters == nil {
			continue
		}
		for k, v := range pack.Filters.Items {
			uniq[k] = v
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
