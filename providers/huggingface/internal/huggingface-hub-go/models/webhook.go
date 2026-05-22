// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package models

import (
	"encoding/json"
)

type Webhook struct {
	ID       string    `json:"id"`
	Watched  []Watched `json:"watched"`
	URL      string    `json:"url"`
	Domains  []string  `json:"domains"`
	Disabled bool      `json:"disabled"`
}

type Watched struct {
	Type string `json:"type"`
	Name string `json:"name"`
}

type WebhookList []Webhook

func (wl *WebhookList) UnmarshalJSON(data []byte) error {
	var webhooks []Webhook
	if err := json.Unmarshal(data, &webhooks); err == nil {
		*wl = webhooks
		return nil
	}

	var obj struct {
		Webhooks []Webhook `json:"webhooks"`
	}
	if err := json.Unmarshal(data, &obj); err != nil {
		return err
	}
	*wl = obj.Webhooks
	return nil
}

type WebhookListOptions struct {
	Search        string
	Filter        string
	SortBy        string
	SortDirection string
	Limit         int
	Full          bool
}

func NewWebhookListOptions() *WebhookListOptions {
	return &WebhookListOptions{
		SortDirection: "1",
		Limit:         20,
		Full:          false,
	}
}
