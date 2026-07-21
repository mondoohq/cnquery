// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers/vercel/connection"
	"go.mondoo.com/mql/v13/types"
)

type certRecord struct {
	ID        string   `json:"id"`
	Cns       []string `json:"cns"`
	AutoRenew bool     `json:"autoRenew"`
	CreatedAt flexTime `json:"createdAt"`
	ExpiresAt flexTime `json:"expiresAt"`
}

func (c *mqlVercelTeam) certificates() ([]any, error) {
	conn := c.MqlRuntime.Connection.(*connection.VercelConnection)
	records, err := connection.GetPaged[certRecord](context.Background(), conn, "/v7/certs", connection.TeamQuery(c.Id.Data), "certs")
	if err != nil {
		if connection.IsForbidden(err) {
			return []any{}, nil
		}
		return nil, err
	}

	var res []any
	for i := range records {
		rec := records[i]
		cert, err := CreateResource(c.MqlRuntime, "vercel.certificate", map[string]*llx.RawData{
			"id":          llx.StringData(rec.ID),
			"commonNames": llx.ArrayData(strSliceToAny(rec.Cns), types.String),
			"autoRenew":   llx.BoolData(rec.AutoRenew),
			"createdAt":   llx.TimeDataPtr(rec.CreatedAt.Time()),
			"expiresAt":   llx.TimeDataPtr(rec.ExpiresAt.Time()),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, cert)
	}
	return res, nil
}

func (c *mqlVercelCertificate) id() (string, error) {
	return c.Id.Data, c.Id.Error
}
