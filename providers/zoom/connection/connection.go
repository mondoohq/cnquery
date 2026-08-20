// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"context"
	"net/url"
	"sync"

	"github.com/cockroachdb/errors"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"

	"go.mondoo.com/mql/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/providers-sdk/v1/plugin"
)

// userIndexPageSize is the page size used when building the account user index.
const userIndexPageSize = 300

// zoomTokenURL is the Server-to-Server OAuth token endpoint.
const zoomTokenURL = "https://zoom.us/oauth/token"

// Platforms is the static catalog of platforms this provider emits. The build
// exports it into dist/zoom.json so the CLI and generated docs can
// list what the provider supports. Add an entry here for every platform your
// asset discovery produces.
var Platforms = []*plugin.PlatformInfo{
	{Name: "zoom", Title: "Zoom", Family: []string{"zoom"}, Kind: []string{"api"}, Runtime: []string{"zoom"}},
}

type ZoomConnection struct {
	plugin.Connection
	Conf      *inventory.Config
	asset     *inventory.Asset
	accountID string
	client    *Client

	userIndexLock sync.Mutex
	userIndex     map[string]*User
}

func NewZoomConnection(id uint32, asset *inventory.Asset, conf *inventory.Config) (*ZoomConnection, error) {
	conn := &ZoomConnection{
		Connection: plugin.NewConnection(id, asset),
		Conf:       conf,
		asset:      asset,
	}

	accountID, ok := GetAccountID(conf)
	if !ok {
		return nil, errors.New("a Zoom account ID is required, set ZOOM_ACCOUNT_ID or use --account-id")
	}
	clientID, ok := GetClientID(conf)
	if !ok {
		return nil, errors.New("a Zoom client ID is required, set ZOOM_CLIENT_ID or use --client-id")
	}
	clientSecret, ok := GetClientSecret(conf)
	if !ok {
		return nil, errors.New("a Zoom client secret is required, set ZOOM_CLIENT_SECRET or use --client-secret")
	}
	conn.accountID = accountID

	oauthCfg := &clientcredentials.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		TokenURL:     zoomTokenURL,
		AuthStyle:    oauth2.AuthStyleInHeader,
		EndpointParams: url.Values{
			"grant_type": {"account_credentials"},
			"account_id": {accountID},
		},
	}
	httpClient := oauthCfg.Client(context.Background())
	conn.client = newClient(httpClient)

	return conn, nil
}

func (c *ZoomConnection) Name() string {
	return "zoom"
}

func (c *ZoomConnection) Asset() *inventory.Asset {
	return c.asset
}

// AccountID returns the Zoom account ID this connection targets.
func (c *ZoomConnection) AccountID() string {
	return c.accountID
}

// Client returns the authenticated Zoom API client.
func (c *ZoomConnection) Client() *Client {
	return c.client
}

// UserIndex returns every active account user keyed by ID, fetching the full
// list at most once per connection. Role and group membership are returned as
// bare user IDs; resolving each through this index turns a role or group with
// N members into a single paginated user list rather than N per-member GetUser
// calls. On error the index is not cached, so a later call can retry.
func (c *ZoomConnection) UserIndex(ctx context.Context) (map[string]*User, error) {
	c.userIndexLock.Lock()
	defer c.userIndexLock.Unlock()
	if c.userIndex != nil {
		return c.userIndex, nil
	}

	index := map[string]*User{}
	nextPageToken := ""
	for {
		list, err := c.client.ListUsers(ctx, userIndexPageSize, nextPageToken)
		if err != nil {
			return nil, err
		}
		for i := range list.Users {
			u := list.Users[i]
			index[u.ID] = &u
		}
		if list.NextPageToken == "" {
			break
		}
		nextPageToken = list.NextPageToken
	}

	c.userIndex = index
	return c.userIndex, nil
}

// Verify validates the Server-to-Server OAuth credentials by issuing the
// cheapest authenticated read against the API.
func (c *ZoomConnection) Verify() error {
	_, err := c.client.GetMe(context.Background())
	if err != nil {
		return errors.Wrap(err, "failed to verify Zoom credentials")
	}
	return nil
}
