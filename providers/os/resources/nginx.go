// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
	"sync"

	"github.com/nginxinc/nginx-go-crossplane"
	"github.com/spf13/afero"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/os/connection/shared"
	"go.mondoo.com/mql/v13/types"
)

type mqlNginxInternal struct {
	lock sync.Mutex
}

// parse runs "nginx -V" once and populates both Version and Modules.
func (n *mqlNginx) parse() error {
	n.lock.Lock()
	defer n.lock.Unlock()

	if n.Version.State == plugin.StateIsSet {
		return nil
	}

	o, err := CreateResource(n.MqlRuntime, "command", map[string]*llx.RawData{
		"command": llx.StringData("nginx -V 2>&1"),
	})
	if err != nil {
		n.Version = plugin.TValue[string]{Error: err, State: plugin.StateIsSet | plugin.StateIsNull}
		n.Modules = plugin.TValue[[]any]{Error: err, State: plugin.StateIsSet | plugin.StateIsNull}
		return err
	}
	cmd := o.(*mqlCommand)
	if exit := cmd.GetExitcode(); exit.Data != 0 {
		err := errors.New("failed to run nginx -V: " + cmd.GetStdout().Data)
		n.Version = plugin.TValue[string]{Error: err, State: plugin.StateIsSet | plugin.StateIsNull}
		n.Modules = plugin.TValue[[]any]{Error: err, State: plugin.StateIsSet | plugin.StateIsNull}
		return err
	}

	output := cmd.GetStdout().Data

	n.Version = plugin.TValue[string]{Data: parseNginxVersion(output), State: plugin.StateIsSet}

	modules := parseNginxModules(output)
	modulesData := make([]any, len(modules))
	for i, m := range modules {
		modulesData[i] = m
	}
	n.Modules = plugin.TValue[[]any]{Data: modulesData, State: plugin.StateIsSet}

	return nil
}

func (n *mqlNginx) version() (string, error) {
	return "", n.parse()
}

func (n *mqlNginx) modules() ([]any, error) {
	return nil, n.parse()
}

// reNginxVersion matches "nginx version: nginx/1.25.3" or similar.
var reNginxVersion = regexp.MustCompile(`nginx version:\s*\S+/(\S+)`)

// parseNginxVersion extracts the version number from nginx -V output.
func parseNginxVersion(output string) string {
	if m := reNginxVersion.FindStringSubmatch(output); len(m) > 1 {
		return m[1]
	}
	return ""
}

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

// parse is the central method that calls crossplane.Parse with afero-backed
// Open/Glob callbacks, then walks the Directive tree to populate all fields.
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

	// Track all files crossplane visits (for the files field).
	visitedFiles := map[string]bool{
		file.Path.Data: true,
	}

	openFn := func(path string) (io.ReadCloser, error) {
		visitedFiles[path] = true
		return afs.Open(path)
	}

	globFn := func(pattern string) ([]string, error) {
		return afero.Glob(afs, pattern)
	}

	payload, err := crossplane.Parse(file.Path.Data, &crossplane.ParseOptions{
		Open:                      openFn,
		Glob:                      globFn,
		SkipDirectiveContextCheck: true,
		SkipDirectiveArgsCheck:    true,
	})

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

	// Walk the crossplane Payload to extract structured data.
	mainParams := map[string]any{}
	httpParams := map[string]any{}
	var servers []nginxServer
	var upstreams []nginxUpstream
	var allListenAddrs []string

	for _, cfg := range payload.Config {
		for _, d := range cfg.Parsed {
			switch d.Directive {
			case "http":
				walkHTTPBlock(d.Block, httpParams, &servers, &upstreams, &allListenAddrs)
			case "events":
				for _, ed := range d.Block {
					if !ed.IsBlock() {
						setNginxParam(mainParams, ed.Directive, strings.Join(ed.Args, " "))
					}
				}
			default:
				if !d.IsBlock() {
					setNginxParam(mainParams, d.Directive, strings.Join(d.Args, " "))
				}
			}
		}
	}

	// Merge main + http params for the top-level params field.
	mergedParams := map[string]any{}
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

	// Deduplicate listen addresses.
	seen := map[string]bool{}
	var uniqueAddrs []any
	for _, addr := range allListenAddrs {
		if !seen[addr] {
			seen[addr] = true
			uniqueAddrs = append(uniqueAddrs, addr)
		}
	}
	s.ListenAddresses = plugin.TValue[[]any]{Data: uniqueAddrs, State: plugin.StateIsSet}

	// Build file resources for all visited files.
	fileResources := make([]any, 0, len(visitedFiles))
	for path := range visitedFiles {
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
func walkHTTPBlock(directives crossplane.Directives, httpParams map[string]any, servers *[]nginxServer, upstreams *[]nginxUpstream, listenAddrs *[]string) {
	for _, d := range directives {
		switch d.Directive {
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
				setNginxParam(httpParams, d.Directive, strings.Join(d.Args, " "))
			}
		}
	}
}

// parseNginxServerBlock extracts structured data from a server{} block.
func parseNginxServerBlock(directives crossplane.Directives) nginxServer {
	srv := nginxServer{
		Params: map[string]any{},
	}

	var listens []string
	for _, d := range directives {
		args := strings.Join(d.Args, " ")

		switch d.Directive {
		case "server_name":
			srv.ServerName = args
			setNginxParam(srv.Params, d.Directive, args)
		case "listen":
			listens = append(listens, args)
			for _, arg := range d.Args {
				if arg == "ssl" {
					srv.SSL = true
				}
			}
			setNginxParam(srv.Params, d.Directive, args)
		case "root":
			srv.Root = args
			setNginxParam(srv.Params, d.Directive, args)
		case "ssl_certificate":
			srv.SSL = true
			setNginxParam(srv.Params, d.Directive, args)
		case "location":
			loc := parseNginxLocationBlock(args, d.Block)
			srv.Locations = append(srv.Locations, loc)
		default:
			if !d.IsBlock() {
				setNginxParam(srv.Params, d.Directive, args)
			}
		}
	}

	srv.Listen = strings.Join(listens, ",")
	return srv
}

// parseNginxLocationBlock extracts structured data from a location{} block.
func parseNginxLocationBlock(path string, directives crossplane.Directives) nginxLocation {
	loc := nginxLocation{
		Path:   path,
		Params: map[string]any{},
	}

	for _, d := range directives {
		if d.IsBlock() {
			continue
		}
		args := strings.Join(d.Args, " ")
		setNginxParam(loc.Params, d.Directive, args)

		switch d.Directive {
		case "proxy_pass":
			loc.ProxyPass = args
		case "root":
			loc.Root = args
		}
	}

	return loc
}

// parseNginxUpstreamBlock extracts structured data from an upstream{} block.
func parseNginxUpstreamBlock(name string, directives crossplane.Directives) nginxUpstream {
	up := nginxUpstream{
		Name:   name,
		Params: map[string]any{},
	}

	for _, d := range directives {
		if d.IsBlock() {
			continue
		}
		args := strings.Join(d.Args, " ")
		if d.Directive == "server" {
			up.Servers = append(up.Servers, args)
		} else {
			setNginxParam(up.Params, d.Directive, args)
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
