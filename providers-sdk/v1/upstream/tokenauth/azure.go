// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package tokenauth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const (
	// envClientID and envTokenAudience are variables this package defines. When set,
	// they select the identity to mint a token with, and the audience to mint it for:
	//
	//	MONDOO_WIF_CLIENT_ID  names one user-assigned identity by its client id, for
	//	                      environments that expose more than one. Unset selects
	//	                      the system-assigned identity.
	//	MONDOO_WIF_AUDIENCE   names the token's aud claim. Azure mints tokens only
	//	                      for a resource principal registered in the tenant, so
	//	                      this is an app registration URI (e.g. api://example).
	//	                      Unset, App Service falls back to the exchange audience,
	//	                      which a working exchange sets to the same URI; set this
	//	                      only when the aud claim must differ from that audience.
	//
	// Both apply only to endpoints that can act on them: Cloud Shell has neither
	// user-assigned identities nor an app registration to target, so it reads
	// neither and always mints for the exchange audience.
	envClientID      = "MONDOO_WIF_CLIENT_ID"
	envTokenAudience = "MONDOO_WIF_AUDIENCE"

	// envIdentityEndpoint and envIdentityHeader are injected by Azure rather than
	// defined here. Their presence identifies an App Service, Function or Container
	// App (incl. jobs): the endpoint is the local token URL, and the header is the
	// secret that authenticates the caller to it.
	envIdentityEndpoint = "IDENTITY_ENDPOINT"
	envIdentityHeader   = "IDENTITY_HEADER"

	// appServiceAPIVersion pins the contract of the App Service token endpoint.
	appServiceAPIVersion = "2019-08-01"

	// cloudShellTokenURL is fixed rather than read from MSI_ENDPOINT. Cloud Shell's
	// token endpoint predates IMDS and always listens on this port.
	cloudShellTokenURL = "http://localhost:50342/oauth2/token"

	tokenFetchTimeout = 2 * time.Second
)

// AzureTokenProvider fetches an Azure AD access token from whichever managed
// identity endpoint the current compute environment exposes: App Service, which
// also covers Functions and Container Apps (incl. jobs), and Cloud Shell.
//
// Azure VMs and VM scale sets are deliberately not supported. They mint tokens
// through IMDS, which is unreachable from every environment above.
//
// Both endpoints request a resource — the app registration whose URI becomes the
// token's aud claim. App Service reads it from MONDOO_WIF_AUDIENCE and falls back
// to the exchange audience, which a working Azure exchange already sets to that
// same URI; the override matters only when the two must differ. Cloud Shell always
// mints for the exchange audience directly.
type AzureTokenProvider struct {
	// cloudShellURL overrides the Cloud Shell endpoint. Empty selects
	// cloudShellTokenURL; only tests set it.
	cloudShellURL string
}

func (p *AzureTokenProvider) GetToken(ctx context.Context, audience string) (string, error) {
	cloudShell := p.cloudShellURL
	if cloudShell == "" {
		cloudShell = cloudShellTokenURL
	}

	reqs, err := azureTokenRequests(audience, cloudShell)
	if err != nil {
		return "", err
	}

	client := &http.Client{Timeout: tokenFetchTimeout}
	var errs []error
	for _, req := range reqs {
		token, err := req.RequestToken(ctx, client)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", req, err))
			continue
		}
		return token, nil
	}

	return "", fmt.Errorf("azure managed identity token request failed on all endpoints: %w", errors.Join(errs...))
}

// azureTokenRequest is one managed identity endpoint we know how to ask for a
// token. Each implementation carries only the parameters its own endpoint needs,
// and owns the wire format that endpoint expects — the two differ in method, in
// where the resource travels, and in how the caller authenticates.
//
// String labels the endpoint when GetToken aggregates the failures. It must not
// leak the App Service secret, which travels in X-IDENTITY-HEADER.
type azureTokenRequest interface {
	fmt.Stringer
	RequestToken(ctx context.Context, client *http.Client) (string, error)
}

// azureTokenRequests builds the ordered list of endpoints to try. An endpoint is
// registered only once the environment supplies everything that endpoint needs.
func azureTokenRequests(audience, cloudShellURL string) ([]azureTokenRequest, error) {
	reqs := make([]azureTokenRequest, 0, 2)

	// App Service announces itself by injecting the endpoint. Neither of the two
	// values the request then needs is guessable, and sending an empty one earns a
	// bodiless 4xx that names nothing, so refuse before the request goes out.
	if endpoint := os.Getenv(envIdentityEndpoint); endpoint != "" {
		// Azure injects the header alongside the endpoint, and the endpoint rejects
		// any request that arrives without it — the header is what defends the local
		// token service against SSRF. Its absence means this is not the environment
		// the endpoint implies, so check it before anything we control.
		header := os.Getenv(envIdentityHeader)
		if header == "" {
			return nil, fmt.Errorf(
				"%s names an azure managed identity endpoint, but %s is unset: the endpoint rejects requests that do not carry it",
				envIdentityEndpoint, envIdentityHeader)
		}

		// The resource is the app registration to mint for. MONDOO_WIF_AUDIENCE
		// overrides it; otherwise it is the exchange audience, which a correctly
		// configured Azure exchange already sets to the same app registration URI.
		// Only when both are empty is there nothing to request — send that and Azure
		// answers with an opaque 400, so refuse before the request goes out.
		resource := os.Getenv(envTokenAudience)
		if resource == "" {
			resource = audience
		}
		if resource == "" {
			return nil, fmt.Errorf(
				"%s names an azure managed identity endpoint, but no resource to mint for: set %s to the app registration URI, or pass an audience",
				envIdentityEndpoint, envTokenAudience)
		}

		reqs = append(reqs, appServiceTokenRequest{
			endpoint: endpoint,
			header:   header,
			resource: resource,
			clientID: os.Getenv(envClientID),
		})
	}

	// Cloud Shell predates the app registration flow and mints for the exchange
	// audience directly, so it needs nothing from the environment. It is always
	// tried, with whatever audience the caller passed — this is the exact request
	// this provider sent before App Service support existed, and an empty audience
	// is preserved rather than guarded so that behavior stays byte-for-byte intact.
	reqs = append(reqs, cloudShellTokenRequest{
		endpoint: cloudShellURL,
		resource: audience,
	})

	return reqs, nil
}

// appServiceTokenRequest covers App Service, Functions and Container Apps. It
// takes the resource in the query string and authenticates with the secret App
// Service injects alongside the endpoint.
type appServiceTokenRequest struct {
	endpoint string
	// header is the secret Azure injects alongside the endpoint. It is required:
	// the endpoint rejects any request that does not carry it.
	header   string
	resource string
	// clientID picks one of several user-assigned identities. Empty falls back to
	// the system-assigned identity.
	clientID string
}

func (r appServiceTokenRequest) String() string {
	identity := "system-assigned identity"
	if r.clientID != "" {
		identity = "client_id=" + r.clientID
	}
	return fmt.Sprintf("app service (%s, resource=%s, %s)", r.endpoint, r.resource, identity)
}

func (r appServiceTokenRequest) RequestToken(ctx context.Context, client *http.Client) (string, error) {
	q := url.Values{}
	q.Set("api-version", appServiceAPIVersion)
	q.Set("resource", r.resource)
	if r.clientID != "" {
		q.Set("client_id", r.clientID)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, r.endpoint+"?"+q.Encode(), nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("X-IDENTITY-HEADER", r.header)

	return requestAzureToken(client, req)
}

// cloudShellTokenRequest carries no clientID: Cloud Shell has no user-assigned
// identities. Its endpoint predates IMDS and wants a form-encoded POST, and its
// resource is the exchange audience the caller passed.
type cloudShellTokenRequest struct {
	endpoint string
	resource string
}

func (r cloudShellTokenRequest) String() string {
	return fmt.Sprintf("cloud shell (%s, resource=%s)", r.endpoint, r.resource)
}

func (r cloudShellTokenRequest) RequestToken(ctx context.Context, client *http.Client) (string, error) {
	form := url.Values{}
	form.Set("resource", r.resource)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, r.endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Metadata", "true")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	return requestAzureToken(client, req)
}

// requestAzureToken sends a built request and extracts the access token. Both
// endpoints answer with the same JSON envelope, so they share the response half.
func requestAzureToken(client *http.Client, req *http.Request) (string, error) {
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// The body carries Azure's actual complaint (invalid resource, no such
		// identity, ...). Without it a 400 is unactionable. It never contains a
		// token: an error response has no access_token.
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		detail := strings.TrimSpace(string(body))
		if detail == "" {
			detail = "<empty body>"
		}
		return "", fmt.Errorf("endpoint returned non-OK status %d: %s", resp.StatusCode, detail)
	}

	var result struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if result.AccessToken == "" {
		return "", fmt.Errorf("endpoint returned an empty access_token")
	}
	return result.AccessToken, nil
}
