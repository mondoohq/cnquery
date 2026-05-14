// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"errors"
	"strings"
	"sync"

	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/vault"
	"go.mondoo.com/mql/v13/providers/iru/connection/client"
)

const (
	OptionAPIURL = "api_url"
)

// IruConnection holds an authenticated Iru REST client and per-device
// caches populated as we walk the inventory.
type IruConnection struct {
	plugin.Connection
	Conf   *inventory.Config
	asset  *inventory.Asset
	Client *client.Client

	// deviceDetails caches the rich GET /devices/{id}/details payload
	// keyed by device ID. It backs most of the iru.device computed
	// methods (filevault, gatekeeper, volumes, …) so iterating
	// `iru.devices { … posture fields … }` only triggers one detail
	// fetch per device rather than one per field.
	detailsMu sync.Mutex
	details   map[string]*client.DeviceDetails

	// deviceApps caches the per-device installed-app listing — the apps
	// endpoint is its own call, independent of the details payload.
	// (The profiles list is read straight off DeviceDetails.InstalledProfiles,
	// so it does not need a second cache layer.)
	appsMu     sync.Mutex
	deviceApps map[string][]client.App

	// blueprintsMu / usersMu / libraryItemsMu memoize the tenant-wide listings
	// so the init functions for typed cross-references (blueprint.libraryItems,
	// libraryItem.blueprints, device.user) can resolve a single ID without
	// re-walking the API for every reference.
	blueprintsMu     sync.Mutex
	blueprintsCached []client.Blueprint
	blueprintsErr    error
	blueprintsDone   bool

	usersMu     sync.Mutex
	usersCached []client.User
	usersErr    error
	usersDone   bool

	libraryItemsMu     sync.Mutex
	libraryItemsCached []client.LibraryItem
	libraryItemsErr    error
	libraryItemsDone   bool
}

// NewIruConnection validates the connection config, builds an Iru REST
// client, and returns a ready-to-use connection. It does not contact the
// API; the first request is issued lazily by resource accessors.
func NewIruConnection(id uint32, asset *inventory.Asset, conf *inventory.Config) (*IruConnection, error) {
	conn := &IruConnection{
		Connection: plugin.NewConnection(id, asset),
		Conf:       conf,
		asset:      asset,
		details:    make(map[string]*client.DeviceDetails),
		deviceApps: make(map[string][]client.App),
	}

	apiURL := conf.Options[OptionAPIURL]

	var token string
	for _, cred := range conf.Credentials {
		if cred.Type == vault.CredentialType_password {
			token = string(cred.Secret)
			break
		}
	}

	if apiURL == "" || token == "" {
		return nil, errors.New("missing required Iru credentials: api-url and token")
	}

	cl, err := client.New(apiURL, token)
	if err != nil {
		return nil, err
	}
	conn.Client = cl
	return conn, nil
}

func (c *IruConnection) Name() string { return "iru" }

func (c *IruConnection) Asset() *inventory.Asset { return c.asset }

// PlatformInfo returns the asset platform metadata for an Iru tenant.
func (c *IruConnection) PlatformInfo() *inventory.Platform {
	return &inventory.Platform{
		Name:                  "iru",
		Title:                 "Iru",
		Family:                []string{"iru"},
		Kind:                  "api",
		Runtime:               "iru",
		TechnologyUrlSegments: []string{"saas", "iru"},
	}
}

// Identifier returns the stable platform ID for the tenant, derived from
// the tenant API URL host. It mirrors the Jamf provider's strategy.
func (c *IruConnection) Identifier() string {
	host := c.Conf.Options[OptionAPIURL]
	if i := strings.Index(host, "://"); i >= 0 {
		host = host[i+3:]
	}
	if i := strings.IndexAny(host, "/?#"); i >= 0 {
		host = host[:i]
	}
	return "//platformid.api.mondoo.app/runtime/iru/" + strings.ToLower(host)
}

// GetDeviceDetails returns the cached detail payload for a device,
// fetching it on first request. Safe for concurrent use.
func (c *IruConnection) GetDeviceDetails(id string) (*client.DeviceDetails, error) {
	c.detailsMu.Lock()
	defer c.detailsMu.Unlock()
	if d, ok := c.details[id]; ok {
		return d, nil
	}
	d, err := c.Client.GetDeviceDetails(id)
	if err != nil {
		return nil, err
	}
	c.details[id] = d
	return d, nil
}

// GetDeviceApps returns the cached installed-app list for a device,
// fetching it on first request. Safe for concurrent use.
func (c *IruConnection) GetDeviceApps(id string) ([]client.App, error) {
	c.appsMu.Lock()
	defer c.appsMu.Unlock()
	if a, ok := c.deviceApps[id]; ok {
		return a, nil
	}
	a, err := c.Client.GetDeviceApps(id)
	if err != nil {
		return nil, err
	}
	c.deviceApps[id] = a
	return a, nil
}

// ListBlueprints returns the tenant's blueprints, memoizing the first
// successful API response for the lifetime of the connection.
func (c *IruConnection) ListBlueprints() ([]client.Blueprint, error) {
	c.blueprintsMu.Lock()
	defer c.blueprintsMu.Unlock()
	if c.blueprintsDone {
		return c.blueprintsCached, c.blueprintsErr
	}
	c.blueprintsCached, c.blueprintsErr = c.Client.ListBlueprints()
	c.blueprintsDone = true
	return c.blueprintsCached, c.blueprintsErr
}

// ListUsers returns the tenant's users, memoizing the first successful
// API response for the lifetime of the connection.
func (c *IruConnection) ListUsers() ([]client.User, error) {
	c.usersMu.Lock()
	defer c.usersMu.Unlock()
	if c.usersDone {
		return c.usersCached, c.usersErr
	}
	c.usersCached, c.usersErr = c.Client.ListUsers()
	c.usersDone = true
	return c.usersCached, c.usersErr
}

// ListLibraryItems returns the tenant's library items, memoizing the
// first successful API response for the lifetime of the connection.
func (c *IruConnection) ListLibraryItems() ([]client.LibraryItem, error) {
	c.libraryItemsMu.Lock()
	defer c.libraryItemsMu.Unlock()
	if c.libraryItemsDone {
		return c.libraryItemsCached, c.libraryItemsErr
	}
	c.libraryItemsCached, c.libraryItemsErr = c.Client.ListLibraryItems()
	c.libraryItemsDone = true
	return c.libraryItemsCached, c.libraryItemsErr
}
