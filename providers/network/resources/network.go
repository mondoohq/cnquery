// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

// This is a small selection of common ports that are supported.
// Outside of this range, users will have to specify ports explicitly.
// We could expand this to cover more of IANA.
var CommonPorts = map[string]int{
	"https":  443,
	"http":   80,
	"ssh":    22,
	"ftp":    21,
	"telnet": 23,
	"smtp":   25,
	"dns":    53,
	"pop3":   110,
	"imap4":  143,
}
