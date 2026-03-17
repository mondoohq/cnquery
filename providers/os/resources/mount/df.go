// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package mount

import (
	"bufio"
	"io"
	"strconv"
	"strings"
)

// DfEntry represents a single line of df output.
type DfEntry struct {
	Filesystem string
	// Size in bytes
	Size int64
	// Used in bytes
	Used int64
	// Available in bytes
	Available int64
	MountedOn string
}

// ParseDf parses the output of "df -P -k" (POSIX portable format, 1024-byte blocks).
func ParseDf(r io.Reader) map[string]*DfEntry {
	entries := map[string]*DfEntry{}

	scanner := bufio.NewScanner(r)
	// Skip header line
	if !scanner.Scan() {
		return entries
	}

	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 6 {
			continue
		}

		// "Mounted on" may contain spaces, rejoin everything from field 5 onward
		mountedOn := strings.Join(fields[5:], " ")

		sizeKB, err := strconv.ParseInt(fields[1], 10, 64)
		if err != nil {
			continue
		}
		usedKB, err := strconv.ParseInt(fields[2], 10, 64)
		if err != nil {
			continue
		}
		availKB, err := strconv.ParseInt(fields[3], 10, 64)
		if err != nil {
			continue
		}

		entries[mountedOn] = &DfEntry{
			Filesystem: fields[0],
			Size:       sizeKB * 1024,
			Used:       usedKB * 1024,
			Available:  availKB * 1024,
			MountedOn:  mountedOn,
		}
	}

	return entries
}
