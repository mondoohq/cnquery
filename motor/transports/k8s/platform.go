package k8s

func (t *Transport) Identifier() (string, error) {
	// TODO: we need a cluster id to make this reliable
	return "//platformid.api.mondoo.app/runtime/k8s", nil
}
