// copyright: 2019, Dominik Richter and Christoph Hartmann
// author: Dominik Richter
// author: Christoph Hartmann

package resources

import (
	"errors"
	"strings"

	"go.mondoo.io/mondoo/lumi"
	"go.mondoo.io/mondoo/lumi/resources/sshd"
)

func (s *lumiSshd) id() (string, error) {
	return "sshd", nil
}

func (s *lumiSshdConfig) init(args *lumi.Args) (*lumi.Args, SshdConfig, error) {
	if x, ok := (*args)["path"]; ok {
		path, ok := x.(string)
		if !ok {
			return nil, nil, errors.New("Wrong type for 'path' in sshd.config initialization, it must be a string")
		}

		f, err := s.Runtime.CreateResource("file", "path", path)
		if err != nil {
			return nil, nil, err
		}
		(*args)["file"] = f
		delete(*args, "path")
	}

	return args, nil, nil
}

const defaultSshdConfig = "/etc/ssh/sshd_config"

func (s *lumiSshdConfig) id() (string, error) {
	r, err := s.File()
	if err != nil {
		return "", err
	}
	return r.Path()
}

func (s *lumiSshdConfig) GetFile() (File, error) {
	f, err := s.Runtime.CreateResource("file", "path", defaultSshdConfig)
	if err != nil {
		return nil, err
	}
	return f.(File), nil
}

func (s *lumiSshdConfig) GetContent(file File) (string, error) {
	// TODO: this can be heavily improved once we do it right, since this is constantly
	// re-registered as the file changes
	err := s.Runtime.WatchAndCompute(file, "content", s, "content")
	if err != nil {
		return "", err
	}

	return file.Content()
}

func (s *lumiSshdConfig) GetParams(content string) (map[string]interface{}, error) {
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

func (s *lumiSshdConfig) parseConfigEntrySlice(raw interface{}) ([]interface{}, error) {
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

func (s *lumiSshdConfig) GetCiphers(params map[string]interface{}) ([]interface{}, error) {
	rawCiphers, ok := params["Ciphers"]
	if !ok {
		return nil, nil
	}

	return s.parseConfigEntrySlice(rawCiphers)
}

func (s *lumiSshdConfig) GetMacs(params map[string]interface{}) ([]interface{}, error) {
	rawMacs, ok := params["MACs"]
	if !ok {
		return nil, nil
	}

	return s.parseConfigEntrySlice(rawMacs)
}

func (s *lumiSshdConfig) GetKexs(params map[string]interface{}) ([]interface{}, error) {
	rawkexs, ok := params["KexAlgorithms"]
	if !ok {
		return nil, nil
	}

	return s.parseConfigEntrySlice(rawkexs)
}

func (s *lumiSshdConfig) GetHostkeys(params map[string]interface{}) ([]interface{}, error) {
	rawHostKeys, ok := params["HostKey"]
	if !ok {
		return nil, nil
	}

	return s.parseConfigEntrySlice(rawHostKeys)
}
