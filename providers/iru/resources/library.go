// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"fmt"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/iru/connection"
	"go.mondoo.com/mql/v13/providers/iru/connection/client"
)

// mqlIruLibraryItemInternal caches the blueprint IDs reported in the
// listing so the typed blueprints() accessor doesn't re-fetch the catalog.
type mqlIruLibraryItemInternal struct {
	cacheBlueprintIds []string
}

func (r *mqlIru) libraryItems() ([]any, error) {
	conn := r.MqlRuntime.Connection.(*connection.IruConnection)
	items, err := conn.ListLibraryItems()
	if err != nil {
		return nil, err
	}

	res := make([]any, 0, len(items))
	for i := range items {
		args, blueprintIds := libraryItemArgs(&items[i])
		item, err := CreateResource(r.MqlRuntime, "iru.libraryItem", args)
		if err != nil {
			return nil, err
		}
		mqlItem := item.(*mqlIruLibraryItem)
		mqlItem.cacheBlueprintIds = blueprintIds
		res = append(res, mqlItem)
	}
	return res, nil
}

func (l *mqlIruLibraryItem) id() (string, error) {
	return "iru.libraryItem/" + l.Id.Data, nil
}

// initIruLibraryItem hydrates an iru.libraryItem created by id (e.g. as
// a cross-reference from iru.blueprint.libraryItems) using the tenant-wide
// library item listing memoized on the connection. Without this, stub
// resources created via NewResource would have empty name/kind/etc.
func initIruLibraryItem(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	idArg, ok := args["id"]
	if !ok || idArg == nil || idArg.Value == nil {
		return args, nil, nil
	}
	id, ok := idArg.Value.(string)
	if !ok || id == "" {
		return args, nil, nil
	}
	conn := runtime.Connection.(*connection.IruConnection)
	items, err := conn.ListLibraryItems()
	if err != nil {
		return args, nil, err
	}
	for i := range items {
		if items[i].ID != id {
			continue
		}
		hydrated, blueprintIds := libraryItemArgs(&items[i])
		res, err := CreateResource(runtime, "iru.libraryItem", hydrated)
		if err != nil {
			return nil, nil, err
		}
		mqlItem := res.(*mqlIruLibraryItem)
		mqlItem.cacheBlueprintIds = blueprintIds
		return nil, mqlItem, nil
	}
	return nil, nil, fmt.Errorf("iru.libraryItem with id %q not found", id)
}

func libraryItemArgs(li *client.LibraryItem) (map[string]*llx.RawData, []string) {
	counts := make(map[string]any, len(li.Counts))
	for k, v := range li.Counts {
		counts[k] = int64(v)
	}
	payload := make(map[string]any, len(li.Payload))
	for k, v := range li.Payload {
		payload[k] = v
	}
	args := map[string]*llx.RawData{
		"id":        llx.StringData(li.ID),
		"name":      llx.StringData(li.Name),
		"kind":      llx.StringData(li.Kind),
		"active":    llx.BoolData(li.Active),
		"counts":    llx.MapData(counts, "int"),
		"payload":   llx.DictData(payload),
		"created":   llx.TimeDataPtr(client.ParseTime(li.Created)),
		"updatedAt": llx.TimeDataPtr(client.ParseTime(li.UpdatedAt)),
	}
	return args, li.BlueprintIDs
}

func (l *mqlIruLibraryItem) blueprints() ([]any, error) {
	if len(l.cacheBlueprintIds) == 0 {
		return []any{}, nil
	}
	res := make([]any, 0, len(l.cacheBlueprintIds))
	for _, id := range l.cacheBlueprintIds {
		item, err := NewResource(l.MqlRuntime, "iru.blueprint", map[string]*llx.RawData{
			"id": llx.StringData(id),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, item)
	}
	return res, nil
}
