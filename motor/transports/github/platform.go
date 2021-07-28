package github

import (
	"context"

	"github.com/google/go-github/v37/github"
)

func (t *Transport) Identifier() (string, error) {
	// TODO: if no organization is provided, we need to see this from the user perspective
	orgId := t.opts["organization"]
	return "//platformid.api.mondoo.app/runtime/github/organization/" + orgId, nil
}

func (t *Transport) Organization() (*github.Organization, error) {
	orgId := t.opts["organization"]
	org, _, err := t.Client().Organizations.Get(context.Background(), orgId)
	return org, err
}
