// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"fmt"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
)

type mqlHetznerImageInternal struct {
	cacheBoundServer *hcloud.Server
}

func (r *mqlHetznerImage) id() (string, error) {
	return fmt.Sprintf("hetzner.image/%d", r.Id.Data), nil
}

func (h *mqlHetzner) images() ([]any, error) {
	c := conn(h.MqlRuntime)
	items, err := paginate(func(opts hcloud.ListOpts) ([]*hcloud.Image, *hcloud.Response, error) {
		return c.Client().Image.List(ctx(), hcloud.ImageListOpts{ListOpts: opts})
	})
	if err != nil {
		return nil, err
	}
	out := make([]any, 0, len(items))
	for _, img := range items {
		res, err := newMqlHetznerImage(h.MqlRuntime, img)
		if err != nil {
			return nil, err
		}
		out = append(out, res)
	}
	return out, nil
}

func newMqlHetznerImage(runtime *plugin.Runtime, img *hcloud.Image) (*mqlHetznerImage, error) {
	res, err := CreateResource(runtime, "hetzner.image", map[string]*llx.RawData{
		"__id":        llx.StringData(fmt.Sprintf("hetzner.image/%d", img.ID)),
		"id":          llx.IntData(img.ID),
		"type":        llx.StringData(string(img.Type)),
		"status":      llx.StringData(string(img.Status)),
		"name":        llx.StringData(img.Name),
		"description": llx.StringData(img.Description),
		"imageSize":   llx.FloatData(float64(img.ImageSize)),
		"diskSize":    llx.FloatData(float64(img.DiskSize)),
		"created":     llx.TimeDataPtr(timePtr(img.Created)),
		"osFlavor":    llx.StringData(img.OSFlavor),
		"osVersion":   llx.StringData(img.OSVersion),
		"rapidDeploy": llx.BoolData(img.RapidDeploy),
		"protection":  llx.DictData(protectionDict(img.Protection.Delete)),
		"deprecated":  llx.TimeDataPtr(timePtr(img.Deprecated)),
		"labels":      labelData(img.Labels),
	})
	if err != nil {
		return nil, err
	}
	m := res.(*mqlHetznerImage)
	m.cacheBoundServer = img.BoundTo
	return m, nil
}

func initHetznerImage(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	id, ok := idArg(args, "id")
	if !ok {
		return nil, nil, errIDRequired("image")
	}
	img, _, err := conn(runtime).Client().Image.GetByID(ctx(), id)
	if err != nil {
		return nil, nil, err
	}
	if img == nil {
		return nil, nil, notFoundErr("image", id)
	}
	res, err := newMqlHetznerImage(runtime, img)
	return args, res, err
}

func (m *mqlHetznerImage) boundServer() (*mqlHetznerServer, error) {
	return resolveTypedResource(&m.BoundServer, m.cacheBoundServer, func(s *hcloud.Server) (*mqlHetznerServer, error) {
		return newMqlHetznerServer(m.MqlRuntime, s)
	})
}
