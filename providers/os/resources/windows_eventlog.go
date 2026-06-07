// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/os/registry"
	"go.mondoo.com/ranger-rpc/codes"
	"go.mondoo.com/ranger-rpc/status"
)

// Registry locations that back an Event Log channel policy. A value enforced
// through Group Policy (the policy path) overrides the channel's effective
// configuration; when neither is set the documented Windows default applies.
const (
	eventlogPolicyPathFmt  = `HKEY_LOCAL_MACHINE\SOFTWARE\Policies\Microsoft\Windows\EventLog\%s`
	eventlogServicePathFmt = `HKEY_LOCAL_MACHINE\SYSTEM\CurrentControlSet\Services\EventLog\%s`
)

// Decoded Retention behaviors.
const (
	retentionOverwriteAsNeeded = "overwrite_as_needed"
	retentionOverwriteByDays   = "overwrite_by_days"
	retentionNeverOverwrite    = "never_overwrite"
)

// neverOverwriteRetention is the Retention sentinel (0xFFFFFFFF) that selects
// "do not overwrite events"; it can also appear as -1 from a signed reading.
const neverOverwriteRetention int64 = 0xFFFFFFFF

// eventlogDefaultMaxSizeKB is the modern Windows default for the classic
// channels (20 MB) used when no policy or effective value is configured.
const eventlogDefaultMaxSizeKB int64 = 20480

type mqlWindowsEventlogInternal struct {
	lock    sync.Mutex
	loaded  atomic.Bool
	loadErr error
	// each map is value name (lower-cased) -> registry item for one key
	policy  map[string]registry.RegistryKeyItem
	service map[string]registry.RegistryKeyItem
}

func (w *mqlWindowsEventlog) id() (string, error) {
	return "windows.eventlog/" + w.Name.Data, nil
}

func initWindowsEventlog(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	// the channel is selected purely by name; the field accessors read the
	// registry lazily, so there is nothing to pre-fetch here
	return args, nil, nil
}

// readKey reads a single registry key and returns its values keyed by the
// lower-cased value name. A missing key yields an empty map rather than an
// error, so resolution can fall through to the next source or the default.
func (w *mqlWindowsEventlog) readKey(path string) (map[string]registry.RegistryKeyItem, error) {
	o, err := CreateResource(w.MqlRuntime, "registrykey", map[string]*llx.RawData{
		"path": llx.StringData(path),
	})
	if err != nil {
		return nil, err
	}

	entries, err := o.(*mqlRegistrykey).getEntries()
	if err != nil {
		if st, ok := status.FromError(err); ok && st.Code() == codes.NotFound {
			return map[string]registry.RegistryKeyItem{}, nil
		}
		return nil, err
	}

	res := make(map[string]registry.RegistryKeyItem, len(entries))
	for i := range entries {
		res[strings.ToLower(entries[i].Key)] = entries[i]
	}
	return res, nil
}

// load reads the policy and effective registry keys exactly once and caches
// them so every field shares a single set of registry reads.
func (w *mqlWindowsEventlog) load() error {
	if w.loaded.Load() {
		return nil
	}
	w.lock.Lock()
	defer w.lock.Unlock()
	if w.loaded.Load() || w.loadErr != nil {
		return w.loadErr
	}

	policy, err := w.readKey(fmt.Sprintf(eventlogPolicyPathFmt, w.Name.Data))
	if err != nil {
		w.loadErr = err
		return err
	}
	service, err := w.readKey(fmt.Sprintf(eventlogServicePathFmt, w.Name.Data))
	if err != nil {
		w.loadErr = err
		return err
	}

	w.policy = policy
	w.service = service
	w.loaded.Store(true)
	return nil
}

// registryItemInt extracts an int64 from a registry item, accepting both the
// DWORD form (effective channel config) and the string form ("0", "0xFFFFFFFF")
// that the Group Policy keys use.
func registryItemInt(item registry.RegistryKeyItem) (int64, bool) {
	switch v := item.GetRawValue().(type) {
	case int64:
		return v, true
	case int:
		return int64(v), true
	case string:
		s := strings.TrimSpace(v)
		if s == "" {
			return 0, false
		}
		// base 0 handles 0x-prefixed hex and plain decimal, including -1
		if n, err := strconv.ParseInt(s, 0, 64); err == nil {
			return n, true
		}
		if n, err := strconv.ParseUint(s, 0, 64); err == nil {
			return int64(n), true
		}
	}
	return 0, false
}

func (w *mqlWindowsEventlog) maxSizeKB() (int64, error) {
	if err := w.load(); err != nil {
		return 0, err
	}
	// the Group Policy MaxSize is already expressed in KB
	if n, ok := lookupInt(w.policy, "maxsize"); ok {
		return n, nil
	}
	// the effective Services\EventLog MaxSize is in bytes
	if n, ok := lookupInt(w.service, "maxsize"); ok {
		return n / 1024, nil
	}
	return eventlogDefaultMaxSizeKB, nil
}

// retentionRaw returns the effective Retention value (policy wins) and whether
// it was set in either source.
func (w *mqlWindowsEventlog) retentionRaw() (int64, bool, error) {
	if err := w.load(); err != nil {
		return 0, false, err
	}
	if n, ok := lookupInt(w.policy, "retention"); ok {
		return n, true, nil
	}
	if n, ok := lookupInt(w.service, "retention"); ok {
		return n, true, nil
	}
	return 0, false, nil
}

func (w *mqlWindowsEventlog) retention() (string, error) {
	n, ok, err := w.retentionRaw()
	if err != nil {
		return "", err
	}
	if !ok {
		// Windows overwrites events as needed by default
		return retentionOverwriteAsNeeded, nil
	}
	return decodeRetention(n), nil
}

// decodeRetention maps a Retention value to a human-readable behavior. The
// encoding is shared by the modern Group Policy form (0 / 0xFFFFFFFF) and the
// legacy seconds form (0 = as needed, 0xFFFFFFFF / -1 = never, N = by days).
func decodeRetention(n int64) string {
	switch {
	case n == 0:
		return retentionOverwriteAsNeeded
	case n < 0 || n == neverOverwriteRetention:
		return retentionNeverOverwrite
	default:
		return retentionOverwriteByDays
	}
}

func (w *mqlWindowsEventlog) overwriteAsNeeded() (bool, error) {
	n, ok, err := w.retentionRaw()
	if err != nil {
		return false, err
	}
	if !ok {
		return true, nil
	}
	return n == 0, nil
}

// lookupInt returns the int64 value of a registry item from the given key map.
func lookupInt(items map[string]registry.RegistryKeyItem, name string) (int64, bool) {
	item, ok := items[name]
	if !ok {
		return 0, false
	}
	return registryItemInt(item)
}
