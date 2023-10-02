package resources

func (r *mqlAtlassian) id() (string, error) {
	return "atlassian", nil
}

func (r *mqlAtlassian) field() (string, error) {
	return "example", nil
}
