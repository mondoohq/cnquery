// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"fmt"
	"strings"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v10/llx"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v10/providers/os/resources/python"
	"go.mondoo.com/cnquery/v10/types"
)

func (k *mqlPythonPackage) id() (string, error) {
	return k.Id.Data, nil
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
		args["id"] = llx.StringData(path)
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

func (k *mqlPythonPackage) authorEmail() (string, error) {
	err := k.populateData()
	if err != nil {
		return "", err
	}
	return k.AuthorEmail.Data, nil
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

	if file.Data == nil || file.Data.Path.Data == "" {
		return fmt.Errorf("file path is empty")
	}

	ppd, err := python.ParseMIME(strings.NewReader(file.Data.Content.Data), file.Data.Path.Data)
	if err != nil {
		return fmt.Errorf("error parsing python package data: %s", err)
	}

	k.Name = plugin.TValue[string]{Data: ppd.Name, State: plugin.StateIsSet}
	k.Version = plugin.TValue[string]{Data: ppd.Version, State: plugin.StateIsSet}
	k.Author = plugin.TValue[string]{Data: ppd.Author, State: plugin.StateIsSet}
	k.AuthorEmail = plugin.TValue[string]{Data: ppd.AuthorEmail, State: plugin.StateIsSet}
	k.Summary = plugin.TValue[string]{Data: ppd.Summary, State: plugin.StateIsSet}
	k.License = plugin.TValue[string]{Data: ppd.License, State: plugin.StateIsSet}
	k.Dependencies = plugin.TValue[[]interface{}]{Data: convert.SliceAnyToInterface(ppd.Dependencies), State: plugin.StateIsSet}

	cpes := []interface{}{}
	for i := range ppd.Cpes {
		cpe, err := k.MqlRuntime.CreateSharedResource("cpe", map[string]*llx.RawData{
			"uri": llx.StringData(ppd.Cpes[i]),
		})
		if err != nil {
			return err
		}
		cpes = append(cpes, cpe)
	}

	k.Cpes = plugin.TValue[[]interface{}]{Data: cpes, State: plugin.StateIsSet}
	k.Purl = plugin.TValue[string]{Data: ppd.Purl, State: plugin.StateIsSet}
	return nil
}

func newMqlPythonPackage(runtime *plugin.Runtime, ppd python.PackageDetails, dependencies []interface{}) (plugin.Resource, error) {
	f, err := CreateResource(runtime, "file", map[string]*llx.RawData{
		"path": llx.StringData(ppd.File),
	})
	if err != nil {
		log.Error().Err(err).Msg("error while creating file resource for python package resource")
		return nil, err
	}

	cpes := []interface{}{}
	for i := range ppd.Cpes {
		cpe, err := runtime.CreateSharedResource("cpe", map[string]*llx.RawData{
			"uri": llx.StringData(ppd.Cpes[i]),
		})
		if err != nil {
			return nil, err
		}
		cpes = append(cpes, cpe)
	}

	r, err := CreateResource(runtime, "python.package", map[string]*llx.RawData{
		"id":           llx.StringData(ppd.File),
		"name":         llx.StringData(ppd.Name),
		"version":      llx.StringData(ppd.Version),
		"author":       llx.StringData(ppd.Author),
		"authorEmail":  llx.StringData(ppd.AuthorEmail),
		"summary":      llx.StringData(ppd.Summary),
		"license":      llx.StringData(ppd.License),
		"file":         llx.ResourceData(f, f.MqlName()),
		"dependencies": llx.ArrayData(dependencies, types.Any),
		"purl":         llx.StringData(ppd.Purl),
		"cpes":         llx.ArrayData(cpes, types.Resource("cpe")),
	})
	if err != nil {
		log.Error().AnErr("err", err).Msg("error while creating MQL resource")
		return nil, err
	}
	return r, nil
}
