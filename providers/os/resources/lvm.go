// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"encoding/json"
	"strconv"
	"strings"

	"go.mondoo.com/mql/v13/llx"
)

func (l *mqlLvm) id() (string, error) {
	return "lvm", nil
}

func (l *mqlLvm) physicalVolumes() ([]any, error) {
	stdout, ok, err := l.runLvmReport("pvs --reportformat json --units b --nosuffix -o pv_name,pv_uuid,vg_name,pv_fmt,pv_attr,pv_size,pv_free")
	if err != nil {
		return nil, err
	}
	if !ok {
		return []any{}, nil
	}

	pvs, err := parseLvmPVs(stdout)
	if err != nil {
		return nil, err
	}

	res := make([]any, 0, len(pvs))
	for _, p := range pvs {
		mqlPV, err := CreateResource(l.MqlRuntime, "lvm.physicalVolume", map[string]*llx.RawData{
			"name":            llx.StringData(p.Name),
			"uuid":            llx.StringData(p.UUID),
			"volumeGroupName": llx.StringData(p.VGName),
			"format":          llx.StringData(p.Format),
			"attributes":      llx.StringData(p.Attributes),
			"sizeBytes":       llx.IntData(p.SizeBytes),
			"freeBytes":       llx.IntData(p.FreeBytes),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlPV)
	}
	return res, nil
}

func (l *mqlLvm) volumeGroups() ([]any, error) {
	stdout, ok, err := l.runLvmReport("vgs --reportformat json --units b --nosuffix -o vg_name,vg_uuid,vg_attr,vg_size,vg_free,pv_count,lv_count,snap_count")
	if err != nil {
		return nil, err
	}
	if !ok {
		return []any{}, nil
	}

	vgs, err := parseLvmVGs(stdout)
	if err != nil {
		return nil, err
	}

	res := make([]any, 0, len(vgs))
	for _, v := range vgs {
		mqlVG, err := CreateResource(l.MqlRuntime, "lvm.volumeGroup", map[string]*llx.RawData{
			"name":                llx.StringData(v.Name),
			"uuid":                llx.StringData(v.UUID),
			"attributes":          llx.StringData(v.Attributes),
			"sizeBytes":           llx.IntData(v.SizeBytes),
			"freeBytes":           llx.IntData(v.FreeBytes),
			"physicalVolumeCount": llx.IntData(v.PVCount),
			"logicalVolumeCount":  llx.IntData(v.LVCount),
			"snapshotCount":       llx.IntData(v.SnapshotCount),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlVG)
	}
	return res, nil
}

func (l *mqlLvm) logicalVolumes() ([]any, error) {
	stdout, ok, err := l.runLvmReport("lvs --reportformat json --units b --nosuffix -o lv_name,lv_path,lv_uuid,vg_name,lv_attr,lv_size,origin,data_percent,pool_lv")
	if err != nil {
		return nil, err
	}
	if !ok {
		return []any{}, nil
	}

	lvs, err := parseLvmLVs(stdout)
	if err != nil {
		return nil, err
	}

	res := make([]any, 0, len(lvs))
	for _, v := range lvs {
		mqlLV, err := CreateResource(l.MqlRuntime, "lvm.logicalVolume", map[string]*llx.RawData{
			"name":            llx.StringData(v.Name),
			"path":            llx.StringData(v.Path),
			"uuid":            llx.StringData(v.UUID),
			"volumeGroupName": llx.StringData(v.VGName),
			"attributes":      llx.StringData(v.Attributes),
			"sizeBytes":       llx.IntData(v.SizeBytes),
			"origin":          llx.StringData(v.Origin),
			"dataPercent":     llx.FloatData(v.DataPercent),
			"poolName":        llx.StringData(v.PoolName),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlLV)
	}
	return res, nil
}

// runLvmReport executes an lvm reporting command via the command resource.
// The second return value is false when lvm is not installed or no objects
// of the requested kind exist on the host — both are treated as "empty list",
// not an error, so a query against a non-LVM host succeeds with `[]`.
func (l *mqlLvm) runLvmReport(cmdline string) (string, bool, error) {
	o, err := CreateResource(l.MqlRuntime, "command", map[string]*llx.RawData{
		"command": llx.StringData(cmdline),
	})
	if err != nil {
		return "", false, err
	}
	cmd := o.(*mqlCommand)
	if exit := cmd.GetExitcode(); exit.Data != 0 {
		return "", false, nil
	}
	return cmd.Stdout.Data, true, nil
}

func (l *mqlLvmPhysicalVolume) id() (string, error) {
	if l.Uuid.Data != "" {
		return "lvm.physicalVolume/" + l.Uuid.Data, nil
	}
	return "lvm.physicalVolume/" + l.Name.Data, nil
}

func (l *mqlLvmVolumeGroup) id() (string, error) {
	if l.Uuid.Data != "" {
		return "lvm.volumeGroup/" + l.Uuid.Data, nil
	}
	return "lvm.volumeGroup/" + l.Name.Data, nil
}

func (l *mqlLvmLogicalVolume) id() (string, error) {
	if l.Uuid.Data != "" {
		return "lvm.logicalVolume/" + l.Uuid.Data, nil
	}
	return "lvm.logicalVolume/" + l.VolumeGroupName.Data + "/" + l.Name.Data, nil
}

// LVM reporting commands print all values as strings, even with --nosuffix
// and --units b. Numeric columns therefore need string-to-int/float parsing,
// and empty strings (e.g. data_percent on a non-thin LV) are treated as
// "not applicable".

type parsedLvmPV struct {
	Name       string
	UUID       string
	VGName     string
	Format     string
	Attributes string
	SizeBytes  int64
	FreeBytes  int64
}

type parsedLvmVG struct {
	Name          string
	UUID          string
	Attributes    string
	SizeBytes     int64
	FreeBytes     int64
	PVCount       int64
	LVCount       int64
	SnapshotCount int64
}

type parsedLvmLV struct {
	Name        string
	Path        string
	UUID        string
	VGName      string
	Attributes  string
	SizeBytes   int64
	Origin      string
	DataPercent float64
	PoolName    string
}

type lvmReport[T any] struct {
	Report []map[string][]T `json:"report"`
}

type rawPV struct {
	PvName string `json:"pv_name"`
	PvUUID string `json:"pv_uuid"`
	VgName string `json:"vg_name"`
	PvFmt  string `json:"pv_fmt"`
	PvAttr string `json:"pv_attr"`
	PvSize string `json:"pv_size"`
	PvFree string `json:"pv_free"`
}

type rawVG struct {
	VgName    string `json:"vg_name"`
	VgUUID    string `json:"vg_uuid"`
	VgAttr    string `json:"vg_attr"`
	VgSize    string `json:"vg_size"`
	VgFree    string `json:"vg_free"`
	PvCount   string `json:"pv_count"`
	LvCount   string `json:"lv_count"`
	SnapCount string `json:"snap_count"`
}

type rawLV struct {
	LvName      string `json:"lv_name"`
	LvPath      string `json:"lv_path"`
	LvUUID      string `json:"lv_uuid"`
	VgName      string `json:"vg_name"`
	LvAttr      string `json:"lv_attr"`
	LvSize      string `json:"lv_size"`
	Origin      string `json:"origin"`
	DataPercent string `json:"data_percent"`
	PoolLv      string `json:"pool_lv"`
}

func parseLvmPVs(stdout string) ([]parsedLvmPV, error) {
	rows, err := decodeLvmReport[rawPV](stdout, "pv")
	if err != nil {
		return nil, err
	}
	out := make([]parsedLvmPV, 0, len(rows))
	for _, r := range rows {
		out = append(out, parsedLvmPV{
			Name:       r.PvName,
			UUID:       r.PvUUID,
			VGName:     r.VgName,
			Format:     r.PvFmt,
			Attributes: strings.TrimSpace(r.PvAttr),
			SizeBytes:  parseLvmInt(r.PvSize),
			FreeBytes:  parseLvmInt(r.PvFree),
		})
	}
	return out, nil
}

func parseLvmVGs(stdout string) ([]parsedLvmVG, error) {
	rows, err := decodeLvmReport[rawVG](stdout, "vg")
	if err != nil {
		return nil, err
	}
	out := make([]parsedLvmVG, 0, len(rows))
	for _, r := range rows {
		out = append(out, parsedLvmVG{
			Name:          r.VgName,
			UUID:          r.VgUUID,
			Attributes:    strings.TrimSpace(r.VgAttr),
			SizeBytes:     parseLvmInt(r.VgSize),
			FreeBytes:     parseLvmInt(r.VgFree),
			PVCount:       parseLvmInt(r.PvCount),
			LVCount:       parseLvmInt(r.LvCount),
			SnapshotCount: parseLvmInt(r.SnapCount),
		})
	}
	return out, nil
}

func parseLvmLVs(stdout string) ([]parsedLvmLV, error) {
	rows, err := decodeLvmReport[rawLV](stdout, "lv")
	if err != nil {
		return nil, err
	}
	out := make([]parsedLvmLV, 0, len(rows))
	for _, r := range rows {
		out = append(out, parsedLvmLV{
			Name:        r.LvName,
			Path:        r.LvPath,
			UUID:        r.LvUUID,
			VGName:      r.VgName,
			Attributes:  strings.TrimSpace(r.LvAttr),
			SizeBytes:   parseLvmInt(r.LvSize),
			Origin:      r.Origin,
			DataPercent: parseLvmFloat(r.DataPercent),
			PoolName:    r.PoolLv,
		})
	}
	return out, nil
}

func decodeLvmReport[T any](stdout, key string) ([]T, error) {
	var report lvmReport[T]
	if err := json.Unmarshal([]byte(stdout), &report); err != nil {
		return nil, err
	}
	var rows []T
	for _, section := range report.Report {
		rows = append(rows, section[key]...)
	}
	return rows, nil
}

func parseLvmInt(s string) int64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0
	}
	return v
}

func parseLvmFloat(s string) float64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return -1
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return -1
	}
	return v
}
