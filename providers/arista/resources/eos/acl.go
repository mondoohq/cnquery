// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package eos

import (
	"fmt"
	"sort"
	"strconv"
)

// AclEntryParsed represents a parsed ACL entry with typed fields
type AclEntryParsed struct {
	SequenceNumber int
	Action         string
	SrcAddress     string
	SrcPrefixLen   int
	Log            bool
}

// ParseAclEntry converts raw ACL entry map values into a typed struct.
// seqNum is the map key (sequence number as string), and the remaining
// fields come from the goeapi AclEntry accessors.
func ParseAclEntry(seqNum, action, srcAddr, srcLen, logVal string) (AclEntryParsed, error) {
	seqNumInt, err := strconv.Atoi(seqNum)
	if err != nil {
		return AclEntryParsed{}, fmt.Errorf("invalid sequence number %q: %w", seqNum, err)
	}

	srcPrefixLen, err := strconv.Atoi(srcLen)
	if err != nil {
		return AclEntryParsed{}, fmt.Errorf("invalid source prefix length %q for seq %s: %w", srcLen, seqNum, err)
	}

	return AclEntryParsed{
		SequenceNumber: seqNumInt,
		Action:         action,
		SrcAddress:     srcAddr,
		SrcPrefixLen:   srcPrefixLen,
		Log:            logVal == "log",
	}, nil
}

// SortAclEntries sorts a slice of AclEntryParsed by SequenceNumber in ascending order
func SortAclEntries(entries []AclEntryParsed) {
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].SequenceNumber < entries[j].SequenceNumber
	})
}
