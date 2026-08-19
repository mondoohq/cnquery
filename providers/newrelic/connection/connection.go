// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"errors"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	mondoovault "go.mondoo.com/mql/v13/providers-sdk/v1/vault"
)

const (
	// requestTimeout bounds a single NerdGraph call. New Relic answers a
	// paginated page quickly, so a slow response means the API is unhealthy
	// rather than the page being large.
	requestTimeout = 60 * time.Second

	// OptionAccountID names the numeric New Relic account to scan. Keys, alert
	// policies, drop rules and retention rules all belong to one account, so it
	// is required rather than inferred.
	OptionAccountID = "account-id"
	// OptionRegion selects the data region the account lives in, which decides
	// the API host. An account in the EU region is unreachable from the US host.
	OptionRegion = "region"

	// RegionUS is the default New Relic data region.
	RegionUS = "us"
	// RegionEU is the European data region, served from a separate API host.
	RegionEU = "eu"

	endpointUS = "https://api.newrelic.com/graphql"
	endpointEU = "https://api.eu.newrelic.com/graphql"
)

// NewrelicConnection holds an authenticated NerdGraph client for one New Relic
// account.
type NewrelicConnection struct {
	plugin.Connection
	Conf  *inventory.Config
	asset *inventory.Asset

	client    *Client
	accountID int
	region    string

	cache sync.Map
}

// NormalizeRegion maps a user-supplied region onto the two New Relic data
// regions and their API hosts. An unrecognized region is refused rather than
// silently defaulted to the US host, which would report an EU account as empty.
func NormalizeRegion(region string) (normalized string, endpoint string, err error) {
	switch strings.ToLower(strings.TrimSpace(region)) {
	case "", RegionUS:
		return RegionUS, endpointUS, nil
	case RegionEU:
		return RegionEU, endpointEU, nil
	default:
		return "", "", errors.New("unknown New Relic region " + strconv.Quote(region) + ", expected \"us\" or \"eu\"")
	}
}

func NewNewrelicConnection(id uint32, asset *inventory.Asset, conf *inventory.Config) (*NewrelicConnection, error) {
	conn := &NewrelicConnection{
		Connection: plugin.NewConnection(id, asset),
		Conf:       conf,
		asset:      asset,
	}

	apiKey := credentialFromConf(conf)
	if apiKey == "" {
		apiKey = os.Getenv("NEW_RELIC_API_KEY")
	}
	if apiKey == "" {
		return nil, errors.New("a New Relic user API key is required (set NEW_RELIC_API_KEY or use --api-key)")
	}

	rawAccountID := option(conf, OptionAccountID)
	if rawAccountID == "" {
		rawAccountID = os.Getenv("NEW_RELIC_ACCOUNT_ID")
	}
	if rawAccountID == "" {
		return nil, errors.New("a New Relic account ID is required (set NEW_RELIC_ACCOUNT_ID or use --account-id)")
	}
	accountID, err := strconv.Atoi(strings.TrimSpace(rawAccountID))
	if err != nil {
		return nil, errors.New("the New Relic account ID must be numeric, got " + strconv.Quote(rawAccountID))
	}
	if accountID <= 0 {
		return nil, errors.New("the New Relic account ID must be a positive number, got " + strconv.Quote(rawAccountID))
	}
	conn.accountID = accountID

	rawRegion := option(conf, OptionRegion)
	if rawRegion == "" {
		rawRegion = os.Getenv("NEW_RELIC_REGION")
	}
	region, endpoint, err := NormalizeRegion(rawRegion)
	if err != nil {
		return nil, err
	}
	conn.region = region

	conn.client = NewClient(&http.Client{Timeout: requestTimeout}, endpoint, apiKey)

	return conn, nil
}

// credentialFromConf pulls the user API key out of the configured credentials.
func credentialFromConf(conf *inventory.Config) string {
	if conf == nil {
		return ""
	}
	for _, cred := range conf.Credentials {
		if cred == nil || len(cred.Secret) == 0 {
			continue
		}
		if cred.Type != mondoovault.CredentialType_password {
			continue
		}
		return string(cred.Secret)
	}
	return ""
}

func option(conf *inventory.Config, key string) string {
	if conf == nil || conf.Options == nil {
		return ""
	}
	return conf.Options[key]
}

func (c *NewrelicConnection) Name() string { return "newrelic" }

func (c *NewrelicConnection) Asset() *inventory.Asset { return c.asset }

// Client returns the authenticated NerdGraph client.
func (c *NewrelicConnection) Client() *Client { return c.client }

// AccountID is the numeric account this connection is scoped to.
func (c *NewrelicConnection) AccountID() int { return c.accountID }

// Region is the normalized data region, either "us" or "eu".
func (c *NewrelicConnection) Region() string { return c.region }

// cacheEntry memoizes one fetch, including its failure. Recording the error
// alongside the value matters: without it a failed lookup would be retried by
// every caller, and a permission failure would turn into a burst of requests.
type cacheEntry struct {
	once sync.Once
	val  any
	err  error
}

// Memoize runs fn at most once per key for the life of the connection and hands
// every later caller the same result. It is what keeps a typed reference from
// costing one API call per row.
func (c *NewrelicConnection) Memoize(key string, fn func() (any, error)) (any, error) {
	stored, _ := c.cache.LoadOrStore(key, &cacheEntry{})
	entry := stored.(*cacheEntry)
	entry.once.Do(func() {
		entry.val, entry.err = fn()
	})
	return entry.val, entry.err
}
