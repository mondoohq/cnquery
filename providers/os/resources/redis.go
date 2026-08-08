// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"io"
	"strings"
	"sync"

	"github.com/spf13/afero"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/os/connection/shared"
	"go.mondoo.com/mql/v13/providers/os/resources/redisconf"
	"go.mondoo.com/mql/v13/types"
)

// redisConfPaths lists the well-known configuration paths, in the order they
// are probed. Valkey ships the same file under its own name, so both products
// are covered by one probe.
var redisConfPaths = []string{
	"/etc/redis/redis.conf",         // Debian, Ubuntu, and RPM packages
	"/etc/redis.conf",               // RPM and older layouts
	"/etc/valkey/valkey.conf",       // Valkey packages
	"/etc/valkey.conf",              // Valkey, flat layout
	"/opt/homebrew/etc/redis.conf",  // Homebrew on Apple silicon
	"/usr/local/etc/redis.conf",     // Homebrew on Intel
	"/opt/homebrew/etc/valkey.conf", // Homebrew Valkey on Apple silicon
	"/usr/local/etc/valkey.conf",    // Homebrew Valkey on Intel
}

// aferoLoader adapts the connection's filesystem to the include resolution
// the parser performs.
type aferoLoader struct{ afs *afero.Afero }

func (l aferoLoader) Read(path string) (string, error) {
	b, err := l.afs.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (l aferoLoader) Glob(pattern string) ([]string, error) {
	return afero.Glob(l.afs.Fs, pattern)
}

// ---------------------------------------------------------------------------
// redis
// ---------------------------------------------------------------------------

// mqlRedisInternal caches the one binary probe both fields read, so asking
// for version and flavor together runs a single command.
type mqlRedisInternal struct {
	once          sync.Once
	cachedProduct string
	cachedVersion string
}

func (r *mqlRedis) id() (string, error) {
	return "redis", nil
}

// probe runs the server binary to read its banner.
//
// Both products are tried because either may be installed, and the banner
// names the product, so one successful call answers both fields.
func (r *mqlRedis) probe() (string, string) {
	r.once.Do(func() {
		conn, ok := r.MqlRuntime.Connection.(shared.Connection)
		if !ok {
			return
		}
		for _, cmd := range []string{"redis-server --version", "valkey-server --version"} {
			res, err := conn.RunCommand(cmd)
			if err != nil || res.ExitStatus != 0 {
				continue
			}
			data, err := io.ReadAll(res.Stdout)
			if err != nil {
				continue
			}
			product, version := redisconf.ParseVersion(string(data))
			if version != "" {
				r.cachedProduct, r.cachedVersion = product, version
				return
			}
		}
	})
	return r.cachedProduct, r.cachedVersion
}

func (r *mqlRedis) version() (string, error) {
	_, version := r.probe()
	if version == "" {
		r.Version.State = plugin.StateIsSet | plugin.StateIsNull
		return "", nil
	}
	return version, nil
}

func (r *mqlRedis) flavor() (string, error) {
	product, _ := r.probe()
	if product == "" {
		r.Flavor.State = plugin.StateIsSet | plugin.StateIsNull
		return "", nil
	}
	return product, nil
}

// ---------------------------------------------------------------------------
// redis.conf
// ---------------------------------------------------------------------------

// mqlRedisConfInternal caches the parse. Every convenience field reads from
// the resolved directive list rather than from a flat map, so they share one
// parse instead of repeating the include walk each time.
type mqlRedisConfInternal struct {
	once       sync.Once
	cachedConf *redisconf.Conf
	cachedErr  error
}

func initRedisConf(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	x, ok := args["path"]
	if !ok {
		return args, nil, nil
	}
	path, ok := x.Value.(string)
	if !ok {
		return nil, nil, errors.New("wrong type for 'path' in redis.conf initialization, it must be a string")
	}
	f, err := CreateResource(runtime, "file", map[string]*llx.RawData{
		"path": llx.StringData(path),
	})
	if err != nil {
		return nil, nil, err
	}
	args["file"] = llx.ResourceData(f, "file")
	delete(args, "path")
	return args, nil, nil
}

func (r *mqlRedisConf) id() (string, error) {
	file := r.GetFile()
	if file.Error != nil {
		return "", file.Error
	}
	if file.Data == nil {
		return "redis.conf", nil
	}
	return file.Data.Path.Data, nil
}

// file locates the configuration file. It is only reached when the resource
// was not initialized with an explicit path, so this is always the probe path.
func (r *mqlRedisConf) file() (*mqlFile, error) {
	conn, ok := r.MqlRuntime.Connection.(shared.Connection)
	if !ok {
		r.File.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	afs := &afero.Afero{Fs: conn.FileSystem()}

	for _, path := range redisConfPaths {
		if ok, _ := afs.Exists(path); ok {
			f, err := CreateResource(r.MqlRuntime, "file", map[string]*llx.RawData{
				"path": llx.StringData(path),
			})
			if err != nil {
				return nil, err
			}
			return f.(*mqlFile), nil
		}
	}

	// No configuration file anywhere, so neither product is installed. Mark
	// the field set and null so dependent fields report defaults rather than
	// cascading a missing-file error.
	r.File.State = plugin.StateIsSet | plugin.StateIsNull
	return nil, nil
}

// parsed resolves the configuration once, including every file an include
// pulls in.
func (r *mqlRedisConf) parsed(file *mqlFile) (*redisconf.Conf, error) {
	r.once.Do(func() {
		if file == nil {
			r.cachedConf = &redisconf.Conf{}
			return
		}
		conn, ok := r.MqlRuntime.Connection.(shared.Connection)
		if !ok {
			r.cachedConf = &redisconf.Conf{}
			return
		}
		if exists := file.GetExists(); exists.Error != nil || !exists.Data {
			r.cachedConf = &redisconf.Conf{}
			return
		}

		afs := &afero.Afero{Fs: conn.FileSystem()}
		conf, err := redisconf.Load(file.Path.Data, aferoLoader{afs: afs})
		if err != nil {
			r.cachedErr = err
			return
		}
		r.cachedConf = conf
	})
	return r.cachedConf, r.cachedErr
}

// conf narrows parsed to the value the accessors want, reporting an empty
// configuration on error so one unreadable file does not fail every field.
func (r *mqlRedisConf) conf(file *mqlFile) (*redisconf.Conf, error) {
	c, err := r.parsed(file)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return &redisconf.Conf{}, nil
	}
	return c, nil
}

func (r *mqlRedisConf) files(file *mqlFile) ([]any, error) {
	c, err := r.conf(file)
	if err != nil {
		return nil, err
	}
	out := make([]any, 0, len(c.Files))
	for _, path := range c.Files {
		f, err := CreateResource(r.MqlRuntime, "file", map[string]*llx.RawData{
			"path": llx.StringData(path),
		})
		if err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, nil
}

// params flattens the directives to a last-wins map.
//
// It is a convenience view for directives with no field of their own. The
// accumulating directives are lossy here by design, which is why save,
// rename-command, and user each have a field that keeps every occurrence.
func (r *mqlRedisConf) params(file *mqlFile) (map[string]any, error) {
	c, err := r.conf(file)
	if err != nil {
		return nil, err
	}
	out := map[string]any{}
	for _, d := range c.Directives {
		out[strings.ToLower(d.Name)] = strings.Join(d.Args, " ")
	}
	return out, nil
}

func (r *mqlRedisConf) flavor(file *mqlFile) (string, error) {
	c, err := r.conf(file)
	if err != nil {
		return "", err
	}
	if c.IsValkey() {
		return "valkey", nil
	}
	return "redis", nil
}

// network exposure

func (r *mqlRedisConf) port(file *mqlFile) (int64, error) {
	c, err := r.conf(file)
	if err != nil {
		return 0, err
	}
	return c.Port(), nil
}

func (r *mqlRedisConf) bind(file *mqlFile) ([]any, error) {
	c, err := r.conf(file)
	if err != nil {
		return nil, err
	}
	return toAnySlice(c.Bind()), nil
}

func (r *mqlRedisConf) bindsAllInterfaces(file *mqlFile) (bool, error) {
	c, err := r.conf(file)
	if err != nil {
		return false, err
	}
	return c.BindsAllInterfaces(), nil
}

func (r *mqlRedisConf) protectedMode(file *mqlFile) (bool, error) {
	c, err := r.conf(file)
	if err != nil {
		return false, err
	}
	return c.ProtectedMode(), nil
}

func (r *mqlRedisConf) requirepassSet(file *mqlFile) (bool, error) {
	c, err := r.conf(file)
	if err != nil {
		return false, err
	}
	return c.RequirepassSet(), nil
}

func (r *mqlRedisConf) unixSocket(file *mqlFile) (string, error) {
	return r.confString(file, func(c *redisconf.Conf) string { return c.UnixSocket() })
}

func (r *mqlRedisConf) unixSocketPerm(file *mqlFile) (string, error) {
	return r.confString(file, func(c *redisconf.Conf) string { return c.UnixSocketPerm() })
}

// TLS

func (r *mqlRedisConf) tlsPort(file *mqlFile) (int64, error) {
	c, err := r.conf(file)
	if err != nil {
		return 0, err
	}
	return c.TLSPort(), nil
}

func (r *mqlRedisConf) tlsEnabled(file *mqlFile) (bool, error) {
	c, err := r.conf(file)
	if err != nil {
		return false, err
	}
	return c.TLSEnabled(), nil
}

func (r *mqlRedisConf) tlsAuthClients(file *mqlFile) (string, error) {
	return r.confString(file, func(c *redisconf.Conf) string { return c.TLSAuthClients() })
}

func (r *mqlRedisConf) tlsCertFile(file *mqlFile) (string, error) {
	return r.confDirective(file, "tls-cert-file")
}

func (r *mqlRedisConf) tlsKeyFile(file *mqlFile) (string, error) {
	return r.confDirective(file, "tls-key-file")
}

func (r *mqlRedisConf) tlsCaCertFile(file *mqlFile) (string, error) {
	return r.confDirective(file, "tls-ca-cert-file")
}

func (r *mqlRedisConf) tlsProtocols(file *mqlFile) (string, error) {
	return r.confDirective(file, "tls-protocols")
}

func (r *mqlRedisConf) tlsCiphers(file *mqlFile) (string, error) {
	return r.confDirective(file, "tls-ciphers")
}

func (r *mqlRedisConf) tlsCiphersuites(file *mqlFile) (string, error) {
	return r.confDirective(file, "tls-ciphersuites")
}

func (r *mqlRedisConf) tlsReplication(file *mqlFile) (bool, error) {
	return r.confBool(file, "tls-replication", false)
}

func (r *mqlRedisConf) tlsCluster(file *mqlFile) (bool, error) {
	return r.confBool(file, "tls-cluster", false)
}

// access control

func (r *mqlRedisConf) aclFile(file *mqlFile) (string, error) {
	return r.confString(file, func(c *redisconf.Conf) string { return c.ACLFile() })
}

func (r *mqlRedisConf) aclPubsubDefault(file *mqlFile) (string, error) {
	return r.confString(file, func(c *redisconf.Conf) string { return c.ACLPubsubDefault() })
}

func (r *mqlRedisConf) users(file *mqlFile) ([]any, error) {
	c, err := r.conf(file)
	if err != nil {
		return nil, err
	}

	confID, err := r.id()
	if err != nil {
		return nil, err
	}

	out := make([]any, 0, len(c.ACLUsers()))
	for _, u := range c.ACLUsers() {
		res, err := CreateResource(r.MqlRuntime, "redis.conf.user", map[string]*llx.RawData{
			// The user name is unique within a file, and the file path keeps
			// two configurations on one host from colliding.
			"__id":            llx.StringData(confID + "/" + u.Name),
			"name":            llx.StringData(u.Name),
			"isDefault":       llx.BoolData(u.IsDefault),
			"enabled":         llx.BoolData(u.Enabled),
			"nopass":          llx.BoolData(u.Nopass),
			"passwordCount":   llx.IntData(u.PasswordCount),
			"keyPatterns":     llx.ArrayData(toAnySlice(u.KeyPatterns), types.String),
			"channelPatterns": llx.ArrayData(toAnySlice(u.ChannelPatterns), types.String),
			"commandRules":    llx.ArrayData(toAnySlice(u.CommandRules), types.String),
		})
		if err != nil {
			return nil, err
		}
		out = append(out, res)
	}
	return out, nil
}

func (r *mqlRedisConf) renamedCommands(file *mqlFile) (map[string]any, error) {
	c, err := r.conf(file)
	if err != nil {
		return nil, err
	}
	out := map[string]any{}
	for from, to := range c.RenamedCommands() {
		out[from] = to
	}
	return out, nil
}

func (r *mqlRedisConf) disabledCommands(file *mqlFile) ([]any, error) {
	c, err := r.conf(file)
	if err != nil {
		return nil, err
	}
	return toAnySlice(c.DisabledCommands()), nil
}

func (r *mqlRedisConf) enableProtectedConfigs(file *mqlFile) (string, error) {
	return r.confString(file, func(c *redisconf.Conf) string { return c.EnableProtectedConfigs() })
}

func (r *mqlRedisConf) enableDebugCommand(file *mqlFile) (string, error) {
	return r.confString(file, func(c *redisconf.Conf) string { return c.EnableDebugCommand() })
}

func (r *mqlRedisConf) enableModuleCommand(file *mqlFile) (string, error) {
	return r.confString(file, func(c *redisconf.Conf) string { return c.EnableModuleCommand() })
}

// replication

func (r *mqlRedisConf) replicationAuthSet(file *mqlFile) (bool, error) {
	c, err := r.conf(file)
	if err != nil {
		return false, err
	}
	return c.ReplicationAuthSet(), nil
}

func (r *mqlRedisConf) replicationUser(file *mqlFile) (string, error) {
	return r.confString(file, func(c *redisconf.Conf) string { return c.ReplicationUser() })
}

// persistence

func (r *mqlRedisConf) savePoints(file *mqlFile) ([]any, error) {
	c, err := r.conf(file)
	if err != nil {
		return nil, err
	}
	points := c.SavePoints()
	out := make([]any, 0, len(points))
	for _, p := range points {
		out = append(out, map[string]any{
			"seconds": p.Seconds,
			"changes": p.Changes,
		})
	}
	return out, nil
}

func (r *mqlRedisConf) rdbEnabled(file *mqlFile) (bool, error) {
	c, err := r.conf(file)
	if err != nil {
		return false, err
	}
	return c.RDBEnabled(), nil
}

func (r *mqlRedisConf) appendOnly(file *mqlFile) (bool, error) {
	c, err := r.conf(file)
	if err != nil {
		return false, err
	}
	return c.AppendOnly(), nil
}

func (r *mqlRedisConf) appendFsync(file *mqlFile) (string, error) {
	return r.confString(file, func(c *redisconf.Conf) string { return c.AppendFsync() })
}

func (r *mqlRedisConf) maxmemory(file *mqlFile) (int64, error) {
	c, err := r.conf(file)
	if err != nil {
		return 0, err
	}
	return c.Bytes("maxmemory", 0), nil
}

func (r *mqlRedisConf) maxmemoryPolicy(file *mqlFile) (string, error) {
	c, err := r.conf(file)
	if err != nil {
		return "", err
	}
	return c.String("maxmemory-policy", redisconf.DefaultMaxmemoryPolicy), nil
}

// logging and storage

func (r *mqlRedisConf) logfile(file *mqlFile) (string, error) {
	return r.confDirective(file, "logfile")
}

func (r *mqlRedisConf) syslogEnabled(file *mqlFile) (bool, error) {
	return r.confBool(file, "syslog-enabled", false)
}

func (r *mqlRedisConf) dir(file *mqlFile) (string, error) {
	c, err := r.conf(file)
	if err != nil {
		return "", err
	}
	return c.String("dir", redisconf.DefaultDir), nil
}

func (r *mqlRedisConf) dbFilename(file *mqlFile) (string, error) {
	c, err := r.conf(file)
	if err != nil {
		return "", err
	}
	return c.String("dbfilename", redisconf.DefaultDbFilename), nil
}

func (r *mqlRedisConf) daemonize(file *mqlFile) (bool, error) {
	return r.confBool(file, "daemonize", false)
}

func (r *mqlRedisConf) supervised(file *mqlFile) (string, error) {
	c, err := r.conf(file)
	if err != nil {
		return "", err
	}
	return c.String("supervised", "no"), nil
}

func (r *mqlRedisConf) pidFile(file *mqlFile) (string, error) {
	return r.confDirective(file, "pidfile")
}

// confString, confDirective, and confBool keep the repetitive accessors to
// one line each.

func (r *mqlRedisConf) confString(file *mqlFile, read func(*redisconf.Conf) string) (string, error) {
	c, err := r.conf(file)
	if err != nil {
		return "", err
	}
	return read(c), nil
}

func (r *mqlRedisConf) confDirective(file *mqlFile, name string) (string, error) {
	c, err := r.conf(file)
	if err != nil {
		return "", err
	}
	return c.String(name, ""), nil
}

func (r *mqlRedisConf) confBool(file *mqlFile, name string, def bool) (bool, error) {
	c, err := r.conf(file)
	if err != nil {
		return def, err
	}
	return c.Bool(name, def), nil
}

// ---------------------------------------------------------------------------
// redis.conf.user
// ---------------------------------------------------------------------------

func (r *mqlRedisConfUser) id() (string, error) {
	return r.__id, nil
}
