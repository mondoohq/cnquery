package sshd

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParser(t *testing.T) {
	t.Run("commented out line", func(t *testing.T) {
		text := []rune(" # This line is commented out")
		line, err := ParseLine(text)
		require.NoError(t, err)
		require.Equal(t, SshdLine{}, line)
	})

	t.Run("key arg", func(t *testing.T) {
		text := []rune(" \tkey arg \t\n")
		line, err := ParseLine(text)
		require.NoError(t, err)
		require.Equal(t, SshdLine{key: "key", args: "arg"}, line)
	})

	t.Run("key arg with equal sign", func(t *testing.T) {
		text := []rune(" \tkey=arg \t\n")
		line, err := ParseLine(text)
		require.NoError(t, err)
		require.Equal(t, SshdLine{key: "key", args: "arg"}, line)
	})

	t.Run("key quoted arg with equal sign", func(t *testing.T) {
		// Need to unescape string
		t.Skip()

		text := []rune(" \tkey=\" \\\"this is an arg \"\t arg1\n")
		line, err := ParseLine(text)
		require.NoError(t, err)
		require.Equal(t, SshdLine{key: "key", args: "\" this is an arg \" arg1"}, line)
	})

	t.Run("fancy quoting requiring escapes", func(t *testing.T) {
		text := []rune(" \tkey=\" \\\"this is an arg \"\t\n")
		line, err := ParseLine(text)
		require.NoError(t, err)
		require.Equal(t, SshdLine{key: "key", args: "\" \\\"this is an arg \""}, line)
	})

	t.Run("key arg with equal sign with varying spaces", func(t *testing.T) {
		text := []rune(" \tkey \t=\t  arg \t\n")
		line, err := ParseLine(text)
		require.NoError(t, err)
		require.Equal(t, SshdLine{key: "key", args: "arg"}, line)
	})

	t.Run("multiple args with varying spaces", func(t *testing.T) {
		text := []rune(" \tkey arg0\targ1   arg2 ")
		line, err := ParseLine(text)
		require.NoError(t, err)
		require.Equal(t, SshdLine{key: "key", args: "arg0 arg1 arg2"}, line)
	})

	t.Run("key with equal and multiple args with varying spaces", func(t *testing.T) {
		text := []rune(" \tkey= arg0\targ1   arg2 ")
		line, err := ParseLine(text)
		require.NoError(t, err)
		require.Equal(t, SshdLine{key: "key", args: "arg0 arg1 arg2"}, line)
	})

	t.Run("inline comment", func(t *testing.T) {
		text := []rune("key arg1 # arg2 ")
		line, err := ParseLine(text)
		require.NoError(t, err)
		require.Equal(t, SshdLine{key: "key", args: "arg1"}, line)
	})

	t.Run("arg with unit minute", func(t *testing.T) {
		text := []rune("LoginGraceTime=1m")
		line, err := ParseLine(text)
		require.NoError(t, err)
		require.Equal(t, SshdLine{key: "LoginGraceTime", args: "60"}, line)
	})

	t.Run("arg with unit second", func(t *testing.T) {
		text := []rune("LoginGraceTime=42s")
		line, err := ParseLine(text)
		require.NoError(t, err)
		require.Equal(t, SshdLine{key: "LoginGraceTime", args: "42"}, line)
	})

	t.Run("arg with unit hour", func(t *testing.T) {
		text := []rune("LoginGraceTime=2h")
		line, err := ParseLine(text)
		require.NoError(t, err)
		require.Equal(t, SshdLine{key: "LoginGraceTime", args: "7200"}, line)
	})

	t.Run("arg with complex duration", func(t *testing.T) {
		text := []rune("LoginGraceTime=2h10m42s")
		line, err := ParseLine(text)
		require.NoError(t, err)
		require.Equal(t, SshdLine{key: "LoginGraceTime", args: "7842"}, line)
	})

	t.Run("arg with complex duration, but non-matching key", func(t *testing.T) {
		text := []rune("key=2h10m42s")
		line, err := ParseLine(text)
		require.NoError(t, err)
		require.Equal(t, SshdLine{key: "key", args: "2h10m42s"}, line)
	})
}
