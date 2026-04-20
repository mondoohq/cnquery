// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/go-cleanhttp"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/providers-sdk/v1/vault"
	"golang.org/x/oauth2"
	googleoauth "golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/api/transport"
)

func (c *GcpConnection) Credentials(scopes ...string) (*googleoauth.Credentials, error) {
	ctx := context.Background()
	credParams := googleoauth.CredentialsParams{
		Scopes:  scopes,
		Subject: c.opts.serviceAccountSubject,
	}
	if c.opts.cred != nil {
		// use service account from secret
		data, err := credsServiceAccountData(c.opts.cred)
		if err != nil {
			return nil, err
		}
		return googleoauth.CredentialsFromJSONWithParams(ctx, data, credParams)
	}

	// otherwise fallback to default google sdk authentication
	log.Debug().Msg("fallback to default google sdk authentication")
	return googleoauth.FindDefaultCredentials(ctx, scopes...)
}

func (c *GcpConnection) Client(scope ...string) (*http.Client, error) {
	ctx := context.Background()

	// use service account from secret if one is provided
	if c.opts.cred != nil {
		data, err := credsServiceAccountData(c.opts.cred)
		if err != nil {
			return nil, err
		}
		return serviceAccountAuth(ctx, c.opts.serviceAccountSubject, data, scope...)
	}

	// otherwise fallback to default google sdk authentication
	log.Debug().Msg("fallback to default google sdk authentication")
	return defaultAuth(ctx, scope...)
}

// defaultAuth builds an HTTP client from Application Default Credentials.
// It routes through transport.NewHTTPClient with option.WithCredentials so that
// the quota project (from quota_project_id in the ADC JSON, or the
// GOOGLE_CLOUD_QUOTA_PROJECT env var) is propagated as the X-Goog-User-Project
// header. Going through googleoauth.DefaultClient skips that plumbing and
// causes APIs that bill the caller (apikeys, serviceusage,
// cloudresourcemanager) to fail with 403 against Google's default SDK project.
func defaultAuth(ctx context.Context, scope ...string) (*http.Client, error) {
	creds, err := googleoauth.FindDefaultCredentials(ctx, scope...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to find google default credentials")
	}

	client, _, err := transport.NewHTTPClient(ctx, option.WithCredentials(creds))
	if err != nil {
		return nil, errors.Wrap(err, "failed to create google http client")
	}
	return client, nil
}

// serviceAccountAuth implements
func serviceAccountAuth(ctx context.Context, subject string, serviceAccount []byte, scopes ...string) (*http.Client, error) {
	credParams := googleoauth.CredentialsParams{
		Scopes:  scopes,
		Subject: subject,
	}

	credentials, err := googleoauth.CredentialsFromJSONWithParams(ctx, serviceAccount, credParams)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create google credentials")
	}

	cleanCtx := context.WithValue(ctx, oauth2.HTTPClient, cleanhttp.DefaultClient())
	client, _, err := transport.NewHTTPClient(cleanCtx, option.WithTokenSource(credentials.TokenSource))
	if err != nil {
		return nil, errors.Wrap(err, "failed to create google http client")
	}

	return client, nil
}

func credsServiceAccountData(cred *vault.Credential) ([]byte, error) {
	switch cred.Type {
	case vault.CredentialType_json:
		return cred.Secret, nil
	default:
		return nil, fmt.Errorf("unsupported credential type: %s", cred.Type)
	}
}
