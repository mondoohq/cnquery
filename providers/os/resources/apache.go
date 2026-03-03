// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/spf13/afero"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/os/connection/shared"
	"go.mondoo.com/mql/v13/providers/os/resources/apache"
	"go.mondoo.com/mql/v13/types"
)

type mqlApacheConfInternal struct {
	lock sync.Mutex
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

func initApacheConf(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if x, ok := args["path"]; ok {
		path, ok := x.Value.(string)
		if !ok {
			return nil, nil, errors.New("wrong type for 'path' in apache.conf initialization, it must be a string")
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

func (s *mqlApacheConf) id() (string, error) {
	file := s.GetFile()
	if file.Error != nil {
		return "", file.Error
	}

	return file.Data.Path.Data, nil
}

func (s *mqlApacheConf) file() (*mqlFile, error) {
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

func (s *mqlApacheConf) expandGlob(pattern string) ([]string, error) {
	conn := s.MqlRuntime.Connection.(shared.Connection)

	// Resolve relative paths against ServerRoot
	if !filepath.IsAbs(pattern) {
		serverRoot := apacheServerRoot(conn)
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

func (s *mqlApacheConf) parse(file *mqlFile) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	if file == nil {
		return errors.New("no base apache config file to read")
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

	cfg, err := apache.ParseWithGlob(file.Path.Data, fileContent, globExpand)

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

func (s *mqlApacheConf) files(file *mqlFile) ([]any, error) {
	return nil, s.parse(file)
}

func (s *mqlApacheConf) params(file *mqlFile) (map[string]any, error) {
	return nil, s.parse(file)
}

func (s *mqlApacheConf) modules(file *mqlFile) ([]any, error) {
	return nil, s.parse(file)
}

func (s *mqlApacheConf) virtualHosts(file *mqlFile) ([]any, error) {
	return nil, s.parse(file)
}

func (s *mqlApacheConf) directories(file *mqlFile) ([]any, error) {
	return nil, s.parse(file)
}

func (s *mqlApacheConf) listenAddresses(params map[string]any) ([]any, error) {
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

func apacheModules2Resources(modules []apache.Module, runtime *plugin.Runtime, ownerID string) ([]any, error) {
	res := make([]any, len(modules))
	for i, mod := range modules {
		obj, err := CreateResource(runtime, "apache.conf.module", map[string]*llx.RawData{
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

func apacheVHosts2Resources(vhosts []apache.VirtualHost, runtime *plugin.Runtime, ownerID string) ([]any, error) {
	res := make([]any, len(vhosts))
	for i, vh := range vhosts {
		// Use address + serverName for unique ID since multiple VHosts can share an address
		id := vh.Address
		if vh.ServerName != "" {
			id += "/" + vh.ServerName
		}

		obj, err := CreateResource(runtime, "apache.conf.virtualHost", map[string]*llx.RawData{
			"__id":         llx.StringData(ownerID + "/vhost/" + id),
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

func apacheDirs2Resources(dirs []apache.Directory, runtime *plugin.Runtime, ownerID string) ([]any, error) {
	res := make([]any, len(dirs))
	for i, d := range dirs {
		obj, err := CreateResource(runtime, "apache.conf.directory", map[string]*llx.RawData{
			"__id":          llx.StringData(ownerID + "/dir/" + d.Path),
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
