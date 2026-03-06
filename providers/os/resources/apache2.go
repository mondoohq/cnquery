// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/spf13/afero"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/os/connection/shared"
	"go.mondoo.com/mql/v13/providers/os/resources/apache2"
	"go.mondoo.com/mql/v13/types"
)

func (s *mqlApache2) id() (string, error) {
	return "apache2", nil
}

// apacheVersionBinaries lists the well-known binary paths for the Apache httpd
// server. The version string (e.g. "Apache/2.4.62") is embedded as a constant in
// the binary, so we can extract it by reading the file directly — no command
// execution required.
var apacheVersionBinaries = []string{
	"/usr/sbin/apache2",
	"/usr/sbin/httpd",
	"/usr/local/sbin/httpd",
	"/usr/local/bin/httpd",
}

// apacheVersionCommands are tried as a fallback when the binary cannot be read.
var apacheVersionCommands = []string{"apache2ctl", "httpd", "apachectl"}

var apacheVersionTag = []byte("Apache/")

func (s *mqlApache2) version() (string, error) {
	conn := s.MqlRuntime.Connection.(shared.Connection)
	afs := &afero.Afero{Fs: conn.FileSystem()}

	// Prefer file-based detection: scan the httpd binary for the embedded
	// "Apache/x.y.z" version string without loading the full binary into memory.
	for _, bin := range apacheVersionBinaries {
		if v := scanBinaryForTag(afs, bin, apacheVersionTag); v != "" {
			return v, nil
		}
	}

	// Fall back to running a command when the binary isn't readable (e.g.
	// non-standard install path).
	for _, bin := range apacheVersionCommands {
		cmd, err := conn.RunCommand(bin + " -v")
		if err != nil {
			continue
		}
		if cmd.ExitStatus != 0 {
			continue
		}
		data, err := io.ReadAll(cmd.Stdout)
		if err != nil {
			continue
		}
		// Output looks like: "Server version: Apache/2.4.62 (Ubuntu)"
		if m := reApacheVersion.FindSubmatch(data); m != nil {
			return string(m[1]), nil
		}
	}

	// Apache is likely not installed; return nil rather than an error.
	s.Version = plugin.TValue[string]{State: plugin.StateIsSet | plugin.StateIsNull}
	return "", nil
}

var reApacheVersion = regexp.MustCompile(`Apache/(\S+)`)

type mqlApache2ConfInternal struct {
	lock       sync.Mutex
	serverRoot string
}

// apacheConfByFamily maps platform families (and a few standalone platform
// names that don't belong to the matching family) to their default Apache
// config path. The lookup checks platform name first, then walks Family[].
var apacheConfByFamily = map[string]string{
	// families
	"debian": "/etc/apache2/apache2.conf",
	"redhat": "/etc/httpd/conf/httpd.conf",
	"suse":   "/etc/apache2/httpd.conf",
	"arch":   "/etc/httpd/conf/httpd.conf",
	"bsd":    "/etc/apache2/httpd.conf",
	// standalone platforms (not in matching families above)
	"amazonlinux": "/etc/httpd/conf/httpd.conf",
	"gentoo":      "/etc/apache2/httpd.conf",
	"freebsd":     "/usr/local/etc/apache24/httpd.conf",
}

const defaultApacheConf = "/etc/httpd/conf/httpd.conf"

func apacheConfPath(conn shared.Connection) string {
	asset := conn.Asset()
	if asset != nil && asset.Platform != nil {
		if p, ok := apacheConfByFamily[asset.Platform.Name]; ok {
			return p
		}
		for _, family := range asset.Platform.Family {
			if p, ok := apacheConfByFamily[family]; ok {
				return p
			}
		}
	}
	return defaultApacheConf
}

// apacheServerRootByFamily maps platform families/names to their default
// ServerRoot directory for resolving relative Include paths.
var apacheServerRootByFamily = map[string]string{
	"debian":  "/etc/apache2",
	"suse":    "/etc/apache2",
	"bsd":     "/etc/apache2",
	"freebsd": "/usr/local/etc/apache24",
	"gentoo":  "/etc/apache2",
}

const defaultApacheServerRoot = "/etc/httpd"

// apacheServerRoot returns the ServerRoot directory for resolving relative
// Include paths. Defaults based on platform.
func apacheServerRoot(conn shared.Connection) string {
	asset := conn.Asset()
	if asset != nil && asset.Platform != nil {
		if p, ok := apacheServerRootByFamily[asset.Platform.Name]; ok {
			return p
		}
		for _, family := range asset.Platform.Family {
			if p, ok := apacheServerRootByFamily[family]; ok {
				return p
			}
		}
	}
	return defaultApacheServerRoot
}

// prescanServerRoot does a quick scan of the config content for a ServerRoot
// directive so that relative Include paths can be resolved before full parsing.
func prescanServerRoot(content string) string {
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || line[0] == '#' {
			continue
		}
		idx := strings.IndexAny(line, " \t")
		if idx < 0 {
			continue
		}
		if strings.EqualFold(line[:idx], "ServerRoot") {
			value := strings.TrimSpace(line[idx+1:])
			if len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"' {
				value = value[1 : len(value)-1]
			}
			return value
		}
	}
	return ""
}

func initApache2Conf(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if x, ok := args["path"]; ok {
		path, ok := x.Value.(string)
		if !ok {
			return nil, nil, errors.New("wrong type for 'path' in apache2.conf initialization, it must be a string")
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

func (s *mqlApache2Conf) id() (string, error) {
	file := s.GetFile()
	if file.Error != nil {
		return "", file.Error
	}

	return file.Data.Path.Data, nil
}

// file is the default getter for the file field. It is only called when
// apache2.conf is created without a path argument. When a path IS provided,
// initApache2Conf converts it to a file resource that the framework stores
// directly, bypassing this method entirely (same pattern as sshd.config).
func (s *mqlApache2Conf) file() (*mqlFile, error) {
	conn := s.MqlRuntime.Connection.(shared.Connection)

	// Try the platform-preferred path first, then fall back to all known paths.
	preferred := apacheConfPath(conn)
	candidates := []string{preferred}
	for _, p := range apacheConfByFamily {
		if p != preferred {
			candidates = append(candidates, p)
		}
	}

	afs := &afero.Afero{Fs: conn.FileSystem()}
	for _, path := range candidates {
		if ok, _ := afs.Exists(path); ok {
			f, err := CreateResource(s.MqlRuntime, "file", map[string]*llx.RawData{
				"path": llx.StringData(path),
			})
			if err != nil {
				return nil, err
			}
			return f.(*mqlFile), nil
		}
	}

	// No config file found; return the platform default so the resource still
	// has a path but parse() will detect non-existence and return empty data.
	f, err := CreateResource(s.MqlRuntime, "file", map[string]*llx.RawData{
		"path": llx.StringData(preferred),
	})
	if err != nil {
		return nil, err
	}
	return f.(*mqlFile), nil
}

var reApacheGlob = regexp.MustCompile(`[*?\[]`)

func (s *mqlApache2Conf) expandGlob(pattern string) ([]string, error) {
	conn := s.MqlRuntime.Connection.(shared.Connection)

	// Resolve relative paths against ServerRoot (prefer config value, fall back to platform default)
	if !filepath.IsAbs(pattern) {
		serverRoot := s.serverRoot
		if serverRoot == "" {
			serverRoot = apacheServerRoot(conn)
		}
		pattern = filepath.Join(serverRoot, pattern)
	}

	if !reApacheGlob.MatchString(pattern) {
		return []string{pattern}, nil
	}

	// Walk the filesystem to expand the glob
	var paths []string
	segments := strings.Split(pattern, "/")
	if segments[0] == "" {
		paths = []string{"/"}
	}

	afs := &afero.Afero{Fs: conn.FileSystem()}

	for _, segment := range segments[1:] {
		if !reApacheGlob.MatchString(segment) {
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
				if match, err := filepath.Match(segment, name); err != nil {
					return nil, err
				} else if match {
					nuPaths = append(nuPaths, filepath.Join(path, name))
				}
			}
		}
		paths = nuPaths
	}

	return paths, nil
}

func (s *mqlApache2Conf) setEmpty() {
	s.Params = plugin.TValue[map[string]any]{Data: map[string]any{}, State: plugin.StateIsSet}
	s.Modules = plugin.TValue[[]any]{Data: []any{}, State: plugin.StateIsSet}
	s.VirtualHosts = plugin.TValue[[]any]{Data: []any{}, State: plugin.StateIsSet}
	s.Directories = plugin.TValue[[]any]{Data: []any{}, State: plugin.StateIsSet}
	s.Files = plugin.TValue[[]any]{Data: []any{}, State: plugin.StateIsSet}
}

func (s *mqlApache2Conf) parse(file *mqlFile) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	if s.Params.State == plugin.StateIsSet {
		return nil
	}

	if file == nil {
		s.setEmpty()
		return nil
	}

	// When the config file doesn't exist (e.g. Apache is not installed),
	// return empty data instead of cascading errors.
	if exists := file.GetExists(); exists.Error != nil || !exists.Data {
		s.setEmpty()
		return nil
	}

	// Pre-scan root file for ServerRoot directive so that relative Include
	// paths are resolved correctly during parsing.
	if content := file.GetContent(); content.Error == nil {
		if sr := prescanServerRoot(content.Data); sr != "" {
			s.serverRoot = sr
		}
	}

	filesIdx := map[string]*mqlFile{
		file.Path.Data: file,
	}

	// fileContent creates file resources and reads their content
	fileContent := func(path string) (string, error) {
		f, ok := filesIdx[path]
		if !ok {
			raw, err := CreateResource(s.MqlRuntime, "file", map[string]*llx.RawData{
				"path": llx.StringData(path),
			})
			if err != nil {
				return "", err
			}
			f = raw.(*mqlFile)
			filesIdx[path] = f
		}

		content := f.GetContent()
		if content.Error != nil {
			return "", content.Error
		}

		return content.Data, nil
	}

	globExpand := func(pattern string) ([]string, error) {
		return s.expandGlob(pattern)
	}

	cfg, err := apache2.ParseWithGlob(file.Path.Data, fileContent, globExpand)

	if err != nil {
		errState := plugin.TValue[map[string]any]{Error: err, State: plugin.StateIsSet | plugin.StateIsNull}
		s.Params = errState
		s.Modules = plugin.TValue[[]any]{Error: err, State: plugin.StateIsSet | plugin.StateIsNull}
		s.VirtualHosts = plugin.TValue[[]any]{Error: err, State: plugin.StateIsSet | plugin.StateIsNull}
		s.Directories = plugin.TValue[[]any]{Error: err, State: plugin.StateIsSet | plugin.StateIsNull}
		s.Files = plugin.TValue[[]any]{Error: err, State: plugin.StateIsSet | plugin.StateIsNull}
	} else {
		s.Params = plugin.TValue[map[string]any]{Data: cfg.Params, State: plugin.StateIsSet}

		modules, err := apacheModules2Resources(cfg.Modules, s.MqlRuntime, s.__id)
		if err != nil {
			return err
		}
		s.Modules = plugin.TValue[[]any]{Data: modules, State: plugin.StateIsSet}

		vhosts, err := apacheVHosts2Resources(cfg.VHosts, s.MqlRuntime, s.__id)
		if err != nil {
			return err
		}
		s.VirtualHosts = plugin.TValue[[]any]{Data: vhosts, State: plugin.StateIsSet}

		dirs, err := apacheDirs2Resources(cfg.Dirs, s.MqlRuntime, s.__id)
		if err != nil {
			return err
		}
		s.Directories = plugin.TValue[[]any]{Data: dirs, State: plugin.StateIsSet}

		files := make([]any, 0, len(filesIdx))
		for _, f := range filesIdx {
			files = append(files, f)
		}
		s.Files = plugin.TValue[[]any]{Data: files, State: plugin.StateIsSet}
	}

	return err
}

func (s *mqlApache2Conf) files(file *mqlFile) ([]any, error) {
	return nil, s.parse(file)
}

func (s *mqlApache2Conf) params(file *mqlFile) (map[string]any, error) {
	return nil, s.parse(file)
}

func (s *mqlApache2Conf) modules(file *mqlFile) ([]any, error) {
	return nil, s.parse(file)
}

func (s *mqlApache2Conf) virtualHosts(file *mqlFile) ([]any, error) {
	return nil, s.parse(file)
}

func (s *mqlApache2Conf) directories(file *mqlFile) ([]any, error) {
	return nil, s.parse(file)
}

func (s *mqlApache2Conf) listenAddresses(params map[string]any) ([]any, error) {
	raw, ok := params["Listen"]
	if !ok {
		return nil, nil
	}

	str, ok := raw.(string)
	if !ok {
		return nil, nil
	}

	parts := strings.Split(str, ",")
	res := make([]any, len(parts))
	for i, p := range parts {
		res[i] = strings.TrimSpace(p)
	}
	return res, nil
}

func apacheModules2Resources(modules []apache2.Module, runtime *plugin.Runtime, ownerID string) ([]any, error) {
	res := make([]any, len(modules))
	for i, mod := range modules {
		obj, err := CreateResource(runtime, "apache2.conf.module", map[string]*llx.RawData{
			"__id": llx.StringData(ownerID + "/module/" + mod.Name),
			"name": llx.StringData(mod.Name),
			"path": llx.StringData(mod.Path),
		})
		if err != nil {
			return nil, err
		}
		res[i] = obj
	}
	return res, nil
}

func apacheVHosts2Resources(vhosts []apache2.VirtualHost, runtime *plugin.Runtime, ownerID string) ([]any, error) {
	res := make([]any, len(vhosts))
	for i, vh := range vhosts {
		obj, err := CreateResource(runtime, "apache2.conf.virtualHost", map[string]*llx.RawData{
			"__id":         llx.StringData(ownerID + "/vhost/" + strconv.Itoa(i) + "/" + vh.Address),
			"address":      llx.StringData(vh.Address),
			"serverName":   llx.StringData(vh.ServerName),
			"documentRoot": llx.StringData(vh.DocumentRoot),
			"ssl":          llx.BoolData(vh.SSL),
			"params":       llx.MapData(vh.Params, types.String),
		})
		if err != nil {
			return nil, err
		}
		res[i] = obj
	}
	return res, nil
}

func apacheDirs2Resources(dirs []apache2.Directory, runtime *plugin.Runtime, ownerID string) ([]any, error) {
	res := make([]any, len(dirs))
	for i, d := range dirs {
		obj, err := CreateResource(runtime, "apache2.conf.directory", map[string]*llx.RawData{
			"__id":          llx.StringData(ownerID + "/dir/" + strconv.Itoa(i) + "/" + d.Path),
			"path":          llx.StringData(d.Path),
			"options":       llx.StringData(d.Options),
			"allowOverride": llx.StringData(d.AllowOverride),
			"params":        llx.MapData(d.Params, types.String),
		})
		if err != nil {
			return nil, err
		}
		res[i] = obj
	}
	return res, nil
}
