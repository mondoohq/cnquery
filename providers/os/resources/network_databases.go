// Copyright Mondoo, Inc. 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"strconv"
	"strings"

	"github.com/spf13/afero"
	"go.mondoo.com/mql/llx"
	"go.mondoo.com/mql/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/providers/os/connection/shared"
)

// network.hosts / network.protocols / network.services expose the OS network
// databases (the hosts, protocols, and services files). They are reached through
// getters on the network resource, e.g. `network.hosts.where(ip == "0.0.0.0")`.

// --- network getters ---

// networkDBResource builds a network-database resource with an explicit id and
// platform-resolved path (mirroring the network.routes() getter pattern).
func (c *mqlNetwork) networkDBResource(name, unixPath, winPath string) (plugin.Resource, error) {
	path := unixPath
	if conn, ok := c.MqlRuntime.Connection.(shared.Connection); ok {
		if pf := conn.Asset().Platform; pf != nil && pf.IsFamily("windows") {
			path = winPath
		}
	}
	return NewResource(c.MqlRuntime, name, map[string]*llx.RawData{
		"__id": llx.StringData(name + ":" + path),
		"path": llx.StringData(path),
	})
}

func (c *mqlNetwork) hosts() (*mqlNetworkHosts, error) {
	r, err := c.networkDBResource("networkHosts", "/etc/hosts", "C:/Windows/System32/drivers/etc/hosts")
	if err != nil {
		return nil, err
	}
	return r.(*mqlNetworkHosts), nil
}

func (c *mqlNetwork) protocols() (*mqlNetworkProtocols, error) {
	r, err := c.networkDBResource("networkProtocols", "/etc/protocols", "C:/Windows/System32/drivers/etc/protocol")
	if err != nil {
		return nil, err
	}
	return r.(*mqlNetworkProtocols), nil
}

func (c *mqlNetwork) services() (*mqlNetworkServices, error) {
	r, err := c.networkDBResource("networkServices", "/etc/services", "C:/Windows/System32/drivers/etc/services")
	if err != nil {
		return nil, err
	}
	return r.(*mqlNetworkServices), nil
}

// --- shared helpers ---

// initNetworkDB defaults the `path` for a network-database resource: the Unix
// path, or the Windows path when the target is Windows.
func initNetworkDB(runtime *plugin.Runtime, args map[string]*llx.RawData, unixPath, winPath string) (map[string]*llx.RawData, plugin.Resource, error) {
	if _, ok := args["path"]; !ok {
		path := unixPath
		if conn, ok := runtime.Connection.(shared.Connection); ok {
			if pf := conn.Asset().Platform; pf != nil && pf.IsFamily("windows") {
				path = winPath
			}
		}
		args["path"] = llx.StringData(path)
	}
	return args, nil, nil
}

// readNetworkDB reads a database file's raw content through the connection's
// filesystem (works over local, ssh, winrm, container, and mounted-image
// transports). A missing file resolves to exists=false rather than an error.
func readNetworkDB(runtime *plugin.Runtime, path string) (content string, exists bool, err error) {
	conn, ok := runtime.Connection.(shared.Connection)
	if !ok {
		return "", false, errors.New("no filesystem connection available")
	}
	afs := &afero.Afero{Fs: conn.FileSystem()}
	ok, err = afs.Exists(path)
	if err != nil || !ok {
		return "", false, nil
	}
	b, err := afs.ReadFile(path)
	return string(b), true, err
}

func networkDBFile(runtime *plugin.Runtime, path string) (*mqlFile, error) {
	f, err := CreateResource(runtime, "file", map[string]*llx.RawData{"path": llx.StringData(path)})
	if err != nil {
		return nil, err
	}
	return f.(*mqlFile), nil
}

// etcRecord is one meaningful (non-comment) line of an /etc-style table.
type etcRecord struct {
	line    int
	fields  []string
	comment string
}

// parseEtcTable strips `#` comments and splits each remaining line into
// whitespace-delimited fields, preserving 1-based line numbers. Blank and
// comment-only lines are dropped.
func parseEtcTable(content string) []etcRecord {
	var out []etcRecord
	for i, raw := range strings.Split(content, "\n") {
		body := raw
		comment := ""
		if idx := strings.IndexByte(body, '#'); idx >= 0 {
			comment = strings.TrimSpace(body[idx+1:])
			body = body[:idx]
		}
		fields := strings.Fields(body)
		if len(fields) == 0 {
			continue
		}
		out = append(out, etcRecord{line: i + 1, fields: fields, comment: comment})
	}
	return out
}

// --- network.hosts ---

func initNetworkHosts(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	return initNetworkDB(runtime, args, "/etc/hosts", "C:/Windows/System32/drivers/etc/hosts")
}

func (x *mqlNetworkHosts) id() (string, error)     { return "networkHosts:" + x.Path.Data, nil }
func (x *mqlNetworkHosts) file() (*mqlFile, error) { return networkDBFile(x.MqlRuntime, x.Path.Data) }
func (x *mqlNetworkHosts) content() (string, error) {
	content, _, err := readNetworkDB(x.MqlRuntime, x.Path.Data)
	return content, err
}

func (x *mqlNetworkHosts) list() ([]any, error) {
	res := []any{}
	content, exists, err := readNetworkDB(x.MqlRuntime, x.Path.Data)
	if err != nil || !exists {
		return res, err
	}
	for _, r := range parseEtcTable(content) {
		entry, err := CreateResource(x.MqlRuntime, "networkHosts.entry", map[string]*llx.RawData{
			"line":      llx.IntData(int64(r.line)),
			"ip":        llx.StringData(r.fields[0]),
			"hostnames": llx.ArrayData(llx.TArr2Raw[string](r.fields[1:]), "string"),
			"comment":   llx.StringData(r.comment),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, entry)
	}
	return res, nil
}

func (x *mqlNetworkHostsEntry) id() (string, error) {
	return "networkHosts.entry/" + strconv.FormatInt(x.Line.Data, 10) + "/" + x.Ip.Data, nil
}

// --- network.protocols ---

func initNetworkProtocols(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	return initNetworkDB(runtime, args, "/etc/protocols", "C:/Windows/System32/drivers/etc/protocol")
}

func (x *mqlNetworkProtocols) id() (string, error) { return "networkProtocols:" + x.Path.Data, nil }
func (x *mqlNetworkProtocols) file() (*mqlFile, error) {
	return networkDBFile(x.MqlRuntime, x.Path.Data)
}
func (x *mqlNetworkProtocols) content() (string, error) {
	content, _, err := readNetworkDB(x.MqlRuntime, x.Path.Data)
	return content, err
}

func (x *mqlNetworkProtocols) list() ([]any, error) {
	res := []any{}
	content, exists, err := readNetworkDB(x.MqlRuntime, x.Path.Data)
	if err != nil || !exists {
		return res, err
	}
	for _, r := range parseEtcTable(content) {
		if len(r.fields) < 2 {
			continue
		}
		number, _ := strconv.Atoi(r.fields[1])
		entry, err := CreateResource(x.MqlRuntime, "networkProtocols.entry", map[string]*llx.RawData{
			"line":    llx.IntData(int64(r.line)),
			"name":    llx.StringData(r.fields[0]),
			"number":  llx.IntData(int64(number)),
			"aliases": llx.ArrayData(llx.TArr2Raw[string](r.fields[2:]), "string"),
			"comment": llx.StringData(r.comment),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, entry)
	}
	return res, nil
}

func (x *mqlNetworkProtocolsEntry) id() (string, error) {
	return "networkProtocols.entry/" + strconv.FormatInt(x.Line.Data, 10) + "/" + x.Name.Data, nil
}

// --- network.services ---

func initNetworkServices(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	return initNetworkDB(runtime, args, "/etc/services", "C:/Windows/System32/drivers/etc/services")
}

func (x *mqlNetworkServices) id() (string, error) { return "networkServices:" + x.Path.Data, nil }
func (x *mqlNetworkServices) file() (*mqlFile, error) {
	return networkDBFile(x.MqlRuntime, x.Path.Data)
}
func (x *mqlNetworkServices) content() (string, error) {
	content, _, err := readNetworkDB(x.MqlRuntime, x.Path.Data)
	return content, err
}

func (x *mqlNetworkServices) list() ([]any, error) {
	res := []any{}
	content, exists, err := readNetworkDB(x.MqlRuntime, x.Path.Data)
	if err != nil || !exists {
		return res, err
	}
	for _, r := range parseEtcTable(content) {
		if len(r.fields) < 2 {
			continue
		}
		port, protocol := 0, ""
		if slash := strings.IndexByte(r.fields[1], '/'); slash >= 0 {
			port, _ = strconv.Atoi(r.fields[1][:slash])
			protocol = r.fields[1][slash+1:]
		} else {
			port, _ = strconv.Atoi(r.fields[1])
		}
		entry, err := CreateResource(x.MqlRuntime, "networkServices.entry", map[string]*llx.RawData{
			"line":     llx.IntData(int64(r.line)),
			"name":     llx.StringData(r.fields[0]),
			"port":     llx.IntData(int64(port)),
			"protocol": llx.StringData(protocol),
			"aliases":  llx.ArrayData(llx.TArr2Raw[string](r.fields[2:]), "string"),
			"comment":  llx.StringData(r.comment),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, entry)
	}
	return res, nil
}

func (x *mqlNetworkServicesEntry) id() (string, error) {
	return "networkServices.entry/" + strconv.FormatInt(x.Line.Data, 10) + "/" + x.Name.Data + "/" +
		strconv.FormatInt(x.Port.Data, 10) + "/" + x.Protocol.Data, nil
}
