// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/spf13/afero"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/os/connection/shared"
	"go.mondoo.com/mql/v13/providers/os/resources/nginx"
	"go.mondoo.com/mql/v13/types"
)

// nginxVersionBinaries lists the well-known binary paths for the nginx server.
// The version string (e.g. "nginx/1.25.3") is embedded as a constant in the
// binary, so we can extract it by reading the file directly — no command
// execution required.
var nginxVersionBinaries = []string{
	"/usr/sbin/nginx",
	"/usr/local/sbin/nginx",
	"/usr/local/bin/nginx",
	"/usr/bin/nginx",
}

var nginxVersionTag = []byte("nginx/")

func (n *mqlNginx) version() (string, error) {
	conn := n.MqlRuntime.Connection.(shared.Connection)
	afs := &afero.Afero{Fs: conn.FileSystem()}

	// Prefer file-based detection: scan the nginx binary for the embedded
	// "nginx/x.y.z" version string without loading the full binary into memory.
	for _, bin := range nginxVersionBinaries {
		if v := scanBinaryForTag(afs, bin, nginxVersionTag); v != "" {
			return v, nil
		}
	}

	// Fall back to running a command when the binary isn't readable (e.g.
	// non-standard install path). We use lowercase -v (not -V) because it
	// prints only the version line and is cheaper than -V which also dumps
	// all configure arguments.
	cmd, err := conn.RunCommand("nginx -v 2>&1")
	if err == nil && cmd.ExitStatus == 0 {
		data, err := io.ReadAll(cmd.Stdout)
		if err == nil {
			if m := reNginxVersion.FindSubmatch(data); m != nil {
				return string(m[1]), nil
			}
		}
	}

	// Nginx is likely not installed; return nil rather than an error.
	n.Version = plugin.TValue[string]{State: plugin.StateIsSet | plugin.StateIsNull}
	return "", nil
}

func (n *mqlNginx) modules() ([]any, error) {
	conn := n.MqlRuntime.Connection.(shared.Connection)

	// Modules require "nginx -V" output (configure arguments are not in the binary).
	cmd, err := conn.RunCommand("nginx -V 2>&1")
	if err != nil {
		n.Modules = plugin.TValue[[]any]{State: plugin.StateIsSet | plugin.StateIsNull}
		return nil, nil
	}
	if cmd.ExitStatus != 0 {
		n.Modules = plugin.TValue[[]any]{State: plugin.StateIsSet | plugin.StateIsNull}
		return nil, nil
	}

	data, err := io.ReadAll(cmd.Stdout)
	if err != nil {
		return nil, err
	}

	modules := parseNginxModules(string(data))
	modulesData := make([]any, len(modules))
	for i, m := range modules {
		modulesData[i] = m
	}
	return modulesData, nil
}

// reNginxVersion matches "nginx version: nginx/1.25.3" or "nginx/1.25.3".
var reNginxVersion = regexp.MustCompile(`nginx/(\S+)`)

// reNginxModule matches --with-*_module flags in configure arguments.
var reNginxModule = regexp.MustCompile(`--with-(\S+_module)`)

// parseNginxModules extracts compiled-in module names from nginx -V output.
func parseNginxModules(output string) []string {
	matches := reNginxModule.FindAllStringSubmatch(output, -1)
	modules := make([]string, 0, len(matches))
	for _, m := range matches {
		modules = append(modules, m[1])
	}
	return modules
}

type mqlNginxConfInternal struct {
	lock sync.Mutex
}

// nginxConfPaths maps platform names to their default nginx config location.
var nginxConfPaths = map[string]string{
	"freebsd":      "/usr/local/etc/nginx/nginx.conf",
	"dragonflybsd": "/usr/local/etc/nginx/nginx.conf",
	"openbsd":      "/etc/nginx/nginx.conf",
	"netbsd":       "/usr/pkg/etc/nginx/nginx.conf",
}

const defaultNginxConf = "/etc/nginx/nginx.conf"

func nginxConfPath(conn shared.Connection) string {
	asset := conn.Asset()
	if asset != nil && asset.Platform != nil {
		if p, ok := nginxConfPaths[asset.Platform.Name]; ok {
			return p
		}
		for _, family := range asset.Platform.Family {
			if p, ok := nginxConfPaths[family]; ok {
				return p
			}
		}
	}
	return defaultNginxConf
}

func initNginxConf(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if x, ok := args["path"]; ok {
		path, ok := x.Value.(string)
		if !ok {
			return nil, nil, errors.New("wrong type for 'path' in nginx.conf initialization, it must be a string")
		}

		f, err := CreateResource(runtime, "file", map[string]*llx.RawData{
			"path": llx.StringData(path),
		})
		if err != nil {
			return nil, nil, err
		}
		args["file"] = llx.ResourceData(f, "file")
		delete(args, "path")
	}

	return args, nil, nil
}

func (s *mqlNginxConf) id() (string, error) {
	file := s.GetFile()
	if file.Error != nil {
		return "", file.Error
	}
	return file.Data.Path.Data, nil
}

func (s *mqlNginxConf) file() (*mqlFile, error) {
	conn := s.MqlRuntime.Connection.(shared.Connection)
	path := nginxConfPath(conn)

	f, err := CreateResource(s.MqlRuntime, "file", map[string]*llx.RawData{
		"path": llx.StringData(path),
	})
	if err != nil {
		return nil, err
	}
	return f.(*mqlFile), nil
}

// reNginxGlob detects filepath glob meta-characters.
var reNginxGlob = regexp.MustCompile(`[*?\[]`)

// expandNginxGlob walks the connection's filesystem to expand an include
// pattern. Matches afero-backed layouts (including serialized asset
// snapshots) — filepath.Glob cannot be used because it hits the host FS.
func (s *mqlNginxConf) expandNginxGlob(pattern string) ([]string, error) {
	conn := s.MqlRuntime.Connection.(shared.Connection)

	if !reNginxGlob.MatchString(pattern) {
		return []string{pattern}, nil
	}

	var paths []string
	segments := strings.Split(pattern, "/")
	if segments[0] == "" {
		paths = []string{"/"}
	}

	afs := &afero.Afero{Fs: conn.FileSystem()}

	for _, segment := range segments[1:] {
		if !reNginxGlob.MatchString(segment) {
			for i := range paths {
				paths[i] = filepath.Join(paths[i], segment)
			}
			continue
		}

		var nuPaths []string
		for _, path := range paths {
			files, err := afs.ReadDir(path)
			if err != nil {
				if os.IsNotExist(err) {
					continue
				}
				return nil, err
			}
			for j := range files {
				name := files[j].Name()
				ok, err := filepath.Match(segment, name)
				if err != nil {
					return nil, err
				}
				if ok {
					nuPaths = append(nuPaths, filepath.Join(path, name))
				}
			}
		}
		paths = nuPaths
	}

	return paths, nil
}

// parse is the central method that invokes the nginx parser, then walks
// the resulting directive tree to populate all fields.
func (s *mqlNginxConf) parse(file *mqlFile) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	if s.Params.State == plugin.StateIsSet {
		return nil
	}

	if file == nil {
		return errors.New("no base nginx config file to read")
	}

	conn := s.MqlRuntime.Connection.(shared.Connection)
	afs := conn.FileSystem()

	openFn := func(path string) (io.ReadCloser, error) {
		return afs.Open(path)
	}
	globFn := func(pattern string) ([]string, error) {
		return s.expandNginxGlob(pattern)
	}

	cfg, err := nginx.ParseFiles(file.Path.Data, openFn, globFn)
	if err != nil {
		errSlice := plugin.TValue[[]any]{Error: err, State: plugin.StateIsSet | plugin.StateIsNull}
		errMap := plugin.TValue[map[string]any]{Error: err, State: plugin.StateIsSet | plugin.StateIsNull}
		s.Params = errMap
		s.HttpParams = errMap
		s.Servers = errSlice
		s.Upstreams = errSlice
		s.ListenAddresses = errSlice
		s.Files = errSlice
		return err
	}

	mainParams := map[string]any{}
	httpParams := map[string]any{}
	var servers []nginxServer
	var upstreams []nginxUpstream
	var allListenAddrs []string

	for _, d := range cfg.Directives {
		switch d.Name {
		case "http":
			walkHTTPBlock(d.Block, httpParams, &servers, &upstreams, &allListenAddrs)
		case "events":
			for _, ed := range d.Block {
				if !ed.IsBlock() {
					setNginxParam(mainParams, ed.Name, strings.Join(ed.Args, " "))
				}
			}
		default:
			if !d.IsBlock() {
				setNginxParam(mainParams, d.Name, strings.Join(d.Args, " "))
			}
		}
	}

	// Merge main + http params for the top-level params field.
	mergedParams := make(map[string]any, len(mainParams)+len(httpParams))
	for k, v := range mainParams {
		mergedParams[k] = v
	}
	for k, v := range httpParams {
		mergedParams[k] = v
	}

	s.Params = plugin.TValue[map[string]any]{Data: mergedParams, State: plugin.StateIsSet}
	s.HttpParams = plugin.TValue[map[string]any]{Data: httpParams, State: plugin.StateIsSet}

	serverResources, err := nginxServers2Resources(servers, s.MqlRuntime, s.__id)
	if err != nil {
		return err
	}
	s.Servers = plugin.TValue[[]any]{Data: serverResources, State: plugin.StateIsSet}

	upstreamResources, err := nginxUpstreams2Resources(upstreams, s.MqlRuntime, s.__id)
	if err != nil {
		return err
	}
	s.Upstreams = plugin.TValue[[]any]{Data: upstreamResources, State: plugin.StateIsSet}

	// Deduplicate listen addresses in first-seen order.
	seen := map[string]bool{}
	var uniqueAddrs []any
	for _, addr := range allListenAddrs {
		if !seen[addr] {
			seen[addr] = true
			uniqueAddrs = append(uniqueAddrs, addr)
		}
	}
	s.ListenAddresses = plugin.TValue[[]any]{Data: uniqueAddrs, State: plugin.StateIsSet}

	// Build file resources for every file visited by the parser.
	fileResources := make([]any, 0, len(cfg.Files))
	for _, path := range cfg.Files {
		f, err := CreateResource(s.MqlRuntime, "file", map[string]*llx.RawData{
			"path": llx.StringData(path),
		})
		if err != nil {
			return err
		}
		fileResources = append(fileResources, f)
	}
	s.Files = plugin.TValue[[]any]{Data: fileResources, State: plugin.StateIsSet}

	return nil
}

// Field methods — all delegate to parse().

func (s *mqlNginxConf) files(file *mqlFile) ([]any, error) {
	return nil, s.parse(file)
}

func (s *mqlNginxConf) params(file *mqlFile) (map[string]any, error) {
	return nil, s.parse(file)
}

func (s *mqlNginxConf) httpParams(file *mqlFile) (map[string]any, error) {
	return nil, s.parse(file)
}

func (s *mqlNginxConf) servers(file *mqlFile) ([]any, error) {
	return nil, s.parse(file)
}

func (s *mqlNginxConf) upstreams(file *mqlFile) ([]any, error) {
	return nil, s.parse(file)
}

func (s *mqlNginxConf) listenAddresses(file *mqlFile) ([]any, error) {
	return nil, s.parse(file)
}

// Derived fields from params.

func (s *mqlNginxConf) user(params map[string]any) (string, error) {
	if v, ok := params["user"]; ok {
		if str, ok := v.(string); ok {
			return str, nil
		}
	}
	return "", nil
}

func (s *mqlNginxConf) workerProcesses(params map[string]any) (string, error) {
	if v, ok := params["worker_processes"]; ok {
		if str, ok := v.(string); ok {
			return str, nil
		}
	}
	return "", nil
}

func (s *mqlNginxConf) errorLog(params map[string]any) (string, error) {
	if v, ok := params["error_log"]; ok {
		if str, ok := v.(string); ok {
			return str, nil
		}
	}
	return "", nil
}

// Internal types for collecting parsed data before converting to MQL resources.

type nginxServer struct {
	ServerName string
	Listen     string
	Root       string
	SSL        bool
	Locations  []nginxLocation
	Params     map[string]any
}

type nginxUpstream struct {
	Name    string
	Servers []string
	Params  map[string]any
}

type nginxLocation struct {
	Path      string
	ProxyPass string
	Root      string
	Params    map[string]any
}

// walkHTTPBlock processes the http{} block's directives.
func walkHTTPBlock(directives []nginx.Directive, httpParams map[string]any, servers *[]nginxServer, upstreams *[]nginxUpstream, listenAddrs *[]string) {
	for _, d := range directives {
		switch d.Name {
		case "server":
			srv := parseNginxServerBlock(d.Block)
			*servers = append(*servers, srv)
			if srv.Listen != "" {
				for _, l := range strings.Split(srv.Listen, ",") {
					*listenAddrs = append(*listenAddrs, strings.TrimSpace(l))
				}
			}
		case "upstream":
			name := ""
			if len(d.Args) > 0 {
				name = d.Args[0]
			}
			up := parseNginxUpstreamBlock(name, d.Block)
			*upstreams = append(*upstreams, up)
		default:
			if !d.IsBlock() {
				setNginxParam(httpParams, d.Name, strings.Join(d.Args, " "))
			}
		}
	}
}

// parseNginxServerBlock extracts structured data from a server{} block.
func parseNginxServerBlock(directives []nginx.Directive) nginxServer {
	srv := nginxServer{
		Params: map[string]any{},
	}

	var listens []string
	for _, d := range directives {
		args := strings.Join(d.Args, " ")

		switch d.Name {
		case "server_name":
			srv.ServerName = args
			setNginxParam(srv.Params, d.Name, args)
		case "listen":
			listens = append(listens, args)
			for _, arg := range d.Args {
				if arg == "ssl" {
					srv.SSL = true
				}
			}
			setNginxParam(srv.Params, d.Name, args)
		case "root":
			srv.Root = args
			setNginxParam(srv.Params, d.Name, args)
		case "ssl_certificate":
			srv.SSL = true
			setNginxParam(srv.Params, d.Name, args)
		case "location":
			loc := parseNginxLocationBlock(args, d.Block)
			srv.Locations = append(srv.Locations, loc)
		default:
			if !d.IsBlock() {
				setNginxParam(srv.Params, d.Name, args)
			}
		}
	}

	srv.Listen = strings.Join(listens, ",")
	return srv
}

// parseNginxLocationBlock extracts structured data from a location{} block.
func parseNginxLocationBlock(path string, directives []nginx.Directive) nginxLocation {
	loc := nginxLocation{
		Path:   path,
		Params: map[string]any{},
	}

	for _, d := range directives {
		if d.IsBlock() {
			continue
		}
		args := strings.Join(d.Args, " ")
		setNginxParam(loc.Params, d.Name, args)

		switch d.Name {
		case "proxy_pass":
			loc.ProxyPass = args
		case "root":
			loc.Root = args
		}
	}

	return loc
}

// parseNginxUpstreamBlock extracts structured data from an upstream{} block.
func parseNginxUpstreamBlock(name string, directives []nginx.Directive) nginxUpstream {
	up := nginxUpstream{
		Name:   name,
		Params: map[string]any{},
	}

	for _, d := range directives {
		if d.IsBlock() {
			continue
		}
		args := strings.Join(d.Args, " ")
		if d.Name == "server" {
			up.Servers = append(up.Servers, args)
		} else {
			setNginxParam(up.Params, d.Name, args)
		}
	}

	return up
}

// setNginxParam sets a directive value. For directives that can appear
// multiple times, values are comma-concatenated (matching the Apache pattern).
func setNginxParam(m map[string]any, key, value string) {
	if isNginxMultiParam[key] {
		if v, ok := m[key]; ok {
			m[key] = v.(string) + "," + value
			return
		}
	}
	m[key] = value
}

// isNginxMultiParam lists directives that can appear multiple times and should
// be concatenated rather than overwritten.
var isNginxMultiParam = map[string]bool{
	"listen":           true,
	"server_name":      true,
	"include":          true,
	"add_header":       true,
	"set":              true,
	"rewrite":          true,
	"allow":            true,
	"deny":             true,
	"fastcgi_param":    true,
	"proxy_set_header": true,
}

// Resource conversion functions.

func nginxServers2Resources(servers []nginxServer, runtime *plugin.Runtime, ownerID string) ([]any, error) {
	res := make([]any, len(servers))
	for i, srv := range servers {
		id := fmt.Sprintf("%s/server/%d-%s-%s", ownerID, i, srv.ServerName, srv.Listen)

		locations, err := nginxLocations2Resources(srv.Locations, runtime, id)
		if err != nil {
			return nil, err
		}

		obj, err := CreateResource(runtime, "nginx.conf.server", map[string]*llx.RawData{
			"__id":       llx.StringData(id),
			"serverName": llx.StringData(srv.ServerName),
			"listen":     llx.StringData(srv.Listen),
			"root":       llx.StringData(srv.Root),
			"ssl":        llx.BoolData(srv.SSL),
			"locations":  llx.ArrayData(locations, types.Resource("nginx.conf.location")),
			"params":     llx.MapData(srv.Params, types.String),
		})
		if err != nil {
			return nil, err
		}
		res[i] = obj
	}
	return res, nil
}

func nginxUpstreams2Resources(upstreams []nginxUpstream, runtime *plugin.Runtime, ownerID string) ([]any, error) {
	res := make([]any, len(upstreams))
	for i, up := range upstreams {
		serversData := make([]any, len(up.Servers))
		for j, s := range up.Servers {
			serversData[j] = s
		}

		obj, err := CreateResource(runtime, "nginx.conf.upstream", map[string]*llx.RawData{
			"__id":    llx.StringData(ownerID + "/upstream/" + up.Name),
			"name":    llx.StringData(up.Name),
			"servers": llx.ArrayData(serversData, types.String),
			"params":  llx.MapData(up.Params, types.String),
		})
		if err != nil {
			return nil, err
		}
		res[i] = obj
	}
	return res, nil
}

func nginxLocations2Resources(locations []nginxLocation, runtime *plugin.Runtime, ownerID string) ([]any, error) {
	res := make([]any, len(locations))
	for i, loc := range locations {
		obj, err := CreateResource(runtime, "nginx.conf.location", map[string]*llx.RawData{
			"__id":      llx.StringData(fmt.Sprintf("%s/location/%d-%s", ownerID, i, loc.Path)),
			"path":      llx.StringData(loc.Path),
			"proxyPass": llx.StringData(loc.ProxyPass),
			"root":      llx.StringData(loc.Root),
			"params":    llx.MapData(loc.Params, types.String),
		})
		if err != nil {
			return nil, err
		}
		res[i] = obj
	}
	return res, nil
}
