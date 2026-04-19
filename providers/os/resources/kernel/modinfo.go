// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package kernel

import (
	"bytes"
	"debug/elf"
	"io"
	"strings"
)

// ModuleInfo represents metadata extracted from a .ko file's .modinfo ELF section.
type ModuleInfo struct {
	Version     string
	Author      string
	License     string
	Description string
}

// ParseModuleInfo reads a .ko kernel module file and extracts metadata
// from its .modinfo ELF section. The section contains null-terminated
// key=value strings.
func ParseModuleInfo(r io.ReaderAt) (*ModuleInfo, error) {
	f, err := elf.NewFile(r)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	section := f.Section(".modinfo")
	if section == nil {
		return &ModuleInfo{}, nil
	}

	data, err := section.Data()
	if err != nil {
		return nil, err
	}

	info := &ModuleInfo{}
	// .modinfo contains null-terminated key=value pairs
	for _, entry := range bytes.Split(data, []byte{0}) {
		s := string(entry)
		if s == "" {
			continue
		}

		key, value, ok := strings.Cut(s, "=")
		if !ok {
			continue
		}

		switch key {
		case "version":
			info.Version = value
		case "author":
			info.Author = value
		case "license":
			info.License = value
		case "description":
			info.Description = value
		}
	}

	return info, nil
}
