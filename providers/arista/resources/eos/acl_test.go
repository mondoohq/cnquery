// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package eos

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseAclEntry(t *testing.T) {
	entry, err := ParseAclEntry("10", "permit", "192.168.1.0", "24", "log")
	require.NoError(t, err)
	assert.Equal(t, 10, entry.SequenceNumber)
	assert.Equal(t, "permit", entry.Action)
	assert.Equal(t, "192.168.1.0", entry.SrcAddress)
	assert.Equal(t, 24, entry.SrcPrefixLen)
	assert.True(t, entry.Log)
}

func TestParseAclEntryNoLog(t *testing.T) {
	entry, err := ParseAclEntry("20", "deny", "10.0.0.0", "8", "")
	require.NoError(t, err)
	assert.Equal(t, 20, entry.SequenceNumber)
	assert.Equal(t, "deny", entry.Action)
	assert.Equal(t, "10.0.0.0", entry.SrcAddress)
	assert.Equal(t, 8, entry.SrcPrefixLen)
	assert.False(t, entry.Log)
}

func TestParseAclEntryHostMask(t *testing.T) {
	// /32 host entry
	entry, err := ParseAclEntry("5", "permit", "10.1.1.1", "32", "")
	require.NoError(t, err)
	assert.Equal(t, 5, entry.SequenceNumber)
	assert.Equal(t, 32, entry.SrcPrefixLen)
}

func TestParseAclEntryInvalidSeqNum(t *testing.T) {
	_, err := ParseAclEntry("abc", "permit", "10.0.0.0", "8", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid sequence number")
}

func TestParseAclEntryInvalidPrefixLen(t *testing.T) {
	_, err := ParseAclEntry("10", "permit", "10.0.0.0", "bad", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid source prefix length")
}

func TestSortAclEntries(t *testing.T) {
	entries := []AclEntryParsed{
		{SequenceNumber: 30, Action: "deny"},
		{SequenceNumber: 10, Action: "permit"},
		{SequenceNumber: 20, Action: "permit"},
		{SequenceNumber: 5, Action: "deny"},
	}

	SortAclEntries(entries)

	assert.Equal(t, 5, entries[0].SequenceNumber)
	assert.Equal(t, 10, entries[1].SequenceNumber)
	assert.Equal(t, 20, entries[2].SequenceNumber)
	assert.Equal(t, 30, entries[3].SequenceNumber)
}

func TestSortAclEntriesEmpty(t *testing.T) {
	entries := []AclEntryParsed{}
	SortAclEntries(entries)
	assert.Empty(t, entries)
}
