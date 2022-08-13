package gcp

import "errors"

func (t *Provider) Identifier() (string, error) {
	switch t.ResourceType() {
	case Project:
		return "//platformid.api.mondoo.app/runtime/gcp/projects/" + t.id, nil
	default:
		return "", errors.New("unsupported resource type")
	}
}

func (t *Provider) ResourceType() ResourceType {
	return t.resourceType
}

func (t *Provider) ResourceID() string {
	return t.id
}
