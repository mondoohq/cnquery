// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"context"
	"net"
	"net/url"
	"strconv"
	"strings"
	"sync"

	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/vault"
	"go.mondoo.com/ranger-rpc/codes"
	"go.mondoo.com/ranger-rpc/status"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// MongoConnection holds the settings to reach a self-hosted MongoDB server and,
// once dialed, the shared client. A single client can query every database, so
// there is one shared handle rather than a pool per database.
type MongoConnection struct {
	plugin.Connection
	Conf  *inventory.Config
	asset *inventory.Asset

	host        string // hostname, or a full mongodb:// connection string
	port        int
	user        string
	password    string
	authDB      string
	tls         bool
	tlsInsecure bool

	// scopedDatabase is set when the asset is a single discovered database.
	scopedDatabase string

	clientOnce sync.Once
	client     *mongo.Client
	clientErr  error
}

func NewMongoConnection(id uint32, asset *inventory.Asset, conf *inventory.Config) (*MongoConnection, error) {
	conn := &MongoConnection{
		Connection: plugin.NewConnection(id, asset),
		Conf:       conf,
		asset:      asset,
	}

	if conf.Options == nil {
		conf.Options = make(map[string]string)
	}

	conn.host = conf.Options[OptionHost]
	if conn.host == "" {
		conn.host = conf.Host
	}
	conn.authDB = conf.Options[OptionAuthDB]
	if conn.authDB == "" {
		conn.authDB = "admin"
	}
	conn.scopedDatabase = conf.Options[OptionScopedDatabase]
	conn.tls = conf.Options[OptionTLS] == "true"
	conn.tlsInsecure = conf.Options[OptionTLSInsecure] == "true"

	conn.port = 27017
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
		return nil, status.Error(codes.InvalidArgument, "missing host for mongo connection")
	}

	return conn, nil
}

func (c *MongoConnection) Name() string {
	return "mongo"
}

func (c *MongoConnection) Asset() *inventory.Asset {
	return c.asset
}

// ScopedDatabase returns the single database this asset is scoped to, or an
// empty string when the asset is the whole server.
func (c *MongoConnection) ScopedDatabase() string {
	return c.scopedDatabase
}

// ServerID returns a stable identifier for the server, used to build asset
// platform ids. When the host is a full connection string it is used verbatim;
// otherwise it is host:port.
func (c *MongoConnection) ServerID() string {
	if strings.HasPrefix(c.host, "mongodb://") || strings.HasPrefix(c.host, "mongodb+srv://") {
		return c.host
	}
	return net.JoinHostPort(c.host, strconv.Itoa(c.port))
}

// uri builds a mongodb:// connection string from the resolved settings, unless
// the host is already a full connection string.
func (c *MongoConnection) uri() string {
	if strings.HasPrefix(c.host, "mongodb://") || strings.HasPrefix(c.host, "mongodb+srv://") {
		return c.host
	}
	q := url.Values{}
	q.Set("authSource", c.authDB)
	if c.tls {
		q.Set("tls", "true")
		if c.tlsInsecure {
			q.Set("tlsInsecure", "true")
		}
	}
	u := &url.URL{
		Scheme:   "mongodb",
		Host:     net.JoinHostPort(c.host, strconv.Itoa(c.port)),
		Path:     "/",
		RawQuery: q.Encode(),
	}
	if c.user != "" {
		u.User = url.UserPassword(c.user, c.password)
	}
	return u.String()
}

// Client returns the shared MongoDB client, dialing on first use.
func (c *MongoConnection) Client() (*mongo.Client, error) {
	c.clientOnce.Do(func() {
		client, err := mongo.Connect(options.Client().ApplyURI(c.uri()))
		if err != nil {
			c.clientErr = err
			return
		}
		c.client = client
	})
	return c.client, c.clientErr
}

// Close disconnects the shared client.
func (c *MongoConnection) Close() {
	if c.client != nil {
		_ = c.client.Disconnect(context.Background())
	}
}

// RunAdminCommand runs a command against the admin database and decodes the
// result into out.
func (c *MongoConnection) RunAdminCommand(cmd any, out any) error {
	return c.RunCommand("admin", cmd, out)
}

// RunCommand runs a command against the named database and decodes the result
// into out.
func (c *MongoConnection) RunCommand(db string, cmd any, out any) error {
	client, err := c.Client()
	if err != nil {
		return err
	}
	return client.Database(db).RunCommand(context.Background(), cmd).Decode(out)
}
