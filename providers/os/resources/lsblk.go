// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"encoding/json"
	"errors"
	"slices"

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

	for i := range blockEntries.Blockdevices {
		d := blockEntries.Blockdevices[i]
		for j := range d.Children {
			entry := d.Children[j]
			// Some versions of the lsblk return [null] instead of empty array
			entry.Mountpoints = slices.Collect(func(yield func(interface{}) bool) {
				for _, m := range entry.Mountpoints {
					if m != nil && !yield(m) {
						return
					}
				}
			})
			// Some versions of the lsblk return the mountpoint instead of the mountpoints array
			if len(entry.Mountpoints) == 0 && entry.Mountpoint != "" {
				entry.Mountpoints = append(entry.Mountpoints, entry.Mountpoint)
			}
			blockEntries.Blockdevices[i].Children[j] = entry
		}
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
