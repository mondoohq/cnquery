// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"sync"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/proxmox/connection"
)

// ---------------------------------------------------------------------------
// node.disks (+ SMART, lazily loaded per disk)
// ---------------------------------------------------------------------------

type mqlProxmoxNodeDiskInternal struct {
	parentNode   string
	smartFetched bool
	smartData    *connection.DiskSMART
	smartErr     error
	lock         sync.Mutex
}

func (r *mqlProxmoxNode) disks() ([]any, error) {
	conn := nodeConn(r)
	disks, err := conn.GetNodeDisks(r.Name.Data)
	if err != nil {
		return nil, err
	}
	list := make([]any, len(disks))
	for i, d := range disks {
		res, err := CreateResource(r.MqlRuntime, "proxmox.node.disk", map[string]*llx.RawData{
			"devPath":  llx.StringData(d.DevPath),
			"model":    llx.StringData(d.Model),
			"vendor":   llx.StringData(d.Vendor),
			"serial":   llx.StringData(d.Serial),
			"wwn":      llx.StringData(d.WWN),
			"byIdLink": llx.StringData(d.ByIDLink),
			"size":     llx.IntData(d.Size),
			"rpm":      llx.IntData(int64(d.RPM)),
			"type":     llx.StringData(d.Type),
			"usedBy":   llx.StringData(d.Used),
			"health":   llx.StringData(d.Health),
			"gpt":      llx.BoolData(d.GPT == 1),
		})
		if err != nil {
			return nil, err
		}
		res.(*mqlProxmoxNodeDisk).parentNode = r.Name.Data
		list[i] = res
	}
	return list, nil
}

func (r *mqlProxmoxNodeDisk) ensureSMART() {
	if r.smartFetched {
		return
	}
	r.lock.Lock()
	defer r.lock.Unlock()
	if r.smartFetched {
		return
	}
	conn := r.MqlRuntime.Connection.(*connection.PveConnection)
	r.smartData, r.smartErr = conn.GetDiskSMART(r.parentNode, r.DevPath.Data)
	r.smartFetched = true
}

func (r *mqlProxmoxNodeDisk) smart() (*mqlProxmoxNodeDiskSmart, error) {
	r.ensureSMART()
	if r.smartErr != nil {
		// SMART is unavailable on USB / virtual / pass-through disks. Treat
		// the "no SMART data" case the same as "couldn't read SMART" — emit
		// a record with UNKNOWN health rather than failing the whole query.
		if connection.IsAccessDeniedOrNotFound(r.smartErr) {
			res, err := CreateResource(r.MqlRuntime, "proxmox.node.disk.smart", map[string]*llx.RawData{
				"__id":       llx.StringData("proxmox.node.disk.smart/" + r.parentNode + "/" + r.DevPath.Data),
				"health":     llx.StringData("UNKNOWN"),
				"type":       llx.StringData(""),
				"text":       llx.StringData(""),
				"attributes": llx.ArrayData([]any{}, "\x07"),
			})
			if err != nil {
				return nil, err
			}
			smartRes := res.(*mqlProxmoxNodeDiskSmart)
			smartRes.parentNode = r.parentNode
			smartRes.parentDev = r.DevPath.Data
			return smartRes, nil
		}
		return nil, r.smartErr
	}
	if r.smartData == nil {
		r.Smart.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	attrs := make([]any, 0, len(r.smartData.Attributes))
	for _, a := range r.smartData.Attributes {
		attrs = append(attrs, map[string]any{
			"id":        int64(a.ID),
			"name":      a.Name,
			"value":     int64(a.Value),
			"worst":     int64(a.Worst),
			"threshold": int64(a.Threshold),
			"raw":       a.Raw,
			"flags":     a.Flags,
			"fail":      a.Fail,
		})
	}
	res, err := CreateResource(r.MqlRuntime, "proxmox.node.disk.smart", map[string]*llx.RawData{
		"__id":       llx.StringData("proxmox.node.disk.smart/" + r.parentNode + "/" + r.DevPath.Data),
		"health":     llx.StringData(r.smartData.Health),
		"type":       llx.StringData(r.smartData.Type),
		"text":       llx.StringData(r.smartData.Text),
		"attributes": llx.ArrayData(attrs, "\x07"),
	})
	if err != nil {
		return nil, err
	}
	smartRes := res.(*mqlProxmoxNodeDiskSmart)
	smartRes.parentNode = r.parentNode
	smartRes.parentDev = r.DevPath.Data
	return smartRes, nil
}

// ---------------------------------------------------------------------------
// node.zfsPools (+ per-pool detail lazily loaded)
// ---------------------------------------------------------------------------

type mqlProxmoxZfsPoolInternal struct {
	parentNode    string
	detailFetched bool
	detail        *connection.ZFSPoolDetail
	detailErr     error
	lock          sync.Mutex
}

func (r *mqlProxmoxNode) zfsPools() ([]any, error) {
	conn := nodeConn(r)
	pools, err := conn.GetNodeZFSPools(r.Name.Data)
	if err != nil {
		return nil, err
	}
	list := make([]any, len(pools))
	for i, p := range pools {
		res, err := CreateResource(r.MqlRuntime, "proxmox.zfs.pool", map[string]*llx.RawData{
			"name":          llx.StringData(p.Name),
			"size":          llx.IntData(p.Size),
			"alloc":         llx.IntData(p.Alloc),
			"free":          llx.IntData(p.Free),
			"fragmentation": llx.IntData(int64(p.Frag)),
			"dedupRatio":    llx.FloatData(p.Dedup),
			"health":        llx.StringData(p.Health),
		})
		if err != nil {
			return nil, err
		}
		res.(*mqlProxmoxZfsPool).parentNode = r.Name.Data
		list[i] = res
	}
	return list, nil
}

func (r *mqlProxmoxZfsPool) ensureDetail() {
	if r.detailFetched {
		return
	}
	r.lock.Lock()
	defer r.lock.Unlock()
	if r.detailFetched {
		return
	}
	conn := r.MqlRuntime.Connection.(*connection.PveConnection)
	r.detail, r.detailErr = conn.GetZFSPoolDetail(r.parentNode, r.Name.Data)
	r.detailFetched = true
}

func (r *mqlProxmoxZfsPool) state() (string, error) {
	r.ensureDetail()
	if r.detailErr != nil {
		return "", r.detailErr
	}
	if r.detail == nil {
		return "", nil
	}
	return r.detail.State, nil
}

func (r *mqlProxmoxZfsPool) scan() (string, error) {
	r.ensureDetail()
	if r.detailErr != nil {
		return "", r.detailErr
	}
	if r.detail == nil {
		return "", nil
	}
	return r.detail.Scan, nil
}

func (r *mqlProxmoxZfsPool) errors() (string, error) {
	r.ensureDetail()
	if r.detailErr != nil {
		return "", r.detailErr
	}
	if r.detail == nil {
		return "", nil
	}
	return r.detail.Errors, nil
}

func (r *mqlProxmoxZfsPool) children() ([]any, error) {
	r.ensureDetail()
	if r.detailErr != nil {
		return nil, r.detailErr
	}
	if r.detail == nil {
		return []any{}, nil
	}
	out := make([]any, len(r.detail.Children))
	for i, c := range r.detail.Children {
		out[i] = zfsChildToDict(c)
	}
	return out, nil
}

func zfsChildToDict(c connection.ZFSPoolChild) map[string]any {
	kids := make([]any, len(c.Children))
	for i, sub := range c.Children {
		kids[i] = zfsChildToDict(sub)
	}
	return map[string]any{
		"name":     c.Name,
		"state":    c.State,
		"read":     int64(c.Read),
		"write":    int64(c.Write),
		"cksum":    int64(c.Cksum),
		"msg":      c.Msg,
		"type":     c.Type,
		"leaf":     c.Leaf == 1,
		"children": kids,
	}
}

// ---------------------------------------------------------------------------
// LVM volume groups + thin pools
// ---------------------------------------------------------------------------

type mqlProxmoxLvmVolumeGroupInternal struct {
	parentNode string
}

type mqlProxmoxLvmThinPoolInternal struct {
	parentNode string
}

func (r *mqlProxmoxNode) volumeGroups() ([]any, error) {
	conn := nodeConn(r)
	vgs, err := conn.GetNodeLVM(r.Name.Data)
	if err != nil {
		return nil, err
	}
	list := make([]any, len(vgs))
	for i, vg := range vgs {
		res, err := CreateResource(r.MqlRuntime, "proxmox.lvm.volumeGroup", map[string]*llx.RawData{
			"name":    llx.StringData(vg.Name),
			"size":    llx.IntData(vg.Size),
			"free":    llx.IntData(vg.Free),
			"lvCount": llx.IntData(int64(vg.LVCount)),
		})
		if err != nil {
			return nil, err
		}
		res.(*mqlProxmoxLvmVolumeGroup).parentNode = r.Name.Data
		list[i] = res
	}
	return list, nil
}

func (r *mqlProxmoxNode) thinPools() ([]any, error) {
	conn := nodeConn(r)
	pools, err := conn.GetNodeLVMThin(r.Name.Data)
	if err != nil {
		return nil, err
	}
	list := make([]any, len(pools))
	for i, p := range pools {
		res, err := CreateResource(r.MqlRuntime, "proxmox.lvm.thinPool", map[string]*llx.RawData{
			"name":         llx.StringData(p.LV),
			"volumeGroup":  llx.StringData(p.VG),
			"size":         llx.IntData(p.LVSize),
			"used":         llx.IntData(p.UsedSize),
			"metadataSize": llx.IntData(p.MetadataSize),
			"metadataUsed": llx.IntData(p.MetadataUsed),
		})
		if err != nil {
			return nil, err
		}
		res.(*mqlProxmoxLvmThinPool).parentNode = r.Name.Data
		list[i] = res
	}
	return list, nil
}
