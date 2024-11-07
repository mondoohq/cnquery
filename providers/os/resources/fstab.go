// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1
package resources

import (
	"bufio"
	"errors"
	"io"
	"strconv"
	"strings"

	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers/os/connection/shared"
)

func (f *mqlFstab) entries() ([]any, error) {
	conn := f.MqlRuntime.Connection.(shared.Connection)

	fs := conn.FileSystem()
	if fs == nil {
		return nil, errors.New("filesystem not available")
	}

	fstabFile, err := fs.Open("/etc/fstab")
	if err != nil {
		return nil, err
	}
	defer fstabFile.Close()

	entries, err := ParseFstab(fstabFile)
	if err != nil {
		return nil, err
	}

	resources := []any{}
	for _, entry := range entries {
		resource, err := CreateResource(f.MqlRuntime, "fstab.entry", map[string]*llx.RawData{
			"device":     llx.StringData(entry.Device),
			"mountpoint": llx.StringData(entry.Mountpoint),
			"fstype":     llx.StringData(entry.Fstype),
			"options":    llx.StringData(entry.Options),
			"dump":       llx.IntDataPtr(entry.Dump),
			"fsck":       llx.IntDataPtr(entry.Fsck),
		})
		if err != nil {
			return nil, err
		}
		resources = append(resources, resource)
	}

	return resources, nil
}

type FstabEntry struct {
	Device     string
	Mountpoint string
	Fstype     string
	Options    string
	Dump       *int
	Fsck       *int
}

func ParseFstab(file io.Reader) ([]FstabEntry, error) {
	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)

	var entries []FstabEntry
	for scanner.Scan() {
		line := scanner.Text()
		// Skip comments and empty lines
		if line == "" || line[0] == '#' {
			continue
		}

		record := strings.Fields(line)
		if len(record) < 4 {
			return nil, errors.New("invalid fstab entry")
		}

		var dump *int
		if len(record) >= 5 {
			_dump, err := strconv.Atoi(record[4])
			if err != nil {
				return nil, err
			}
			dump = &_dump
		}

		var fsck *int
		if len(record) >= 6 {
			_fsck, err := strconv.Atoi(record[5])
			if err != nil {
				return nil, err
			}
			fsck = &_fsck
		}

		entry := FstabEntry{
			Device:     record[0],
			Mountpoint: record[1],
			Fstype:     record[2],
			Options:    record[3],
			Dump:       dump,
			Fsck:       fsck,
		}

		entries = append(entries, entry)
	}

	return entries, nil
}

func (e *mqlFstabEntry) id() (string, error) {
	return e.Device.Data, nil
}
