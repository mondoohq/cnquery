// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package connection

type SharepointOnlineReport struct {
	SPOTenant                      interface{} `json:"SPOTenant"`
	SPOTenantSyncClientRestriction interface{} `json:"SPOTenantSyncClientRestriction"`
}

type Microsoft365Report struct {
	ExchangeOnline   ExchangeOnlineReport   `json:"ExchangeOnline"`
	SharepointOnline SharepointOnlineReport `json:"SharepointOnline"`
	MsTeams          MsTeamsReport          `json:"MsTeams"`
}
