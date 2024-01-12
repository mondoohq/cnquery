// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package updates

import (
	"fmt"
	"io"
	"strings"

	"go.mondoo.com/cnquery/v10/providers/os/connection/shared"
	"howett.net/plist"
)

const (
	MacosUpdateFormat = "macos"
)

type MacosUpdateManager struct {
	conn shared.Connection
}

func (um *MacosUpdateManager) Name() string {
	return "macOS Update Manager"
}

func (um *MacosUpdateManager) Format() string {
	return MacosUpdateFormat
}

func (um *MacosUpdateManager) List() ([]OperatingSystemUpdate, error) {
	f, err := um.conn.FileSystem().Open("/Library/Preferences/com.apple.SoftwareUpdate.plist")
	defer f.Close()
	if err != nil {
		return nil, fmt.Errorf("could not read package list")
	}
	return ParseSoftwarePlistUpdates(f)
}

// parse macos system version property list
func ParseSoftwarePlistUpdates(input io.Reader) ([]OperatingSystemUpdate, error) {
	var r io.ReadSeeker
	r, ok := input.(io.ReadSeeker)

	// if the read seaker is not implemented lets cache stdout in-memory
	if !ok {
		packageList, err := io.ReadAll(input)
		if err != nil {
			return nil, err
		}
		r = strings.NewReader(string(packageList))
	}

	type recommendedUpdate struct {
		Identifier           string `plist:"Identifier"`
		DisplayName          string `plist:"Display Name"`
		Version              string `plist:"Display Version"`
		MobileSoftwareUpdate bool   `plist:"MobileSoftwareUpdate"`
		ProductKey           string `plist:"Product Key"`
	}

	type softwareUpdate struct {
		RecommendedUpdates []recommendedUpdate `plist:"RecommendedUpdates"`
	}

	var data softwareUpdate
	decoder := plist.NewDecoder(r)
	err := decoder.Decode(&data)
	if err != nil {
		return nil, err
	}

	updates := make([]OperatingSystemUpdate, len(data.RecommendedUpdates))
	for i, entry := range data.RecommendedUpdates {
		updates[i].ID = entry.ProductKey
		updates[i].Name = entry.Identifier
		updates[i].Description = entry.DisplayName
		updates[i].Version = entry.Version

		updates[i].Format = MacosUpdateFormat
	}

	return updates, nil
}
