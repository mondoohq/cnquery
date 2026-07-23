// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"sync"
	"sync/atomic"

	tsclient "github.com/tailscale/tailscale-client-go/v2"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/tailscale/connection"
)

// mqlTailscaleInternal caches tailnet-level settings so multiple field accessors
// share the result of a single TailnetSettings API call. settingsFetched is
// atomic because the accessors that read it are distinct MQL fields, which the
// runtime may resolve concurrently.
type mqlTailscaleInternal struct {
	settingsLock    sync.Mutex
	settingsFetched atomic.Bool
	settings        *tsclient.TailnetSettings
}

func (r *mqlTailscale) id() (string, error) {
	return r.Tailnet.Data, nil
}

func initTailscale(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	conn := runtime.Connection.(*connection.TailscaleConnection)
	mqlResource, err := CreateResource(runtime, "tailscale",
		map[string]*llx.RawData{
			"tailnet": llx.StringData(conn.ResolveTailnet()),
		})
	if err != nil {
		return args, nil, err
	}
	return args, mqlResource, nil
}

func (t *mqlTailscale) devices() ([]any, error) {
	conn := t.MqlRuntime.Connection.(*connection.TailscaleConnection)
	devices, err := conn.Client().Devices().List(context.Background())
	if err != nil {
		return nil, err
	}

	var resources []any
	for _, device := range devices {
		resource, err := createTailscaleDeviceResource(t.MqlRuntime, &device)
		if err != nil {
			return nil, err
		}
		resources = append(resources, resource)
	}

	return resources, nil
}

func (t *mqlTailscale) users() ([]any, error) {
	conn := t.MqlRuntime.Connection.(*connection.TailscaleConnection)
	// TODO we can do filter here for user type and role
	users, err := conn.Client().Users().List(context.Background(), nil, nil)
	if err != nil {
		return nil, err
	}

	var resources []any
	for _, user := range users {
		resource, err := createTailscaleUserResource(t.MqlRuntime, &user)
		if err != nil {
			return nil, err
		}
		resources = append(resources, resource)
	}

	return resources, nil
}

func (t *mqlTailscale) nameservers() ([]any, error) {
	conn := t.MqlRuntime.Connection.(*connection.TailscaleConnection)
	nameservers, err := conn.Client().DNS().Nameservers(context.Background())
	if err != nil {
		return nil, err
	}
	return convert.SliceAnyToInterface(nameservers), nil
}

// fetchSettings returns the cached TailnetSettings, fetching it on first call.
// Multiple tailnet-level field accessors share the same response.
func (t *mqlTailscale) fetchSettings() (*tsclient.TailnetSettings, error) {
	if t.settingsFetched.Load() {
		return t.settings, nil
	}
	t.settingsLock.Lock()
	defer t.settingsLock.Unlock()
	if t.settingsFetched.Load() {
		return t.settings, nil
	}
	conn := t.MqlRuntime.Connection.(*connection.TailscaleConnection)
	settings, err := conn.Client().TailnetSettings().Get(context.Background())
	if err != nil {
		return nil, err
	}
	t.settings = settings
	t.settingsFetched.Store(true)
	return t.settings, nil
}

func (t *mqlTailscale) deviceApprovalRequired() (bool, error) {
	s, err := t.fetchSettings()
	if err != nil {
		return false, err
	}
	return s.DevicesApprovalOn, nil
}

func (t *mqlTailscale) userApprovalRequired() (bool, error) {
	s, err := t.fetchSettings()
	if err != nil {
		return false, err
	}
	return s.UsersApprovalOn, nil
}

func (t *mqlTailscale) devicesAutoUpdatesEnabled() (bool, error) {
	s, err := t.fetchSettings()
	if err != nil {
		return false, err
	}
	return s.DevicesAutoUpdatesOn, nil
}

func (t *mqlTailscale) devicesKeyDurationDays() (int64, error) {
	s, err := t.fetchSettings()
	if err != nil {
		return 0, err
	}
	return int64(s.DevicesKeyDurationDays), nil
}

func (t *mqlTailscale) networkFlowLoggingEnabled() (bool, error) {
	s, err := t.fetchSettings()
	if err != nil {
		return false, err
	}
	return s.NetworkFlowLoggingOn, nil
}

func (t *mqlTailscale) postureIdentityCollectionEnabled() (bool, error) {
	s, err := t.fetchSettings()
	if err != nil {
		return false, err
	}
	return s.PostureIdentityCollectionOn, nil
}

func (t *mqlTailscale) usersRoleAllowedToJoinExternalTailnets() (string, error) {
	s, err := t.fetchSettings()
	if err != nil {
		return "", err
	}
	return string(s.UsersRoleAllowedToJoinExternalTailnets), nil
}

func (t *mqlTailscale) aclPolicy() (*mqlTailscaleAclPolicy, error) {
	conn := t.MqlRuntime.Connection.(*connection.TailscaleConnection)
	ctx := context.Background()

	acl, err := conn.Client().PolicyFile().Get(ctx)
	if err != nil {
		return nil, err
	}

	resource, err := createTailscaleAclPolicyResource(t.MqlRuntime, t.Tailnet.Data, acl)
	if err != nil {
		return nil, err
	}
	return resource.(*mqlTailscaleAclPolicy), nil
}
