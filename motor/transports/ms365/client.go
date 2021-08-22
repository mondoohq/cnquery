package ms365

import (
	"context"
	"net/http"

	msgraph "github.com/yaegashi/msgraph.go/beta"
	msgraphbeta "github.com/yaegashi/msgraph.go/beta"
	"github.com/yaegashi/msgraph.go/msauth"
	"golang.org/x/oauth2"
)

var DefaultMSGraphScopes = []string{msauth.DefaultMSGraphScope}

func (t *Transport) GraphClient() (*msgraph.GraphServiceRequestBuilder, error) {
	httpClient, err := t.httpClient()
	if err != nil {
		return nil, err
	}

	graphClient := msgraph.NewClient(httpClient)
	return graphClient, nil
}

func (t *Transport) GraphBetaClient() (*msgraphbeta.GraphServiceRequestBuilder, error) {
	httpClient, err := t.httpClient()
	if err != nil {
		return nil, err
	}

	graphBetaClient := msgraphbeta.NewClient(httpClient)
	return graphBetaClient, nil
}

// httpClient prepares the agent client with oauth2 bearer token
func (t *Transport) httpClient() (*http.Client, error) {
	ctx := context.Background()
	m := msauth.NewManager()
	ts, err := m.ClientCredentialsGrant(ctx, t.tenantID, t.clientID, t.clientSecret, DefaultMSGraphScopes)
	if err != nil {
		return nil, err
	}

	httpClient := oauth2.NewClient(ctx, ts)
	return httpClient, nil
}
