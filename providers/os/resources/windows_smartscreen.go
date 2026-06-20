// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"sync"

	"go.mondoo.com/mql/v13/providers/os/connection/shared"
	"go.mondoo.com/mql/v13/providers/os/resources/windows"
)

// mqlWindowsSmartScreenInternal caches the policy read so every accessor shares
// a single PowerShell invocation.
type mqlWindowsSmartScreenInternal struct {
	lock     sync.Mutex
	fetched  bool
	settings *windows.SmartScreenSettings
	err      error
}

func (s *mqlWindowsSmartScreen) get() (*windows.SmartScreenSettings, error) {
	s.lock.Lock()
	defer s.lock.Unlock()
	if s.fetched {
		return s.settings, s.err
	}
	conn := s.MqlRuntime.Connection.(shared.Connection)
	s.settings, s.err = windows.GetSmartScreenSettings(conn)
	s.fetched = true
	return s.settings, s.err
}

func (s *mqlWindowsSmartScreen) explorerEnabled() (bool, error) {
	v, err := s.get()
	if err != nil {
		return false, err
	}
	return v.ExplorerEnabled(), nil
}

func (s *mqlWindowsSmartScreen) explorerLevel() (string, error) {
	v, err := s.get()
	if err != nil {
		return "", err
	}
	return v.ShellSmartScreenLevel, nil
}

func (s *mqlWindowsSmartScreen) edgeEnabled() (bool, error) {
	v, err := s.get()
	if err != nil {
		return false, err
	}
	return v.EdgeEnabled(), nil
}

func (s *mqlWindowsSmartScreen) edgePuaEnabled() (bool, error) {
	v, err := s.get()
	if err != nil {
		return false, err
	}
	return v.EdgePuaEnabled(), nil
}

func (s *mqlWindowsSmartScreen) edgePreventOverride() (bool, error) {
	v, err := s.get()
	if err != nil {
		return false, err
	}
	return v.EdgePreventOverrideEnabled(), nil
}

func (s *mqlWindowsSmartScreen) edgePreventOverrideForFiles() (bool, error) {
	v, err := s.get()
	if err != nil {
		return false, err
	}
	return v.EdgePreventOverrideForFilesEnabled(), nil
}

func (s *mqlWindowsSmartScreen) storeAppsEnabled() (bool, error) {
	v, err := s.get()
	if err != nil {
		return false, err
	}
	return v.StoreAppsEnabled(), nil
}
