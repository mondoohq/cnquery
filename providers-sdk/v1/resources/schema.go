// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

func (s *Schema) Add(other *Schema) *Schema {
	if other == nil {
		return s
	}

	for k, v := range other.Resources {
		if existing, ok := s.Resources[k]; ok {
			// We will merge resources into it until we find one that is not extending.
			// Technically, this should only happen with one resource and one only,
			// i.e. the root resource. This is more of a protection.
			if existing.IsExtension {
				existing.IsExtension = v.IsExtension
				existing.Provider = v.Provider
				existing.Init = v.Init
			} else if !v.IsExtension {
				// TODO: clean up any resource that clashes right now. There are a few
				// implicit extensions that cause this behavior at the moment.
				// log.Warn().Str("resource", k).Msg("found a resource that is not flagged as `extends` properly")
			}

			if v.Title != "" {
				existing.Title = v.Title
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
				existing.Fields = v.Fields
			} else {
				for fk, fv := range v.Fields {
					existing.Fields[fk] = fv
				}
			}
		} else {
			s.Resources[k] = v
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
