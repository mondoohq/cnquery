package equinix

func (t *Transport) Identifier() (string, error) {
	return "//platformid.api.mondoo.app/runtime/equinix/projects/" + t.projectId, nil // TODO: this is not specific enough
}
