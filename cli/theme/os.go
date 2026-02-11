// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package theme

var DefaultTheme = OperatingSystemTheme

func init() {
	DefaultTheme.PolicyPrinter.Error = DefaultTheme.Error
	DefaultTheme.PolicyPrinter.Primary = DefaultTheme.Primary
	DefaultTheme.PolicyPrinter.Secondary = DefaultTheme.Secondary
}

// logo for the shell
const Logo = ` _ __ ___   __ _| |
| '_ ` + "`" + ` _ \ / _` + "`" + ` | |
| | | | | | (_| | |
|_| |_| |_|\__, |_|
  mondoo™     |_|`
