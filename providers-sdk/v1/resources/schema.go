// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"go.mondoo.com/cnquery/v11/types"
)

type ResourcesSchema interface {
	Lookup(resource string) *ResourceInfo
	LookupField(resource string, field string) (*ResourceInfo, *Field)
	FindField(resource *ResourceInfo, field string) (FieldPath, []*Field, bool)
	AllResources() map[string]*ResourceInfo
	AllDependencies() map[string]*ProviderInfo
}

// Add another schema and return yourself. other may be nil.
// The other schema overrides specifications in this schema, unless
// it is trying to extend a resource whose base is already defined.
func (s *Schema) Add(other ResourcesSchema) ResourcesSchema {
	if other == nil {
		return s
	}

	for k, v := range other.AllResources() {
		if existing, ok := s.Resources[k]; ok {
			// If neither resource is an extension, we can't merge them. We store both references.
			if !v.IsExtension && !existing.IsExtension && v.Provider != existing.Provider {
				existing.Others = append(existing.Others, v)
				continue
			}

			// We will merge resources into it until we find one that is not extending.
			// Technically, this should only happen with one resource and one only,
			// i.e. the root resource. In case they are incorrectly specified, the
			// last added resource wins (as is the case with all other fields below).
			if !v.IsExtension || existing.IsExtension {
				existing.IsExtension = v.IsExtension
				existing.Provider = v.Provider
				existing.Init = v.Init
			}
			// TODO: clean up any resource that clashes right now. There are a few
			//       implicit extensions that cause this behavior at the moment.
			//       log.Warn().Str("resource", k).Msg("found a resource that is not flagged as `extends` properly")
			// else if !v.IsExtension {}

			if v.Title != "" {
				existing.Title = v.Title
			}
			if v.Name != "" {
				existing.Name = v.Name
			}
			if v.MinMondooVersion != "" {
				existing.MinMondooVersion = v.MinMondooVersion
			}
			if v.Desc != "" {
				existing.Desc = v.Desc
			}
			if !v.Private {
				existing.Private = false
			}
			if v.Defaults != "" {
				existing.Defaults = v.Defaults
			}
			if v.Context != "" {
				existing.Context = v.Context
			}

			if existing.Fields == nil {
				existing.Fields = map[string]*Field{}
			}
			for fk, fv := range v.Fields {
				// If the field exists in the current resource, but is from a different provider,
				// we store it as an "other"
				if fExisting, ok := existing.Fields[fk]; ok && fv.Provider != fExisting.Provider {
					fExisting.Others = append(fExisting.Others, fv)
				} else {
					existing.Fields[fk] = fv
				}
			}
		} else {
			ri := &ResourceInfo{
				Id:               v.Id,
				Name:             v.Name,
				Fields:           make(map[string]*Field, len(v.Fields)),
				Init:             v.Init,
				ListType:         v.ListType,
				Title:            v.Title,
				Desc:             v.Desc,
				Private:          v.Private,
				IsExtension:      v.IsExtension,
				MinMondooVersion: v.MinMondooVersion,
				Defaults:         v.Defaults,
				Context:          v.Context,
				Provider:         v.Provider,
			}
			for k, v := range v.Fields {
				ri.Fields[k] = v
			}
			s.Resources[k] = ri
		}
	}

	for k, v := range other.AllDependencies() {
		if existing, ok := s.Dependencies[k]; ok {
			if v.Name != "" {
				existing.Name = v.Name
			}
		} else {
			pi := &ProviderInfo{
				Id:   v.Id,
				Name: v.Name,
			}
			if s.Dependencies == nil {
				s.Dependencies = make(map[string]*ProviderInfo)
			}
			s.Dependencies[k] = pi
		}
	}

	return s
}

func (s *Schema) Lookup(name string) *ResourceInfo {
	return s.Resources[name]
}

func (s *Schema) LookupField(resource string, field string) (*ResourceInfo, *Field) {
	res := s.Lookup(resource)
	if res == nil {
		return res, nil
	}

	// If the fields don't exist in the current resource, check the other instances of it
	if res.Fields == nil {
		for _, o := range res.Others {
			if o.Fields != nil && o.Fields[field] != nil {
				res = o
				break
			}
		}
	}
	return res, res.Fields[field]
}

type FieldPath []string

func (s *Schema) FindField(resource *ResourceInfo, field string) (FieldPath, []*Field, bool) {
	fieldInfo, ok := resource.Fields[field]
	if ok {
		return FieldPath{field}, []*Field{fieldInfo}, true
	}

	for _, f := range resource.Fields {
		if f.IsEmbedded {
			typ := types.Type(f.Type)
			nextResource := s.Lookup(typ.ResourceName())
			if nextResource == nil {
				continue
			}
			childFieldPath, childFieldInfos, ok := s.FindField(nextResource, field)
			if ok {
				fp := make(FieldPath, len(childFieldPath)+1)
				fieldInfos := make([]*Field, len(childFieldPath)+1)
				fp[0] = f.Name
				fieldInfos[0] = f
				for i, n := range childFieldPath {
					fp[i+1] = n
				}
				for i, f := range childFieldInfos {
					fieldInfos[i+1] = f
				}
				return fp, fieldInfos, true
			}
		}
	}
	return nil, nil, false
}

func (s *Schema) AllResources() map[string]*ResourceInfo {
	return s.Resources
}

func (s *Schema) AllDependencies() map[string]*ProviderInfo {
	return s.Dependencies
}
