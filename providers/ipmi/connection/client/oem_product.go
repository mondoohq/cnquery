// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package client

type OemProductID uint16

const (
	OemProductUnknown = OemProductID(0)
)

var oemProductStrings = map[OemProductID]string{
	OemProductUnknown: "Unknown",
}

func (id OemProductID) String() string {
	if s, ok := oemProductStrings[id]; ok {
		return s
	}
	return "Unknown"
}
