// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"strings"

	"github.com/google/go-github/v82/github"
	"go.mondoo.com/cnquery/v12/llx"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v12/providers/github/connection"
	"go.mondoo.com/cnquery/v12/types"
)

func (g *mqlGithubAuditLogEntry) id() (string, error) {
	if g.DocumentId.Error != nil {
		return "", g.DocumentId.Error
	}
	return "github.auditLogEntry/" + g.DocumentId.Data, nil
}

// auditLog returns the audit log entries for an organization.
// Note: This requires GitHub Enterprise Cloud.
func (g *mqlGithubOrganization) auditLog() ([]any, error) {
	conn := g.MqlRuntime.Connection.(*connection.GithubConnection)

	if g.Login.Error != nil {
		return nil, g.Login.Error
	}
	orgLogin := g.Login.Data

	opts := &github.GetAuditLogOptions{
		ListCursorOptions: github.ListCursorOptions{PerPage: paginationPerPage},
	}

	var allEntries []*github.AuditEntry
	for {
		entries, resp, err := conn.Client().Organizations.GetAuditLog(conn.Context(), orgLogin, opts)
		if err != nil {
			// Audit log is only available for GitHub Enterprise Cloud
			if strings.Contains(err.Error(), "404") || strings.Contains(err.Error(), "403") {
				return nil, nil
			}
			return nil, err
		}
		allEntries = append(allEntries, entries...)
		if resp.After == "" {
			break
		}
		opts.ListCursorOptions.After = resp.After
	}

	res := []any{}
	for _, entry := range allEntries {
		var actorLocation string
		if entry.ActorLocation != nil && entry.ActorLocation.CountryCode != nil {
			actorLocation = *entry.ActorLocation.CountryCode
		}

		data, _ := convert.JsonToDict(entry.Data)
		additionalFields, _ := convert.JsonToDict(entry.AdditionalFields)

		r, err := CreateResource(g.MqlRuntime, "github.auditLogEntry", map[string]*llx.RawData{
			"documentId":               llx.StringDataPtr(entry.DocumentID),
			"action":                   llx.StringDataPtr(entry.Action),
			"actor":                    llx.StringDataPtr(entry.Actor),
			"actorId":                  llx.IntDataDefault(entry.ActorID, 0),
			"actorLocation":            llx.StringData(actorLocation),
			"business":                 llx.StringDataPtr(entry.Business),
			"businessId":               llx.IntDataDefault(entry.BusinessID, 0),
			"org":                      llx.StringDataPtr(entry.Org),
			"orgId":                    llx.IntDataDefault(entry.OrgID, 0),
			"user":                     llx.StringDataPtr(entry.User),
			"userId":                   llx.IntDataDefault(entry.UserID, 0),
			"createdAt":                llx.TimeDataPtr(githubTimestamp(entry.CreatedAt)),
			"timestamp":                llx.TimeDataPtr(githubTimestamp(entry.Timestamp)),
			"externalIdentityNameId":   llx.StringDataPtr(entry.ExternalIdentityNameID),
			"externalIdentityUsername": llx.StringDataPtr(entry.ExternalIdentityUsername),
			"hashedToken":              llx.StringDataPtr(entry.HashedToken),
			"tokenId":                  llx.IntDataDefault(entry.TokenID, 0),
			"tokenScopes":              llx.StringDataPtr(entry.TokenScopes),
			"data":                     llx.MapData(data, types.Any),
			"additionalFields":         llx.MapData(additionalFields, types.Any),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, r)
	}

	return res, nil
}
