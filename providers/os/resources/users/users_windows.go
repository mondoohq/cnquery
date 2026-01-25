// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

//go:build windows
// +build windows

package users

import (
	"syscall"
	"unsafe"

	"github.com/rs/zerolog/log"
	"golang.org/x/sys/windows"
)

var (
	modNetapi32          = windows.NewLazySystemDLL("netapi32.dll")
	procNetUserEnum      = modNetapi32.NewProc("NetUserEnum")
	procNetApiBufferFree = modNetapi32.NewProc("NetApiBufferFree")
)

const (
	// Filter for NetUserEnum - FILTER_NORMAL_ACCOUNT
	FILTER_NORMAL_ACCOUNT = 0x0002

	// Info levels for NetUserEnum
	USER_INFO_LEVEL_1 = 1

	// User flags
	UF_ACCOUNTDISABLE = 0x0002
)

// USER_INFO_1 structure
// https://learn.microsoft.com/en-us/windows/win32/api/lmaccess/ns-lmaccess-user_info_1
type userInfo1 struct {
	Name        *uint16
	Password    *uint16
	PasswordAge uint32
	Priv        uint32
	HomeDir     *uint16
	Comment     *uint16
	Flags       uint32
	ScriptPath  *uint16
}

// GetNativeUsers retrieves local users using the Windows NetUserEnum API.
// This is significantly faster than PowerShell (~1-10ms vs ~200-500ms).
func GetNativeUsers() ([]*User, error) {
	log.Debug().Msg("enumerating local users using native Windows API")

	var (
		buf         *byte
		entriesRead uint32
		totalUsers  uint32
		resumeH     uint32
	)

	users := []*User{}

	for {
		// NetUserEnum with level 1 to get account flags (enabled/disabled status)
		ret, _, _ := procNetUserEnum.Call(
			0,                                // servername (NULL for local)
			USER_INFO_LEVEL_1,                // info level
			FILTER_NORMAL_ACCOUNT,            // filter
			uintptr(unsafe.Pointer(&buf)),    // buffer pointer
			0xFFFFFFFF,                        // preferred max length (MAX_PREFERRED_LENGTH)
			uintptr(unsafe.Pointer(&entriesRead)),
			uintptr(unsafe.Pointer(&totalUsers)),
			uintptr(unsafe.Pointer(&resumeH)),
		)

		if ret != 0 && ret != 234 { // 0 = NERR_Success, 234 = ERROR_MORE_DATA
			if buf != nil {
				procNetApiBufferFree.Call(uintptr(unsafe.Pointer(buf)))
			}
			return nil, syscall.Errno(ret)
		}

		if buf != nil && entriesRead > 0 {
			// Calculate the size of userInfo1 structure
			infoSize := unsafe.Sizeof(userInfo1{})

			for i := uint32(0); i < entriesRead; i++ {
				// Get pointer to current USER_INFO_1 structure
				info := (*userInfo1)(unsafe.Pointer(uintptr(unsafe.Pointer(buf)) + uintptr(i)*infoSize))

				name := windows.UTF16PtrToString(info.Name)
				homeDir := ""
				if info.HomeDir != nil {
					homeDir = windows.UTF16PtrToString(info.HomeDir)
				}

				// Get SID for the user
				sid := getUserSID(name)

				// Check if account is enabled (not disabled)
				enabled := (info.Flags & UF_ACCOUNTDISABLE) == 0

				user := &User{
					ID:      sid,
					Sid:     sid,
					Uid:     -1, // Windows doesn't have Unix UIDs
					Gid:     -1, // Windows doesn't have Unix GIDs
					Name:    name,
					Home:    homeDir,
					Enabled: enabled,
				}
				users = append(users, user)
			}

			procNetApiBufferFree.Call(uintptr(unsafe.Pointer(buf)))
			buf = nil
		}

		// If no more data, exit the loop
		if ret != 234 {
			break
		}
	}

	log.Debug().Int("count", len(users)).Msg("native Windows API enumerated users")
	return users, nil
}

// getUserSID retrieves the SID string for a given username
func getUserSID(username string) string {
	// First, look up the account to get the SID
	sid, _, _, err := syscall.LookupSID("", username)
	if err != nil {
		log.Debug().Str("username", username).Err(err).Msg("failed to lookup SID for user")
		return ""
	}

	sidStr, err := sid.String()
	if err != nil {
		log.Debug().Str("username", username).Err(err).Msg("failed to convert SID to string")
		return ""
	}

	return sidStr
}
