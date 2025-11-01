// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package firewalld

import (
	"slices"
	"strings"
)

func ParseBool(rawValue string) bool {
	validBoolValues := []string{"yes", "true", "on", "1"}
	value := strings.ToLower(strings.TrimSpace(rawValue))
	return slices.Contains(validBoolValues, value)
}
