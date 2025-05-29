// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package ibmcompute

import (
	"fmt"
	"strings"
)

const (
	baseUnix     = `-H "Authorization: Bearer %s" -v http://169.254.169.254/%s`
	tokenURLUnix = `-H "Metadata-Flavor: ibm" -X PUT "http://169.254.169.254/instance_identity/v1/token?version=2025-05-20" -d '{}'`
)

func unixCurlParams(token, path string) string {
	return fmt.Sprintf(baseUnix, token, path)
}

func unixTokenCmdString() string {
	return "curl " + tokenURLUnix
}

func unixMetadataCmdString(token, metadataPath string) string {
	return fmt.Sprintf("curl %s", unixCurlParams(token, strings.TrimPrefix(metadataPath, "/")))
}
