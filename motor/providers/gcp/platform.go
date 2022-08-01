package gcp

import "errors"

func (t *Transport) Identifier() (string, error) {
	switch t.ResourceType() {
	case Project:
		return "//platformid.api.mondoo.app/runtime/gcp/projects/" + t.id, nil
	default:
		return "", errors.New("unsupported resource type")
	}
}

func (t *Transport) ResourceType() ResourceType {
	return t.resourceType
}

func (t *Transport) ResourceID() string {
	return t.id
}
