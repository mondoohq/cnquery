// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"context"
	"net"
	"net/url"
	"strconv"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/vault"
	"go.mondoo.com/ranger-rpc/codes"
	"go.mondoo.com/ranger-rpc/status"
)

// PostgresConnection holds the settings needed to reach a PostgreSQL server.
// Because PostgreSQL cannot query across databases, it keeps one connection
// pool per database, opened lazily and shared across resources.
type PostgresConnection struct {
	plugin.Connection
	Conf  *inventory.Config
	asset *inventory.Asset

	host        string
	port        int
	user        string
	password    string
	database    string // database used for the server-level connection
	sslmode     string
	sslrootcert string
	sslcert     string
	sslkey      string

	// scopedDatabase is set when the asset is a single discovered database.
	scopedDatabase string

	poolsMu sync.Mutex
	pools   map[string]*pgxpool.Pool

	systemIDOnce sync.Once
	systemID     string
	systemIDErr  error
}

func NewPostgresConnection(id uint32, asset *inventory.Asset, conf *inventory.Config) (*PostgresConnection, error) {
	conn := &PostgresConnection{
		Connection: plugin.NewConnection(id, asset),
		Conf:       conf,
		asset:      asset,
		pools:      map[string]*pgxpool.Pool{},
	}

	if conf.Options == nil {
		conf.Options = make(map[string]string)
	}

	conn.host = conf.Options[OptionHost]
	if conn.host == "" {
		conn.host = conf.Host
	}
	conn.database = conf.Options[OptionDatabase]
	if conn.database == "" {
		conn.database = "postgres"
	}
	conn.scopedDatabase = conf.Options[OptionScopedDatabase]
	conn.sslmode = conf.Options[OptionSSLMode]
	if conn.sslmode == "" {
		conn.sslmode = "prefer"
	}
	conn.sslrootcert = conf.Options[OptionSSLRootCert]
	conn.sslcert = conf.Options[OptionSSLCert]
	conn.sslkey = conf.Options[OptionSSLKey]

	conn.port = 5432
	if p := conf.Options[OptionPort]; p != "" {
		v, err := strconv.Atoi(p)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid port %q: %v", p, err)
		}
		conn.port = v
	} else if conf.Port > 0 {
		conn.port = int(conf.Port)
	}

	for i := range conf.Credentials {
		cred := conf.Credentials[i]
		if cred.Type == vault.CredentialType_password {
			conn.user = cred.User
			conn.password = string(cred.Secret)
		}
	}

	if conn.host == "" {
		return nil, status.Error(codes.InvalidArgument, "missing host for postgres connection")
	}
	if conn.user == "" {
		return nil, status.Error(codes.InvalidArgument, "missing user for postgres connection")
	}

	return conn, nil
}

func (c *PostgresConnection) Name() string {
	return "postgres"
}

func (c *PostgresConnection) Asset() *inventory.Asset {
	return c.asset
}

// ScopedDatabase returns the single database this asset is scoped to, or an
// empty string when the asset is the whole server.
func (c *PostgresConnection) ScopedDatabase() string {
	return c.scopedDatabase
}

// connString builds a postgres:// URL for a specific database.
func (c *PostgresConnection) connString(database string) string {
	query := url.Values{}
	query.Set("sslmode", c.sslmode)
	if c.sslrootcert != "" {
		query.Set("sslrootcert", c.sslrootcert)
	}
	if c.sslcert != "" {
		query.Set("sslcert", c.sslcert)
	}
	if c.sslkey != "" {
		query.Set("sslkey", c.sslkey)
	}
	u := &url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(c.user, c.password),
		Host:     net.JoinHostPort(c.host, strconv.Itoa(c.port)),
		Path:     "/" + database,
		RawQuery: query.Encode(),
	}
	return u.String()
}

// Client returns a connection pool for the given database, opening it on first
// use. An empty database name resolves to the server-level database.
func (c *PostgresConnection) Client(database string) (*pgxpool.Pool, error) {
	if database == "" {
		database = c.database
	}

	c.poolsMu.Lock()
	defer c.poolsMu.Unlock()
	if pool, ok := c.pools[database]; ok {
		return pool, nil
	}

	pool, err := pgxpool.New(context.Background(), c.connString(database))
	if err != nil {
		return nil, err
	}
	c.pools[database] = pool
	return pool, nil
}

// Close releases every open pool.
func (c *PostgresConnection) Close() {
	c.poolsMu.Lock()
	defer c.poolsMu.Unlock()
	for _, pool := range c.pools {
		pool.Close()
	}
	c.pools = map[string]*pgxpool.Pool{}
}

// SystemID returns the cluster system identifier, used to build stable asset
// platform ids. It is resolved once and shared.
func (c *PostgresConnection) SystemID() (string, error) {
	c.systemIDOnce.Do(func() {
		pool, err := c.Client("")
		if err != nil {
			c.systemIDErr = err
			return
		}
		c.systemIDErr = pool.QueryRow(context.Background(),
			"SELECT system_identifier::text FROM pg_control_system()").Scan(&c.systemID)
	})
	return c.systemID, c.systemIDErr
}
