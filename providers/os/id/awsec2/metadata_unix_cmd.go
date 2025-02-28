// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package awsec2

import (
	"fmt"
	"strings"
)

const (
	baseUnix = `-H "X-aws-ec2-metadata-token: %s" -v http://169.254.169.254/latest/%s`

	// identityUrl = `-H "X-aws-ec2-metadata-token: %s" -v http://169.254.169.254/latest/dynamic/instance-identity/document`
	// tagNameUrl  = `-H "X-aws-ec2-metadata-token: %s" -v http://169.254.169.254/latest/meta-data/tags/instance/Name`
	tokenUrlUnix = `-H "X-aws-ec2-metadata-token-ttl-seconds: 21600" -X PUT "http://169.254.169.254/latest/api/token"`
)

func unixCurlParams(token, path string) string {
	return fmt.Sprintf(baseUnix, token, path)
}

func unixTokenCmdString() string {
	return "curl " + tokenUrlUnix
}

func unixMetadataCmdString(token, metadataPath string) string {
	return fmt.Sprintf("curl %s", unixCurlParams(token, strings.TrimPrefix(metadataPath, "/")))
}
