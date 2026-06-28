// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"errors"
	"net"
	"net/url"
	"os"
	"strings"
	"sync"

	"github.com/portainer/client-api-go/v2/client"
	"github.com/portainer/client-api-go/v2/pkg/models"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/vault"
)

type PortainerConnection struct {
	plugin.Connection
	Conf       *inventory.Config
	asset      *inventory.Asset
	client     *client.PortainerClient
	instanceID string
	version    string
	hostname   string

	// Instance-wide lists are shared by many resources (e.g. user.teams and
	// team.members both walk memberships/users/teams). Cache them on the
	// connection so they are fetched at most once instead of once per instance.
	cacheMu        sync.Mutex
	users          cachedList[*models.PortainereeUser]
	teams          cachedList[*models.PortainerTeam]
	memberships    cachedList[*models.PortainerTeamMembership]
	tags           cachedList[*models.PortainerTag]
	endpoints      cachedList[*models.PortainereeEndpoint]
	endpointGroups cachedList[*models.PortainerEndpointGroup]
	edgeGroups     cachedList[*models.EdgegroupsDecoratedEdgeGroup]
}

// cachedList memoizes a single API list call (including its error) so callers
// share one result across resource resolutions.
type cachedList[T any] struct {
	items   []T
	err     error
	fetched bool
}

func NewPortainerConnection(id uint32, asset *inventory.Asset, conf *inventory.Config) (*PortainerConnection, error) {
	conn := &PortainerConnection{
		Connection: plugin.NewConnection(id, asset),
		Conf:       conf,
		asset:      asset,
	}

	address := conf.Options[OptionAddress]
	if address == "" {
		address = os.Getenv("PORTAINER_ADDRESS")
	}
	if address == "" {
		return nil, errors.New("a Portainer address is required, pass --address '<url>' or set PORTAINER_ADDRESS")
	}

	// if a secret was provided, it always overrides the env variable since it has precedence
	accessToken := strings.TrimSpace(os.Getenv("PORTAINER_ACCESS_TOKEN"))
	if len(conf.Credentials) > 0 {
		for i := range conf.Credentials {
			cred := conf.Credentials[i]
			if cred.Type == vault.CredentialType_password {
				// trim trailing whitespace/newlines that file- or vault-backed
				// secrets often carry, which would corrupt the x-api-key header
				accessToken = strings.TrimSpace(string(cred.Secret))
				// the first password credential wins; stop so a later one can't
				// silently override it
				break
			}
			log.Warn().Str("credential-type", cred.Type.String()).Msg("unsupported credential type for Portainer provider")
		}
	}
	if accessToken == "" {
		return nil, errors.New("a Portainer access token is required, pass --access-token '<token>' or set PORTAINER_ACCESS_TOKEN")
	}

	host, scheme, basePath, err := parseAddress(address)
	if err != nil {
		return nil, err
	}
	// the bare hostname (no scheme, no port) is used to label the asset
	conn.hostname = host
	if h, _, splitErr := net.SplitHostPort(host); splitErr == nil {
		conn.hostname = h
	}

	opts := []client.ClientOption{
		client.WithScheme(scheme),
		client.WithBasePath(basePath),
	}
	// Honor both the provider --insecure flag and the global --insecure (-k),
	// which the runtime surfaces as conf.Insecure.
	if conf.Insecure || conf.Options[OptionInsecure] == "true" {
		opts = append(opts, client.WithSkipTLSVerify(true))
	}

	cli := client.NewPortainerClient(host, accessToken, opts...)

	// validate credentials early and capture instance metadata for the platform id
	status, err := cli.GetSystemStatus()
	if err != nil {
		return nil, errors.New("failed to connect to Portainer: " + err.Error())
	}

	conn.client = cli
	if status != nil {
		conn.instanceID = status.InstanceID
		conn.version = status.Version
	}
	return conn, nil
}

// parseAddress splits a user-provided address into the host, scheme and base
// path expected by the Portainer client. The client takes a bare host (no
// scheme) plus separate scheme and base-path options.
//
// It is deliberately lenient: both "https://portainer.example.com" and a bare
// "portainer.example.com" are accepted (the scheme defaults to https), as is a
// reverse-proxy path prefix and surrounding whitespace from a pasted value.
func parseAddress(address string) (host, scheme, basePath string, err error) {
	scheme = "https"
	basePath = "/api"

	address = strings.TrimSpace(address)
	if address == "" {
		return "", "", "", errors.New("invalid Portainer address: empty address")
	}

	if !strings.Contains(address, "://") {
		address = scheme + "://" + address
	}
	u, err := url.Parse(address)
	if err != nil {
		return "", "", "", errors.New("invalid Portainer address: " + err.Error())
	}
	if u.Scheme != "" {
		scheme = u.Scheme
	}
	if u.Host == "" {
		return "", "", "", errors.New("invalid Portainer address: missing host")
	}
	if p := strings.TrimRight(u.Path, "/"); p != "" {
		basePath = p
	}
	return u.Host, scheme, basePath, nil
}

func (c *PortainerConnection) Name() string {
	return "portainer"
}

func (c *PortainerConnection) Asset() *inventory.Asset {
	return c.asset
}

func (c *PortainerConnection) Client() *client.PortainerClient {
	return c.client
}

// Users returns all Portainer users, fetched once and cached on the connection.
func (c *PortainerConnection) Users() ([]*models.PortainereeUser, error) {
	c.cacheMu.Lock()
	defer c.cacheMu.Unlock()
	if !c.users.fetched {
		c.users.items, c.users.err = c.client.ListUsers()
		// only memoize a successful result so a transient failure is retried
		// on the next call instead of being cached for the connection lifetime
		if c.users.err == nil {
			c.users.fetched = true
		}
	}
	return c.users.items, c.users.err
}

// Teams returns all Portainer teams, fetched once and cached on the connection.
func (c *PortainerConnection) Teams() ([]*models.PortainerTeam, error) {
	c.cacheMu.Lock()
	defer c.cacheMu.Unlock()
	if !c.teams.fetched {
		c.teams.items, c.teams.err = c.client.ListTeams()
		if c.teams.err == nil {
			c.teams.fetched = true
		}
	}
	return c.teams.items, c.teams.err
}

// TeamMemberships returns all Portainer team memberships, fetched once and
// cached on the connection.
func (c *PortainerConnection) TeamMemberships() ([]*models.PortainerTeamMembership, error) {
	c.cacheMu.Lock()
	defer c.cacheMu.Unlock()
	if !c.memberships.fetched {
		c.memberships.items, c.memberships.err = c.client.ListTeamMemberships()
		if c.memberships.err == nil {
			c.memberships.fetched = true
		}
	}
	return c.memberships.items, c.memberships.err
}

// Tags returns all Portainer tags, fetched once and cached on the connection.
func (c *PortainerConnection) Tags() ([]*models.PortainerTag, error) {
	c.cacheMu.Lock()
	defer c.cacheMu.Unlock()
	if !c.tags.fetched {
		c.tags.items, c.tags.err = c.client.ListTags()
		if c.tags.err == nil {
			c.tags.fetched = true
		}
	}
	return c.tags.items, c.tags.err
}

// Endpoints returns all Portainer environments (endpoints), fetched once and
// cached on the connection.
func (c *PortainerConnection) Endpoints() ([]*models.PortainereeEndpoint, error) {
	c.cacheMu.Lock()
	defer c.cacheMu.Unlock()
	if !c.endpoints.fetched {
		c.endpoints.items, c.endpoints.err = c.client.ListEndpoints()
		if c.endpoints.err == nil {
			c.endpoints.fetched = true
		}
	}
	return c.endpoints.items, c.endpoints.err
}

// EndpointGroups returns all Portainer environment groups (endpoint groups),
// fetched once and cached on the connection.
func (c *PortainerConnection) EndpointGroups() ([]*models.PortainerEndpointGroup, error) {
	c.cacheMu.Lock()
	defer c.cacheMu.Unlock()
	if !c.endpointGroups.fetched {
		c.endpointGroups.items, c.endpointGroups.err = c.client.ListEndpointGroups()
		if c.endpointGroups.err == nil {
			c.endpointGroups.fetched = true
		}
	}
	return c.endpointGroups.items, c.endpointGroups.err
}

// EdgeGroups returns all Portainer edge groups, fetched once and cached on the
// connection.
func (c *PortainerConnection) EdgeGroups() ([]*models.EdgegroupsDecoratedEdgeGroup, error) {
	c.cacheMu.Lock()
	defer c.cacheMu.Unlock()
	if !c.edgeGroups.fetched {
		c.edgeGroups.items, c.edgeGroups.err = c.client.ListEdgeGroups()
		if c.edgeGroups.err == nil {
			c.edgeGroups.fetched = true
		}
	}
	return c.edgeGroups.items, c.edgeGroups.err
}

func (c *PortainerConnection) InstanceID() string {
	return c.instanceID
}

func (c *PortainerConnection) Version() string {
	return c.version
}

// Hostname returns the bare instance hostname (no scheme, no port), used to
// label the asset.
func (c *PortainerConnection) Hostname() string {
	return c.hostname
}
