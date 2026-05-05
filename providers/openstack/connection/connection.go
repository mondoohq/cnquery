// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
)

var PlatformIdOpenstackProject = "//platformid.api.mondoo.app/runtime/openstack/project/"

type OpenstackConnection struct {
	plugin.Connection
	Conf  *inventory.Config
	asset *inventory.Asset

	provider *gophercloud.ProviderClient
	region   string
	authOpts gophercloud.AuthOptions

	clientLock   sync.Mutex
	identity     *gophercloud.ServiceClient
	compute      *gophercloud.ServiceClient
	network      *gophercloud.ServiceClient
	blockStorage *gophercloud.ServiceClient
	image        *gophercloud.ServiceClient
	keyManager   *gophercloud.ServiceClient
	loadBalancer *gophercloud.ServiceClient

	SGNameCacheLock sync.Mutex
	SGNameCache     map[string]string
	SGNameCacheDone bool

	FlavorNameCacheLock sync.Mutex
	FlavorNameCache     map[string]string
	FlavorNameCacheDone bool
}

func NewOpenstackConnection(id uint32, asset *inventory.Asset, conf *inventory.Config) (*OpenstackConnection, error) {
	auth, err := resolveAuth(conf)
	if err != nil {
		return nil, err
	}

	if auth.authOpts.IdentityEndpoint == "" {
		return nil, errors.New("OpenStack auth URL is required (use --auth-url, --cloud, or set OS_AUTH_URL)")
	}

	provider, err := openstack.NewClient(auth.authOpts.IdentityEndpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize OpenStack client: %w", err)
	}
	if conf.Options[OPTION_INSECURE] == "true" {
		provider.HTTPClient = http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec // user-controlled flag for lab/test clouds
			},
		}
	}
	if err := openstack.Authenticate(context.Background(), provider, auth.authOpts); err != nil {
		return nil, fmt.Errorf("failed to authenticate with OpenStack: %w", err)
	}

	return &OpenstackConnection{
		Connection: plugin.NewConnection(id, asset),
		Conf:       conf,
		asset:      asset,
		provider:   provider,
		region:     auth.region,
		authOpts:   auth.authOpts,
	}, nil
}

func (c *OpenstackConnection) Name() string                          { return "openstack" }
func (c *OpenstackConnection) Asset() *inventory.Asset               { return c.asset }
func (c *OpenstackConnection) Provider() *gophercloud.ProviderClient { return c.provider }
func (c *OpenstackConnection) Region() string                        { return c.region }

// AuthURL returns the Keystone endpoint without trailing slashes, suitable for
// embedding into platform IDs.
func (c *OpenstackConnection) AuthURL() string {
	return strings.TrimRight(c.authOpts.IdentityEndpoint, "/")
}

// ProjectID returns the scoped project ID. Empty when the token is not
// project-scoped.
func (c *OpenstackConnection) ProjectID() string {
	if c.authOpts.TenantID != "" {
		return c.authOpts.TenantID
	}
	if c.authOpts.Scope != nil {
		return c.authOpts.Scope.ProjectID
	}
	return ""
}

func (c *OpenstackConnection) endpointOpts() gophercloud.EndpointOpts {
	return gophercloud.EndpointOpts{Region: c.region}
}

func (c *OpenstackConnection) IdentityClient() (*gophercloud.ServiceClient, error) {
	c.clientLock.Lock()
	defer c.clientLock.Unlock()
	if c.identity != nil {
		return c.identity, nil
	}
	client, err := openstack.NewIdentityV3(c.provider, c.endpointOpts())
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Keystone client: %w", err)
	}
	c.identity = client
	return client, nil
}

func (c *OpenstackConnection) ComputeClient() (*gophercloud.ServiceClient, error) {
	c.clientLock.Lock()
	defer c.clientLock.Unlock()
	if c.compute != nil {
		return c.compute, nil
	}
	client, err := openstack.NewComputeV2(c.provider, c.endpointOpts())
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Nova client: %w", err)
	}
	c.compute = client
	return client, nil
}

func (c *OpenstackConnection) NetworkClient() (*gophercloud.ServiceClient, error) {
	c.clientLock.Lock()
	defer c.clientLock.Unlock()
	if c.network != nil {
		return c.network, nil
	}
	client, err := openstack.NewNetworkV2(c.provider, c.endpointOpts())
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Neutron client: %w", err)
	}
	c.network = client
	return client, nil
}

func (c *OpenstackConnection) BlockStorageClient() (*gophercloud.ServiceClient, error) {
	c.clientLock.Lock()
	defer c.clientLock.Unlock()
	if c.blockStorage != nil {
		return c.blockStorage, nil
	}
	client, err := openstack.NewBlockStorageV3(c.provider, c.endpointOpts())
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Cinder client: %w", err)
	}
	c.blockStorage = client
	return client, nil
}

func (c *OpenstackConnection) ImageClient() (*gophercloud.ServiceClient, error) {
	c.clientLock.Lock()
	defer c.clientLock.Unlock()
	if c.image != nil {
		return c.image, nil
	}
	client, err := openstack.NewImageV2(c.provider, c.endpointOpts())
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Glance client: %w", err)
	}
	c.image = client
	return client, nil
}

func (c *OpenstackConnection) KeyManagerClient() (*gophercloud.ServiceClient, error) {
	c.clientLock.Lock()
	defer c.clientLock.Unlock()
	if c.keyManager != nil {
		return c.keyManager, nil
	}
	client, err := openstack.NewKeyManagerV1(c.provider, c.endpointOpts())
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Barbican client: %w", err)
	}
	c.keyManager = client
	return client, nil
}

func (c *OpenstackConnection) LoadBalancerClient() (*gophercloud.ServiceClient, error) {
	c.clientLock.Lock()
	defer c.clientLock.Unlock()
	if c.loadBalancer != nil {
		return c.loadBalancer, nil
	}
	client, err := openstack.NewLoadBalancerV2(c.provider, c.endpointOpts())
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Octavia client: %w", err)
	}
	c.loadBalancer = client
	return client, nil
}
