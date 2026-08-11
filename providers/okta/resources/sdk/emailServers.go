// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package sdk

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/okta/okta-sdk-golang/v5/okta"
)

// ListEmailServers fetches the org's custom SMTP servers from
// `/api/v1/email-servers`.
//
// The v5 SDK's generated EmailServerAPI.ListEmailServers types the response as
// an EmailServerListResponse object, but the endpoint answers with a bare array
// of servers, so the generated call fails to unmarshal and every org that has
// configured an SMTP server gets an error instead of its servers. Both shapes
// are accepted here: the array the API sends today, and the wrapped object the
// SDK expects, so the resource keeps working whichever one an org is served.
func (m *ApiExtension) ListEmailServers(ctx context.Context) ([]okta.EmailServerResponse, *http.Response, error) {
	var raw json.RawMessage
	resp, err := m.get(ctx, m.url("/api/v1/email-servers"), &raw)
	if err != nil {
		return nil, resp, err
	}

	servers, err := decodeOktaEmailServers(raw)
	if err != nil {
		return nil, resp, err
	}
	return servers, resp, nil
}

// decodeOktaEmailServers accepts either the bare array the API returns or the
// `{"email-servers": [...]}` envelope the SDK's model describes.
func decodeOktaEmailServers(raw json.RawMessage) ([]okta.EmailServerResponse, error) {
	if len(raw) == 0 {
		return nil, nil
	}

	var list []okta.EmailServerResponse
	if err := json.Unmarshal(raw, &list); err == nil {
		return list, nil
	}

	var wrapped struct {
		EmailServers []okta.EmailServerResponse `json:"email-servers"`
	}
	if err := json.Unmarshal(raw, &wrapped); err != nil {
		return nil, err
	}
	return wrapped.EmailServers, nil
}
