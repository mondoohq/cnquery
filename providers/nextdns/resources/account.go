// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"go.mondoo.com/mql/v13/providers/nextdns/connection"
)

func (r *mqlNextdnsAccount) id() (string, error) {
	return "nextdns.account/" + r.Id.Data, nil
}

func (r *mqlNextdnsAccount) profiles() ([]any, error) {
	conn := r.MqlRuntime.Connection.(*connection.NextdnsConnection)
	return profilesToResources(r.MqlRuntime, conn)
}
