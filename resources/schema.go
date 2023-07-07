package resources

func (s *Schema) Add(other *Schema) {
	if other == nil {
		return
	}

	for k, v := range other.Resources {
		s.Resources[k] = v
	}
}
