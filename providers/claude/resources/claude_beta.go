// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"
	"go.mondoo.com/mql/v13/llx"
)

// claude.agent

func (r *mqlClaude) agents() ([]interface{}, error) {
	c := conn(r.MqlRuntime)
	client := c.Client()

	pager := client.Beta.Agents.ListAutoPaging(context.Background(), anthropic.BetaAgentListParams{})

	var res []interface{}
	for pager.Next() {
		a := pager.Current()

		mqlAgent, err := CreateResource(r.MqlRuntime, "claude.agent", map[string]*llx.RawData{
			"__id":        llx.StringData(a.ID),
			"id":          llx.StringData(a.ID),
			"name":        llx.StringData(a.Name),
			"description": llx.StringData(a.Description),
			"system":      llx.StringData(a.System),
			"model":       llx.StringData(string(a.Model.ID)),
			"version":     llx.IntData(a.Version),
			"createdAt":   llx.TimeData(a.CreatedAt),
			"updatedAt":   llx.TimeData(a.UpdatedAt),
			"archivedAt":  llx.TimeData(a.ArchivedAt),
			"type":        llx.StringData(string(a.Type)),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlAgent)
	}
	if err := pager.Err(); err != nil {
		return nil, fmt.Errorf("listing agents: %w", err)
	}

	return res, nil
}

// claude.environment

func (r *mqlClaude) environments() ([]interface{}, error) {
	c := conn(r.MqlRuntime)
	client := c.Client()

	pager := client.Beta.Environments.ListAutoPaging(context.Background(), anthropic.BetaEnvironmentListParams{})

	var res []interface{}
	for pager.Next() {
		e := pager.Current()

		createdAt, err := parseTime(e.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("parsing environment createdAt: %w", err)
		}
		updatedAt, err := parseTime(e.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("parsing environment updatedAt: %w", err)
		}
		archivedAt, err := parseTime(e.ArchivedAt)
		if err != nil {
			return nil, fmt.Errorf("parsing environment archivedAt: %w", err)
		}

		mqlEnv, err := CreateResource(r.MqlRuntime, "claude.environment", map[string]*llx.RawData{
			"__id":        llx.StringData(e.ID),
			"id":          llx.StringData(e.ID),
			"name":        llx.StringData(e.Name),
			"description": llx.StringData(e.Description),
			"scope":       llx.StringData(string(e.Scope)),
			"createdAt":   llx.TimeData(createdAt),
			"updatedAt":   llx.TimeData(updatedAt),
			"archivedAt":  llx.TimeData(archivedAt),
			"type":        llx.StringData(string(e.Type)),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlEnv)
	}
	if err := pager.Err(); err != nil {
		return nil, fmt.Errorf("listing environments: %w", err)
	}

	return res, nil
}

// claude.session

func (r *mqlClaude) sessions() ([]interface{}, error) {
	c := conn(r.MqlRuntime)
	client := c.Client()

	pager := client.Beta.Sessions.ListAutoPaging(context.Background(), anthropic.BetaSessionListParams{})

	var res []interface{}
	for pager.Next() {
		s := pager.Current()

		mqlSession, err := CreateResource(r.MqlRuntime, "claude.session", map[string]*llx.RawData{
			"__id":          llx.StringData(s.ID),
			"id":            llx.StringData(s.ID),
			"title":         llx.StringData(s.Title),
			"status":        llx.StringData(string(s.Status)),
			"environmentId": llx.StringData(s.EnvironmentID),
			"createdAt":     llx.TimeData(s.CreatedAt),
			"updatedAt":     llx.TimeData(s.UpdatedAt),
			"archivedAt":    llx.TimeData(s.ArchivedAt),
			"type":          llx.StringData(string(s.Type)),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlSession)
	}
	if err := pager.Err(); err != nil {
		return nil, fmt.Errorf("listing sessions: %w", err)
	}

	return res, nil
}

// claude.file

func (r *mqlClaude) files() ([]interface{}, error) {
	c := conn(r.MqlRuntime)
	client := c.Client()

	pager := client.Beta.Files.ListAutoPaging(context.Background(), anthropic.BetaFileListParams{})

	var res []interface{}
	for pager.Next() {
		f := pager.Current()

		mqlFile, err := CreateResource(r.MqlRuntime, "claude.file", map[string]*llx.RawData{
			"__id":         llx.StringData(f.ID),
			"id":           llx.StringData(f.ID),
			"filename":     llx.StringData(f.Filename),
			"mimeType":     llx.StringData(f.MimeType),
			"sizeBytes":    llx.IntData(f.SizeBytes),
			"downloadable": llx.BoolData(f.Downloadable),
			"createdAt":    llx.TimeData(f.CreatedAt),
			"type":         llx.StringData(string(f.Type)),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlFile)
	}
	if err := pager.Err(); err != nil {
		return nil, fmt.Errorf("listing files: %w", err)
	}

	return res, nil
}

// claude.skill

func (r *mqlClaude) skills() ([]interface{}, error) {
	c := conn(r.MqlRuntime)
	client := c.Client()

	pager := client.Beta.Skills.ListAutoPaging(context.Background(), anthropic.BetaSkillListParams{})

	var res []interface{}
	for pager.Next() {
		s := pager.Current()

		createdAt, err := parseTime(s.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("parsing skill createdAt: %w", err)
		}
		updatedAt, err := parseTime(s.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("parsing skill updatedAt: %w", err)
		}

		mqlSkill, err := CreateResource(r.MqlRuntime, "claude.skill", map[string]*llx.RawData{
			"__id":          llx.StringData(s.ID),
			"id":            llx.StringData(s.ID),
			"displayTitle":  llx.StringData(s.DisplayTitle),
			"source":        llx.StringData(s.Source),
			"latestVersion": llx.StringData(s.LatestVersion),
			"createdAt":     llx.TimeData(createdAt),
			"updatedAt":     llx.TimeData(updatedAt),
			"type":          llx.StringData(s.Type),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlSkill)
	}
	if err := pager.Err(); err != nil {
		return nil, fmt.Errorf("listing skills: %w", err)
	}

	return res, nil
}

// claude.vault

func (r *mqlClaude) vaults() ([]interface{}, error) {
	c := conn(r.MqlRuntime)
	client := c.Client()

	pager := client.Beta.Vaults.ListAutoPaging(context.Background(), anthropic.BetaVaultListParams{})

	var res []interface{}
	for pager.Next() {
		v := pager.Current()

		mqlVault, err := CreateResource(r.MqlRuntime, "claude.vault", map[string]*llx.RawData{
			"__id":        llx.StringData(v.ID),
			"id":          llx.StringData(v.ID),
			"displayName": llx.StringData(v.DisplayName),
			"createdAt":   llx.TimeData(v.CreatedAt),
			"updatedAt":   llx.TimeData(v.UpdatedAt),
			"archivedAt":  llx.TimeData(v.ArchivedAt),
			"type":        llx.StringData(string(v.Type)),
		})
		if err != nil {
			return nil, err
		}

		res = append(res, mqlVault)
	}
	if err := pager.Err(); err != nil {
		return nil, fmt.Errorf("listing vaults: %w", err)
	}

	return res, nil
}

func (r *mqlClaudeVault) credentials() ([]interface{}, error) {
	c := conn(r.MqlRuntime)
	client := c.Client()

	pager := client.Beta.Vaults.Credentials.ListAutoPaging(context.Background(), r.GetId().Data, anthropic.BetaVaultCredentialListParams{})

	var res []interface{}
	for pager.Next() {
		cred := pager.Current()

		mqlCred, err := CreateResource(r.MqlRuntime, "claude.vault.credential", map[string]*llx.RawData{
			"__id":        llx.StringData(cred.ID),
			"id":          llx.StringData(cred.ID),
			"displayName": llx.StringData(cred.DisplayName),
			"vaultId":     llx.StringData(cred.VaultID),
			"createdAt":   llx.TimeData(cred.CreatedAt),
			"updatedAt":   llx.TimeData(cred.UpdatedAt),
			"archivedAt":  llx.TimeData(cred.ArchivedAt),
			"type":        llx.StringData(string(cred.Type)),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlCred)
	}
	if err := pager.Err(); err != nil {
		return nil, fmt.Errorf("listing vault credentials: %w", err)
	}

	return res, nil
}

// claude.memoryStore

func (r *mqlClaude) memoryStores() ([]interface{}, error) {
	c := conn(r.MqlRuntime)
	client := c.Client()

	pager := client.Beta.MemoryStores.ListAutoPaging(context.Background(), anthropic.BetaMemoryStoreListParams{})

	var res []interface{}
	for pager.Next() {
		ms := pager.Current()

		mqlMS, err := CreateResource(r.MqlRuntime, "claude.memoryStore", map[string]*llx.RawData{
			"__id":        llx.StringData(ms.ID),
			"id":          llx.StringData(ms.ID),
			"name":        llx.StringData(ms.Name),
			"description": llx.StringData(ms.Description),
			"createdAt":   llx.TimeData(ms.CreatedAt),
			"updatedAt":   llx.TimeData(ms.UpdatedAt),
			"archivedAt":  llx.TimeData(ms.ArchivedAt),
			"type":        llx.StringData(string(ms.Type)),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlMS)
	}
	if err := pager.Err(); err != nil {
		return nil, fmt.Errorf("listing memory stores: %w", err)
	}

	return res, nil
}

// claude.messageBatch

func (r *mqlClaude) messageBatches() ([]interface{}, error) {
	c := conn(r.MqlRuntime)
	client := c.Client()

	pager := client.Beta.Messages.Batches.ListAutoPaging(context.Background(), anthropic.BetaMessageBatchListParams{})

	var res []interface{}
	for pager.Next() {
		b := pager.Current()

		mqlBatch, err := CreateResource(r.MqlRuntime, "claude.messageBatch", map[string]*llx.RawData{
			"__id":              llx.StringData(b.ID),
			"id":                llx.StringData(b.ID),
			"processingStatus":  llx.StringData(string(b.ProcessingStatus)),
			"createdAt":         llx.TimeData(b.CreatedAt),
			"endedAt":           llx.TimeData(b.EndedAt),
			"expiresAt":         llx.TimeData(b.ExpiresAt),
			"archivedAt":        llx.TimeData(b.ArchivedAt),
			"cancelInitiatedAt": llx.TimeData(b.CancelInitiatedAt),
			"type":              llx.StringData(string(b.Type)),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlBatch)
	}
	if err := pager.Err(); err != nil {
		return nil, fmt.Errorf("listing message batches: %w", err)
	}

	return res, nil
}

// claude.userProfile

func (r *mqlClaude) userProfiles() ([]interface{}, error) {
	c := conn(r.MqlRuntime)
	client := c.Client()

	pager := client.Beta.UserProfiles.ListAutoPaging(context.Background(), anthropic.BetaUserProfileListParams{})

	var res []interface{}
	for pager.Next() {
		p := pager.Current()

		mqlProfile, err := CreateResource(r.MqlRuntime, "claude.userProfile", map[string]*llx.RawData{
			"__id":         llx.StringData(p.ID),
			"id":           llx.StringData(p.ID),
			"name":         llx.StringData(p.Name),
			"externalId":   llx.StringData(p.ExternalID),
			"relationship": llx.StringData(string(p.Relationship)),
			"createdAt":    llx.TimeData(p.CreatedAt),
			"updatedAt":    llx.TimeData(p.UpdatedAt),
			"type":         llx.StringData(string(p.Type)),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlProfile)
	}
	if err := pager.Err(); err != nil {
		return nil, fmt.Errorf("listing user profiles: %w", err)
	}

	return res, nil
}
