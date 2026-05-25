// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/openai/openai-go/v3"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/openai/connection"
)

func openaiConn(runtime *plugin.Runtime) *connection.OpenaiConnection {
	return runtime.Connection.(*connection.OpenaiConnection)
}

func unixToTime(ts int64) time.Time {
	if ts == 0 {
		return time.Time{}
	}
	return time.Unix(ts, 0)
}

func isAccessDenied(err error) bool {
	if err == nil {
		return false
	}
	var apiErr *openai.Error
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode == 401 || apiErr.StatusCode == 403
	}
	return false
}

func (r *mqlOpenai) id() (string, error) {
	return "openai", nil
}

func (r *mqlOpenai) organization() (string, error) {
	return openaiConn(r.MqlRuntime).Organization(), nil
}

func (r *mqlOpenai) projectId() (string, error) {
	return openaiConn(r.MqlRuntime).Project(), nil
}

func (r *mqlOpenai) models() ([]interface{}, error) {
	conn := openaiConn(r.MqlRuntime)
	client := conn.Client()
	if client == nil {
		return []interface{}{}, nil
	}
	ctx := context.Background()

	page, err := client.Models.List(ctx)
	if err != nil {
		if isAccessDenied(err) {
			return []interface{}{}, nil
		}
		return nil, fmt.Errorf("failed to list models: %w", err)
	}

	res := make([]interface{}, 0, len(page.Data))
	for _, m := range page.Data {
		created := unixToTime(m.Created)
		mqlModel, err := CreateResource(r.MqlRuntime, "openai.model", map[string]*llx.RawData{
			"__id":      llx.StringData(m.ID),
			"id":        llx.StringData(m.ID),
			"createdAt": llx.TimeData(created),
			"ownedBy":   llx.StringData(m.OwnedBy),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlModel)
	}
	return res, nil
}

func (r *mqlOpenai) files() ([]interface{}, error) {
	conn := openaiConn(r.MqlRuntime)
	client := conn.Client()
	if client == nil {
		return []interface{}{}, nil
	}
	ctx := context.Background()

	iter := client.Files.ListAutoPaging(ctx, openai.FileListParams{})
	var res []interface{}
	for iter.Next() {
		f := iter.Current()
		created := unixToTime(f.CreatedAt)
		mqlFile, err := CreateResource(r.MqlRuntime, "openai.file", map[string]*llx.RawData{
			"__id":      llx.StringData(f.ID),
			"id":        llx.StringData(f.ID),
			"filename":  llx.StringData(f.Filename),
			"bytes":     llx.IntData(f.Bytes),
			"createdAt": llx.TimeData(created),
			"purpose":   llx.StringData(string(f.Purpose)),
			"status":    llx.StringData(string(f.Status)),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlFile)
	}
	if err := iter.Err(); err != nil {
		if isAccessDenied(err) {
			return []interface{}{}, nil
		}
		return nil, fmt.Errorf("failed to list files: %w", err)
	}
	return res, nil
}

func (r *mqlOpenai) fineTuningJobs() ([]interface{}, error) {
	conn := openaiConn(r.MqlRuntime)
	client := conn.Client()
	if client == nil {
		return []interface{}{}, nil
	}
	ctx := context.Background()

	iter := client.FineTuning.Jobs.ListAutoPaging(ctx, openai.FineTuningJobListParams{})
	var res []interface{}
	for iter.Next() {
		j := iter.Current()
		created := unixToTime(j.CreatedAt)

		var finishedAt *time.Time
		if j.FinishedAt != 0 {
			t := unixToTime(j.FinishedAt)
			finishedAt = &t
		}

		var trainedTokens int64
		if j.TrainedTokens != 0 {
			trainedTokens = j.TrainedTokens
		}

		hyperparams := map[string]interface{}{
			"n_epochs": j.Hyperparameters.NEpochs.OfInt,
		}

		var errInfo interface{}
		if j.Error.Code != "" {
			errInfo = map[string]interface{}{
				"code":    j.Error.Code,
				"message": j.Error.Message,
				"param":   j.Error.Param,
			}
		}

		mqlJob, err := CreateResource(r.MqlRuntime, "openai.fineTuningJob", map[string]*llx.RawData{
			"__id":            llx.StringData(j.ID),
			"id":              llx.StringData(j.ID),
			"model":           llx.StringData(j.Model),
			"status":          llx.StringData(string(j.Status)),
			"createdAt":       llx.TimeData(created),
			"finishedAt":      llx.TimeDataPtr(finishedAt),
			"fineTunedModel":  llx.StringData(j.FineTunedModel),
			"trainedTokens":   llx.IntData(trainedTokens),
			"seed":            llx.IntData(j.Seed),
			"organizationId":  llx.StringData(j.OrganizationID),
			"hyperparameters": llx.DictData(hyperparams),
			"error":           llx.DictData(errInfo),
		})
		if err != nil {
			return nil, err
		}

		job := mqlJob.(*mqlOpenaiFineTuningJob)
		job.cacheTrainingFileID = j.TrainingFile
		job.cacheValidationFileID = j.ValidationFile

		res = append(res, mqlJob)
	}
	if err := iter.Err(); err != nil {
		if isAccessDenied(err) {
			return []interface{}{}, nil
		}
		return nil, fmt.Errorf("failed to list fine-tuning jobs: %w", err)
	}
	return res, nil
}

func (r *mqlOpenai) vectorStores() ([]interface{}, error) {
	conn := openaiConn(r.MqlRuntime)
	client := conn.Client()
	if client == nil {
		return []interface{}{}, nil
	}
	ctx := context.Background()

	iter := client.VectorStores.ListAutoPaging(ctx, openai.VectorStoreListParams{})
	var res []interface{}
	for iter.Next() {
		vs := iter.Current()
		created := unixToTime(vs.CreatedAt)

		var lastActiveAt *time.Time
		if vs.LastActiveAt != 0 {
			t := unixToTime(vs.LastActiveAt)
			lastActiveAt = &t
		}

		var expiresAt *time.Time
		if vs.ExpiresAt != 0 {
			t := unixToTime(vs.ExpiresAt)
			expiresAt = &t
		}

		fileCounts := map[string]interface{}{
			"in_progress": vs.FileCounts.InProgress,
			"completed":   vs.FileCounts.Completed,
			"failed":      vs.FileCounts.Failed,
			"cancelled":   vs.FileCounts.Cancelled,
			"total":       vs.FileCounts.Total,
		}

		var expiresAfter interface{}
		if vs.ExpiresAfter.Days != 0 {
			expiresAfter = map[string]interface{}{
				"anchor": string(vs.ExpiresAfter.Anchor),
				"days":   vs.ExpiresAfter.Days,
			}
		}

		metadata := make(map[string]interface{})
		for k, v := range vs.Metadata {
			metadata[k] = v
		}

		mqlVS, err := CreateResource(r.MqlRuntime, "openai.vectorStore", map[string]*llx.RawData{
			"__id":         llx.StringData(vs.ID),
			"id":           llx.StringData(vs.ID),
			"name":         llx.StringData(vs.Name),
			"status":       llx.StringData(string(vs.Status)),
			"usageBytes":   llx.IntData(vs.UsageBytes),
			"createdAt":    llx.TimeData(created),
			"lastActiveAt": llx.TimeDataPtr(lastActiveAt),
			"fileCounts":   llx.DictData(fileCounts),
			"expiresAfter": llx.DictData(expiresAfter),
			"expiresAt":    llx.TimeDataPtr(expiresAt),
			"metadata":     llx.DictData(metadata),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlVS)
	}
	if err := iter.Err(); err != nil {
		if isAccessDenied(err) {
			return []interface{}{}, nil
		}
		return nil, fmt.Errorf("failed to list vector stores: %w", err)
	}
	return res, nil
}

func (r *mqlOpenai) projects() ([]interface{}, error) {
	conn := openaiConn(r.MqlRuntime)
	client := conn.AdminClient()
	if client == nil {
		return []interface{}{}, nil
	}
	ctx := context.Background()

	iter := client.Admin.Organization.Projects.ListAutoPaging(ctx, openai.AdminOrganizationProjectListParams{})
	var res []interface{}
	for iter.Next() {
		p := iter.Current()
		created := unixToTime(p.CreatedAt)

		var archivedAt *time.Time
		if p.ArchivedAt != 0 {
			t := unixToTime(p.ArchivedAt)
			archivedAt = &t
		}

		mqlProject, err := CreateResource(r.MqlRuntime, "openai.project", map[string]*llx.RawData{
			"__id":       llx.StringData(p.ID),
			"id":         llx.StringData(p.ID),
			"name":       llx.StringData(p.Name),
			"status":     llx.StringData(string(p.Status)),
			"createdAt":  llx.TimeData(created),
			"archivedAt": llx.TimeDataPtr(archivedAt),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlProject)
	}
	if err := iter.Err(); err != nil {
		if isAccessDenied(err) {
			return []interface{}{}, nil
		}
		return nil, fmt.Errorf("failed to list projects: %w", err)
	}
	return res, nil
}

func (r *mqlOpenai) users() ([]interface{}, error) {
	conn := openaiConn(r.MqlRuntime)
	client := conn.AdminClient()
	if client == nil {
		return []interface{}{}, nil
	}
	ctx := context.Background()

	iter := client.Admin.Organization.Users.ListAutoPaging(ctx, openai.AdminOrganizationUserListParams{})
	var res []interface{}
	for iter.Next() {
		u := iter.Current()
		addedAt := unixToTime(u.AddedAt)
		created := unixToTime(u.Created)

		var apiKeyLastUsedAt *time.Time
		if u.APIKeyLastUsedAt != 0 {
			t := unixToTime(u.APIKeyLastUsedAt)
			apiKeyLastUsedAt = &t
		}

		mqlUser, err := CreateResource(r.MqlRuntime, "openai.organizationUser", map[string]*llx.RawData{
			"__id":             llx.StringData(u.ID),
			"id":               llx.StringData(u.ID),
			"email":            llx.StringData(u.Email),
			"name":             llx.StringData(u.Name),
			"role":             llx.StringData(u.Role),
			"isDefault":        llx.BoolData(u.IsDefault),
			"isScimManaged":    llx.BoolData(u.IsScimManaged),
			"isServiceAccount": llx.BoolData(u.IsServiceAccount),
			"addedAt":          llx.TimeData(addedAt),
			"createdAt":        llx.TimeData(created),
			"apiKeyLastUsedAt": llx.TimeDataPtr(apiKeyLastUsedAt),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlUser)
	}
	if err := iter.Err(); err != nil {
		if isAccessDenied(err) {
			return []interface{}{}, nil
		}
		return nil, fmt.Errorf("failed to list organization users: %w", err)
	}
	return res, nil
}

func (r *mqlOpenai) invites() ([]interface{}, error) {
	conn := openaiConn(r.MqlRuntime)
	client := conn.AdminClient()
	if client == nil {
		return []interface{}{}, nil
	}
	ctx := context.Background()

	iter := client.Admin.Organization.Invites.ListAutoPaging(ctx, openai.AdminOrganizationInviteListParams{})
	var res []interface{}
	for iter.Next() {
		inv := iter.Current()
		created := unixToTime(inv.CreatedAt)

		var acceptedAt *time.Time
		if inv.AcceptedAt != 0 {
			t := unixToTime(inv.AcceptedAt)
			acceptedAt = &t
		}

		var expiresAt *time.Time
		if inv.ExpiresAt != 0 {
			t := unixToTime(inv.ExpiresAt)
			expiresAt = &t
		}

		mqlInvite, err := CreateResource(r.MqlRuntime, "openai.invite", map[string]*llx.RawData{
			"__id":       llx.StringData(inv.ID),
			"id":         llx.StringData(inv.ID),
			"email":      llx.StringData(inv.Email),
			"role":       llx.StringData(string(inv.Role)),
			"status":     llx.StringData(string(inv.Status)),
			"createdAt":  llx.TimeData(created),
			"acceptedAt": llx.TimeDataPtr(acceptedAt),
			"expiresAt":  llx.TimeDataPtr(expiresAt),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlInvite)
	}
	if err := iter.Err(); err != nil {
		if isAccessDenied(err) {
			return []interface{}{}, nil
		}
		return nil, fmt.Errorf("failed to list invites: %w", err)
	}
	return res, nil
}

func (r *mqlOpenai) auditLogs() ([]interface{}, error) {
	conn := openaiConn(r.MqlRuntime)
	client := conn.AdminClient()
	if client == nil {
		return []interface{}{}, nil
	}
	ctx := context.Background()

	iter := client.Admin.Organization.AuditLogs.ListAutoPaging(ctx, openai.AdminOrganizationAuditLogListParams{})
	var res []interface{}
	for iter.Next() {
		entry := iter.Current()
		effectiveAt := unixToTime(entry.EffectiveAt)

		actorType := entry.Actor.Type
		var actorId string
		switch actorType {
		case "session":
			actorId = entry.Actor.Session.User.Email
		case "api_key":
			actorId = entry.Actor.APIKey.ID
		}

		mqlLog, err := CreateResource(r.MqlRuntime, "openai.auditLog", map[string]*llx.RawData{
			"__id":        llx.StringData(entry.ID),
			"id":          llx.StringData(entry.ID),
			"type":        llx.StringData(string(entry.Type)),
			"effectiveAt": llx.TimeData(effectiveAt),
			"actorType":   llx.StringData(actorType),
			"actorId":     llx.StringData(actorId),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlLog)
	}
	if err := iter.Err(); err != nil {
		if isAccessDenied(err) {
			return []interface{}{}, nil
		}
		return nil, fmt.Errorf("failed to list audit logs: %w", err)
	}
	return res, nil
}

func (r *mqlOpenaiProject) apiKeys() ([]interface{}, error) {
	conn := openaiConn(r.MqlRuntime)
	client := conn.AdminClient()
	if client == nil {
		return []interface{}{}, nil
	}
	ctx := context.Background()

	iter := client.Admin.Organization.Projects.APIKeys.ListAutoPaging(ctx, r.Id.Data, openai.AdminOrganizationProjectAPIKeyListParams{})
	var res []interface{}
	for iter.Next() {
		k := iter.Current()
		created := unixToTime(k.CreatedAt)
		lastUsed := unixToTime(k.LastUsedAt)

		ownerType := k.Owner.Type
		var ownerName, ownerId string
		switch ownerType {
		case "user":
			ownerName = k.Owner.User.Email
			ownerId = k.Owner.User.ID
		case "service_account":
			ownerName = k.Owner.ServiceAccount.Name
			ownerId = k.Owner.ServiceAccount.ID
		}

		mqlKey, err := CreateResource(r.MqlRuntime, "openai.project.apiKey", map[string]*llx.RawData{
			"__id":          llx.StringData(k.ID),
			"id":            llx.StringData(k.ID),
			"name":          llx.StringData(k.Name),
			"redactedValue": llx.StringData(k.RedactedValue),
			"createdAt":     llx.TimeData(created),
			"lastUsedAt":    llx.TimeData(lastUsed),
			"ownerType":     llx.StringData(ownerType),
			"ownerName":     llx.StringData(ownerName),
			"ownerId":       llx.StringData(ownerId),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlKey)
	}
	if err := iter.Err(); err != nil {
		if isAccessDenied(err) {
			return []interface{}{}, nil
		}
		return nil, fmt.Errorf("failed to list project API keys: %w", err)
	}
	return res, nil
}

func (r *mqlOpenaiProject) serviceAccounts() ([]interface{}, error) {
	conn := openaiConn(r.MqlRuntime)
	client := conn.AdminClient()
	if client == nil {
		return []interface{}{}, nil
	}
	ctx := context.Background()

	iter := client.Admin.Organization.Projects.ServiceAccounts.ListAutoPaging(ctx, r.Id.Data, openai.AdminOrganizationProjectServiceAccountListParams{})
	var res []interface{}
	for iter.Next() {
		sa := iter.Current()
		created := unixToTime(sa.CreatedAt)

		mqlSA, err := CreateResource(r.MqlRuntime, "openai.project.serviceAccount", map[string]*llx.RawData{
			"__id":      llx.StringData(sa.ID),
			"id":        llx.StringData(sa.ID),
			"name":      llx.StringData(sa.Name),
			"role":      llx.StringData(string(sa.Role)),
			"createdAt": llx.TimeData(created),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlSA)
	}
	if err := iter.Err(); err != nil {
		if isAccessDenied(err) {
			return []interface{}{}, nil
		}
		return nil, fmt.Errorf("failed to list project service accounts: %w", err)
	}
	return res, nil
}

func (r *mqlOpenaiModel) isFineTuned() (bool, error) {
	return strings.HasPrefix(r.Id.Data, "ft:"), nil
}

func (r *mqlOpenaiModel) baseModel() (string, error) {
	if !strings.HasPrefix(r.Id.Data, "ft:") {
		return "", nil
	}
	// Fine-tuned model ID format: ft:<base-model>:<org>:<suffix>:<id>
	parts := strings.SplitN(r.Id.Data, ":", 3)
	if len(parts) >= 2 {
		return parts[1], nil
	}
	return "", nil
}

// initOpenaiFile fetches file details from the API when only the ID is known
// (e.g., when referenced from a fine-tuning job).
func initOpenaiFile(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}

	idRaw, ok := args["id"]
	if !ok || idRaw == nil || idRaw.Value == nil {
		return args, nil, nil
	}
	fileID, ok := idRaw.Value.(string)
	if !ok || fileID == "" {
		return args, nil, nil
	}

	conn := openaiConn(runtime)
	client := conn.Client()
	if client == nil {
		return args, nil, nil
	}
	f, err := client.Files.Get(context.Background(), fileID)
	if err != nil {
		if isAccessDenied(err) {
			return args, nil, nil
		}
		return nil, nil, fmt.Errorf("failed to get file %s: %w", fileID, err)
	}

	created := unixToTime(f.CreatedAt)
	args["__id"] = llx.StringData(f.ID)
	args["id"] = llx.StringData(f.ID)
	args["filename"] = llx.StringData(f.Filename)
	args["bytes"] = llx.IntData(f.Bytes)
	args["createdAt"] = llx.TimeData(created)
	args["purpose"] = llx.StringData(string(f.Purpose))
	args["status"] = llx.StringData(string(f.Status))

	return args, nil, nil
}

// mqlOpenaiFineTuningJobInternal stores file IDs for lazy-loading typed references.
type mqlOpenaiFineTuningJobInternal struct {
	cacheTrainingFileID   string
	cacheValidationFileID string
}

func (r *mqlOpenaiFineTuningJob) trainingFile() (*mqlOpenaiFile, error) {
	if r.cacheTrainingFileID == "" {
		r.TrainingFile.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	res, err := NewResource(r.MqlRuntime, "openai.file", map[string]*llx.RawData{
		"__id": llx.StringData(r.cacheTrainingFileID),
		"id":   llx.StringData(r.cacheTrainingFileID),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlOpenaiFile), nil
}

func (r *mqlOpenaiFineTuningJob) validationFile() (*mqlOpenaiFile, error) {
	if r.cacheValidationFileID == "" {
		r.ValidationFile.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	res, err := NewResource(r.MqlRuntime, "openai.file", map[string]*llx.RawData{
		"__id": llx.StringData(r.cacheValidationFileID),
		"id":   llx.StringData(r.cacheValidationFileID),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlOpenaiFile), nil
}
