package core_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/resources/packs/core"
	"go.mondoo.com/cnquery/resources/packs/testutils"
)

const passwdContent = `root:x:0:0::/root:/bin/bash
chris:x:1000:1001::/home/chris:/bin/bash
christopher:x:1000:1001::/home/christopher:/bin/bash
chris:x:1002:1003::/home/chris:/bin/bash
bin:x:1:1::/:/usr/bin/nologin
`

func TestResource_File(t *testing.T) {
	x.TestSimple(t, []testutils.SimpleTest{
		{
			"file(\"/etc/passwd\").exists",
			0, true,
		},
		{
			"file(\"/etc/passwd\").basename",
			0, "passwd",
		},
		{
			"file(\"/etc/passwd\").dirname",
			0, "/etc",
		},
		{
			"file(\"/etc/passwd\").size",
			0, int64(193),
		},
		{
			"file(\"/etc/passwd\").permissions.mode",
			0, int64(420),
		},
		{
			"file(\"/etc/passwd\").content",
			0, passwdContent,
		},
	})
}

func TestResource_File_NotExist(t *testing.T) {
	res := x.TestQuery(t, "file('Nope').content")
	assert.ErrorIs(t, res[0].Data.Error, resources.NotFound)
}

func TestResource_File_Permissions(t *testing.T) {
	runtime := resources.NewRuntime(core.Registry, testutils.LinuxMock())

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

	for _, tc := range testCases {
		if !tc.focus {
			continue
		}
		permRaw, err := runtime.CreateResource("file.permissions",
			"mode", int64(tc.mode),
			"user_readable", tc.userReadable,
			"user_writeable", tc.userWriteable,
			"user_executable", tc.userExecutable,
			"group_readable", tc.groupReadable,
			"group_writeable", tc.groupWriteable,
			"group_executable", tc.groupExecutable,
			"other_readable", tc.otherReadable,
			"other_writeable", tc.otherWriteable,
			"other_executable", tc.otherExecutable,
			"suid", tc.suid,
			"sgid", tc.sgid,
			"sticky", tc.sticky,
			"isDirectory", tc.isDir,
			"isFile", tc.isFile,
			"isSymlink", tc.isSymlink,
		)
		require.NoError(t, err)
		require.Equal(t, tc.expectedID, permRaw.MqlResource().Id)
	}
}
