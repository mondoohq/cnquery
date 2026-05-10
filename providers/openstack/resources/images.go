// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"github.com/gophercloud/gophercloud/v2/openstack/image/v2/images"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
)

// ---- openstack.image ----

type mqlOpenstackImageInternal struct {
	cacheOwnerID string
}

func (r *mqlOpenstackImage) id() (string, error) {
	return "openstack.image/" + r.Id.Data, nil
}

func initOpenstackImage(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	id, ok := stringArg(args, "id")
	if !ok || id == "" {
		return args, nil, nil
	}
	root, err := CreateResource(runtime, "openstack", map[string]*llx.RawData{})
	if err != nil {
		return nil, nil, err
	}
	list := root.(*mqlOpenstack).GetImages()
	if list.Error == nil {
		for _, raw := range list.Data {
			img := raw.(*mqlOpenstackImage)
			if img.Id.Data == id {
				return args, img, nil
			}
		}
	}
	initSyntheticID("openstack.image", "id", args)
	return args, nil, nil
}

func (o *mqlOpenstack) images() ([]any, error) {
	c := conn(o.MqlRuntime)
	client, err := c.ImageClient()
	if err != nil {
		return nil, err
	}
	pages, err := images.List(client, images.ListOpts{}).AllPages(ctx())
	if err != nil {
		if translateOpenstackError(err) == nil {
			return []any{}, nil
		}
		return nil, err
	}
	items, err := images.ExtractImages(pages)
	if err != nil {
		return nil, err
	}

	out := make([]any, 0, len(items))
	for i := range items {
		img := &items[i]
		res, err := CreateResource(o.MqlRuntime, "openstack.image", map[string]*llx.RawData{
			"__id":             llx.StringData("openstack.image/" + img.ID),
			"id":               llx.StringData(img.ID),
			"name":             llx.StringData(img.Name),
			"status":           llx.StringData(string(img.Status)),
			"visibility":       llx.StringData(string(img.Visibility)),
			"protected":        llx.BoolData(img.Protected),
			"hidden":           llx.BoolData(img.Hidden),
			"containerFormat":  llx.StringData(img.ContainerFormat),
			"diskFormat":       llx.StringData(img.DiskFormat),
			"minDiskGigabytes": llx.IntData(int64(img.MinDiskGigabytes)),
			"minRamMegabytes":  llx.IntData(int64(img.MinRAMMegabytes)),
			"checksum":         llx.StringData(img.Checksum),
			"sizeBytes":        llx.IntData(img.SizeBytes),
			"virtualSize":      llx.IntData(img.VirtualSize),
			"tags":             stringSliceData(img.Tags),
			"metadata":         stringMapData(img.Metadata),
			"properties":       llx.DictData(toDict(img.Properties)),
			"createdAt":        llx.TimeDataPtr(timePtr(img.CreatedAt)),
			"updatedAt":        llx.TimeDataPtr(timePtr(img.UpdatedAt)),
		})
		if err != nil {
			return nil, err
		}
		mqlImage := res.(*mqlOpenstackImage)
		mqlImage.cacheOwnerID = img.Owner
		out = append(out, mqlImage)
	}
	return out, nil
}

func (r *mqlOpenstackImage) owner() (*mqlOpenstackProject, error) {
	if r.cacheOwnerID == "" {
		r.Owner.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	res, err := NewResource(r.MqlRuntime, "openstack.project", map[string]*llx.RawData{
		"id": llx.StringData(r.cacheOwnerID),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlOpenstackProject), nil
}
