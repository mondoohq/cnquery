// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package registry

import (
	"encoding/json"
	"io"
)

func ParsePowershellRegistryKeyItems(r io.Reader) ([]RegistryKeyItem, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	var items []RegistryKeyItem
	err = json.Unmarshal(data, &items)
	if err != nil {
		return nil, err
	}

	return items, nil
}

func ParsePowershellRegistryKeyChildren(r io.Reader) ([]RegistryKeyChild, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	var children []RegistryKeyChild
	err = json.Unmarshal(data, &children)
	if err != nil {
		return nil, err
	}

	return children, nil
}
