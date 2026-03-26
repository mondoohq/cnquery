// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
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
	return int64(config["globalstate"].(float64)), nil
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
	return int64(config["stealthenabled"].(float64)) != 0, nil
}

func (m *mqlMacosFirewall) loggingEnabled() (bool, error) {
	config, err := m.fetchConfig()
	if err != nil {
		return false, err
	}
	return int64(config["loggingenabled"].(float64)) != 0, nil
}

func (m *mqlMacosFirewall) loggingDetail() (string, error) {
	config, err := m.fetchConfig()
	if err != nil {
		return "", err
	}
	option := int64(config["loggingoption"].(float64))
	switch option {
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
	return int64(config["allowsignedenabled"].(float64)) != 0, nil
}

func (m *mqlMacosFirewall) allowDownloadSignedApps() (bool, error) {
	config, err := m.fetchConfig()
	if err != nil {
		return false, err
	}
	return int64(config["allowdownloadsignedenabled"].(float64)) != 0, nil
}

func (m *mqlMacosFirewall) version() (string, error) {
	config, err := m.fetchConfig()
	if err != nil {
		return "", err
	}
	return config["version"].(string), nil
}

func (m *mqlMacosFirewall) exceptions() ([]any, error) {
	config, err := m.fetchConfig()
	if err != nil {
		return nil, err
	}
	return config["exceptions"].([]any), nil
}

func (m *mqlMacosFirewall) explicitAuths() ([]any, error) {
	config, err := m.fetchConfig()
	if err != nil {
		return nil, err
	}
	explicitAuthsRaw := config["explicitauths"].([]any)
	result := []any{}
	for i := range explicitAuthsRaw {
		entry := explicitAuthsRaw[i].(map[string]any)
		result = append(result, entry["id"])
	}
	return result, nil
}

func (m *mqlMacosFirewall) applications() ([]any, error) {
	config, err := m.fetchConfig()
	if err != nil {
		return nil, err
	}

	appsRaw := config["applications"].([]any)
	apps := make([]any, 0, len(appsRaw))
	for _, raw := range appsRaw {
		entry := raw.(map[string]any)

		name, _ := entry["bundleid"].(string)
		bundleId := name
		if path, ok := entry["path"].(string); ok && path != "" {
			name = path
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
