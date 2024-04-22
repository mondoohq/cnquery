// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package urlx

import (
	"fmt"
	"strings"
)

// ParseGitSshUrl retrieves the
func ParseGitSshUrl(url string) (string, string, string, error) {
	parts := strings.Split(url, "@")
	if len(parts) != 2 {
		return "", "", "", fmt.Errorf("malformed URL")
	}

	// Get the provider
	providerParts := strings.Split(parts[1], ":")
	if len(providerParts) != 2 {
		return "", "", "", fmt.Errorf("malformed URL")
	}
	provider := providerParts[0]

	// Now split the second part at the slash to separate the org and repo
	orgRepoParts := strings.Split(providerParts[1], "/")

	// The repo name is the last part after the split. It includes .git,
	// so we remove that
	repo := strings.TrimSuffix(orgRepoParts[len(orgRepoParts)-1], ".git")

	return provider, orgRepoParts[0], repo, nil
}
