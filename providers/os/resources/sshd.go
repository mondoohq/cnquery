// copyright: 2019, Dominik Richter and Christoph Hartmann
// author: Dominik Richter
// author: Christoph Hartmann

package resources

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/spf13/afero"
	"go.mondoo.com/cnquery/v12/llx"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v12/providers/os/connection/shared"
	"go.mondoo.com/cnquery/v12/providers/os/resources/sshd"
	"go.mondoo.com/cnquery/v12/types"
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

		fobj, err := CreateResource(runtime, "file", map[string]*llx.RawData{
			"path": llx.StringData(cur.Context.Path),
		})
		if err != nil {
			return nil, err
		}

		cobj, err := CreateResource(runtime, "file.context", map[string]*llx.RawData{
			"file":  llx.ResourceData(fobj, "file"),
			"range": llx.RangeData(cur.Context.Range),
		})
		if err != nil {
			return nil, err
		}

		obj, err := CreateResource(runtime, "sshd.config.matchBlock", map[string]*llx.RawData{
			"__id":     llx.StringData(ownerID + "/" + cur.Criteria),
			"criteria": llx.StringData(cur.Criteria),
			"params":   llx.MapData(cur.Params, types.String),
			"context":  llx.ResourceData(cobj, "file.context"),
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
				// If the directory doesn't exist, treat it as "no matches" (empty result)
				// This is consistent with standard glob behavior where a non-existent directory
				// results in an empty match set, not an error
				if os.IsNotExist(err) {
					continue
				}
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

	// Function to get file content by path
	fileContent := func(path string) (string, error) {
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

		content := fileContent.Data + "\n"
		allContents.WriteString(content)
		return fileContent.Data, nil
	}

	// Function to expand glob patterns
	globExpand := func(glob string) ([]string, error) {
		return s.expandGlob(glob)
	}

	matchBlocks, err := sshd.ParseBlocksWithGlob(file.Path.Data, fileContent, globExpand)
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

func parseConfigEntrySlice(raw any) ([]any, error) {
	str, ok := raw.(string)
	if !ok {
		return nil, errors.New("value is not a valid string")
	}

	res := []any{}
	entries := strings.Split(str, ",")
	for i := range entries {
		val := strings.TrimSpace(entries[i])
		res = append(res, val)
	}

	return res, nil
}

func (s *mqlSshdConfig) ciphers(params map[string]any) ([]any, error) {
	rawCiphers, ok := params["Ciphers"]
	if !ok {
		return nil, nil
	}

	return parseConfigEntrySlice(rawCiphers)
}

func (s *mqlSshdConfig) macs(params map[string]any) ([]any, error) {
	rawMacs, ok := params["MACs"]
	if !ok {
		return nil, nil
	}

	return parseConfigEntrySlice(rawMacs)
}

func (s *mqlSshdConfig) kexs(params map[string]any) ([]any, error) {
	rawkexs, ok := params["KexAlgorithms"]
	if !ok {
		return nil, nil
	}

	return parseConfigEntrySlice(rawkexs)
}

func (s *mqlSshdConfig) hostkeys(params map[string]any) ([]any, error) {
	rawHostKeys, ok := params["HostKey"]
	if !ok {
		return nil, nil
	}

	return parseConfigEntrySlice(rawHostKeys)
}

func (s *mqlSshdConfig) permitRootLogin(params map[string]any) ([]any, error) {
	rawHostKeys, ok := params["PermitRootLogin"]
	if !ok {
		return nil, nil
	}

	return parseConfigEntrySlice(rawHostKeys)
}

func (s *mqlSshdConfigMatchBlock) context() (*mqlFileContext, error) {
	return nil, errors.New("context was not provided for sshd.config match block")
}

func (s *mqlSshdConfigMatchBlock) ciphers(params map[string]any) ([]any, error) {
	rawCiphers, ok := params["Ciphers"]
	if !ok {
		return nil, nil
	}

	return parseConfigEntrySlice(rawCiphers)
}

func (s *mqlSshdConfigMatchBlock) macs(params map[string]any) ([]any, error) {
	rawMacs, ok := params["MACs"]
	if !ok {
		return nil, nil
	}

	return parseConfigEntrySlice(rawMacs)
}

func (s *mqlSshdConfigMatchBlock) kexs(params map[string]any) ([]any, error) {
	rawkexs, ok := params["KexAlgorithms"]
	if !ok {
		return nil, nil
	}

	return parseConfigEntrySlice(rawkexs)
}

func (s *mqlSshdConfigMatchBlock) hostkeys(params map[string]any) ([]any, error) {
	rawHostKeys, ok := params["HostKey"]
	if !ok {
		return nil, nil
	}

	return parseConfigEntrySlice(rawHostKeys)
}

func (s *mqlSshdConfigMatchBlock) permitRootLogin(params map[string]any) ([]any, error) {
	rawHostKeys, ok := params["PermitRootLogin"]
	if !ok {
		return nil, nil
	}

	return parseConfigEntrySlice(rawHostKeys)
}
