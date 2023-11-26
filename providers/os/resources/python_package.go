// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"fmt"
	"go.mondoo.com/cnquery/v9/providers-sdk/v1/util/convert"

	"github.com/spf13/afero"
	"go.mondoo.com/cnquery/v9/llx"
	"go.mondoo.com/cnquery/v9/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v9/providers/os/connection/shared"
)

func (k *mqlPythonPackage) id() (string, error) {
	file := k.GetFile()
	if file.Error != nil {
		return "", file.Error
	}

	mqlFile := file.Data
	metadataPath := mqlFile.Path.Data
	return metadataPath, nil
}

func initPythonPackage(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 1 {
		return args, nil, nil
	}
	if x, ok := args["path"]; ok {
		path, ok := x.Value.(string)
		if !ok {
			return nil, nil, errors.New("Wrong type for 'path' in python.package initialization, it must be a string")
		}

		file, err := CreateResource(runtime, "file", map[string]*llx.RawData{
			"path": llx.StringData(path),
		})
		if err != nil {
			return nil, nil, err
		}
		args["file"] = llx.ResourceData(file, "file")

		delete(args, "path")
	}
	return args, nil, nil
}

func (k *mqlPythonPackage) name() (string, error) {
	err := k.populateData()
	if err != nil {
		return "", err
	}
	return k.Name.Data, nil
}

func (k *mqlPythonPackage) version() (string, error) {
	err := k.populateData()
	if err != nil {
		return "", err
	}
	return k.Version.Data, nil
}

func (k *mqlPythonPackage) license() (string, error) {
	err := k.populateData()
	if err != nil {
		return "", err
	}
	return k.License.Data, nil
}

func (k *mqlPythonPackage) author() (string, error) {
	err := k.populateData()
	if err != nil {
		return "", err
	}
	return k.Author.Data, nil
}

func (k *mqlPythonPackage) summary() (string, error) {
	err := k.populateData()
	if err != nil {
		return "", err
	}
	return k.Summary.Data, nil
}

func (k *mqlPythonPackage) purl() (string, error) {
	err := k.populateData()
	if err != nil {
		return "", err
	}
	return k.Purl.Data, nil
}

func (k *mqlPythonPackage) cpes() ([]interface{}, error) {
	err := k.populateData()
	if err != nil {
		return nil, err
	}
	return k.Cpes.Data, nil
}

func (k *mqlPythonPackage) dependencies() ([]interface{}, error) {
	err := k.populateData()
	if err != nil {
		return nil, err
	}
	return k.Dependencies.Data, nil
}

func (k *mqlPythonPackage) populateData() error {
	file := k.GetFile()
	if file.Error != nil {
		return file.Error
	}
	mqlFile := file.Data
	conn := k.MqlRuntime.Connection.(shared.Connection)
	afs := &afero.Afero{Fs: conn.FileSystem()}
	metadataPath := mqlFile.Path.Data
	ppd, err := parseMIME(afs, metadataPath)
	if err != nil {
		return fmt.Errorf("error parsing python package data: %s", err)
	}

	k.Name = plugin.TValue[string]{Data: ppd.name, State: plugin.StateIsSet}
	k.Version = plugin.TValue[string]{Data: ppd.version, State: plugin.StateIsSet}
	k.Author = plugin.TValue[string]{Data: ppd.author, State: plugin.StateIsSet}
	k.Summary = plugin.TValue[string]{Data: ppd.summary, State: plugin.StateIsSet}
	k.License = plugin.TValue[string]{Data: ppd.license, State: plugin.StateIsSet}
	k.Dependencies = plugin.TValue[[]interface{}]{Data: convert.SliceAnyToInterface(ppd.dependencies), State: plugin.StateIsSet}

	cpes := []interface{}{}
	for i := range ppd.cpes {
		cpe, err := k.MqlRuntime.CreateSharedResource("cpe", map[string]*llx.RawData{
			"uri": llx.StringData(k.Cpes.Data[i].(string)),
		})
		if err != nil {
			return err
		}
		cpes = append(cpes, cpe)
	}

	k.Cpes = plugin.TValue[[]interface{}]{Data: cpes, State: plugin.StateIsSet}
	k.Purl = plugin.TValue[string]{Data: ppd.purl, State: plugin.StateIsSet}

	return nil
}
