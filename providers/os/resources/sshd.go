// copyright: 2019, Dominik Richter and Christoph Hartmann
// author: Dominik Richter
// author: Christoph Hartmann

package resources

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	"go.mondoo.com/cnquery/v10/llx"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v10/providers/os/connection/shared"
	"go.mondoo.com/cnquery/v10/providers/os/resources/sshd"
	"go.mondoo.com/cnquery/v10/types"
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

func (s *mqlSshdConfig) files(file *mqlFile) ([]interface{}, error) {
	if !file.GetExists().Data {
		return nil, errors.New("sshd config does not exist in " + file.GetPath().Data)
	}

	conn := s.MqlRuntime.Connection.(shared.Connection)
	allFiles, err := sshd.GetAllSshdIncludedFiles(file.Path.Data, conn)
	if err != nil {
		return nil, err
	}

	// Return a list of lumi files
	lumiFiles := make([]interface{}, len(allFiles))
	for i, path := range allFiles {
		lumiFile, err := CreateResource(s.MqlRuntime, "file", map[string]*llx.RawData{
			"path": llx.StringData(path),
		})
		if err != nil {
			return nil, err
		}

		lumiFiles[i] = lumiFile.(*mqlFile)
	}

	return lumiFiles, nil
}

func (s *mqlSshdConfig) content(files []interface{}) (string, error) {
	// TODO: this can be heavily improved once we do it right, since this is constantly
	// re-registered as the file changes

	// files is in the "dependency" order that files were discovered while
	// parsing the base/root config file. We will essentially re-parse the
	// config and insert the contents of those dependent files in-place where
	// they appear in the base/root config.
	if len(files) < 1 {
		return "", fmt.Errorf("no base sshd config file to read")
	}

	lumiFiles := make([]*mqlFile, len(files))
	for i, file := range files {
		lumiFile, ok := file.(*mqlFile)
		if !ok {
			return "", fmt.Errorf("failed to type assert list of files to File interface")
		}
		lumiFiles[i] = lumiFile
	}

	// The first entry in our list is the base/root of the sshd configuration tree
	baseConfigFilePath := lumiFiles[0].Path.Data

	conn := s.MqlRuntime.Connection.(shared.Connection)
	fullContent, err := sshd.GetSshdUnifiedContent(baseConfigFilePath, conn)
	if err != nil {
		return "", err
	}

	return fullContent, nil
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

func (s *mqlSshdConfig) parse(content string) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	params, err := sshd.ParseBlocks(content)
	if err != nil {
		s.Params = plugin.TValue[map[string]any]{Error: err, State: plugin.StateIsSet | plugin.StateIsNull}
		s.Blocks = plugin.TValue[[]any]{Error: err, State: plugin.StateIsSet | plugin.StateIsNull}
	} else {
		blocks, err := matchBlocks2Resources(params, s.MqlRuntime, s.__id)
		if err != nil {
			return err
		}
		s.Params = plugin.TValue[map[string]any]{Data: params.Flatten(), State: plugin.StateIsSet}
		s.Blocks = plugin.TValue[[]any]{Data: blocks, State: plugin.StateIsSet}
	}

	return err
}

func (s *mqlSshdConfig) params(content string) (map[string]any, error) {
	return nil, s.parse(content)
}

func (s *mqlSshdConfig) blocks(content string) ([]any, error) {
	return nil, s.parse(content)
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
