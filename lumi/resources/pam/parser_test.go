package pam

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/stretchr/testify/require"
)

func TestParseLine(t *testing.T) {
	t.Run("parsing conf lines", func(t *testing.T) {
		line := "account    required       pam_opendirectory.so"
		expected := &PamLine{
			PamType: "account",
			Control: "required",
			Module:  "pam_opendirectory.so",
			Options: []interface{}{},
		}
		result, err := ParseLine(line)
		require.NoError(t, err)
		require.Equal(t, expected, result)
	})

	t.Run("parsing conf lines with options", func(t *testing.T) {
		line := "account    required       pam_opendirectory.so no_warn group=admin,wheel"
		expected := &PamLine{
			PamType: "account",
			Control: "required",
			Module:  "pam_opendirectory.so",
			Options: []interface{}{"no_warn", "group=admin,wheel"},
		}
		result, err := ParseLine(line)
		require.NoError(t, err)
		require.Equal(t, expected, result)
	})

	t.Run("parsing conf lines with complicated control", func(t *testing.T) {
		line := "account     [default=bad success=ok user_unknown=ignore] pam_sss.so"
		expected := &PamLine{
			PamType: "account",
			Control: "[default=bad success=ok user_unknown=ignore]",
			Module:  "pam_sss.so",
			Options: []interface{}{},
		}
		result, err := ParseLine(line)
		require.NoError(t, err)
		require.Equal(t, expected, result)
	})

	t.Run("parsing conf lines with complicated control and options", func(t *testing.T) {
		line := "account    [default=bad success=ok user_unknown=ignore]       pam_opendirectory.so no_warn group=admin,wheel"
		expected := &PamLine{
			PamType: "account",
			Control: "[default=bad success=ok user_unknown=ignore]",
			Module:  "pam_opendirectory.so",
			Options: []interface{}{"no_warn", "group=admin,wheel"},
		}
		result, err := ParseLine(line)
		require.NoError(t, err)
		require.Equal(t, expected, result)
	})

	t.Run("parsing conf lines with complicated control and options", func(t *testing.T) {
		line := "account    [default=bad]       pam_opendirectory.so no_warn group=admin,wheel"
		expected := &PamLine{
			PamType: "account",
			Control: "[default=bad]",
			Module:  "pam_opendirectory.so",
			Options: []interface{}{"no_warn", "group=admin,wheel"},
		}
		result, err := ParseLine(line)
		require.NoError(t, err)
		require.Equal(t, expected, result)
	})

	t.Run("parsing conf line with include", func(t *testing.T) {
		line := "@include common-password"
		expected := &PamLine{
			PamType: "@include",
			Control: "common-password",
			Module:  "",
			Options: []interface{}{},
		}
		result, err := ParseLine(line)
		require.NoError(t, err)
		require.Equal(t, expected, result)
	})
}

func parsePamContent(content string) ([]*PamLine, error) {
	entries := []*PamLine{}
	lines := strings.Split(content, "\n")
	for i := range lines {
		line := lines[i]
		entry, err := ParseLine(line)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

func TestSimplePamConfigurationFile(t *testing.T) {
	data, err := os.ReadFile("./testdata/simple")
	require.NoError(t, err)
	content := string(data)

	expected := []*PamLine{
		nil,
		{
			PamType: "auth",
			Control: "required",
			Module:  "pam_securetty.so",
			Options: []interface{}{},
		},
		{
			PamType: "auth",
			Control: "required",
			Module:  "pam_unix.so",
			Options: []interface{}{
				"nullok",
			},
		},
		{
			PamType: "auth",
			Control: "required",
			Module:  "pam_nologin.so",
			Options: []interface{}{},
		},
		{
			PamType: "account",
			Control: "required",
			Module:  "pam_unix.so",
			Options: []interface{}{},
		},
		{
			PamType: "password",
			Control: "required",
			Module:  "pam_cracklib.so",
			Options: []interface{}{
				"retry=3",
			},
		},
		{
			PamType: "password",
			Control: "required",
			Module:  "pam_unix.so",
			Options: []interface{}{
				"shadow",
				"nullok",
				"use_authtok",
			},
		},
		{
			PamType: "session",
			Control: "required",
			Module:  "pam_unix.so",
			Options: []interface{}{},
		},
	}
	entries, err := parsePamContent(content)
	require.NoError(t, err)
	assert.Equal(t, expected, entries)
}

func TestRebootPamConfigurationFile(t *testing.T) {
	data, err := os.ReadFile("./testdata/reboot")
	require.NoError(t, err)
	content := string(data)

	expected := []*PamLine{
		nil,
		{
			PamType: "auth",
			Control: "sufficient",
			Module:  "pam_rootok.so",
			Options: []interface{}{},
		},
		{
			PamType: "auth",
			Control: "required",
			Module:  "pam_console.so",
			Options: []interface{}{},
		},
		nil,
		{
			PamType: "account",
			Control: "required",
			Module:  "pam_permit.so",
			Options: []interface{}{},
		},
	}
	entries, err := parsePamContent(content)
	require.NoError(t, err)
	assert.Equal(t, expected, entries)
}

func TestCommonSessionNonInteractivePamConfigurationFile(t *testing.T) {
	data, err := os.ReadFile("./testdata/common-session-noninteractive")
	require.NoError(t, err)
	content := string(data)

	expected := []*PamLine{
		{
			PamType: "session",
			Control: "[default=1]",
			Module:  "pam_permit.so",
			Options: []interface{}{},
		},
		{
			PamType: "session",
			Control: "requisite",
			Module:  "pam_deny.so",
			Options: []interface{}{},
		},
		{
			PamType: "session",
			Control: "required",
			Module:  "pam_permit.so",
			Options: []interface{}{},
		},
		{
			PamType: "session",
			Control: "optional",
			Module:  "pam_umask.so",
			Options: []interface{}{},
		},
		{
			PamType: "session",
			Control: "required",
			Module:  "pam_unix.so",
			Options: []interface{}{},
		},
		{
			PamType: "session",
			Control: "optional",
			Module:  "pam_ecryptfs.so",
			Options: []interface{}{
				"unwrap",
			},
		},
	}
	entries, err := parsePamContent(content)
	require.NoError(t, err)
	assert.Equal(t, expected, entries)
}

func TestIncludePamConfigurationFile(t *testing.T) {
	data, err := os.ReadFile("./testdata/atd")
	require.NoError(t, err)
	content := string(data)

	expected := []*PamLine{
		{
			PamType: "auth",
			Control: "required",
			Module:  "pam_env.so",
			Options: []interface{}{},
		},
		{
			PamType: "@include",
			Control: "common-auth",
			Module:  "",
			Options: []interface{}{},
		},
		{
			PamType: "@include",
			Control: "common-account",
			Module:  "",
			Options: []interface{}{},
		},
		{
			PamType: "@include",
			Control: "common-session-noninteractive",
			Module:  "",
			Options: []interface{}{},
		},
		{
			PamType: "session",
			Control: "required",
			Module:  "pam_limits.so",
			Options: []interface{}{},
		},
	}
	entries, err := parsePamContent(content)
	require.NoError(t, err)
	assert.Equal(t, expected, entries)
}
