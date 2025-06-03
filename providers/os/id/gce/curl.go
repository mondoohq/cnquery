// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package gce

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
)

// curl fetches a single metadata path from the instance metadata service
func (m *CommandInstanceMetadata) curl(metadataPath string) (string, error) {
	var commandString string
	switch {
	case m.platform.IsFamily(inventory.FAMILY_UNIX):
		commandString = unixMetadataCmdString(metadataPath)
	case m.platform.IsFamily(inventory.FAMILY_WINDOWS):
		commandString = windowsMetadataCmdString(metadataPath)
	default:
		return "", errors.New("your platform is not supported by aws metadata identifier resource")
	}

	cmd, err := m.connection.RunCommand(commandString)
	if err != nil {
		return "", err
	}
	data, err := io.ReadAll(cmd.Stdout)
	return strings.TrimSpace(string(data)), err
}

func unixMetadataCmdString(metadataPath string) string {
	return fmt.Sprintf(`curl --noproxy '*' -H "Metadata-Flavor: Google" %s%s`, metadataSvcURL, strings.TrimPrefix(metadataPath, "/"))
}

func windowsMetadataCmdString(metadataPath string) string {
	pipe := ""
	if windowsPathNeedsJSONConvertion(metadataPath) {
		pipe = "| ConvertTo-Json"
	}
	return fmt.Sprintf(`
$Headers = @{
    "Metadata-Flavor" = "Google"
}
Invoke-RestMethod -TimeoutSec 1 -Headers $Headers -URI "%s%s" -UseBasicParsing %s
`, metadataSvcURL, strings.TrimPrefix(metadataPath, "/"), pipe)
}

func windowsPathNeedsJSONConvertion(path string) bool {
	return strings.HasSuffix(path, "/token") ||
		strings.HasSuffix(path, "instance/tags")
}
