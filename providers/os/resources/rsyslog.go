// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"strings"

	"go.mondoo.com/cnquery/v11/checksums"
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/resources"
)

const defaultRsyslogConf = "/etc/rsyslog.conf"

func (s *mqlRsyslogConf) id() (string, error) {
	files := s.GetFiles()
	if files.Error != nil {
		return "", files.Error
	}

	checksum := checksums.New
	for i := range files.Data {
		path := files.Data[i].(*mqlFile).Path.Data
		checksum = checksum.Add(path)
	}

	return checksum.String(), nil
}

func (s *mqlRsyslogConf) path() (string, error) {
	return defaultRsyslogConf, nil
}

func (s *mqlRsyslogConf) files(path string) ([]interface{}, error) {
	if !strings.HasSuffix(path, ".conf") {
		return nil, errors.New("failed to initialize, path must end in `.conf` so we can find files in `.d` directory")
	}

	f, err := CreateResource(s.MqlRuntime, "file", map[string]*llx.RawData{
		"path": llx.StringData(path),
	})
	if err != nil {
		return nil, err
	}

	confD := path[0:len(path)-5] + ".d"
	o, err := CreateResource(s.MqlRuntime, "files.find", map[string]*llx.RawData{
		"from": llx.StringData(confD),
		"type": llx.StringData("file"),
	})
	if err != nil {
		return nil, err
	}

	list := o.(*mqlFilesFind).GetList()
	if list.Error != nil {
		return nil, list.Error
	}

	return append([]interface{}{f.(*mqlFile)}, list.Data...), nil
}

func (s *mqlRsyslogConf) content(files []interface{}) (string, error) {
	var res strings.Builder

	// TODO: this can be heavily improved once we do it right, since this is constantly
	// re-registered as the file changes
	for i := range files {
		file := files[i].(*mqlFile)
		content := file.GetContent()
		if content.Error != nil {
			if errors.Is(content.Error, resources.NotFoundError{}) {
				continue
			}
		}

		res.WriteString(content.Data)
		res.WriteString("\n")
	}

	return res.String(), nil
}

func (s *mqlRsyslogConf) settings(content string) ([]interface{}, error) {
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

	return settings, nil
}
