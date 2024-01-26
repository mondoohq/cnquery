// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

// Add another schema and return yourself. other may be nil.
// The other schema overrides specifications in this schema, unless
// it is trying to extend a resource whose base is already defined.
func (s *Schema) Add(other *Schema) *Schema {
	if other == nil {
		return s
	}

	for k, v := range other.Resources {
		if existing, ok := s.Resources[k]; ok {
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

			if existing.Fields == nil {
				existing.Fields = map[string]*Field{}
			}
			for fk, fv := range v.Fields {
				existing.Fields[fk] = fv
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
				Provider:         v.Provider,
			}
			for k, v := range v.Fields {
				ri.Fields[k] = v
			}
			s.Resources[k] = ri
		}
	}

	return s
}

func (s *Schema) Lookup(name string) *ResourceInfo {
	return s.Resources[name]
}

func (s *Schema) LookupField(resource string, field string) (*ResourceInfo, *Field) {
	res := s.Lookup(resource)
	if res == nil || res.Fields == nil {
		return res, nil
	}
	return res, res.Fields[field]
}

func (s *Schema) AllResources() map[string]*ResourceInfo {
	return s.Resources
}
