package gcp

import (
	"context"
	"net/http"

	"golang.org/x/oauth2/google"
)

func (t *Provider) Client(scope ...string) (*http.Client, error) {
	return Client(scope...)
}

func Client(scope ...string) (*http.Client, error) {
	ctx := context.Background()
	return google.DefaultClient(ctx, scope...)
}
