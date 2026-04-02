// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"sync"

	"github.com/spf13/afero"
	"go.mondoo.com/mql/v13/providers/os/connection/shared"
)

// EFI variable GUID for global Secure Boot variables.
const efiGlobalVariable = "8be4df61-93ca-11d2-aa0d-00e098032b8c"

type mqlMachineSecurebootInternal struct {
	once           sync.Once
	cacheEfi       bool
	cacheEnabled   bool
	cacheSetupMode bool
	fetchErr       error
}

func (s *mqlMachineSecureboot) id() (string, error) {
	return "machine.secureboot", nil
}

// fetchStatus reads the EFI firmware variables once and caches the result.
func (s *mqlMachineSecureboot) fetchStatus() error {
	s.once.Do(func() {
		conn := s.MqlRuntime.Connection.(shared.Connection)
		fs := conn.FileSystem()

		// Check if the system is booted in EFI mode by looking for /sys/firmware/efi.
		_, err := fs.Stat("/sys/firmware/efi")
		if err != nil {
			// No EFI directory means legacy BIOS boot — no Secure Boot possible.
			return
		}
		s.cacheEfi = true

		s.cacheEnabled = readEfiVarBool(fs, "SecureBoot-"+efiGlobalVariable)
		s.cacheSetupMode = readEfiVarBool(fs, "SetupMode-"+efiGlobalVariable)
	})
	return s.fetchErr
}

// readEfiVarBool reads an EFI variable from /sys/firmware/efi/efivars/ and
// returns true if its data byte is 1. EFI variable files contain a 4-byte
// attribute header followed by the variable data.
func readEfiVarBool(fs afero.Fs, name string) bool {
	data, err := afero.ReadFile(fs, "/sys/firmware/efi/efivars/"+name)
	if err != nil {
		return false
	}
	// Must have at least 4 bytes of attributes + 1 byte of data.
	if len(data) < 5 {
		return false
	}
	// The data portion starts after the 4-byte EFI variable attributes header.
	// For SecureBoot/SetupMode the data is a single uint8: 1 = on, 0 = off.
	return data[4] == 1
}

func (s *mqlMachineSecureboot) efi() (bool, error) {
	if err := s.fetchStatus(); err != nil {
		return false, err
	}
	return s.cacheEfi, nil
}

func (s *mqlMachineSecureboot) enabled() (bool, error) {
	if err := s.fetchStatus(); err != nil {
		return false, err
	}
	return s.cacheEnabled, nil
}

func (s *mqlMachineSecureboot) setupMode() (bool, error) {
	if err := s.fetchStatus(); err != nil {
		return false, err
	}
	return s.cacheSetupMode, nil
}
