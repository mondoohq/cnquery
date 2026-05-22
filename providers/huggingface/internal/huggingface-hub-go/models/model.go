// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package models

import (
	"encoding/json"
	"fmt"
)

type GatedValue struct {
	IsGated bool
	Mode    string
}

func (g *GatedValue) UnmarshalJSON(data []byte) error {
	var b bool
	if err := json.Unmarshal(data, &b); err == nil {
		g.IsGated = b
		if b {
			g.Mode = "true"
		} else {
			g.Mode = "false"
		}
		return nil
	}

	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		g.IsGated = s != "" && s != "false"
		g.Mode = s
		return nil
	}

	return fmt.Errorf("gated: unexpected type %s", string(data))
}

type ModelConfig struct {
	Architectures   []string       `json:"architectures"`
	ModelType       string         `json:"model_type"`
	TokenizerConfig map[string]any `json:"tokenizer_config"`
}

type ModelIndexResult struct {
	Task struct {
		Name string `json:"name"`
		Type string `json:"type"`
	} `json:"task"`
	Dataset struct {
		Name   string `json:"name"`
		Type   string `json:"type"`
		Config string `json:"config"`
		Split  string `json:"split"`
	} `json:"dataset"`
	Metrics []struct {
		Name     string  `json:"name"`
		Type     string  `json:"type"`
		Value    float64 `json:"value"`
		Verified bool    `json:"verified"`
	} `json:"metrics"`
}

type ModelIndex struct {
	Name    string             `json:"name"`
	Results []ModelIndexResult `json:"results"`
}

type CardData struct {
	Language    []string `json:"language"`
	Tags        []string `json:"tags"`
	PipelineTag string   `json:"pipeline_tag"`
	License     string   `json:"license"`
}

type TransformersInfo struct {
	AutoModel   string `json:"auto_model"`
	PipelineTag string `json:"pipeline_tag"`
	Processor   string `json:"processor"`
}

type Sibling struct {
	Rfilename string `json:"rfilename"`
}

type Model struct {
	ID               string           `json:"id"`
	ModelID          string           `json:"modelId"`
	Private          bool             `json:"private"`
	PipelineTag      string           `json:"pipeline_tag"`
	LibraryName      string           `json:"library_name"`
	Tags             []string         `json:"tags"`
	Downloads        int              `json:"downloads"`
	Likes            int              `json:"likes"`
	Author           string           `json:"author"`
	Sha              string           `json:"sha"`
	LastModified     string           `json:"lastModified"`
	Gated            GatedValue       `json:"gated"`
	Disabled         bool             `json:"disabled"`
	Config           ModelConfig      `json:"config"`
	CardData         CardData         `json:"cardData"`
	TransformersInfo TransformersInfo `json:"transformersInfo"`
	Siblings         []Sibling        `json:"siblings"`
}

// ModelDetail is the full model response from GET /api/models/{id}.
// CardData, Config, and Siblings intentionally shadow the embedded Model fields
// with different types (map[string]any vs typed structs) because the detail API
// returns richer untyped JSON that we need to pass through as-is.
type ModelDetail struct {
	Model
	CreatedAt string         `json:"createdAt"`
	CardData  map[string]any `json:"cardData"`
	Config    map[string]any `json:"config"`
	Siblings  []Sibling      `json:"siblings"`
}

type ModelList struct {
	Models []Model
}

func (ml *ModelList) UnmarshalJSON(data []byte) error {
	var models []Model
	if err := json.Unmarshal(data, &models); err == nil {
		ml.Models = models
		return nil
	}

	type Alias ModelList
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(ml),
	}
	return json.Unmarshal(data, aux)
}
