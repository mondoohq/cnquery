// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"io"
	"strconv"
	"strings"

	"go.mondoo.com/cnquery/v10/checksums"
	"go.mondoo.com/cnquery/v10/llx"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v10/providers/os/connection/shared"
	"go.mondoo.com/cnquery/v10/providers/os/resources/pam"
	"go.mondoo.com/cnquery/v10/types"
)

const (
	defaultPamConf = "/etc/pam.conf"
	defaultPamDir  = "/etc/pam.d"
)

func initPamConf(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if x, ok := args["path"]; ok {
		path, ok := x.Value.(string)
		if !ok {
			return nil, nil, errors.New("wrong type for 'path' it must be a string")
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

func (s *mqlPamConf) id() (string, error) {
	checksum := checksums.New
	for i := range s.Files.Data {
		path := s.Files.Data[i].(*mqlFile).Path.Data
		checksum = checksum.Add(path)
	}

	return checksum.String(), nil
}

func (se *mqlPamConfServiceEntry) id() (string, error) {
	ptype := se.PamType.Data
	mod := se.Module.Data
	s := se.Service.Data
	ln := se.LineNumber.Data
	lnstr := strconv.FormatInt(ln, 10)

	id := s + "/" + lnstr + "/" + ptype

	// for include mod is empty
	if mod != "" {
		id += "/" + mod
	}

	return id, nil
}

func (s *mqlPamConf) getFiles(confPath string) ([]interface{}, error) {
	// check if the pam.d directory or pam config file exists
	raw, err := CreateResource(s.MqlRuntime, "file", map[string]*llx.RawData{
		"path": llx.StringData(confPath),
	})
	if err != nil {
		return nil, err
	}
	f := raw.(*mqlFile)
	exists := f.GetExists()
	if exists.Error != nil {
		return nil, exists.Error
	}

	if !exists.Data {
		return nil, errors.New(" could not load pam configuration: " + confPath)
	}

	perm := f.GetPermissions()
	if perm.Error != nil {
		return nil, perm.Error
	}

	if perm.Data.IsDirectory.Data {
		return s.getConfDFiles(confPath)
	} else {
		return []interface{}{f}, nil
	}
}

func (s *mqlPamConf) getConfDFiles(confD string) ([]interface{}, error) {
	files, err := CreateResource(s.MqlRuntime, "files.find", map[string]*llx.RawData{
		"from": llx.StringData(confD),
		"type": llx.StringData("file"),
	})
	if err != nil {
		return nil, err
	}

	res := files.(*mqlFilesFind).GetList()
	return res.Data, res.Error
}

// GetFiles is called when the user has not provided a custom path. Otherwise files are set in the init
// method and this function is never called then since the data is already cached.
func (s *mqlPamConf) files() ([]interface{}, error) {
	// check if the pam.d directory exists and is a directory
	// according to the pam spec, pam prefers the directory if it  exists over the single file config
	// see http://www.linux-pam.org/Linux-PAM-html/sag-configuration.html
	raw, err := CreateResource(s.MqlRuntime, "file", map[string]*llx.RawData{
		"path": llx.StringData(defaultPamDir),
	})
	if err != nil {
		return nil, err
	}
	f := raw.(*mqlFile)
	exist := f.GetExists()
	if exist.Error != nil {
		return nil, exist.Error
	}

	if exist.Data {
		return s.getFiles(defaultPamDir)
	} else {
		return s.getFiles(defaultPamConf)
	}
}

func (s *mqlPamConf) content(files []interface{}) (string, error) {
	conn := s.MqlRuntime.Connection.(shared.Connection)

	var res strings.Builder
	var notReadyError error = nil

	for i := range files {
		file := files[i].(*mqlFile)

		f, err := conn.FileSystem().Open(file.Path.Data)
		if err != nil {
			return "", err
		}

		raw, err := io.ReadAll(f)
		f.Close()
		if err != nil {
			return "", err
		}

		res.WriteString(string(raw))
		res.WriteString("\n")
	}

	if notReadyError != nil {
		return "", notReadyError
	}

	return res.String(), nil
}

func (s *mqlPamConf) services(files []interface{}) (map[string]interface{}, error) {
	conn := s.MqlRuntime.Connection.(shared.Connection)

	contents := map[string]string{}
	var notReadyError error = nil

	for i := range files {
		file := files[i].(*mqlFile)

		f, err := conn.FileSystem().Open(file.Path.Data)
		if err != nil {
			return nil, err
		}

		raw, err := io.ReadAll(f)
		f.Close()
		if err != nil {
			return nil, err
		}

		contents[file.Path.Data] = string(raw)
	}

	if notReadyError != nil {
		return nil, notReadyError
	}

	services := map[string]interface{}{}
	for basename, content := range contents {
		lines := strings.Split(content, "\n")
		settings := []interface{}{}
		var line string
		for i := range lines {
			line = lines[i]

			if idx := strings.Index(line, "#"); idx >= 0 {
				line = line[0:idx]
			}
			line = strings.Trim(line, " \t\r")

			if line != "" {
				settings = append(settings, line)
			}
		}
		services[basename] = settings
	}

	return services, nil
}

func (s *mqlPamConf) entries(files []interface{}) (map[string]interface{}, error) {
	conn := s.MqlRuntime.Connection.(shared.Connection)

	contents := map[string]string{}
	var notReadyError error = nil

	for i := range files {
		file := files[i].(*mqlFile)

		f, err := conn.FileSystem().Open(file.Path.Data)
		if err != nil {
			return nil, err
		}

		raw, err := io.ReadAll(f)
		f.Close()
		if err != nil {
			return nil, err
		}

		contents[file.Path.Data] = string(raw)
	}

	if notReadyError != nil {
		return nil, notReadyError
	}

	services := map[string]interface{}{}
	for basename, content := range contents {
		lines := strings.Split(content, "\n")
		settings := []interface{}{}
		var line string
		for i := range lines {
			line = lines[i]

			entry, err := pam.ParseLine(line)
			if err != nil {
				return nil, err
			}

			// empty lines parse as empty object
			if entry == nil {
				continue
			}

			pamEntry, err := CreateResource(s.MqlRuntime, "pam.conf.serviceEntry", map[string]*llx.RawData{
				"service":    llx.StringData(basename),
				"lineNumber": llx.IntData(int64(i)), // Used for ID
				"pamType":    llx.StringData(entry.PamType),
				"control":    llx.StringData(entry.Control),
				"module":     llx.StringData(entry.Module),
				"options":    llx.ArrayData(entry.Options, types.String),
			})
			if err != nil {
				return nil, err
			}
			settings = append(settings, pamEntry.(*mqlPamConfServiceEntry))

		}

		services[basename] = settings
	}

	return services, nil
}
