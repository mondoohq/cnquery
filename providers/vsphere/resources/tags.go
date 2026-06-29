// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"fmt"

	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vapi/tags"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/vsphere/connection"
	"go.mondoo.com/mql/v13/types"
)

// mqlVsphereTagInternal caches the category ID so vsphere.tag.category can
// resolve the typed reference against the (memoized) vsphere.categories list
// without re-querying the vAPI per tag.
type mqlVsphereTagInternal struct {
	cacheCategoryID string
}

func (v *mqlVsphere) categories() ([]any, error) {
	conn := v.MqlRuntime.Connection.(*connection.VsphereConnection)
	ctx := context.Background()

	rc, err := conn.RestClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to vAPI: %w", err)
	}

	cats, err := tags.NewManager(rc).GetCategories(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list tag categories: %w", err)
	}

	res := make([]any, 0, len(cats))
	for i := range cats {
		cat := cats[i]
		mqlCat, err := CreateResource(v.MqlRuntime, "vsphere.category", map[string]*llx.RawData{
			"__id":            llx.StringData(cat.ID),
			"id":              llx.StringData(cat.ID),
			"name":            llx.StringData(cat.Name),
			"description":     llx.StringData(cat.Description),
			"cardinality":     llx.StringData(cat.Cardinality),
			"associableTypes": llx.ArrayData(convert.SliceAnyToInterface(cat.AssociableTypes), types.String),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlCat)
	}
	return res, nil
}

func (v *mqlVsphere) tags() ([]any, error) {
	conn := v.MqlRuntime.Connection.(*connection.VsphereConnection)
	ctx := context.Background()

	rc, err := conn.RestClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to vAPI: %w", err)
	}

	vsphereTags, err := tags.NewManager(rc).GetTags(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list tags: %w", err)
	}

	res := make([]any, 0, len(vsphereTags))
	for i := range vsphereTags {
		tag := vsphereTags[i]
		mqlTag, err := CreateResource(v.MqlRuntime, "vsphere.tag", map[string]*llx.RawData{
			"__id":        llx.StringData(tag.ID),
			"id":          llx.StringData(tag.ID),
			"name":        llx.StringData(tag.Name),
			"description": llx.StringData(tag.Description),
		})
		if err != nil {
			return nil, err
		}
		mqlTag.(*mqlVsphereTag).cacheCategoryID = tag.CategoryID
		res = append(res, mqlTag)
	}
	return res, nil
}

func (t *mqlVsphereTag) category() (*mqlVsphereCategory, error) {
	if t.cacheCategoryID == "" {
		t.Category.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}

	res, err := CreateResource(t.MqlRuntime, "vsphere", map[string]*llx.RawData{})
	if err != nil {
		return nil, err
	}
	cats := res.(*mqlVsphere).GetCategories()
	if cats.Error != nil {
		return nil, cats.Error
	}
	for _, c := range cats.Data {
		cat := c.(*mqlVsphereCategory)
		if cat.Id.Data == t.cacheCategoryID {
			return cat, nil
		}
	}

	t.Category.State = plugin.StateIsSet | plugin.StateIsNull
	return nil, nil
}

func (v *mqlVsphere) customFields() ([]any, error) {
	conn := v.MqlRuntime.Connection.(*connection.VsphereConnection)
	ctx := context.Background()

	m, err := object.GetCustomFieldsManager(conn.Client().Client)
	if err != nil {
		return nil, fmt.Errorf("failed to get custom fields manager: %w", err)
	}
	defs, err := m.Field(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list custom field definitions: %w", err)
	}

	res := make([]any, 0, len(defs))
	for _, def := range defs {
		mqlField, err := CreateResource(v.MqlRuntime, "vsphere.customField", map[string]*llx.RawData{
			"__id":              llx.StringData(fmt.Sprintf("vsphere.customField/%d", def.Key)),
			"key":               llx.IntData(int64(def.Key)),
			"name":              llx.StringData(def.Name),
			"managedObjectType": llx.StringData(def.ManagedObjectType),
			"type":              llx.StringData(def.Type),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlField)
	}
	return res, nil
}
