// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package cloud

type InstanceMetadata struct {
	PublicHostname  string
	PrivateHostname string

	PublicIpv4  []Ipv4Address
	PrivateIpv4 []Ipv4Address

	Metadata any
}
