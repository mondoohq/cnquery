package ms365

import (
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"

	msgraph "github.com/yaegashi/msgraph.go/beta"
	"github.com/yaegashi/msgraph.go/msauth"
	"golang.org/x/oauth2"

	msgraphbeta "github.com/yaegashi/msgraph.go/beta"
)

func (t *Transport) GraphClient() (*msgraph.GraphServiceRequestBuilder, error) {
	var scopes = []string{msauth.DefaultMSGraphScope}

	httpClient, err := graphHttpClient(t.tenantID, t.clientID, t.clientSecret, scopes)
	if err != nil {
		return nil, err
	}

	graphClient := msgraph.NewClient(httpClient)
	return graphClient, nil
}

func (t *Transport) GraphBetaClient() (*msgraphbeta.GraphServiceRequestBuilder, error) {
	var scopes = []string{msauth.DefaultMSGraphScope}

	httpClient, err := graphHttpClient(t.tenantID, t.clientID, t.clientSecret, scopes)
	if err != nil {
		return nil, err
	}

	graphBetaClient := msgraphbeta.NewClient(httpClient)
	return graphBetaClient, nil
}

func graphHttpClient(tenantID string, clientID string, clientSecret string, scopes []string) (*http.Client, error) {
	ctx := context.Background()
	m := msauth.NewManager()
	ts, err := m.ClientCredentialsGrant(ctx, tenantID, clientID, clientSecret, scopes)
	if err != nil {
		return nil, err
	}

	httpClient := oauth2.NewClient(ctx, ts)
	return httpClient, nil

}

func ParseMicrosoftAuth(r io.Reader) (*MicrosoftAuth, error) {
	var adAccount MicrosoftAuth

	data, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(data, &adAccount)
	if err != nil {
		return nil, err
	}
	return &adAccount, nil
}

type MicrosoftAuth struct {
	TenantId       string `json:"tenantId"`
	SubscriptionId string `json:"subscriptionId"`
	ClientId       string `json:"clientId"`
	ClientSecret   string `json:"clientSecret"`
}
