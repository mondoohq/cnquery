// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"fmt"

	"github.com/openai/openai-go/v3"
	"go.mondoo.com/mql/llx"
	"go.mondoo.com/mql/providers-sdk/v1/plugin"
)

// mapVectorStore builds the resource args for an openai.vectorStore. Both the
// collection path and the single-object init share it so the two paths cannot
// diverge.
func mapVectorStore(vs openai.VectorStore) map[string]*llx.RawData {
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

	return map[string]*llx.RawData{
		"__id":         llx.StringData(vs.ID),
		"id":           llx.StringData(vs.ID),
		"name":         llx.StringData(vs.Name),
		"status":       llx.StringData(string(vs.Status)),
		"usageBytes":   llx.IntData(vs.UsageBytes),
		"createdAt":    llx.TimeDataPtr(unixToNullableTime(vs.CreatedAt)),
		"lastActiveAt": llx.TimeDataPtr(unixToNullableTime(vs.LastActiveAt)),
		"fileCounts":   llx.DictData(fileCounts),
		"expiresAfter": llx.DictData(expiresAfter),
		"expiresAt":    llx.TimeDataPtr(unixToNullableTime(vs.ExpiresAt)),
		"metadata":     llx.DictData(metadata),
	}
}

func (r *mqlOpenai) vectorStores() ([]any, error) {
	conn := openaiConn(r.MqlRuntime)
	client, err := dataPlaneClient(conn, "openai.vectorStores")
	if err != nil {
		return nil, err
	}
	if client == nil {
		return []any{}, nil
	}
	ctx := context.Background()

	iter := client.VectorStores.ListAutoPaging(ctx, openai.VectorStoreListParams{})
	var res []any
	for iter.Next() {
		vs := iter.Current()
		mqlVS, err := CreateResource(r.MqlRuntime, "openai.vectorStore", mapVectorStore(vs))
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

func initOpenaiVectorStore(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}
	idRaw, ok := args["id"]
	if !ok || idRaw == nil || idRaw.Value == nil {
		return args, nil, nil
	}
	vsID, ok := idRaw.Value.(string)
	if !ok || vsID == "" {
		return args, nil, nil
	}

	conn := openaiConn(runtime)
	client, err := dataPlaneClient(conn, "openai.vectorStore")
	if err != nil {
		return nil, nil, err
	}
	if client == nil {
		return nil, nil, fmt.Errorf("cannot fetch vector store %s: no project API key configured", vsID)
	}
	vs, err := client.VectorStores.Get(context.Background(), vsID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get vector store %s: %w", vsID, err)
	}
	return mapVectorStore(*vs), nil, nil
}
