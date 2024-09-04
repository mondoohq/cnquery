// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/go-cleanhttp"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/vault"
	"golang.org/x/oauth2"
	googleoauth "golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/api/transport"
)

func (c *GcpConnection) Credentials(scopes ...string) (*googleoauth.Credentials, error) {
	ctx := context.Background()
	credParams := googleoauth.CredentialsParams{
		Scopes:  scopes,
		Subject: c.serviceAccountSubject,
	}
	if c.cred != nil {
		// use service account from secret
		data, err := credsServiceAccountData(c.cred)
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
	if c.cred != nil {
		data, err := credsServiceAccountData(c.cred)
		if err != nil {
			return nil, err
		}
		return serviceAccountAuth(ctx, c.serviceAccountSubject, data, scope...)
	}

	// otherwise fallback to default google sdk authentication
	log.Debug().Msg("fallback to default google sdk authentication")
	return defaultAuth(ctx, scope...)
}

// defaultAuth implements the
func defaultAuth(ctx context.Context, scope ...string) (*http.Client, error) {
	return googleoauth.DefaultClient(ctx, scope...)
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
