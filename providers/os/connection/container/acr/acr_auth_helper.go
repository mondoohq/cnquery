// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package acr

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/util/azauth"
)

const (
	// nullGuid is used to let docker perform the login for us
	nullGuid = "00000000-0000-0000-0000-000000000000"
	msScope  = "https://management.core.windows.net//.default"
)

// implemented according to https://github.com/Azure/acr/blob/main/docs/AAD-OAuth.md
type acrAuthHelper struct {
	httpClient *http.Client
	aadToken   azcore.TokenCredential
	cache      map[string]string
}

func newAcrAuthHelperFromToken(token azcore.TokenCredential) *acrAuthHelper {
	return &acrAuthHelper{
		httpClient: http.DefaultClient,
		aadToken:   token,
		cache:      make(map[string]string),
	}
}

func NewAcrAuthHelper() (*acrAuthHelper, error) {
	token, err := azauth.GetChainedToken(nil)
	if err != nil {
		return nil, err
	}
	return newAcrAuthHelperFromToken(token), nil
}

func (a *acrAuthHelper) getRefreshUrl(serverUrl string) string {
	return fmt.Sprintf("https://%s/oauth2/exchange", serverUrl)
}

// https://github.com/Azure/acr/blob/main/docs/AAD-OAuth.md#calling-post-oauth2exchange-to-get-an-acr-refresh-token
func (a *acrAuthHelper) getRefreshToken(ctx context.Context, serverUrl string) (string, error) {
	if refreshToken, ok := a.cache[serverUrl]; ok {
		log.Debug().Str("server", serverUrl).Msg("using cached acr refresh token")
		return refreshToken, nil
	}
	rawAadToken, err := a.aadToken.GetToken(ctx, policy.TokenRequestOptions{
		Scopes: []string{msScope},
	})
	if err != nil {
		return "", err
	}
	regUrl := a.getRefreshUrl(serverUrl)
	form := url.Values{
		"grant_type":   {"access_token"},
		"service":      {serverUrl},
		"access_token": {rawAadToken.Token},
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, regUrl, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	httpResp, err := a.httpClient.Do(httpReq)
	if err != nil {
		return "", err
	}
	resp, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return "", err
	}
	if httpResp.StatusCode != http.StatusOK {
		return "", errors.New(string(resp))
	}

	var tokenResp refreshTokenResp
	if err := json.Unmarshal(resp, &tokenResp); err != nil {
		return "", err
	}

	a.cache[serverUrl] = tokenResp.RefreshToken
	return tokenResp.RefreshToken, nil
}

type refreshTokenResp struct {
	RefreshToken string `json:"refresh_token"`
}

func (a *acrAuthHelper) Get(serverURL string) (string, string, error) {
	log.Debug().Str("server", serverURL).Msg("using acr auth helper")
	ctx := context.Background()
	refreshToken, err := a.getRefreshToken(ctx, serverURL)
	if err != nil {
		log.Error().Err(err).Msg("failed to fetch acr refresh token")
		return "", "", err
	}
	return nullGuid, refreshToken, nil
}
