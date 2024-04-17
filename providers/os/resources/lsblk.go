// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"encoding/json"
	"errors"

	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/types"
)

func (l *mqlLsblk) id() (string, error) {
	return "lsblk", nil
}

func (l *mqlLsblk) list() ([]interface{}, error) {
	o, err := CreateResource(l.MqlRuntime, "command", map[string]*llx.RawData{
		"command": llx.StringData("lsblk --json --fs"),
	})
	cmd := o.(*mqlCommand)
	if exit := cmd.GetExitcode(); exit.Data != 0 {
		return nil, errors.New("could not retrieve lsblk: " + cmd.Stderr.Data)
	}

	blockEntries, err := parseBlockEntries([]byte(cmd.Stdout.Data))
	if err != nil {
		return nil, err
	}

	mqlBlockEntries := []interface{}{}
	for i := range blockEntries.Blockdevices {
		d := blockEntries.Blockdevices[i]
		for i := range d.Children {
			entry := d.Children[i]
			entry.Mountpoints = append(entry.Mountpoints, entry.Mountpoint)
			mqlLsblkEntry, err := CreateResource(l.MqlRuntime, "lsblk.entry", map[string]*llx.RawData{
				"name":        llx.StringData(entry.Name),
				"fstype":      llx.StringData(entry.Fstype),
				"label":       llx.StringData(entry.Label),
				"uuid":        llx.StringData(entry.Uuid),
				"mountpoints": llx.ArrayData(entry.Mountpoints, types.String),
			})
			if err != nil {
				return nil, err
			}
			mqlBlockEntries = append(mqlBlockEntries, mqlLsblkEntry)
		}
	}
	return mqlBlockEntries, nil
}

func parseBlockEntries(data []byte) (blockdevices, error) {
	blockEntries := blockdevices{}
	if err := json.Unmarshal(data, &blockEntries); err != nil {
		return blockEntries, err
	}
	return blockEntries, nil
}

func (l *mqlLsblkEntry) id() (string, error) {
	return l.Name.Data + "-" + l.Fstype.Data, nil
}

type blockdevices struct {
	Blockdevices []blockdevice `json:"blockdevices,omitempty"`
}

type blockdevice struct {
	Name        string        `json:"name,omitempty"`
	Fstype      string        `json:"fstype,omitempty"`
	Label       string        `json:"label,omitempty"`
	Uuid        string        `json:"uuid,omitempty"`
	Mountpoints []interface{} `json:"mountpoints,omitempty"`
	Mountpoint  string        `json:"mountpoint,omitempty"`
	Children    []blockdevice `json:"children,omitempty"`
}
