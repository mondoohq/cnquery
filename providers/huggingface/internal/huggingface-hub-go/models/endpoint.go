// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package models

type InferenceEndpoint struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Model       string `json:"model"`
	Framework   string `json:"framework"`
	Status      string `json:"status"`
	EndpointURL string `json:"endpoint_url"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

type InferenceEndpointList struct {
	Endpoints []InferenceEndpoint `json:"items"`
}
