package resources

func (s *Schema) Add(other *Schema) *Schema {
	if other == nil {
		return s
	}

	for k, v := range other.Resources {
		if existing, ok := s.Resources[k]; ok {
			if v.Title != "" {
				existing.Title = v.Title
			}
			if v.Desc != "" {
				existing.Desc = v.Desc
			}
			if !v.Private {
				existing.Private = false
			}
			if v.Init != nil {
				existing.Init = v.Init
				existing.Provider = v.Provider
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
