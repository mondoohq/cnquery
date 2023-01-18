package google

import (
	"context"
	"net/http"

	"github.com/hashicorp/go-cleanhttp"
	"github.com/rs/zerolog/log"
	"golang.org/x/oauth2"
	googleoauth "golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/api/transport"
)

func (t *Provider) Credentials(scopes ...string) (*googleoauth.Credentials, error) {
	ctx := context.Background()
	credParams := googleoauth.CredentialsParams{
		Scopes:  scopes,
		Subject: t.serviceAccountSubject,
	}
	if t.cred != nil {
		// use service account from secret
		return googleoauth.CredentialsFromJSONWithParams(ctx, t.cred.Secret, credParams)
	}

	if t.serviceAccountSubject != "" {
		// use custom service account provided by user
		return googleoauth.CredentialsFromJSONWithParams(ctx, t.serviceAccount, credParams)
	}

	// otherwise fallback to default google sdk authentication
	log.Debug().Msg("fallback to default google sdk authentication")
	return googleoauth.FindDefaultCredentials(ctx, scopes...)
}

func (t *Provider) Client(scope ...string) (*http.Client, error) {
	ctx := context.Background()

	// use service account from secret if one is provided
	if t.cred != nil {
		return serviceAccountAuth(ctx, t.serviceAccountSubject, t.cred.Secret, scope...)
	}

	// use service account authentication if we loaded a service account
	if t.serviceAccount != nil {
		return serviceAccountAuth(ctx, t.serviceAccountSubject, t.serviceAccount, scope...)
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
		return nil, err
	}

	cleanCtx := context.WithValue(ctx, oauth2.HTTPClient, cleanhttp.DefaultClient())
	client, _, err := transport.NewHTTPClient(cleanCtx, option.WithTokenSource(credentials.TokenSource))
	if err != nil {
		return nil, err
	}

	return client, nil
}
