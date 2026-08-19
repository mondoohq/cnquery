// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package powershell_test

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/providers/os/connection/shared"
	"go.mondoo.com/mql/providers/os/resources/powershell"
)

// readOnlyFs is the shape both cat.Fs shims have: it advertises a filesystem
// and refuses every mutating call. winrm and `ssh --sudo` both hand one of
// these back, which is the reason the chunked fallback exists at all.
type readOnlyFs struct{ afero.Fs }

func (readOnlyFs) Create(string) (afero.File, error) {
	return nil, errors.New("not implemented")
}

type fakeConn struct {
	connType shared.ConnectionType
	platform *inventory.Platform
	fs       afero.Fs
	caps     shared.Capabilities
	// commands records every RunCommand in order, which is what the chunked
	// write has to be asserted on: the file never exists locally.
	commands []string
	exit     int
}

func (c *fakeConn) ID() uint32                        { return 1 }
func (c *fakeConn) Name() string                      { return "fake" }
func (c *fakeConn) Type() shared.ConnectionType       { return c.connType }
func (c *fakeConn) Capabilities() shared.Capabilities { return c.caps }
func (c *fakeConn) UpdateAsset(a *inventory.Asset)    {}
func (c *fakeConn) ParentID() uint32                  { return 0 }
func (c *fakeConn) SetParentID(uint32)                {}
func (c *fakeConn) Asset() *inventory.Asset {
	return &inventory.Asset{Platform: c.platform}
}
func (c *fakeConn) FileSystem() afero.Fs { return c.fs }
func (c *fakeConn) FileInfo(string) (shared.FileInfoDetails, error) {
	return shared.FileInfoDetails{}, errors.New("not implemented")
}
func (c *fakeConn) RunCommand(cmd string) (*shared.Command, error) {
	c.commands = append(c.commands, cmd)
	return &shared.Command{
		Command: cmd, ExitStatus: c.exit,
		Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{},
	}, nil
}

var _ shared.Connection = (*fakeConn)(nil)
var _ plugin.Connection = (*fakeConn)(nil)

func windowsPlatform() *inventory.Platform {
	return &inventory.Platform{Name: "windows", Family: []string{"windows", "os"}}
}

// The hash is what keeps two staged scripts from colliding under one recording
// key, so it has to depend on the content and on nothing else.
func TestStagedNameIsContentAddressed(t *testing.T) {
	a := powershell.StagedName("iis", "Get-Item A")
	b := powershell.StagedName("iis", "Get-Item B")
	assert.NotEqual(t, a, b, "two different scripts must not stage to one path")
	assert.Equal(t, a, powershell.StagedName("iis", "Get-Item A"),
		"the same script must stage to the same path on every run")
	assert.True(t, strings.HasPrefix(a, "iis-") && strings.HasSuffix(a, ".ps1"), a)
}

// -File is execution-policy sensitive where -EncodedCommand is not, and the
// RemoteSigned default hides that. Losing the Bypass would leave a resource
// that works everywhere except on a hardened host.
func TestStagedCommandBypassesTheExecutionPolicy(t *testing.T) {
	cmd := powershell.StagedCommand(`C:\Windows\Temp\iis-abc.ps1`)
	assert.Contains(t, cmd, "-ExecutionPolicy Bypass")
	assert.Contains(t, cmd, "-File")
	assert.NotContains(t, cmd, "-EncodedCommand")
}

// The staged directory must come from the client's view of the asset, never
// from the target. A `$env:TEMP` round trip would put a host-dependent string
// into the command that keys the recording, and replay would stop matching.
func TestStagedPathIsAClientSideLiteral(t *testing.T) {
	conn := &fakeConn{
		connType: "mock", platform: windowsPlatform(),
		caps: shared.Capability_RunCommand,
	}
	staged, err := powershell.Stage(conn, "iis", "Get-Item C:\\; ConvertTo-Json @{}")
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(staged.Path, `C:\Windows\Temp\`), staged.Path)
	assert.Empty(t, conn.commands, "a mock connection must not be written to")

	// And the same script, on a second connection to a different host, must
	// produce byte-identical commands — that is the replay guarantee.
	other := &fakeConn{
		connType: "mock", platform: windowsPlatform(),
		caps: shared.Capability_RunCommand,
	}
	again, err := powershell.Stage(other, "iis", "Get-Item C:\\; ConvertTo-Json @{}")
	require.NoError(t, err)
	assert.Equal(t, staged.Command, again.Command)
}

func TestStageWritesThroughAWritableFilesystem(t *testing.T) {
	fs := afero.NewMemMapFs()
	conn := &fakeConn{
		connType: shared.Type_SSH, platform: windowsPlatform(),
		fs: fs, caps: shared.Capability_RunCommand | shared.Capability_File,
	}
	script := "$ErrorActionPreference = 'Stop'\nConvertTo-Json @{ a = 1 }\n"
	staged, err := powershell.Stage(conn, "iis", script)
	require.NoError(t, err)

	body, err := afero.ReadFile(fs, staged.Path)
	require.NoError(t, err)
	assert.Equal(t, script, string(body))
	assert.Empty(t, conn.commands, "the filesystem path must not shell out")

	staged.Remove()
	require.Len(t, conn.commands, 1)
	assert.Contains(t, conn.commands[0], "Remove-Item")
	assert.Contains(t, conn.commands[0], staged.Path)
	// The removal is the command most likely to break unnoticed: it runs after
	// the answer has already been produced, and its failure only logs at debug.
	assertStagingCommandShape(t, conn.commands)
}

// winrm and `ssh --sudo`: the filesystem is there and refuses to write, so the
// script has to go over RunCommand in pieces. The assertion that matters is
// that no single command approaches the 8191 ceiling — the whole point is that
// the script is longer than one.
func TestStageFallsBackToChunkedCommandsOnAReadOnlyFilesystem(t *testing.T) {
	conn := &fakeConn{
		connType: shared.Type_Winrm, platform: windowsPlatform(),
		fs:   readOnlyFs{afero.NewMemMapFs()},
		caps: shared.Capability_RunCommand | shared.Capability_File,
	}
	// Comfortably past the cap, and past one chunk.
	script := "$ErrorActionPreference = 'Stop'\n" + strings.Repeat("# padding line\n", 1200)
	require.Greater(t, len(powershell.Encode(script)), powershell.MaxCommandLength,
		"the fixture script has to be over the cap or this proves nothing")

	staged, err := powershell.Stage(conn, "iis", script)
	require.NoError(t, err)
	require.Greater(t, len(conn.commands), 2, "expected a clear, several chunks and a decode")

	for i, cmd := range conn.commands {
		assert.LessOrEqual(t, len(cmd), powershell.MaxCommandLength,
			"staging command %d is %d chars, which is over the very ceiling "+
				"staging exists to avoid", i, len(cmd))
	}
	assert.Contains(t, conn.commands[0], "Remove-Item")
	assert.Contains(t, conn.commands[len(conn.commands)-1], "FromBase64String")
	assert.Contains(t, staged.Command, staged.Path)

	assertStagingCommandShape(t, conn.commands)
}

// assertStagingCommandShape pins the two properties every staging command needs,
// both of which were established by a command that failed on a live host.
func assertStagingCommandShape(t *testing.T, commands []string) {
	t.Helper()
	for i, cmd := range commands {
		// 1. The exit code is computed, not inherited. `powershell -Command`
		// exits 1 whenever $? is false at the end, and $? is false after a
		// *suppressed* error too — so `Remove-Item -ErrorAction SilentlyContinue`
		// on a file that is not there exits 1 with empty stderr. Read at face
		// value that aborts a scan that would have worked.
		assert.Contains(t, cmd, "exit 0", "staging command %d does not set its own exit code", i)
		assert.Contains(t, cmd, "exit 1", "staging command %d has no failure exit", i)

		// 2. No `$`, anywhere. Over SSH to a Windows host whose sshd
		// DefaultShell is PowerShell — the normal configuration — the command
		// line is parsed twice, and the outer pass expands any variable inside
		// the double-quoted argument before the inner powershell.exe sees it.
		// `$ErrorActionPreference='Stop'` becomes `='Stop'`, which is a syntax
		// error, and on the cleanup path that failure is silent: the staged file
		// is simply left on the target after every scan.
		assert.NotContains(t, cmd, "$",
			"staging command %d contains a variable, which an outer PowerShell "+
				"login shell will expand before powershell.exe sees it", i)
	}
}

// A write that fails has to be reported. RunCommand reports a failed *command*
// through ExitStatus and not through err, so a helper that only checks err
// stages nothing and says it succeeded — and the caller then runs a -File that
// is not there.
func TestStageReportsANonZeroExitFromTheWrite(t *testing.T) {
	conn := &fakeConn{
		connType: shared.Type_Winrm, platform: windowsPlatform(),
		fs:   readOnlyFs{afero.NewMemMapFs()},
		caps: shared.Capability_RunCommand | shared.Capability_File,
		exit: 1,
	}
	_, err := powershell.Stage(conn, "iis", "ConvertTo-Json @{ a = 1 }\n"+strings.Repeat("#x\n", 50))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exited 1")
}

func TestCanStage(t *testing.T) {
	assert.False(t, powershell.CanStage(nil))
	assert.True(t, powershell.CanStage(&fakeConn{
		connType: "mock", platform: windowsPlatform(), caps: shared.Capability_None}))
	assert.False(t, powershell.CanStage(&fakeConn{
		connType: shared.Type_ContainerRegistry, platform: windowsPlatform(),
		caps: shared.Capability_File}),
		"Capability_File alone is a read-only shim, which cannot be staged to")
}

// StagedWindowsPath is what a test derives a recording key from, so it has to
// agree with what Stage actually produces on a Windows asset. If they drift, a
// resource test builds a recording under a key the resource never asks for and
// fails with "command not found" — which reads as a broken resource.
func TestStagedWindowsPathMatchesStage(t *testing.T) {
	script := "$ErrorActionPreference = 'Stop'\nConvertTo-Json @{ a = 1 }\n"
	conn := &fakeConn{
		connType: "mock", platform: windowsPlatform(),
		caps: shared.Capability_RunCommand,
	}
	staged, err := powershell.Stage(conn, "iis", script)
	require.NoError(t, err)
	assert.Equal(t, staged.Path, powershell.StagedWindowsPath("iis", script))
	assert.Equal(t, staged.Command,
		powershell.StagedCommand(powershell.StagedWindowsPath("iis", script)))
}
