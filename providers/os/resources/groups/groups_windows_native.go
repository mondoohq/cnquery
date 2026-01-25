// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

//go:build windows

package groups

import (
	"unsafe"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v12/providers/os/connection/shared"
	"golang.org/x/sys/windows"
)

var (
	netapi32              = windows.NewLazySystemDLL("netapi32.dll")
	procNetLocalGroupEnum = netapi32.NewProc("NetLocalGroupEnum")
	procNetApiBufferFree  = netapi32.NewProc("NetApiBufferFree")
)

// LOCALGROUP_INFO_1 structure
// https://learn.microsoft.com/en-us/windows/win32/api/lmaccess/ns-lmaccess-localgroup_info_1
type localGroupInfo1 struct {
	Name    *uint16
	Comment *uint16
}

const (
	NERR_Success       = 0
	ERROR_MORE_DATA    = 234
	MAX_PREFERRED_SIZE = 0xFFFFFFFF
)

// List returns local groups on Windows.
// Uses native Windows API when running locally, falls back to PowerShell for remote connections.
func (s *WindowsGroupManager) List() ([]*Group, error) {
	// Check if we're running locally on Windows - use native API for performance
	if s.conn.Type() == shared.Type_Local {
		groups, err := s.listNative()
		if err == nil {
			return groups, nil
		}
		log.Debug().Err(err).Msg("native Windows groups API failed, falling back to PowerShell")
	}

	// Fallback to PowerShell for remote connections or if native API fails
	return s.listViaPowershell()
}

// listNative uses the Windows NetLocalGroupEnum API to enumerate local groups
// https://learn.microsoft.com/en-us/windows/win32/api/lmaccess/nf-lmaccess-netlocalgropenum
func (s *WindowsGroupManager) listNative() ([]*Group, error) {
	log.Debug().Msg("enumerating local groups using native Windows API")

	var (
		buf          uintptr
		entriesRead  uint32
		totalEntries uint32
		resumeHandle uint32
	)

	// Call NetLocalGroupEnum with level 1 to get name and comment
	// Level 1 returns LOCALGROUP_INFO_1 structures
	ret, _, _ := procNetLocalGroupEnum.Call(
		0, // servername (NULL = local)
		1, // level (1 = LOCALGROUP_INFO_1)
		uintptr(unsafe.Pointer(&buf)),
		MAX_PREFERRED_SIZE,
		uintptr(unsafe.Pointer(&entriesRead)),
		uintptr(unsafe.Pointer(&totalEntries)),
		uintptr(unsafe.Pointer(&resumeHandle)),
	)

	if buf != 0 {
		defer procNetApiBufferFree.Call(buf)
	}

	if ret != NERR_Success && ret != ERROR_MORE_DATA {
		return nil, windows.Errno(ret)
	}

	groups := make([]*Group, 0, entriesRead)

	// Parse the returned buffer
	entrySize := unsafe.Sizeof(localGroupInfo1{})
	for i := uint32(0); i < entriesRead; i++ {
		entry := (*localGroupInfo1)(unsafe.Pointer(buf + uintptr(i)*entrySize))

		name := ""
		if entry.Name != nil {
			name = windows.UTF16PtrToString(entry.Name)
		}

		// Get the group SID using LookupAccountName
		sid, err := lookupGroupSID(name)
		if err != nil {
			log.Debug().Str("group", name).Err(err).Msg("failed to lookup SID for group")
			// Continue without SID - better to have partial data than fail completely
			sid = ""
		}

		groups = append(groups, &Group{
			ID:      sid,
			Sid:     sid,
			Gid:     -1, // Windows doesn't use numeric GIDs in the same way as Unix
			Name:    name,
			Members: []string{},
		})
	}

	return groups, nil
}

// lookupGroupSID looks up the SID for a local group name
func lookupGroupSID(groupName string) (string, error) {
	sid, _, _, err := windows.LookupSID("", groupName)
	if err != nil {
		return "", err
	}
	return sid.String(), nil
}
