// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package parsers

import (
	"io"
	"strings"

	"github.com/coreos/go-systemd/unit"
)

// UnitSection represents a section in a systemd unit file
type UnitSection struct {
	Name   string
	Params []UnitParam
}

// UnitParam represents a key-value pair in a systemd unit file
type UnitParam struct {
	Name  string
	Value string
}

// Unit contains the parsed contents of a systemd unit file
type Unit struct {
	Sections []UnitSection
}

// ParseUnit parses the raw text contents of a systemd unit file
// Returns sections and params as arrays to support duplicate keys
func ParseUnit(raw string) (*Unit, error) {
	opts, err := unit.Deserialize(strings.NewReader(raw))
	if err != nil {
		return nil, err
	}

	// Group options by section, preserving order and duplicates
	sectionMap := make(map[string][]UnitParam)
	sectionOrder := []string{}

	for _, opt := range opts {
		sectionName := opt.Section
		if sectionName == "" {
			sectionName = "" // empty string for default section
		}

		// Track section order (first occurrence)
		if _, exists := sectionMap[sectionName]; !exists {
			sectionOrder = append(sectionOrder, sectionName)
		}

		// Append param (allows duplicates)
		sectionMap[sectionName] = append(sectionMap[sectionName], UnitParam{
			Name:  opt.Name,
			Value: opt.Value,
		})
	}

	// Build sections array in order
	sections := make([]UnitSection, 0, len(sectionOrder))
	for _, sectionName := range sectionOrder {
		sections = append(sections, UnitSection{
			Name:   sectionName,
			Params: sectionMap[sectionName],
		})
	}

	return &Unit{Sections: sections}, nil
}

// ParseUnitFromReader parses a systemd unit file from an io.Reader
func ParseUnitFromReader(reader io.Reader) (*Unit, error) {
	opts, err := unit.Deserialize(reader)
	if err != nil {
		return nil, err
	}

	// Group options by section, preserving order and duplicates
	sectionMap := make(map[string][]UnitParam)
	sectionOrder := []string{}

	for _, opt := range opts {
		sectionName := opt.Section
		if sectionName == "" {
			sectionName = "" // empty string for default section
		}

		// Track section order (first occurrence)
		if _, exists := sectionMap[sectionName]; !exists {
			sectionOrder = append(sectionOrder, sectionName)
		}

		// Append param (allows duplicates)
		sectionMap[sectionName] = append(sectionMap[sectionName], UnitParam{
			Name:  opt.Name,
			Value: opt.Value,
		})
	}

	// Build sections array in order
	sections := make([]UnitSection, 0, len(sectionOrder))
	for _, sectionName := range sectionOrder {
		sections = append(sections, UnitSection{
			Name:   sectionName,
			Params: sectionMap[sectionName],
		})
	}

	return &Unit{Sections: sections}, nil
}
