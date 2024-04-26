// copyright: 2019, Dominik Richter and Christoph Hartmann
// author: Dominik Richter
// author: Christoph Hartmann

package resources

import (
	"errors"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/spf13/afero"
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers/os/connection/shared"
	"go.mondoo.com/cnquery/v11/providers/os/resources/sshd"
	"go.mondoo.com/cnquery/v11/types"
)

type mqlSshdConfigInternal struct {
	lock sync.Mutex
}

func initSshdConfig(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if x, ok := args["path"]; ok {
		path, ok := x.Value.(string)
		if !ok {
			return nil, nil, errors.New("wrong type for 'path' in sshd.config initialization, it must be a string")
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

const defaultSshdConfig = "/etc/ssh/sshd_config"

func (s *mqlSshdConfig) id() (string, error) {
	file := s.GetFile()
	if file.Error != nil {
		return "", file.Error
	}

	return file.Data.Path.Data, nil
}

func (s *mqlSshdConfig) file() (*mqlFile, error) {
	f, err := CreateResource(s.MqlRuntime, "file", map[string]*llx.RawData{
		"path": llx.StringData(defaultSshdConfig),
	})
	if err != nil {
		return nil, err
	}
	return f.(*mqlFile), nil
}

func matchBlocks2Resources(m sshd.MatchBlocks, runtime *plugin.Runtime, ownerID string) ([]any, error) {
	res := make([]any, len(m))
	for i := range m {
		cur := m[i]
		obj, err := CreateResource(runtime, "sshd.config.matchBlock", map[string]*llx.RawData{
			"__id":     llx.StringData(ownerID + "\x00" + cur.Criteria),
			"criteria": llx.StringData(cur.Criteria),
			"params":   llx.MapData(cur.Params, types.String),
		})
		if err != nil {
			return nil, err
		}
		res[i] = obj
	}
	return res, nil
}

var reGlob = regexp.MustCompile(`.*\*.*`)

func (s *mqlSshdConfig) expandGlob(glob string) ([]string, error) {
	if !reGlob.MatchString(glob) {
		if !filepath.IsAbs(glob) {
			glob = filepath.Join("/etc/ssh", glob)
		}
		return []string{glob}, nil
	}

	var paths []string
	segments := strings.Split(glob, "/")
	if segments[0] == "" {
		paths = []string{"/"}
	} else {
		// https://man7.org/linux/man-pages/man5/sshd_config.5.html
		// Relative files are always expanded from `/ssh`
		paths = []string{"/etc/ssh"}
	}

	conn := s.MqlRuntime.Connection.(shared.Connection)
	afs := &afero.Afero{Fs: conn.FileSystem()}

	for _, segment := range segments[1:] {
		if !reGlob.MatchString(segment) {
			for i := range paths {
				paths[i] = filepath.Join(paths[i], segment)
			}
			continue
		}

		var nuPaths []string
		for _, path := range paths {
			files, err := afs.ReadDir(path)
			if err != nil {
				return nil, err
			}

			for j := range files {
				file := files[j]
				name := file.Name()
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

func (s *mqlSshdConfig) parse(file *mqlFile) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	if file == nil {
		return errors.New("no base sshd config file to read")
	}

	filesIdx := map[string]*mqlFile{
		file.Path.Data: file,
	}
	var allContents strings.Builder
	globPathContent := func(glob string) (string, error) {
		paths, err := s.expandGlob(glob)
		if err != nil {
			return "", err
		}

		var content strings.Builder
		for _, path := range paths {
			file, ok := filesIdx[path]
			if !ok {
				raw, err := CreateResource(s.MqlRuntime, "file", map[string]*llx.RawData{
					"path": llx.StringData(path),
				})
				if err != nil {
					return "", err
				}
				file = raw.(*mqlFile)
				filesIdx[path] = file
			}

			fileContent := file.GetContent()
			if fileContent.Error != nil {
				return "", fileContent.Error
			}

			content.WriteString(fileContent.Data)
			content.WriteString("\n")
		}

		res := content.String()
		allContents.WriteString(res)
		return res, nil
	}

	matchBlocks, err := sshd.ParseBlocks(file.Path.Data, globPathContent)
	// TODO: check if not ready on I/O
	if err != nil {
		s.Params = plugin.TValue[map[string]any]{Error: err, State: plugin.StateIsSet | plugin.StateIsNull}
		s.Blocks = plugin.TValue[[]any]{Error: err, State: plugin.StateIsSet | plugin.StateIsNull}
		s.Content = plugin.TValue[string]{Error: err, State: plugin.StateIsSet | plugin.StateIsNull}
		s.Files = plugin.TValue[[]any]{Error: err, State: plugin.StateIsSet | plugin.StateIsNull}

	} else {
		s.Params = plugin.TValue[map[string]any]{Data: matchBlocks.Flatten(), State: plugin.StateIsSet}

		blocks, err := matchBlocks2Resources(matchBlocks, s.MqlRuntime, s.__id)
		if err != nil {
			return err
		}
		s.Blocks = plugin.TValue[[]any]{Data: blocks, State: plugin.StateIsSet}

		s.Content = plugin.TValue[string]{Data: allContents.String(), State: plugin.StateIsSet}

		files := make([]any, len(filesIdx))
		i := 0
		for _, v := range filesIdx {
			files[i] = v
			i++
		}
		s.Files = plugin.TValue[[]any]{Data: files, State: plugin.StateIsSet}
	}

	return err
}

func (s *mqlSshdConfig) files(file *mqlFile) ([]any, error) {
	return nil, s.parse(file)
}

func (s *mqlSshdConfig) content(file *mqlFile) (string, error) {
	return "", s.parse(file)
}

func (s *mqlSshdConfig) params(file *mqlFile) (map[string]any, error) {
	return nil, s.parse(file)
}

func (s *mqlSshdConfig) blocks(file *mqlFile) ([]any, error) {
	return nil, s.parse(file)
}

func (s *mqlSshdConfig) parseConfigEntrySlice(raw interface{}) ([]interface{}, error) {
	str, ok := raw.(string)
	if !ok {
		return nil, errors.New("value is not a valid string")
	}

	res := []interface{}{}
	entries := strings.Split(str, ",")
	for i := range entries {
		val := strings.TrimSpace(entries[i])
		res = append(res, val)
	}

	return res, nil
}

func (s *mqlSshdConfig) ciphers(params map[string]interface{}) ([]interface{}, error) {
	rawCiphers, ok := params["Ciphers"]
	if !ok {
		return nil, nil
	}

	return s.parseConfigEntrySlice(rawCiphers)
}

func (s *mqlSshdConfig) macs(params map[string]interface{}) ([]interface{}, error) {
	rawMacs, ok := params["MACs"]
	if !ok {
		return nil, nil
	}

	return s.parseConfigEntrySlice(rawMacs)
}

func (s *mqlSshdConfig) kexs(params map[string]interface{}) ([]interface{}, error) {
	rawkexs, ok := params["KexAlgorithms"]
	if !ok {
		return nil, nil
	}

	return s.parseConfigEntrySlice(rawkexs)
}

func (s *mqlSshdConfig) hostkeys(params map[string]interface{}) ([]interface{}, error) {
	rawHostKeys, ok := params["HostKey"]
	if !ok {
		return nil, nil
	}

	return s.parseConfigEntrySlice(rawHostKeys)
}

func (s *mqlSshdConfig) permitRootLogin(params map[string]interface{}) ([]interface{}, error) {
	rawHostKeys, ok := params["PermitRootLogin"]
	if !ok {
		return nil, nil
	}

	return s.parseConfigEntrySlice(rawHostKeys)
}
