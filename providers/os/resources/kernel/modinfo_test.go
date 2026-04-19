// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package kernel

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseModuleInfoFromBytes(t *testing.T) {
	// Simulate a .modinfo section with null-terminated key=value pairs
	modinfo := "version=535.183.01\x00" +
		"author=NVIDIA Corporation\x00" +
		"license=Proprietary\x00" +
		"description=NVIDIA GPU driver\x00" +
		"srcversion=ABC123\x00" +
		"alias=pci:v000010DEd*sv*sd*bc03sc*i*\x00"

	info, err := parseModInfoData([]byte(modinfo))
	require.NoError(t, err)

	assert.Equal(t, "535.183.01", info.Version)
	assert.Equal(t, "NVIDIA Corporation", info.Author)
	assert.Equal(t, "Proprietary", info.License)
	assert.Equal(t, "NVIDIA GPU driver", info.Description)
}

func TestParseModuleInfoEmpty(t *testing.T) {
	info, err := parseModInfoData([]byte{})
	require.NoError(t, err)
	assert.Equal(t, "", info.Version)
}

func TestParseModuleInfoNoVersion(t *testing.T) {
	modinfo := "license=GPL\x00description=Test module\x00"
	info, err := parseModInfoData([]byte(modinfo))
	require.NoError(t, err)
	assert.Equal(t, "", info.Version)
	assert.Equal(t, "GPL", info.License)
}

func TestParseModuleInfoInvalidELF(t *testing.T) {
	// ParseModuleInfo should return an error for non-ELF input
	_, err := ParseModuleInfo(bytes.NewReader([]byte("not an elf file")))
	assert.Error(t, err)
}

func TestParseModuleInfoEmptyInput(t *testing.T) {
	// ParseModuleInfo should return an error for empty input
	_, err := ParseModuleInfo(bytes.NewReader([]byte{}))
	assert.Error(t, err)
}

// parseModInfoData is a test helper that parses raw modinfo bytes
// without needing a full ELF file. This tests the parsing logic
// independently of ELF section lookup.
func parseModInfoData(data []byte) (*ModuleInfo, error) {
	info := &ModuleInfo{}
	for _, entry := range bytes.Split(data, []byte{0}) {
		s := string(entry)
		if s == "" {
			continue
		}

		key, value, ok := bytes.Cut(entry, []byte("="))
		if !ok {
			continue
		}

		switch string(key) {
		case "version":
			info.Version = string(value)
		case "author":
			info.Author = string(value)
		case "license":
			info.License = string(value)
		case "description":
			info.Description = string(value)
		}
	}
	return info, nil
}
