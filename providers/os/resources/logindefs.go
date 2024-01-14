// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"strings"

	"go.mondoo.com/cnquery/v10/llx"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v10/providers/os/resources/logindefs"
)

func initLogindefs(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if x, ok := args["path"]; ok {
		path, ok := x.Value.(string)
		if !ok {
			return nil, nil, errors.New("Wrong type for 'path' in logindefs initialization, it must be a string")
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

const defaultLoginDefsConfig = "/etc/login.defs"

func (s *mqlLogindefs) id() (string, error) {
	file := s.GetFile()
	if file.Data == nil {
		return "", errors.New("no file for logindefs")
	}
	return file.Data.Path.Data, nil
}

func (s *mqlLogindefs) file() (*mqlFile, error) {
	f, err := CreateResource(s.MqlRuntime, "file", map[string]*llx.RawData{
		"path": llx.StringData(defaultLoginDefsConfig),
	})
	if err != nil {
		return nil, err
	}
	return f.(*mqlFile), nil
}

// borrowed from ssh resource
func (s *mqlLogindefs) content(file *mqlFile) (string, error) {
	c := file.GetContent()
	return c.Data, c.Error
}

func (s *mqlLogindefs) params(content string) (map[string]interface{}, error) {
	res := make(map[string]interface{})

	params := logindefs.Parse(strings.NewReader(content))

	for k, v := range params {
		res[k] = v
	}

	return res, nil
}
