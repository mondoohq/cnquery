// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"strconv"
	"strings"

	"github.com/google/go-github/v82/github"
	"go.mondoo.com/cnquery/v12/llx"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/plugin"
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

func (g *mqlGithubRunner) id() (string, error) {
	if g.Id.Error != nil {
		return "", g.Id.Error
	}
	return "github.runner/" + strconv.FormatInt(g.Id.Data, 10), nil
}

func (g *mqlGithubRunnerLabel) id() (string, error) {
	if g.Id.Error != nil {
		return "", g.Id.Error
	}
	return "github.runnerLabel/" + strconv.FormatInt(g.Id.Data, 10), nil
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

// runners returns the self-hosted runners for an organization.
func (g *mqlGithubOrganization) runners() ([]any, error) {
	conn := g.MqlRuntime.Connection.(*connection.GithubConnection)

	if g.Login.Error != nil {
		return nil, g.Login.Error
	}
	orgLogin := g.Login.Data

	opts := &github.ListRunnersOptions{
		ListOptions: github.ListOptions{PerPage: paginationPerPage},
	}

	var allRunners []*github.Runner
	for {
		runners, resp, err := conn.Client().Actions.ListOrganizationRunners(conn.Context(), orgLogin, opts)
		if err != nil {
			if strings.Contains(err.Error(), "404") || strings.Contains(err.Error(), "403") {
				return nil, nil
			}
			return nil, err
		}
		allRunners = append(allRunners, runners.Runners...)
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return runnersToMql(g.MqlRuntime, allRunners)
}

// runners returns the self-hosted runners for a repository.
func (g *mqlGithubRepository) runners() ([]any, error) {
	conn := g.MqlRuntime.Connection.(*connection.GithubConnection)

	if g.Name.Error != nil {
		return nil, g.Name.Error
	}
	repoName := g.Name.Data

	if g.Owner.Error != nil {
		return nil, g.Owner.Error
	}
	owner := g.Owner.Data

	if owner.Login.Error != nil {
		return nil, owner.Login.Error
	}
	ownerLogin := owner.Login.Data

	opts := &github.ListRunnersOptions{
		ListOptions: github.ListOptions{PerPage: paginationPerPage},
	}

	var allRunners []*github.Runner
	for {
		runners, resp, err := conn.Client().Actions.ListRunners(conn.Context(), ownerLogin, repoName, opts)
		if err != nil {
			if strings.Contains(err.Error(), "404") || strings.Contains(err.Error(), "403") {
				return nil, nil
			}
			return nil, err
		}
		allRunners = append(allRunners, runners.Runners...)
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return runnersToMql(g.MqlRuntime, allRunners)
}

// runnersToMql converts a list of GitHub runners to MQL resources.
func runnersToMql(runtime *plugin.Runtime, runners []*github.Runner) ([]any, error) {
	res := []any{}
	for _, runner := range runners {
		labels := []any{}
		for _, label := range runner.Labels {
			labelRes, err := CreateResource(runtime, "github.runnerLabel", map[string]*llx.RawData{
				"id":   llx.IntDataDefault(label.ID, 0),
				"name": llx.StringDataPtr(label.Name),
				"type": llx.StringDataPtr(label.Type),
			})
			if err != nil {
				return nil, err
			}
			labels = append(labels, labelRes)
		}

		r, err := CreateResource(runtime, "github.runner", map[string]*llx.RawData{
			"id":     llx.IntDataDefault(runner.ID, 0),
			"name":   llx.StringDataPtr(runner.Name),
			"os":     llx.StringDataPtr(runner.OS),
			"status": llx.StringDataPtr(runner.Status),
			"busy":   llx.BoolDataPtr(runner.Busy),
			"labels": llx.ArrayData(labels, types.Resource("github.runnerLabel")),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, r)
	}

	return res, nil
}
