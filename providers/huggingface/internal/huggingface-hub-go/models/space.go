// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package models

import (
	"encoding/json"
)

type Space struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
	Author      string   `json:"author"`
	Likes       int      `json:"likes"`
	Private     bool     `json:"private"`
}

type SpaceListOptions struct {
	Author        string
	Search        string
	Filter        string
	SortBy        string
	SortDirection string
	Limit         int
	Full          bool
}

func NewSpaceListOptions() *SpaceListOptions {
	return &SpaceListOptions{
		SortDirection: "1",
		Limit:         20,
		Full:          false,
	}
}

type SpaceList struct {
	Spaces []Space
}

func (sl *SpaceList) UnmarshalJSON(data []byte) error {
	var spaces []Space
	if err := json.Unmarshal(data, &spaces); err == nil {
		sl.Spaces = spaces
		return nil
	}

	type Alias SpaceList
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(sl),
	}
	return json.Unmarshal(data, aux)
}
