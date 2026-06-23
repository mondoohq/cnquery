// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package detector

import (
	"debug/elf"
	"encoding/binary"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// minimalELF64 builds a minimal but valid 64-bit ELF header for the given
// machine type and data encoding. It has no program or section headers, which
// is enough for debug/elf to parse the machine type.
func minimalELF64(machine elf.Machine, data elf.Data) []byte {
	var order binary.ByteOrder = binary.LittleEndian
	if data == elf.ELFDATA2MSB {
		order = binary.BigEndian
	}

	b := make([]byte, 64)
	// e_ident
	b[0], b[1], b[2], b[3] = 0x7f, 'E', 'L', 'F'
	b[4] = byte(elf.ELFCLASS64)
	b[5] = byte(data)
	b[6] = byte(elf.EV_CURRENT)

	order.PutUint16(b[16:], uint16(elf.ET_EXEC)) // e_type
	order.PutUint16(b[18:], uint16(machine))     // e_machine
	order.PutUint32(b[20:], uint32(elf.EV_CURRENT))
	// e_entry, e_phoff, e_shoff all zero -> no program/section headers
	order.PutUint16(b[52:], 64) // e_ehsize
	return b
}

func TestElfMachineToArch(t *testing.T) {
	tests := []struct {
		machine elf.Machine
		data    elf.Data
		want    string
	}{
		{elf.EM_X86_64, elf.ELFDATA2LSB, "x86_64"},
		{elf.EM_AARCH64, elf.ELFDATA2LSB, "aarch64"},
		{elf.EM_386, elf.ELFDATA2LSB, "i386"},
		{elf.EM_ARM, elf.ELFDATA2LSB, "arm"},
		{elf.EM_PPC64, elf.ELFDATA2LSB, "ppc64le"},
		{elf.EM_PPC64, elf.ELFDATA2MSB, "ppc64"},
		{elf.EM_S390, elf.ELFDATA2MSB, "s390x"},
		{elf.EM_RISCV, elf.ELFDATA2LSB, "riscv64"},
		{elf.EM_AVR, elf.ELFDATA2LSB, ""}, // unknown -> empty
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, elfMachineToArch(tt.machine, tt.data), tt.machine.String())
	}
}

func TestArchFromELF(t *testing.T) {
	t.Run("x86_64 binary at /bin/ls", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		require.NoError(t, afero.WriteFile(fs, "/bin/ls", minimalELF64(elf.EM_X86_64, elf.ELFDATA2LSB), 0o755))
		assert.Equal(t, "x86_64", archFromELF(fs))
	})

	t.Run("falls through to a later candidate", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		// /bin/ls is not an ELF file; /sbin/init is.
		require.NoError(t, afero.WriteFile(fs, "/bin/ls", []byte("#!/bin/sh\n"), 0o755))
		require.NoError(t, afero.WriteFile(fs, "/sbin/init", minimalELF64(elf.EM_AARCH64, elf.ELFDATA2LSB), 0o755))
		assert.Equal(t, "aarch64", archFromELF(fs))
	})

	t.Run("no candidates present", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		assert.Equal(t, "", archFromELF(fs))
	})

	t.Run("nil filesystem", func(t *testing.T) {
		assert.Equal(t, "", archFromELF(nil))
	})
}
