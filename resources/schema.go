package resources

func (s *Schema) Add(other *Schema) {
	if other == nil {
		return
	}

	for k, v := range other.Resources {
		s.Resources[k] = v
	}
}

func (s *Schema) Lookup(name string) *ResourceInfo {
	return s.Resources[name]
}

func (s *Schema) AllResources() map[string]*ResourceInfo {
	return s.Resources
}
