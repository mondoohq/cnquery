// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

// standard nmap scan
// nmap -sT -T4 192.168.178.0/24
//
// include service and version Detection
// nmap -sT -T4 -sV 192.168.178.0/24
//
// fast discovery
// nmap -sn -n -T4 192.168.178.0/24
func (r *mqlNmap) id() (string, error) {
	return "nmap", nil
}
