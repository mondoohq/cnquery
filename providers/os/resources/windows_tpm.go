// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"io"
	"sync"
	"sync/atomic"

	"go.mondoo.com/mql/v13/providers/os/connection/shared"
	"go.mondoo.com/mql/v13/providers/os/resources/powershell"
	"go.mondoo.com/mql/v13/providers/os/resources/windows"
)

type mqlWindowsTpmInternal struct {
	lock    sync.Mutex
	loaded  atomic.Bool
	loadErr error
	info    *windows.TpmInfo
}

func (w *mqlWindowsTpm) id() (string, error) {
	return "windows.tpm", nil
}

// load runs the TPM query exactly once and caches the result so every field
// shares a single PowerShell invocation.
func (w *mqlWindowsTpm) load() (*windows.TpmInfo, error) {
	if w.loaded.Load() {
		return w.info, nil
	}
	w.lock.Lock()
	defer w.lock.Unlock()
	if w.loaded.Load() || w.loadErr != nil {
		return w.info, w.loadErr
	}

	conn := w.MqlRuntime.Connection.(shared.Connection)
	executedCmd, err := conn.RunCommand(powershell.Encode(windows.PSGetTpm))
	if err != nil {
		w.loadErr = err
		return nil, err
	}
	if executedCmd.ExitStatus != 0 {
		stderr, _ := io.ReadAll(executedCmd.Stderr)
		w.loadErr = errors.New("failed to retrieve TPM information: " + string(stderr))
		return nil, w.loadErr
	}

	info, err := windows.ParseTpm(executedCmd.Stdout)
	if err != nil {
		w.loadErr = err
		return nil, err
	}

	w.info = info
	w.loaded.Store(true)
	return info, nil
}

func (w *mqlWindowsTpm) present() (bool, error) {
	info, err := w.load()
	if err != nil {
		return false, err
	}
	return info.TpmPresent, nil
}

func (w *mqlWindowsTpm) ready() (bool, error) {
	info, err := w.load()
	if err != nil {
		return false, err
	}
	return info.TpmReady, nil
}

func (w *mqlWindowsTpm) enabled() (bool, error) {
	info, err := w.load()
	if err != nil {
		return false, err
	}
	return info.TpmEnabled, nil
}

func (w *mqlWindowsTpm) activated() (bool, error) {
	info, err := w.load()
	if err != nil {
		return false, err
	}
	return info.TpmActivated, nil
}

func (w *mqlWindowsTpm) specVersion() (string, error) {
	info, err := w.load()
	if err != nil {
		return "", err
	}
	return info.MajorSpecVersion(), nil
}

func (w *mqlWindowsTpm) manufacturerVersion() (string, error) {
	info, err := w.load()
	if err != nil {
		return "", err
	}
	return info.ManufacturerVersion, nil
}
