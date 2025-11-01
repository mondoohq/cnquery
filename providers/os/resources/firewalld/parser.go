// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package firewalld

import (
	"encoding/xml"
	"slices"
	"strings"
)

func ParseBool(rawValue string) bool {
	validBoolValues := []string{"yes", "true", "on", "1"}
	value := strings.ToLower(strings.TrimSpace(rawValue))
	return slices.Contains(validBoolValues, value)
}

func ParseZone(content []byte) (*Zone, error) {
	var zoneXML Zone
	if err := xml.Unmarshal(content, &zoneXML); err != nil {
		return nil, err
	}
	return &zoneXML, nil
}
