// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"go.mondoo.com/mql/providers/network/connection"
)

// network.host is the asset this provider connects to (ADR 031). Before it
// existed the connected host was only reachable through resources that quietly
// defaulted their target to it - `dns.records` meaning *this* host's records -
// which left nothing for a query to be rooted at.
func (r *mqlNetworkHost) id() (string, error) {
	return "network.host/" + r.MqlRuntime.Connection.(*connection.HostConnection).FQDN(), nil
}

func (r *mqlNetworkHost) fqdn() (string, error) {
	return r.MqlRuntime.Connection.(*connection.HostConnection).FQDN(), nil
}

// scheme is what the target was given with. The connection stores it as the
// runtime, and leaves it empty when the target carried none - which it then
// reads as HTTPS.
func (r *mqlNetworkHost) scheme() (string, error) {
	conn := r.MqlRuntime.Connection.(*connection.HostConnection)
	if conn.Conf == nil {
		return "", nil
	}
	return conn.Conf.Runtime, nil
}
