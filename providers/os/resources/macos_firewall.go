// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"fmt"
	"sync"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers/os/connection/shared"
	"go.mondoo.com/mql/v13/providers/os/resources/plist"
)

type mqlMacosFirewallInternal struct {
	lock    sync.Mutex
	fetched bool
	config  plist.Data
}

func alfConfigFloat(config plist.Data, key string) (float64, error) {
	v, ok := config[key].(float64)
	if !ok {
		return 0, fmt.Errorf("ALF config key %q not found or not a number", key)
	}
	return v, nil
}

func alfConfigString(config plist.Data, key string) (string, error) {
	v, ok := config[key].(string)
	if !ok {
		return "", fmt.Errorf("ALF config key %q not found or not a string", key)
	}
	return v, nil
}

func alfConfigSlice(config plist.Data, key string) ([]any, error) {
	v, ok := config[key].([]any)
	if !ok {
		return nil, fmt.Errorf("ALF config key %q not found or not an array", key)
	}
	return v, nil
}

func (m *mqlMacosFirewall) fetchConfig() (plist.Data, error) {
	if m.fetched {
		return m.config, nil
	}
	m.lock.Lock()
	defer m.lock.Unlock()
	if m.fetched {
		return m.config, nil
	}

	conn := m.MqlRuntime.Connection.(shared.Connection)
	fs := conn.FileSystem()

	var plistLocation string
	for _, loc := range alfPlistLocations {
		log.Debug().Str("location", loc).Msg("Checking for ALF configuration")
		s, err := fs.Stat(loc)
		if err == nil && !s.IsDir() {
			log.Debug().Str("location", loc).Msg("Found ALF configuration")
			plistLocation = loc
			break
		}
	}

	if plistLocation == "" {
		return nil, errors.New("ALF configuration not found")
	}

	f, err := fs.Open(plistLocation)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	config, err := plist.Decode(f)
	if err != nil {
		return nil, err
	}

	m.fetched = true
	m.config = config
	return config, nil
}

func (m *mqlMacosFirewall) globalState() (int64, error) {
	config, err := m.fetchConfig()
	if err != nil {
		return 0, err
	}
	v, err := alfConfigFloat(config, "globalstate")
	if err != nil {
		return 0, err
	}
	return int64(v), nil
}

func (m *mqlMacosFirewall) enabled() (bool, error) {
	state, err := m.globalState()
	if err != nil {
		return false, err
	}
	return state >= 1, nil
}

func (m *mqlMacosFirewall) blockAllIncoming() (bool, error) {
	state, err := m.globalState()
	if err != nil {
		return false, err
	}
	return state == 2, nil
}

func (m *mqlMacosFirewall) stealthEnabled() (bool, error) {
	config, err := m.fetchConfig()
	if err != nil {
		return false, err
	}
	v, err := alfConfigFloat(config, "stealthenabled")
	if err != nil {
		return false, err
	}
	return int64(v) != 0, nil
}

func (m *mqlMacosFirewall) loggingEnabled() (bool, error) {
	config, err := m.fetchConfig()
	if err != nil {
		return false, err
	}
	v, err := alfConfigFloat(config, "loggingenabled")
	if err != nil {
		return false, err
	}
	return int64(v) != 0, nil
}

func (m *mqlMacosFirewall) loggingDetail() (string, error) {
	config, err := m.fetchConfig()
	if err != nil {
		return "", err
	}
	v, err := alfConfigFloat(config, "loggingoption")
	if err != nil {
		return "", err
	}
	switch int64(v) {
	case 0:
		return "disabled", nil
	case 1:
		return "detail", nil
	case 2:
		return "brief", nil
	case 3:
		return "throttled", nil
	default:
		return "unknown", nil
	}
}

func (m *mqlMacosFirewall) allowSignedApps() (bool, error) {
	config, err := m.fetchConfig()
	if err != nil {
		return false, err
	}
	v, err := alfConfigFloat(config, "allowsignedenabled")
	if err != nil {
		return false, err
	}
	return int64(v) != 0, nil
}

func (m *mqlMacosFirewall) allowDownloadSignedApps() (bool, error) {
	config, err := m.fetchConfig()
	if err != nil {
		return false, err
	}
	v, err := alfConfigFloat(config, "allowdownloadsignedenabled")
	if err != nil {
		return false, err
	}
	return int64(v) != 0, nil
}

func (m *mqlMacosFirewall) version() (string, error) {
	config, err := m.fetchConfig()
	if err != nil {
		return "", err
	}
	return alfConfigString(config, "version")
}

func (m *mqlMacosFirewall) exceptions() ([]any, error) {
	config, err := m.fetchConfig()
	if err != nil {
		return nil, err
	}
	return alfConfigSlice(config, "exceptions")
}

func (m *mqlMacosFirewall) explicitAuths() ([]any, error) {
	config, err := m.fetchConfig()
	if err != nil {
		return nil, err
	}
	explicitAuthsRaw, err := alfConfigSlice(config, "explicitauths")
	if err != nil {
		return nil, err
	}
	result := []any{}
	for i := range explicitAuthsRaw {
		entry, ok := explicitAuthsRaw[i].(map[string]any)
		if !ok {
			continue
		}
		result = append(result, entry["id"])
	}
	return result, nil
}

func (m *mqlMacosFirewall) applications() ([]any, error) {
	config, err := m.fetchConfig()
	if err != nil {
		return nil, err
	}

	appsRaw, err := alfConfigSlice(config, "applications")
	if err != nil {
		return nil, err
	}
	apps := make([]any, 0, len(appsRaw))
	for i, raw := range appsRaw {
		entry, ok := raw.(map[string]any)
		if !ok {
			continue
		}

		bundleId, _ := entry["bundleid"].(string)
		name := bundleId
		if path, ok := entry["path"].(string); ok && path != "" {
			name = path
		}
		if name == "" {
			name = fmt.Sprintf("unknown-%d", i)
		}
		state := int64(0)
		if s, ok := entry["state"].(float64); ok {
			state = int64(s)
		}

		app, err := CreateResource(m.MqlRuntime, "macos.firewall.app", map[string]*llx.RawData{
			"__id":     llx.StringData("macos.firewall.app/" + name),
			"name":     llx.StringData(name),
			"bundleId": llx.StringData(bundleId),
			"state":    llx.IntData(state),
		})
		if err != nil {
			return nil, err
		}
		apps = append(apps, app)
	}

	return apps, nil
}

// id is required for the macos.firewall.app sub-resource
func (m *mqlMacosFirewallApp) id() (string, error) {
	return "macos.firewall.app/" + m.Name.Data, nil
}
