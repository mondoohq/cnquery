// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseGrubDefaults(t *testing.T) {
	input := `# If you change this file, run 'update-grub' afterwards to update
# /boot/grub/grub.cfg.
GRUB_DEFAULT=0
GRUB_TIMEOUT_STYLE=hidden
GRUB_TIMEOUT=5
GRUB_DISTRIBUTOR="$(lsb_release -i -s 2> /dev/null) || echo Debian"
GRUB_CMDLINE_LINUX_DEFAULT="quiet splash"
GRUB_CMDLINE_LINUX="audit=1 apparmor=1 security=apparmor"
GRUB_DISABLE_RECOVERY="true"

# Uncomment to enable BadRAM filtering
#GRUB_BADRAM="0x01234567,0xfefefefe,0x89abcdef,0xefefefef"
`
	params, err := ParseGrubDefaults(strings.NewReader(input))
	require.NoError(t, err)

	assert.Equal(t, "0", params["GRUB_DEFAULT"])
	assert.Equal(t, "hidden", params["GRUB_TIMEOUT_STYLE"])
	assert.Equal(t, "5", params["GRUB_TIMEOUT"])
	assert.Equal(t, "quiet splash", params["GRUB_CMDLINE_LINUX_DEFAULT"])
	assert.Equal(t, "audit=1 apparmor=1 security=apparmor", params["GRUB_CMDLINE_LINUX"])
	assert.Equal(t, "true", params["GRUB_DISABLE_RECOVERY"])
	// Unquoted shell expression
	assert.Equal(t, "$(lsb_release -i -s 2> /dev/null) || echo Debian", params["GRUB_DISTRIBUTOR"])
	// Commented lines should not appear
	_, ok := params["GRUB_BADRAM"]
	assert.False(t, ok)
}

func TestParseGrubDefaultsSingleQuotes(t *testing.T) {
	input := `GRUB_CMDLINE_LINUX='audit=1'
GRUB_TIMEOUT=10
`
	params, err := ParseGrubDefaults(strings.NewReader(input))
	require.NoError(t, err)

	assert.Equal(t, "audit=1", params["GRUB_CMDLINE_LINUX"])
	assert.Equal(t, "10", params["GRUB_TIMEOUT"])
}

func TestParseGrubCfgEntries(t *testing.T) {
	input := `#!/bin/sh
exec tail -n +3 $0
# This file provides an easy way to add custom menu entries.
set default=0
set timeout=5

menuentry 'Ubuntu' --class ubuntu {
	load_video
	set gfxpayload=keep
	insmod gzio
	linux /vmlinuz-5.4.0-42-generic root=/dev/mapper/ubuntu--vg-ubuntu--lv ro quiet splash
	initrd /initrd.img-5.4.0-42-generic
}

menuentry 'Ubuntu, with Linux 5.4.0-40-generic' --class ubuntu {
	linux /vmlinuz-5.4.0-40-generic root=/dev/mapper/ubuntu--vg-ubuntu--lv ro quiet splash
	initrd /initrd.img-5.4.0-40-generic
}

submenu 'Advanced options for Ubuntu' --class ubuntu {
	menuentry 'Ubuntu, with Linux 5.4.0-42-generic (recovery mode)' {
		linux /vmlinuz-5.4.0-42-generic root=/dev/mapper/ubuntu--vg-ubuntu--lv ro recovery
		initrd /initrd.img-5.4.0-42-generic
	}
}
`
	entries, err := ParseGrubCfgEntries(strings.NewReader(input))
	require.NoError(t, err)
	require.Len(t, entries, 4)

	assert.Equal(t, "Ubuntu", entries[0].Title)
	assert.Equal(t, "/vmlinuz-5.4.0-42-generic root=/dev/mapper/ubuntu--vg-ubuntu--lv ro quiet splash", entries[0].Cmdline)
	assert.Equal(t, "/initrd.img-5.4.0-42-generic", entries[0].Initrd)
	assert.False(t, entries[0].IsSubmenu)

	assert.Equal(t, "Ubuntu, with Linux 5.4.0-40-generic", entries[1].Title)
	assert.False(t, entries[1].IsSubmenu)

	assert.Equal(t, "Advanced options for Ubuntu", entries[2].Title)
	assert.True(t, entries[2].IsSubmenu)

	// Nested entry inside submenu
	assert.Equal(t, "Ubuntu, with Linux 5.4.0-42-generic (recovery mode)", entries[3].Title)
	assert.Contains(t, entries[3].Cmdline, "recovery")
	assert.False(t, entries[3].IsSubmenu)
}

func TestParseGrubCfgEntriesLinux16(t *testing.T) {
	input := `menuentry 'CentOS Linux (3.10.0-1160.el7.x86_64)' {
	linux16 /vmlinuz-3.10.0-1160.el7.x86_64 root=/dev/mapper/centos-root ro crashkernel=auto
	initrd16 /initramfs-3.10.0-1160.el7.x86_64.img
}
`
	entries, err := ParseGrubCfgEntries(strings.NewReader(input))
	require.NoError(t, err)
	require.Len(t, entries, 1)

	assert.Equal(t, "CentOS Linux (3.10.0-1160.el7.x86_64)", entries[0].Title)
	assert.Contains(t, entries[0].Cmdline, "vmlinuz-3.10.0")
	assert.Contains(t, entries[0].Initrd, "initramfs-3.10.0")
}

func TestParseGrubCfgEntriesLinuxefi(t *testing.T) {
	input := `menuentry 'Fedora (6.2.0-20.fc38.x86_64)' {
	linuxefi /vmlinuz-6.2.0-20.fc38.x86_64 root=UUID=abc-123 ro rhgb quiet
	initrdefi /initramfs-6.2.0-20.fc38.x86_64.img
}
`
	entries, err := ParseGrubCfgEntries(strings.NewReader(input))
	require.NoError(t, err)
	require.Len(t, entries, 1)

	assert.Equal(t, "Fedora (6.2.0-20.fc38.x86_64)", entries[0].Title)
	assert.Contains(t, entries[0].Cmdline, "vmlinuz-6.2.0")
	assert.Contains(t, entries[0].Initrd, "initramfs-6.2.0")
}

func TestParseGrubCfgEntriesBraceOnNextLine(t *testing.T) {
	input := `menuentry 'Test Entry'
{
	linux /vmlinuz root=/dev/sda1 ro
	initrd /initrd.img
}
`
	entries, err := ParseGrubCfgEntries(strings.NewReader(input))
	require.NoError(t, err)
	require.Len(t, entries, 1)

	assert.Equal(t, "Test Entry", entries[0].Title)
	assert.Contains(t, entries[0].Cmdline, "vmlinuz")
	assert.Contains(t, entries[0].Initrd, "initrd.img")
}

func TestParseGrubCfgEntriesDuplicateTitles(t *testing.T) {
	input := `menuentry 'Ubuntu' {
	linux /vmlinuz-5.4 root=/dev/sda1 ro
	initrd /initrd.img-5.4
}
menuentry 'Ubuntu' {
	linux /vmlinuz-5.3 root=/dev/sda1 ro
	initrd /initrd.img-5.3
}
`
	entries, err := ParseGrubCfgEntries(strings.NewReader(input))
	require.NoError(t, err)
	require.Len(t, entries, 2)

	assert.Equal(t, "Ubuntu", entries[0].Title)
	assert.Contains(t, entries[0].Cmdline, "vmlinuz-5.4")
	assert.Equal(t, "Ubuntu", entries[1].Title)
	assert.Contains(t, entries[1].Cmdline, "vmlinuz-5.3")
}

func TestParseGrubCfgEntriesCommentWithBraces(t *testing.T) {
	input := `menuentry 'Ubuntu' {
	# echo "Use {grub} menu"
	linux /vmlinuz root=/dev/sda1 ro
	initrd /initrd.img
}
`
	entries, err := ParseGrubCfgEntries(strings.NewReader(input))
	require.NoError(t, err)
	require.Len(t, entries, 1)

	assert.Equal(t, "Ubuntu", entries[0].Title)
	assert.Contains(t, entries[0].Cmdline, "vmlinuz")
}

func TestParseGrubPasswordProtected(t *testing.T) {
	t.Run("protected with pbkdf2", func(t *testing.T) {
		input := `set superusers="root"
password_pbkdf2 root grub.pbkdf2.sha512.10000.ABC123
`
		assert.True(t, ParseGrubPasswordProtected([]byte(input)))
	})

	t.Run("protected with plaintext password", func(t *testing.T) {
		input := `set superusers="root"
password root secret
`
		assert.True(t, ParseGrubPasswordProtected([]byte(input)))
	})

	t.Run("no superusers", func(t *testing.T) {
		input := `password_pbkdf2 root grub.pbkdf2.sha512.10000.ABC123
`
		assert.False(t, ParseGrubPasswordProtected([]byte(input)))
	})

	t.Run("no password", func(t *testing.T) {
		input := `set superusers="root"
`
		assert.False(t, ParseGrubPasswordProtected([]byte(input)))
	})

	t.Run("empty config", func(t *testing.T) {
		assert.False(t, ParseGrubPasswordProtected([]byte("")))
	})
}

func TestParseGrubPasswordProtectedCommentedOut(t *testing.T) {
	t.Run("commented superusers and password", func(t *testing.T) {
		input := `# set superusers="root"
# password_pbkdf2 root grub.pbkdf2.sha512.10000.ABC123
`
		assert.False(t, ParseGrubPasswordProtected([]byte(input)))
	})

	t.Run("commented superusers only", func(t *testing.T) {
		input := `# set superusers="root"
password_pbkdf2 root grub.pbkdf2.sha512.10000.ABC123
`
		assert.False(t, ParseGrubPasswordProtected([]byte(input)))
	})

	t.Run("commented password only", func(t *testing.T) {
		input := `set superusers="root"
# password_pbkdf2 root grub.pbkdf2.sha512.10000.ABC123
`
		assert.False(t, ParseGrubPasswordProtected([]byte(input)))
	})
}

func TestParseGrubCfgEntriesShellVariableExpansion(t *testing.T) {
	input := `menuentry 'Ubuntu' {
	set root='hd0,msdos1'
	linux /vmlinuz root=${rootdev} ro quiet
	echo "Loading ${kernel_name}"
	initrd /initrd.img
}
menuentry 'Recovery' {
	linux /vmlinuz root=/dev/sda1 ro single
	initrd /initrd-recovery.img
}
`
	entries, err := ParseGrubCfgEntries(strings.NewReader(input))
	require.NoError(t, err)
	require.Len(t, entries, 2)

	assert.Equal(t, "Ubuntu", entries[0].Title)
	assert.Contains(t, entries[0].Cmdline, "${rootdev}")
	assert.Equal(t, "/initrd.img", entries[0].Initrd)

	assert.Equal(t, "Recovery", entries[1].Title)
	assert.Contains(t, entries[1].Cmdline, "single")
}

func TestStripQuotes(t *testing.T) {
	assert.Equal(t, "hello", stripQuotes(`"hello"`))
	assert.Equal(t, "hello", stripQuotes(`'hello'`))
	assert.Equal(t, "hello", stripQuotes("hello"))
	assert.Equal(t, "", stripQuotes(`""`))
	assert.Equal(t, "", stripQuotes(`''`))
	assert.Equal(t, "a", stripQuotes("a"))
}
