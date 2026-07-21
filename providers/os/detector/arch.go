// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package detector

import (
	"debug/elf"

	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
)

// archBinaryCandidates is an ordered list of well-known binaries that are
// present on virtually every Linux installation. The first one that exists and
// parses as an ELF file is used to determine the machine architecture.
var archBinaryCandidates = []string{
	"/bin/ls",
	"/usr/bin/ls",
	"/bin/sh",
	"/sbin/init",
	"/usr/lib/systemd/systemd",
	"/bin/busybox",
}

// archFromELF determines the platform architecture by inspecting the ELF header
// of a well-known binary on the target filesystem. It is used as a fallback for
// command-less scans (e.g. filesystem scans of k8s nodes) where `uname -m` is
// not available. It returns an empty string if the architecture could not be
// determined.
func archFromELF(fs afero.Fs) string {
	if fs == nil {
		return ""
	}

	for _, path := range archBinaryCandidates {
		arch := archFromBinary(fs, path)
		if arch != "" {
			log.Debug().Str("path", path).Str("arch", arch).Msg("detected platform architecture from ELF header")
			return arch
		}
	}

	log.Debug().Msg("could not determine platform architecture from filesystem")
	return ""
}

// archFromBinary opens a single file and returns its ELF machine architecture,
// or an empty string if the file does not exist or is not a recognized ELF
// binary.
func archFromBinary(fs afero.Fs, path string) string {
	f, err := fs.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()

	ef, err := elf.NewFile(f)
	if err != nil {
		return ""
	}

	return elfMachineToArch(ef.Machine, ef.Data)
}

// elfMachineToArch maps an ELF machine type to the uname-style architecture
// string used throughout platform detection (i.e. the value `uname -m` would
// report). Unknown machine types return an empty string.
func elfMachineToArch(machine elf.Machine, data elf.Data) string {
	switch machine {
	case elf.EM_X86_64:
		return "x86_64"
	case elf.EM_AARCH64:
		return "aarch64"
	case elf.EM_386:
		return "i386"
	case elf.EM_ARM:
		return "arm"
	case elf.EM_PPC64:
		// ppc64 (big endian) and ppc64le (little endian) share a machine type
		// and are distinguished by the ELF data encoding.
		if data == elf.ELFDATA2LSB {
			return "ppc64le"
		}
		return "ppc64"
	case elf.EM_PPC:
		return "ppc"
	case elf.EM_S390:
		return "s390x"
	case elf.EM_RISCV:
		return "riscv64"
	case elf.EM_MIPS:
		return "mips"
	default:
		return ""
	}
}
