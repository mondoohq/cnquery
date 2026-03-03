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

// apacheVersionBinaries lists commands to try for getting the Apache version.
// Debian/Ubuntu use apache2ctl, RHEL/CentOS use httpd, others may use apachectl.
var apacheVersionBinaries = []string{"apache2ctl", "httpd", "apachectl"}

func (s *mqlApache2) version() (string, error) {
	conn := s.MqlRuntime.Connection.(shared.Connection)

	for _, bin := range apacheVersionBinaries {
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

	return "", errors.New("could not determine apache version")
}

var reApacheVersion = regexp.MustCompile(`Apache/(\S+)`)

type mqlApache2ConfInternal struct {
	lock       sync.Mutex
	serverRoot string
}

// apacheConfPaths maps platform names to their default Apache config location.
// Debian/Ubuntu use apache2, RHEL/CentOS use httpd.
var apacheConfPaths = map[string]string{
	"ubuntu":  "/etc/apache2/apache2.conf",
	"debian":  "/etc/apache2/apache2.conf",
	"redhat":  "/etc/httpd/conf/httpd.conf",
	"centos":  "/etc/httpd/conf/httpd.conf",
	"fedora":  "/etc/httpd/conf/httpd.conf",
	"rocky":   "/etc/httpd/conf/httpd.conf",
	"alma":    "/etc/httpd/conf/httpd.conf",
	"oracle":  "/etc/httpd/conf/httpd.conf",
	"amazon":  "/etc/httpd/conf/httpd.conf",
	"suse":    "/etc/apache2/httpd.conf",
	"opensuse":   "/etc/apache2/httpd.conf",
	"freebsd":    "/usr/local/etc/apache24/httpd.conf",
	"openbsd":    "/etc/apache2/httpd.conf",
	"arch":       "/etc/httpd/conf/httpd.conf",
	"gentoo":     "/etc/apache2/httpd.conf",
}

const defaultApacheConf = "/etc/httpd/conf/httpd.conf"

func apacheConfPath(conn shared.Connection) string {
	asset := conn.Asset()
	if asset != nil && asset.Platform != nil {
		if p, ok := apacheConfPaths[asset.Platform.Name]; ok {
			return p
		}
		// Check family for broader matches
		for _, family := range asset.Platform.Family {
			if p, ok := apacheConfPaths[family]; ok {
				return p
			}
		}
	}
	return defaultApacheConf
}

// apacheServerRoot returns the ServerRoot directory for resolving relative
// Include paths. Defaults based on platform.
func apacheServerRoot(conn shared.Connection) string {
	asset := conn.Asset()
	if asset != nil && asset.Platform != nil {
		switch asset.Platform.Name {
		case "ubuntu", "debian":
			return "/etc/apache2"
		case "freebsd":
			return "/usr/local/etc/apache24"
		}
		for _, family := range asset.Platform.Family {
			if family == "debian" {
				return "/etc/apache2"
			}
		}
	}
	return "/etc/httpd"
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

func (s *mqlApache2Conf) file() (*mqlFile, error) {
	conn := s.MqlRuntime.Connection.(shared.Connection)
	path := apacheConfPath(conn)

	f, err := CreateResource(s.MqlRuntime, "file", map[string]*llx.RawData{
		"path": llx.StringData(path),
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

func (s *mqlApache2Conf) parse(file *mqlFile) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	if file == nil {
		return errors.New("no base apache config file to read")
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
