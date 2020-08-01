package gcp

func (t *Transport) Identifier() (string, error) {
	return "//platformid.api.mondoo.app/runtime/gcp/projects/" + t.projectid, nil
}

func (t *Transport) ProjectID() string {
	return t.projectid
}
