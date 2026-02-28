// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"strings"

	"go.mondoo.com/mql/v13/checksums"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/resources"
	"go.mondoo.com/mql/v13/providers/os/connection/shared"
)

// rsyslogConfPaths maps platform names to their rsyslog.conf location.
// BSD variants install rsyslog via package managers to non-default prefixes.
var rsyslogConfPaths = map[string]string{
	"freebsd":      "/usr/local/etc/rsyslog.conf",
	"dragonflybsd": "/usr/local/etc/rsyslog.conf",
	"openbsd":      "/usr/local/etc/rsyslog.conf",
	"netbsd":       "/usr/pkg/etc/rsyslog.conf",
}

func rsyslogConfPath(conn shared.Connection) string {
	asset := conn.Asset()
	if asset != nil && asset.Platform != nil {
		if p, ok := rsyslogConfPaths[asset.Platform.Name]; ok {
			return p
		}
	}
	return "/etc/rsyslog.conf"
}

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
	conn := s.MqlRuntime.Connection.(shared.Connection)
	return rsyslogConfPath(conn), nil
}

func (s *mqlRsyslogConf) files(path string) ([]any, error) {
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

	return append([]any{f.(*mqlFile)}, list.Data...), nil
}

func (s *mqlRsyslogConf) content(files []any) (string, error) {
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

func (s *mqlRsyslogConf) settings(content string) ([]any, error) {
	lines := strings.Split(content, "\n")

	settings := []any{}
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
