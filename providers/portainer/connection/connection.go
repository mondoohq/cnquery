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
	// Each cachedList owns its mutex, so independent fetches run in parallel.
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
	mu      sync.Mutex
	items   []T
	err     error
	fetched bool
}

// get fetches and memoizes the list on first call. Only a successful result is
// cached, so a transient failure is retried on the next call.
func (cl *cachedList[T]) get(fetch func() ([]T, error)) ([]T, error) {
	cl.mu.Lock()
	defer cl.mu.Unlock()
	if !cl.fetched {
		cl.items, cl.err = fetch()
		if cl.err == nil {
			cl.fetched = true
		}
	}
	return cl.items, cl.err
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

	// reach the instance early and capture its metadata for the platform id
	status, err := cli.GetSystemStatus()
	if err != nil {
		return nil, errors.New("failed to connect to Portainer: " + err.Error())
	}

	conn.client = cli
	if status != nil {
		conn.instanceID = status.InstanceID
		conn.version = status.Version
	}

	// The system status endpoint is public, so reaching it proves the address
	// points at a Portainer instance but says nothing about the access token.
	// Make one authenticated call so an invalid or expired token fails here,
	// with a message that names the cause, instead of surfacing later as an
	// unattributed error on every single resource. The result is memoized on
	// the connection, so discovery and portainer.environments reuse it.
	if _, err := conn.Endpoints(); err != nil {
		return nil, errors.New("failed to authenticate against Portainer, check the access token: " + err.Error())
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
		// a path prefix is a reverse-proxy mount point; the Portainer API still
		// lives under <prefix>/api, so append it rather than replace the default
		basePath = p + "/api"
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
	return c.users.get(c.client.ListUsers)
}

// Teams returns all Portainer teams, fetched once and cached on the connection.
func (c *PortainerConnection) Teams() ([]*models.PortainerTeam, error) {
	return c.teams.get(c.client.ListTeams)
}

// TeamMemberships returns all Portainer team memberships, fetched once and
// cached on the connection.
func (c *PortainerConnection) TeamMemberships() ([]*models.PortainerTeamMembership, error) {
	return c.memberships.get(c.client.ListTeamMemberships)
}

// Tags returns all Portainer tags, fetched once and cached on the connection.
func (c *PortainerConnection) Tags() ([]*models.PortainerTag, error) {
	return c.tags.get(c.client.ListTags)
}

// Endpoints returns all Portainer environments (endpoints), fetched once and
// cached on the connection.
func (c *PortainerConnection) Endpoints() ([]*models.PortainereeEndpoint, error) {
	return c.endpoints.get(c.client.ListEndpoints)
}

// EndpointGroups returns all Portainer environment groups (endpoint groups),
// fetched once and cached on the connection.
func (c *PortainerConnection) EndpointGroups() ([]*models.PortainerEndpointGroup, error) {
	return c.endpointGroups.get(c.client.ListEndpointGroups)
}

// EdgeGroups returns all Portainer edge groups, fetched once and cached on the
// connection.
func (c *PortainerConnection) EdgeGroups() ([]*models.EdgegroupsDecoratedEdgeGroup, error) {
	return c.edgeGroups.get(c.client.ListEdgeGroups)
}

func (c *PortainerConnection) InstanceID() string {
	return c.instanceID
}

// InstanceKey returns the identifier platform ids are built from. It prefers
// the instance id reported by the server and falls back to the hostname, so
// that instances which report no id do not all collapse onto the same platform
// id (and therefore onto the same asset).
func (c *PortainerConnection) InstanceKey() string {
	if c.instanceID != "" {
		return c.instanceID
	}
	if c.hostname != "" {
		return c.hostname
	}
	return "unknown"
}

func (c *PortainerConnection) Version() string {
	return c.version
}

// Hostname returns the bare instance hostname (no scheme, no port), used to
// label the asset.
func (c *PortainerConnection) Hostname() string {
	return c.hostname
}
