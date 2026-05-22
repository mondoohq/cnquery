// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package models

type ModelCard struct {
	ID           string     `json:"id"`
	Private      bool       `json:"private"`
	PipelineTag  string     `json:"pipeline_tag"`
	LibraryName  string     `json:"library_name"`
	Tags         []string   `json:"tags"`
	Downloads    int        `json:"downloads"`
	Likes        int        `json:"likes"`
	ModelID      string     `json:"modelId"`
	Author       string     `json:"author"`
	Sha          string     `json:"sha"`
	LastModified string     `json:"lastModified"`
	Gated        GatedValue `json:"gated"`
	Disabled     bool       `json:"disabled"`
	CardData     struct {
		Language    []string `json:"language"`
		License     string   `json:"license"`
		LibraryName string   `json:"library_name"`
		Tags        []string `json:"tags"`
		Datasets    []string `json:"datasets"`
		BaseModel   []string `json:"base_model"`
	} `json:"cardData"`
}

type DatasetCard struct {
	ID           string     `json:"id"`
	Private      bool       `json:"private"`
	Tags         []string   `json:"tags"`
	Downloads    int        `json:"downloads"`
	Likes        int        `json:"likes"`
	Author       string     `json:"author"`
	Sha          string     `json:"sha"`
	LastModified string     `json:"lastModified"`
	Gated        GatedValue `json:"gated"`
	Disabled     bool       `json:"disabled"`
	Description  string     `json:"description"`
	CardData     struct {
		Language       []string `json:"language"`
		License        string   `json:"license"`
		Tags           []string `json:"tags"`
		PrettyName     string   `json:"pretty_name"`
		SizeCategories []string `json:"size_categories"`
		TaskCategories []string `json:"task_categories"`
	} `json:"cardData"`
}
