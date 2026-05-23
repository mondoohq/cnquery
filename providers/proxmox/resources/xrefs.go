// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"strings"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
)

// ---------------------------------------------------------------------------
// vm.disk / container.mountPoint → storage
// ---------------------------------------------------------------------------

func (r *mqlProxmoxVmDisk) storageRef() (*mqlProxmoxStorage, error) {
	return resolveStorageRef(r.MqlRuntime, r.Storage.Data, &r.StorageRef)
}

func (r *mqlProxmoxContainerMountPoint) storageRef() (*mqlProxmoxStorage, error) {
	return resolveStorageRef(r.MqlRuntime, r.Storage.Data, &r.StorageRef)
}

func resolveStorageRef(runtime *plugin.Runtime, storageID string, slot *plugin.TValue[*mqlProxmoxStorage]) (*mqlProxmoxStorage, error) {
	if storageID == "" {
		slot.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	res, err := NewResource(runtime, "proxmox.storage", map[string]*llx.RawData{
		"id": llx.StringData(storageID),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlProxmoxStorage), nil
}

// ---------------------------------------------------------------------------
// token → owner user
// ---------------------------------------------------------------------------

func (r *mqlProxmoxToken) owner() (*mqlProxmoxUser, error) {
	// Token id shape is `user@realm!tokenid`. Anything else is a token
	// the API would have refused to create, so treat unparseable ids as
	// "no resolvable owner" rather than failing the whole query.
	bang := strings.LastIndex(r.Id.Data, "!")
	if bang <= 0 {
		r.Owner.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	ownerID := r.Id.Data[:bang]
	res, err := NewResource(r.MqlRuntime, "proxmox.user", map[string]*llx.RawData{
		"id": llx.StringData(ownerID),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlProxmoxUser), nil
}

// ---------------------------------------------------------------------------
// user → groupRefs
// ---------------------------------------------------------------------------

func (r *mqlProxmoxUser) groupRefs() ([]any, error) {
	out := make([]any, 0, len(r.Groups.Data))
	for _, raw := range r.Groups.Data {
		g, ok := raw.(string)
		if !ok || g == "" {
			continue
		}
		res, err := NewResource(r.MqlRuntime, "proxmox.group", map[string]*llx.RawData{
			"id": llx.StringData(g),
		})
		if err != nil {
			return nil, err
		}
		out = append(out, res)
	}
	return out, nil
}
