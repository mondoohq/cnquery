package gcp

import (
	"context"
	"net/http"

	"golang.org/x/oauth2/google"
)

func gcpClient(scope ...string) (*http.Client, error) {
	ctx := context.Background()
	return google.DefaultClient(ctx, scope...)
}
