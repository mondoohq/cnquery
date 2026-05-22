// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package models

import (
	"encoding/json"
)

type Dataset struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
	Author      string   `json:"author"`
	Downloads   int      `json:"downloads"`
	Likes       int      `json:"likes"`
	Private     bool     `json:"private"`
}

type DatasetListOptions struct {
	Author        string
	Search        string
	Filter        string
	SortBy        string
	SortDirection string
	Limit         int
	Full          bool
}

func NewDatasetListOptions() *DatasetListOptions {
	return &DatasetListOptions{
		SortDirection: "1",
		Limit:         20,
		Full:          false,
	}
}

type DatasetList struct {
	Datasets []Dataset
}

func (dl *DatasetList) UnmarshalJSON(data []byte) error {
	var datasets []Dataset
	if err := json.Unmarshal(data, &datasets); err == nil {
		dl.Datasets = datasets
		return nil
	}

	type Alias DatasetList
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(dl),
	}
	return json.Unmarshal(data, aux)
}
