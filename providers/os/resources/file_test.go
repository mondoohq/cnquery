// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/v10/llx"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/testutils"
	"go.mondoo.com/cnquery/v10/providers/os/resources"
)

const passwdContent = `root:x:0:0::/root:/bin/bash
bin:x:1:1::/:/usr/bin/nologin
daemon:x:2:2::/:/usr/bin/nologin
mail:x:8:12::/var/spool/mail:/usr/bin/nologin
`

func TestResource_File(t *testing.T) {
	x.TestSimple(t, []testutils.SimpleTest{
		{
			Code:        "file('/etc/passwd').exists",
			ResultIndex: 0, Expectation: true,
		},
		{
			Code:        "file('/etc/passwd').basename",
			ResultIndex: 0, Expectation: "passwd",
		},
		{
			Code:        "file('/etc/passwd').dirname",
			ResultIndex: 0, Expectation: "/etc",
		},
		{
			Code:        "file('/etc/passwd').size",
			ResultIndex: 0, Expectation: int64(len(passwdContent)),
		},
		{
			Code:        "file('/etc/passwd').permissions.mode",
			ResultIndex: 0, Expectation: int64(420),
		},
		{
			Code:        "file('/etc/passwd').content",
			ResultIndex: 0, Expectation: passwdContent,
		},
	})
}

func TestResource_File_NotExist(t *testing.T) {
	res := x.TestQuery(t, "file('Nope').content")
	assert.EqualError(t, res[0].Data.Error, "file 'Nope' not found")
}

func TestResource_File_Permissions(t *testing.T) {
	testCases := []struct {
		mode            int64
		userReadable    bool
		userWriteable   bool
		userExecutable  bool
		groupReadable   bool
		groupWriteable  bool
		groupExecutable bool
		otherReadable   bool
		otherWriteable  bool
		otherExecutable bool
		suid            bool
		sgid            bool
		sticky          bool
		isDir           bool
		isFile          bool
		isSymlink       bool

		focus      bool
		expectedID string
	}{
		{
			mode:            0o755,
			userReadable:    true,
			userWriteable:   true,
			userExecutable:  true,
			groupReadable:   true,
			groupExecutable: true,
			otherReadable:   true,
			otherExecutable: true,
			isFile:          true,

			expectedID: "-rwxr-xr-x",
		},
		{
			mode:            0o755,
			userReadable:    true,
			userWriteable:   true,
			userExecutable:  true,
			groupReadable:   true,
			groupExecutable: true,
			otherReadable:   true,
			otherExecutable: true,
			isFile:          true,
			suid:            true,

			expectedID: "-rwsr-xr-x",
		},
		{
			mode:            0o655,
			userReadable:    true,
			userWriteable:   true,
			userExecutable:  false,
			groupReadable:   true,
			groupExecutable: true,
			otherReadable:   true,
			otherExecutable: true,
			isFile:          true,
			suid:            true,

			expectedID: "-rwSr-xr-x",
		},
		{
			mode:            0o755,
			userReadable:    true,
			userWriteable:   true,
			userExecutable:  true,
			groupReadable:   true,
			groupExecutable: true,
			otherReadable:   true,
			otherExecutable: true,
			isDir:           true,

			expectedID: "drwxr-xr-x",
		},
		{
			mode:            0o755,
			userReadable:    true,
			userWriteable:   true,
			userExecutable:  true,
			groupReadable:   true,
			groupExecutable: true,
			otherReadable:   true,
			otherExecutable: true,
			isDir:           true,
			sticky:          true,

			expectedID: "drwxr-xr-t",
		},
		{
			mode:            0o754,
			userReadable:    true,
			userWriteable:   true,
			userExecutable:  true,
			groupReadable:   true,
			groupExecutable: true,
			otherReadable:   true,
			otherExecutable: false,
			isDir:           true,
			sticky:          true,

			expectedID: "drwxr-xr-T",
		},
		{
			mode:            0o755,
			userReadable:    true,
			userWriteable:   true,
			userExecutable:  true,
			groupReadable:   true,
			groupExecutable: true,
			otherReadable:   true,
			otherExecutable: true,
			isFile:          true,
			sgid:            true,
			focus:           true,
			expectedID:      "-rwxr-sr-x",
		},
		{
			mode:            0o754,
			userReadable:    true,
			userWriteable:   true,
			userExecutable:  true,
			groupReadable:   true,
			groupExecutable: false,
			otherReadable:   true,
			otherExecutable: true,
			isFile:          true,
			sgid:            true,

			expectedID: "-rwxr-Sr-x",
		},
		{
			mode:            0o755,
			userReadable:    true,
			userWriteable:   true,
			userExecutable:  true,
			groupReadable:   true,
			groupExecutable: true,
			otherReadable:   true,
			otherExecutable: true,
			isSymlink:       true,

			expectedID: "lrwxr-xr-x",
		},
	}

	runtime := &plugin.Runtime{}

	for _, tc := range testCases {
		if !tc.focus {
			continue
		}

		permRaw, err := resources.CreateResource(
			runtime,
			"file.permissions",
			map[string]*llx.RawData{
				"mode":             llx.IntData(int64(tc.mode)),
				"user_readable":    llx.BoolData(tc.userReadable),
				"user_writeable":   llx.BoolData(tc.userWriteable),
				"user_executable":  llx.BoolData(tc.userExecutable),
				"group_readable":   llx.BoolData(tc.groupReadable),
				"group_writeable":  llx.BoolData(tc.groupWriteable),
				"group_executable": llx.BoolData(tc.groupExecutable),
				"other_readable":   llx.BoolData(tc.otherReadable),
				"other_writeable":  llx.BoolData(tc.otherWriteable),
				"other_executable": llx.BoolData(tc.otherExecutable),
				"suid":             llx.BoolData(tc.suid),
				"sgid":             llx.BoolData(tc.sgid),
				"sticky":           llx.BoolData(tc.sticky),
				"isDirectory":      llx.BoolData(tc.isDir),
				"isFile":           llx.BoolData(tc.isFile),
				"isSymlink":        llx.BoolData(tc.isSymlink),
			},
		)
		require.NoError(t, err)
		require.Equal(t, tc.expectedID, permRaw.MqlID())
	}
}
