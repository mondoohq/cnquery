package resources

func (s *Schema) Add(other *Schema) *Schema {
	if other == nil {
		return s
	}

	for k, v := range other.Resources {
		s.Resources[k] = v
	}

	return s
}

func (s *Schema) Lookup(name string) *ResourceInfo {
	return s.Resources[name]
}

func (s *Schema) AllResources() map[string]*ResourceInfo {
	return s.Resources
}
