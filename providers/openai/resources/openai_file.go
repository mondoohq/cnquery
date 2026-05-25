// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"fmt"

	"github.com/openai/openai-go/v3"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
)

func (r *mqlOpenai) files() ([]any, error) {
	conn := openaiConn(r.MqlRuntime)
	client := conn.Client()
	if client == nil {
		return []any{}, nil
	}
	ctx := context.Background()

	iter := client.Files.ListAutoPaging(ctx, openai.FileListParams{})
	var res []any
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
			return []any{}, nil
		}
		return nil, fmt.Errorf("failed to list files: %w", err)
	}
	return res, nil
}

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
		return nil, nil, fmt.Errorf("cannot fetch file %s: no project API key configured", fileID)
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
