// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"fmt"
	"strings"

	"go.mondoo.com/mql/v13/llx"
)

func (r *mqlOpenai) models() ([]any, error) {
	conn := openaiConn(r.MqlRuntime)
	client := conn.Client()
	if client == nil {
		return []any{}, nil
	}
	ctx := context.Background()

	iter := client.Models.ListAutoPaging(ctx)
	var res []any
	for iter.Next() {
		m := iter.Current()
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
	if err := iter.Err(); err != nil {
		if isAccessDenied(err) {
			return []any{}, nil
		}
		return nil, fmt.Errorf("failed to list models: %w", err)
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
