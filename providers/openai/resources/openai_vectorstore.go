// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"fmt"
	"time"

	"github.com/openai/openai-go/v3"
	"go.mondoo.com/mql/v13/llx"
)

func (r *mqlOpenai) vectorStores() ([]any, error) {
	conn := openaiConn(r.MqlRuntime)
	client := conn.Client()
	if client == nil {
		return []any{}, nil
	}
	ctx := context.Background()

	iter := client.VectorStores.ListAutoPaging(ctx, openai.VectorStoreListParams{})
	var res []any
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

		fileCounts := map[string]any{
			"in_progress": vs.FileCounts.InProgress,
			"completed":   vs.FileCounts.Completed,
			"failed":      vs.FileCounts.Failed,
			"cancelled":   vs.FileCounts.Cancelled,
			"total":       vs.FileCounts.Total,
		}

		var expiresAfter any
		if vs.ExpiresAfter.Days != 0 {
			expiresAfter = map[string]any{
				"anchor": string(vs.ExpiresAfter.Anchor),
				"days":   vs.ExpiresAfter.Days,
			}
		}

		metadata := make(map[string]any)
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
			return []any{}, nil
		}
		return nil, fmt.Errorf("failed to list vector stores: %w", err)
	}
	return res, nil
}
