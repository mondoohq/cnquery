// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"strings"
	"sync"

	"go.mondoo.com/mql/v13/llx"
)

// systemProfilerSharingCmd populates the unified Sharing panel view —
// the same data shown by `system_profiler SPSharingDataType`. Output
// is one `Service Name: On|Off` line per toggle, indented under a
// `Sharing:` header, so the line parser doesn't need to deal with
// `system_profiler -json`'s schema drift across macOS versions.
const systemProfilerSharingCmd = "system_profiler SPSharingDataType"

type mqlMacosSharingInternal struct {
	lock    sync.Mutex
	fetched bool
	state   map[string]bool
}

func (s *mqlMacosSharing) id() (string, error) {
	return "macos.sharing", nil
}

// fetchState runs `system_profiler SPSharingDataType` once and
// returns the parsed map of service name → enabled. Non-Darwin hosts
// (or hosts where system_profiler is unreachable) get an empty map,
// which surfaces as `false` on every accessor — fail-soft for
// systems that simply don't have a Sharing panel.
func (s *mqlMacosSharing) fetchState() (map[string]bool, error) {
	if s.fetched {
		return s.state, nil
	}
	s.lock.Lock()
	defer s.lock.Unlock()
	if s.fetched {
		return s.state, nil
	}

	res, err := NewResource(s.MqlRuntime, "command", map[string]*llx.RawData{
		"command": llx.StringData(systemProfilerSharingCmd),
	})
	if err != nil {
		return nil, err
	}
	cmd := res.(*mqlCommand)
	if exit := cmd.GetExitcode(); exit.Data != 0 {
		s.state = map[string]bool{}
		s.fetched = true
		return s.state, nil
	}

	s.state = parseSharingOutput(cmd.GetStdout().Data)
	s.fetched = true
	return s.state, nil
}

// parseSharingOutput parses the human-readable output of
// `system_profiler SPSharingDataType`. The format has been stable
// across the last decade of macOS releases:
//
//	Sharing:
//
//	    Computer Name: My Mac
//	    Bluetooth Sharing: Off
//	    File Sharing: On
//	    Screen Sharing: Off
//	    AirPlay Receiver: On
//
// Each `Name: Value` line whose value is exactly `On` or `Off` is
// captured into the returned map. Lines like `Computer Name: ...`
// don't match the On/Off shape and are quietly skipped.
func parseSharingOutput(stdout string) map[string]bool {
	out := map[string]bool{}
	for _, raw := range strings.Split(stdout, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		idx := strings.Index(line, ": ")
		if idx <= 0 {
			continue
		}
		key := strings.TrimSpace(line[:idx])
		value := strings.TrimSpace(line[idx+2:])
		switch value {
		case "On":
			out[key] = true
		case "Off":
			out[key] = false
		}
	}
	return out
}

// sharingFlag returns the bool for one Sharing panel entry. Missing
// entries (some macOS versions omit services that aren't installed,
// e.g. DVD or CD Sharing on Apple Silicon) read as `false` — the
// "service is not present" outcome is operationally equivalent to
// "service is off" for audit purposes.
func (s *mqlMacosSharing) sharingFlag(name string) (bool, error) {
	state, err := s.fetchState()
	if err != nil {
		return false, err
	}
	return state[name], nil
}

func (s *mqlMacosSharing) screenSharing() (bool, error) {
	return s.sharingFlag("Screen Sharing")
}

func (s *mqlMacosSharing) remoteManagement() (bool, error) {
	return s.sharingFlag("Remote Management")
}

func (s *mqlMacosSharing) fileSharing() (bool, error) {
	return s.sharingFlag("File Sharing")
}

func (s *mqlMacosSharing) printerSharing() (bool, error) {
	return s.sharingFlag("Printer Sharing")
}

func (s *mqlMacosSharing) internetSharing() (bool, error) {
	return s.sharingFlag("Internet Sharing")
}

func (s *mqlMacosSharing) bluetoothSharing() (bool, error) {
	return s.sharingFlag("Bluetooth Sharing")
}

func (s *mqlMacosSharing) mediaSharing() (bool, error) {
	return s.sharingFlag("Media Sharing")
}

func (s *mqlMacosSharing) contentCaching() (bool, error) {
	return s.sharingFlag("Content Caching")
}

func (s *mqlMacosSharing) airplayReceiver() (bool, error) {
	return s.sharingFlag("AirPlay Receiver")
}

func (s *mqlMacosSharing) dvdSharing() (bool, error) {
	return s.sharingFlag("DVD or CD Sharing")
}
