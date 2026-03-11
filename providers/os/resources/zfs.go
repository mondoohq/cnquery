// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/os/resources/zfs"
	"go.mondoo.com/mql/v13/types"
)

// validZfsName matches valid ZFS pool, dataset, and snapshot names.
// ZFS names consist of alphanumerics, underscores, hyphens, periods, colons, and slashes.
// Snapshot names include an @ separator.
var validZfsName = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.\-/:@]*$`)

func (z *mqlZfs) id() (string, error) {
	return "zfs", nil
}

func (z *mqlZfs) version() (string, error) {
	o, err := CreateResource(z.MqlRuntime, "command", map[string]*llx.RawData{
		"command": llx.StringData("zfs version"),
	})
	if err != nil {
		return "", err
	}
	cmd := o.(*mqlCommand)
	if exit := cmd.GetExitcode(); exit.Data != 0 {
		return "", errors.New("could not retrieve zfs version: " + cmd.Stderr.Data)
	}
	version := strings.TrimSpace(cmd.Stdout.Data)
	if i := strings.IndexByte(version, '\n'); i != -1 {
		version = version[:i]
	}
	return version, nil
}

func (z *mqlZfs) pools() ([]any, error) {
	o, err := CreateResource(z.MqlRuntime, "command", map[string]*llx.RawData{
		"command": llx.StringData("zpool get -jp all"),
	})
	if err != nil {
		return nil, err
	}
	cmd := o.(*mqlCommand)
	if exit := cmd.GetExitcode(); exit.Data != 0 {
		return nil, errors.New("could not retrieve zfs pools: " + cmd.Stderr.Data)
	}

	pools, err := zfs.ParsePools(cmd.Stdout.Data)
	if err != nil {
		return nil, err
	}

	res := make([]any, 0, len(pools))
	for _, p := range pools {
		r, err := CreateResource(z.MqlRuntime, "zfs.pool", map[string]*llx.RawData{
			"name":           llx.StringData(p.Name),
			"guid":           llx.StringData(p.GUID),
			"health":         llx.StringData(p.Health),
			"sizeBytes":      llx.IntData(p.Size),
			"allocatedBytes": llx.IntData(p.Allocated),
			"freeBytes":      llx.IntData(p.Free),
			"fragmentation":  llx.IntData(p.Fragmentation),
			"percentUsed":    llx.IntData(p.PercentUsed),
			"dedupratio":     llx.FloatData(p.Dedupratio),
			"readonly":       llx.BoolData(p.Readonly),
			"autoexpand":     llx.BoolData(p.Autoexpand),
			"autoreplace":    llx.BoolData(p.Autoreplace),
			"autotrim":       llx.BoolData(p.Autotrim),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, r)
	}
	return res, nil
}

func (z *mqlZfs) datasets() ([]any, error) {
	o, err := CreateResource(z.MqlRuntime, "command", map[string]*llx.RawData{
		"command": llx.StringData("zfs get -jp all"),
	})
	if err != nil {
		return nil, err
	}
	cmd := o.(*mqlCommand)
	if exit := cmd.GetExitcode(); exit.Data != 0 {
		return nil, errors.New("could not retrieve zfs datasets: " + cmd.Stderr.Data)
	}

	datasets, err := zfs.ParseDatasets(cmd.Stdout.Data)
	if err != nil {
		return nil, err
	}

	res := make([]any, 0, len(datasets))
	for _, ds := range datasets {
		r, err := CreateResource(z.MqlRuntime, "zfs.dataset", map[string]*llx.RawData{
			"name":             llx.StringData(ds.Name),
			"type":             llx.StringData(ds.Type),
			"usedBytes":        llx.IntData(ds.Used),
			"availableBytes":   llx.IntData(ds.Available),
			"referencedBytes":  llx.IntData(ds.Referenced),
			"mountpoint":       llx.StringData(ds.Mountpoint),
			"compression":      llx.StringData(ds.Compression),
			"compressratio":    llx.FloatData(ds.Compressratio),
			"mounted":          llx.BoolData(ds.Mounted),
			"recordsizeBytes":  llx.IntData(ds.Recordsize),
			"quotaBytes":       llx.IntData(ds.Quota),
			"reservationBytes": llx.IntData(ds.Reservation),
			"origin":           llx.StringData(ds.Origin),
			"creation":         llx.TimeDataPtr(ds.Creation),
			"encryption":       llx.StringData(ds.Encryption),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, r)
	}
	return res, nil
}

// zfs.pool

func (p *mqlZfsPool) id() (string, error) {
	return "zfs.pool/" + p.Name.Data, nil
}

func initZfsPool(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}

	nameRaw := args["name"]
	if nameRaw == nil {
		return args, nil, nil
	}

	name, ok := nameRaw.Value.(string)
	if !ok {
		return args, nil, nil
	}

	obj, err := CreateResource(runtime, "zfs", map[string]*llx.RawData{})
	if err != nil {
		return nil, nil, err
	}
	z := obj.(*mqlZfs)
	pools := z.GetPools()
	if pools.Error != nil {
		return nil, nil, pools.Error
	}

	for i := range pools.Data {
		pool := pools.Data[i].(*mqlZfsPool)
		if pool.Name.Data == name {
			return nil, pool, nil
		}
	}

	return nil, nil, errors.New("zfs pool not found: " + name)
}

func (p *mqlZfsPool) vdevs() ([]any, error) {
	if !validZfsName.MatchString(p.Name.Data) {
		return nil, fmt.Errorf("invalid zfs pool name: %q", p.Name.Data)
	}
	o, err := CreateResource(p.MqlRuntime, "command", map[string]*llx.RawData{
		"command": llx.StringData(fmt.Sprintf("zpool status -jp %q", p.Name.Data)),
	})
	if err != nil {
		return nil, err
	}
	cmd := o.(*mqlCommand)
	if exit := cmd.GetExitcode(); exit.Data != 0 {
		return nil, errors.New("could not retrieve zfs pool vdevs: " + cmd.Stderr.Data)
	}

	vdevs, err := zfs.ParseVdevs(cmd.Stdout.Data)
	if err != nil {
		return nil, err
	}

	return createVdevResources(p.MqlRuntime, p.Name.Data, vdevs)
}

func createVdevResources(runtime *plugin.Runtime, poolName string, vdevs []zfs.Vdev) ([]any, error) {
	res := make([]any, 0, len(vdevs))
	for _, v := range vdevs {
		children, err := createVdevResources(runtime, poolName, v.Devices)
		if err != nil {
			return nil, err
		}

		r, err := CreateResource(runtime, "zfs.pool.vdev", map[string]*llx.RawData{
			"name":           llx.StringData(v.Name),
			"type":           llx.StringData(v.Type),
			"state":          llx.StringData(v.State),
			"path":           llx.StringData(v.Path),
			"readErrors":     llx.IntData(v.ReadErrors),
			"writeErrors":    llx.IntData(v.WriteErrors),
			"checksumErrors": llx.IntData(v.ChecksumErrors),
			"slowIos":        llx.IntData(v.SlowIOs),
			"numDevices":     llx.IntData(int64(len(v.Devices))),
			"devices":        llx.ArrayData(children, types.Resource("zfs.pool.vdev")),
		})
		if err != nil {
			return nil, err
		}
		vdev := r.(*mqlZfsPoolVdev)
		vdev.poolName = poolName
		res = append(res, r)
	}
	return res, nil
}

// zfs.pool.vdev

type mqlZfsPoolVdevInternal struct {
	poolName string
}

func (v *mqlZfsPoolVdev) id() (string, error) {
	return "zfs.pool.vdev/" + v.poolName + "/" + v.Name.Data, nil
}

func (p *mqlZfsPool) properties() (map[string]any, error) {
	if !validZfsName.MatchString(p.Name.Data) {
		return nil, fmt.Errorf("invalid zfs pool name: %q", p.Name.Data)
	}
	o, err := CreateResource(p.MqlRuntime, "command", map[string]*llx.RawData{
		"command": llx.StringData(fmt.Sprintf("zpool get -jp all %q", p.Name.Data)),
	})
	if err != nil {
		return nil, err
	}
	cmd := o.(*mqlCommand)
	if exit := cmd.GetExitcode(); exit.Data != 0 {
		return nil, errors.New("could not retrieve zfs pool properties: " + cmd.Stderr.Data)
	}

	props, err := zfs.ParseProperties(cmd.Stdout.Data)
	if err != nil {
		return nil, err
	}

	result := make(map[string]any, len(props))
	for k, v := range props {
		result[k] = v
	}
	return result, nil
}

// zfs.dataset

func (d *mqlZfsDataset) id() (string, error) {
	return "zfs.dataset/" + d.Name.Data, nil
}

func initZfsDataset(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}

	nameRaw := args["name"]
	if nameRaw == nil {
		return args, nil, nil
	}

	name, ok := nameRaw.Value.(string)
	if !ok {
		return args, nil, nil
	}

	obj, err := CreateResource(runtime, "zfs", map[string]*llx.RawData{})
	if err != nil {
		return nil, nil, err
	}
	z := obj.(*mqlZfs)
	datasets := z.GetDatasets()
	if datasets.Error != nil {
		return nil, nil, datasets.Error
	}

	for i := range datasets.Data {
		ds := datasets.Data[i].(*mqlZfsDataset)
		if ds.Name.Data == name {
			return nil, ds, nil
		}
	}

	return nil, nil, errors.New("zfs dataset not found: " + name)
}

func (d *mqlZfsDataset) properties() (map[string]any, error) {
	if !validZfsName.MatchString(d.Name.Data) {
		return nil, fmt.Errorf("invalid zfs dataset name: %q", d.Name.Data)
	}
	o, err := CreateResource(d.MqlRuntime, "command", map[string]*llx.RawData{
		"command": llx.StringData(fmt.Sprintf("zfs get -jp all %q", d.Name.Data)),
	})
	if err != nil {
		return nil, err
	}
	cmd := o.(*mqlCommand)
	if exit := cmd.GetExitcode(); exit.Data != 0 {
		return nil, errors.New("could not retrieve zfs dataset properties: " + cmd.Stderr.Data)
	}

	props, err := zfs.ParseProperties(cmd.Stdout.Data)
	if err != nil {
		return nil, err
	}

	result := make(map[string]any, len(props))
	for k, v := range props {
		result[k] = v
	}
	return result, nil
}

func (d *mqlZfsDataset) snapshots() ([]any, error) {
	obj, err := CreateResource(d.MqlRuntime, "zfs", map[string]*llx.RawData{})
	if err != nil {
		return nil, err
	}
	z := obj.(*mqlZfs)
	datasets := z.GetDatasets()
	if datasets.Error != nil {
		return nil, datasets.Error
	}

	prefix := d.Name.Data + "@"
	var snapshots []any
	for i := range datasets.Data {
		ds := datasets.Data[i].(*mqlZfsDataset)
		if ds.Type.Data == "snapshot" && strings.HasPrefix(ds.Name.Data, prefix) {
			snapshots = append(snapshots, ds)
		}
	}
	return snapshots, nil
}
