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

// mqlIruBlueprintInternal caches the library item IDs from the blueprint
// listing so the typed libraryItems() accessor can resolve refs without
// re-fetching the blueprint.
type mqlIruBlueprintInternal struct {
	cacheLibraryItemIds []string
}

func (r *mqlIru) blueprints() ([]any, error) {
	conn := r.MqlRuntime.Connection.(*connection.IruConnection)
	bps, err := conn.ListBlueprints()
	if err != nil {
		return nil, err
	}

	res := make([]any, 0, len(bps))
	for i := range bps {
		args, libraryItemIds := blueprintArgs(&bps[i])
		item, err := CreateResource(r.MqlRuntime, "iru.blueprint", args)
		if err != nil {
			return nil, err
		}
		bp := item.(*mqlIruBlueprint)
		bp.cacheLibraryItemIds = libraryItemIds
		res = append(res, bp)
	}
	return res, nil
}

func (b *mqlIruBlueprint) id() (string, error) {
	return "iru.blueprint/" + b.Id.Data, nil
}

// initIruBlueprint hydrates an iru.blueprint created by id (e.g. as
// a cross-reference from iru.libraryItem.blueprints or iru.device.blueprint)
// using the tenant-wide blueprint listing memoized on the connection.
// Without this, stub resources created via NewResource would have empty
// name/description/etc.
func initIruBlueprint(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	idArg, ok := args["id"]
	if !ok || idArg == nil || idArg.Value == nil {
		return args, nil, nil
	}
	id, ok := idArg.Value.(string)
	if !ok || id == "" {
		return args, nil, nil
	}
	conn := runtime.Connection.(*connection.IruConnection)
	bps, err := conn.ListBlueprints()
	if err != nil {
		return args, nil, err
	}
	for i := range bps {
		if bps[i].ID != id {
			continue
		}
		hydrated, libraryItemIds := blueprintArgs(&bps[i])
		res, err := CreateResource(runtime, "iru.blueprint", hydrated)
		if err != nil {
			return nil, nil, err
		}
		bp := res.(*mqlIruBlueprint)
		bp.cacheLibraryItemIds = libraryItemIds
		return nil, bp, nil
	}
	return nil, nil, fmt.Errorf("iru.blueprint with id %q not found", id)
}

func blueprintArgs(b *client.Blueprint) (map[string]*llx.RawData, []string) {
	args := map[string]*llx.RawData{
		"id":             llx.StringData(b.ID),
		"name":           llx.StringData(b.Name),
		"description":    llx.StringData(b.Description),
		"enrollmentCode": llx.StringData(b.EnrollmentCode),
		"devicesCount":   llx.IntData(int64(b.DevicesCount)),
		"created":        llx.TimeDataPtr(client.ParseTime(b.Created)),
		"updatedAt":      llx.TimeDataPtr(client.ParseTime(b.UpdatedAt)),
	}
	return args, b.LibraryItems
}

func (b *mqlIruBlueprint) libraryItems() ([]any, error) {
	if len(b.cacheLibraryItemIds) == 0 {
		return []any{}, nil
	}
	res := make([]any, 0, len(b.cacheLibraryItemIds))
	for _, id := range b.cacheLibraryItemIds {
		item, err := NewResource(b.MqlRuntime, "iru.libraryItem", map[string]*llx.RawData{
			"id": llx.StringData(id),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, item)
	}
	return res, nil
}
