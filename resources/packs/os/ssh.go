// copyright: 2019, Dominik Richter and Christoph Hartmann
// author: Dominik Richter
// author: Christoph Hartmann

package os

import (
	"errors"
	"fmt"
	"strings"

	"go.mondoo.io/mondoo/resources"
	"go.mondoo.io/mondoo/resources/packs/core"
	"go.mondoo.io/mondoo/resources/packs/os/sshd"
)

func (s *mqlSshd) id() (string, error) {
	return "sshd", nil
}

func (s *mqlSshdConfig) init(args *resources.Args) (*resources.Args, SshdConfig, error) {
	if x, ok := (*args)["path"]; ok {
		path, ok := x.(string)
		if !ok {
			return nil, nil, errors.New("Wrong type for 'path' in sshd.config initialization, it must be a string")
		}

		f, err := s.MotorRuntime.CreateResource("file", "path", path)
		if err != nil {
			return nil, nil, err
		}
		(*args)["file"] = f

		files, err := s.getFiles(path)
		if err != nil {
			return nil, nil, err
		}
		(*args)["files"] = files
		delete(*args, "path")
	}

	return args, nil, nil
}

const defaultSshdConfig = "/etc/ssh/sshd_config"

func (s *mqlSshdConfig) id() (string, error) {
	r, err := s.File()
	if err != nil {
		return "", err
	}
	return r.Path()
}

func (s *mqlSshdConfig) getFiles(confPath string) ([]interface{}, error) {
	lumiFile, err := s.MotorRuntime.CreateResource("file", "path", confPath)
	if err != nil {
		return nil, err
	}
	f := lumiFile.(core.File)
	exists, err := f.Exists()
	if err != nil {
		return nil, err
	}

	if !exists {
		return nil, errors.New("could not load sshd configuration: " + confPath)
	}

	osProv, err := osProvider(s.MotorRuntime.Motor)
	if err != nil {
		return nil, err
	}

	// Get the list of all files involved in defining the runtime sshd configuration
	allFiles, err := sshd.GetAllSshdIncludedFiles(confPath, osProv)
	if err != nil {
		return nil, err
	}

	// Return a list of lumi files
	lumiFiles := make([]interface{}, len(allFiles))
	for i, v := range allFiles {

		lumiFile, err := s.MotorRuntime.CreateResource("file", "path", v)
		if err != nil {
			return nil, err
		}

		lumiFiles[i] = lumiFile.(core.File)
	}

	return lumiFiles, nil
}

func (s *mqlSshdConfig) GetFile() (core.File, error) {
	f, err := s.MotorRuntime.CreateResource("file", "path", defaultSshdConfig)
	if err != nil {
		return nil, err
	}
	return f.(core.File), nil
}

func (s *mqlSshdConfig) GetFiles() ([]interface{}, error) {
	lumiFile, err := s.MotorRuntime.CreateResource("file", "path", defaultSshdConfig)
	if err != nil {
		return nil, err
	}
	f := lumiFile.(core.File)
	exists, err := f.Exists()
	if err != nil {
		return nil, err
	}

	if !exists {
		return nil, errors.New(fmt.Sprintf(" could not read sshd config file %s", defaultSshdConfig))
	}
	files, err := s.getFiles(defaultSshdConfig)
	if err != nil {
		return nil, err
	}

	return files, nil
}

func (s *mqlSshdConfig) GetContent(files []interface{}) (string, error) {
	// TODO: this can be heavily improved once we do it right, since this is constantly
	// re-registered as the file changes

	// files is in the "dependency" order that files were discovered while
	// parsing the base/root config file. We will essentially re-parse the
	// config and insert the contents of those dependent files in-place where
	// they appear in the base/root config.
	if len(files) < 1 {
		return "", fmt.Errorf("no base sshd config file to read")
	}

	lumiFiles := make([]core.File, len(files))
	for i, file := range files {
		lumiFile, ok := file.(core.File)
		if !ok {
			return "", fmt.Errorf("failed to type assert list of files to File interface")
		}
		lumiFiles[i] = lumiFile
	}

	// The first entry in our list is the base/root of the sshd configuration tree
	baseConfigFilePath, err := lumiFiles[0].Path()
	if err != nil {
		return "", err
	}

	osProv, err := osProvider(s.MotorRuntime.Motor)
	if err != nil {
		return "", err
	}

	fullContent, err := sshd.GetSshdUnifiedContent(baseConfigFilePath, osProv)
	if err != nil {
		return "", err
	}

	return fullContent, nil
}

func (s *mqlSshdConfig) GetParams(content string) (map[string]interface{}, error) {
	params, err := sshd.Params(content)
	if err != nil {
		return nil, err
	}

	// convert  map
	res := map[string]interface{}{}
	for k, v := range params {
		res[k] = v
	}

	return res, nil
}

func (s *mqlSshdConfig) parseConfigEntrySlice(raw interface{}) ([]interface{}, error) {
	strCipher, ok := raw.(string)
	if !ok {
		return nil, errors.New("value is not a valid string")
	}

	res := []interface{}{}
	entries := strings.Split(strCipher, ",")
	for i := range entries {
		val := strings.TrimSpace(entries[i])
		res = append(res, val)
	}

	return res, nil
}

func (s *mqlSshdConfig) GetCiphers(params map[string]interface{}) ([]interface{}, error) {
	rawCiphers, ok := params["Ciphers"]
	if !ok {
		return nil, nil
	}

	return s.parseConfigEntrySlice(rawCiphers)
}

func (s *mqlSshdConfig) GetMacs(params map[string]interface{}) ([]interface{}, error) {
	rawMacs, ok := params["MACs"]
	if !ok {
		return nil, nil
	}

	return s.parseConfigEntrySlice(rawMacs)
}

func (s *mqlSshdConfig) GetKexs(params map[string]interface{}) ([]interface{}, error) {
	rawkexs, ok := params["KexAlgorithms"]
	if !ok {
		return nil, nil
	}

	return s.parseConfigEntrySlice(rawkexs)
}

func (s *mqlSshdConfig) GetHostkeys(params map[string]interface{}) ([]interface{}, error) {
	rawHostKeys, ok := params["HostKey"]
	if !ok {
		return nil, nil
	}

	return s.parseConfigEntrySlice(rawHostKeys)
}
